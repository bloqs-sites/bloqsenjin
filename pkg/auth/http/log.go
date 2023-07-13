package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	bloqs_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/proto"
	p "google.golang.org/protobuf/proto"

	_ "github.com/joho/godotenv/autoload"
)

func LogRoute(w http.ResponseWriter, r *http.Request, segs []string) {
	var (
		err    error
		v      *proto.TokenValidation
		status uint32

		a proto.AuthServer
	)

	h := w.Header()
	status, err = helpers.CheckOriginHeader(&h, r, true)

	types_route := conf.MustGetConfOrDefault("/types/", "auth", "paths", "types")

	switch r.Method {
	case http.MethodPost: // log in
		if err != nil {
			v = &proto.TokenValidation{
				Validation: auth.ErrorToValidation(err, &status),
			}
			goto respond
		}

		var ask *proto.AskPermissions
		var creds *proto.Credentials

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_helpers.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.X_WWW_FORM_URLENCODED, err), &status),
				}
				goto respond
			}
		} else if strings.HasPrefix(ct, bloqs_helpers.FORM_DATA) {
			if err = r.ParseMultipartForm(32 << 20); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.FORM_DATA, err), &status),
				}
				goto respond
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_helpers.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("could not read the HTTP request body:\t %s", err), &status),
				}
				goto respond
			} else {
				ask = new(proto.AskPermissions)
				if err := p.Unmarshal(buf, ask); err != nil {
					status = http.StatusBadRequest
					v = &proto.TokenValidation{
						Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.GRPC, err), &status),
					}
					goto respond
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			status = http.StatusUnsupportedMediaType
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.X_WWW_FORM_URLENCODED)
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.FORM_DATA)
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.GRPC)
			v = &proto.TokenValidation{
				Validation: auth.Invalid(fmt.Sprintf("request has the usupported media type `%s`", ct), &status),
			}
			goto respond
		}

		t := conf.MustGetConfOrDefault("type", "auth", "queryParams", "type")
		if !r.URL.Query().Has(t) {
			status = http.StatusBadRequest
			v = &proto.TokenValidation{
				Validation: auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` that specifies the method to use for authentication/authorization was not defined. Define it with one of the supported values (.%s).\n", t, types_route), &status),
			}
			goto respond
		}

		method := r.URL.Query().Get(t)

		switch method {
		case "basic":
			if !auth.IsAuthMethodSupported(method) {
				status = http.StatusUnprocessableEntity
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` value `%s` it's unsupported. Define it with one of the supported values (.%s).\n", t, method, types_route), &status),
				}
				goto respond
			}

			if ask == nil {
				email := r.FormValue("email")

				if email == "" {
					status = http.StatusUnprocessableEntity
					v = &proto.TokenValidation{
						Validation: auth.Invalid("`email` body field is empty and needs to be defined to proceed.\n", &status),
					}
					goto respond
				}

				pass := r.FormValue("pass")

				if pass == "" {
					status = http.StatusUnprocessableEntity
					v = &proto.TokenValidation{
						Validation: auth.Invalid("`pass` body field is empty and needs to be defined to proceed.\n", &status),
					}
					goto respond
				}

				creds = &proto.Credentials{
					Credentials: &proto.Credentials_Basic{
						Basic: &proto.Credentials_BasicCredentials{
							Email:    email,
							Password: pass,
						},
					},
				}
			}
		default:
			status = http.StatusBadRequest
			v = &proto.TokenValidation{
				Validation: auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` has an unsupported value. Define it with one of the supported values (.%s).\n", t, types_route), &status),
			}
			goto respond
		}

		a, err = authSrv(r.Context())
		if err != nil {
			status = http.StatusInternalServerError
			v = &proto.TokenValidation{
				Validation: auth.ErrorToValidation(err, &status),
			}
			goto respond
		}

		var validation *proto.Validation

		if ask != nil && creds == nil {
			creds = ask.Credentials
		}

		validation, err = a.IsSuper(r.Context(), creds)

		if ask == nil {
			perm := conf.MustGetConfOrDefault("permissions", "auth", "queryParams", "permissions")
			permissions := auth.DEFAULT_PERMISSIONS
			list := auth.GetPermissionsList(validation.Valid)
			if r.URL.Query().Has(perm) {
				ps, ok := r.URL.Query()[perm]
				if ok {
					permissions = auth.NIL
					for _, i := range ps {
						p, ok := list[i]
						if !ok {
							status = http.StatusBadRequest
							v = &proto.TokenValidation{
								Validation: auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` that specifies the permissions for the token to have was has an invalid value. Check which values are supported (.%s).\n", perm, "TODO"), &status),
							}
							goto respond
						}
						permissions |= p
					}
				}
			}

			ask = &proto.AskPermissions{
				Credentials: creds,
				Permissions: uint64(permissions),
			}
		}

		v, err = a.LogIn(r.Context(), ask)
		if err == nil {
			bloqs_helpers.SetToken(w, r, v.Token.Jwt)
		}

		goto respond
	case http.MethodDelete: // log out
		if err != nil {
			v = &proto.TokenValidation{
				Validation: auth.ErrorToValidation(err, &status),
			}
			goto respond
		}

		var tk *proto.Token

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_helpers.PLAIN) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("could not read the HTTP request body:\t %s", err), &status),
				}
				goto respond
			} else {
				tk = &proto.Token{
					Jwt: string(buf),
				}
			}
		} else if strings.HasPrefix(ct, bloqs_helpers.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.X_WWW_FORM_URLENCODED, err), &status),
				}
				goto respond
			}

			tk = &proto.Token{
				Jwt: r.FormValue("token"),
			}
		} else if strings.HasPrefix(ct, bloqs_helpers.FORM_DATA) {
			if err = r.ParseMultipartForm(32 << 20); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.FORM_DATA, err), &status),
				}
				goto respond
			}

			tk = &proto.Token{
				Jwt: r.FormValue("token"),
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_helpers.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: auth.Invalid(fmt.Sprintf("could not read the HTTP request body:\t %s", err), &status),
				}
				goto respond
			} else {
				tk = new(proto.Token)
				if err := p.Unmarshal(buf, tk); err != nil {
					status = http.StatusBadRequest
					v = &proto.TokenValidation{
						Validation: auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.GRPC, err), &status),
					}
					goto respond
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			status = http.StatusUnsupportedMediaType
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.PLAIN)
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.X_WWW_FORM_URLENCODED)
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.FORM_DATA)
			bloqs_helpers.Append(&h, "Accept", bloqs_helpers.GRPC)
			v = &proto.TokenValidation{
				Validation: auth.Invalid(fmt.Sprintf("request has the usupported media type `%s`", ct), &status),
			}
			goto respond
		}

		if (tk == nil) || len(tk.Jwt) <= 0 {
			var jwt []byte
			jwt, err = bloqs_helpers.ExtractToken(w, r)

			if err != nil {
				status = http.StatusInternalServerError
				if err, ok := err.(*mux.HttpError); ok {
					status = uint32(err.Status)
				}

				v = &proto.TokenValidation{
					Validation: auth.ErrorToValidation(err, &status),
				}

				goto respond
			}

			tk = &proto.Token{
				Jwt: string(jwt),
			}
		}

		a, err = authSrv(r.Context())
		if err != nil {
			status = http.StatusInternalServerError
			v = &proto.TokenValidation{
				Validation: auth.ErrorToValidation(err, &status),
			}
			goto respond
		}

		var valid *proto.Validation
		valid, err = a.LogOut(r.Context(), tk)
		v = &proto.TokenValidation{
			Validation: valid,
		}

		goto respond
	case http.MethodOptions:
		bloqs_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodPost)
		bloqs_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodDelete)
		bloqs_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodOptions)
		h.Set("Access-Control-Allow-Credentials", "true")
		//bloqs_helpers.Append(&h, "Access-Control-Allow-Headers", "")
		//bloqs_helpers.Append(&h, "Access-Control-Expose-Headers", "")
		//h.Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		h.Set("Access-Control-Max-Age", "0")
		var msg string
		if err != nil {
			msg = err.Error()
		}
		v = &proto.TokenValidation{
			Validation: &proto.Validation{
				Valid:          err == nil,
				Message:        &msg,
				HttpStatusCode: &status,
			},
		}
		goto respond
	default:
		status = http.StatusMethodNotAllowed
		v = &proto.TokenValidation{
			Validation: auth.Invalid("", &status),
		}
		goto respond
	}

respond:
	see_other := redirect(r)

	if valid := v.Validation; valid != nil {
		if code := valid.HttpStatusCode; code != nil {
			status = *code
			valid.HttpStatusCode = nil

			if (status >= 200) && (status < 300) && (see_other != nil) {
				status = 303
				w.Header().Set("Location", *see_other)
			}
		} else {
			if err != nil {
				status = http.StatusInternalServerError
			} else {
				if valid.Valid {
					status = http.StatusOK

					if see_other != nil {
						status = 303
						w.Header().Set("Location", *see_other)
					}
				} else {
					status = http.StatusInternalServerError
				}
			}
		}

		w.WriteHeader(int(status))

		if status != http.StatusNoContent {
			json.NewEncoder(w).Encode(v)
			h.Set("Content-Type", "application/json")
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func redirect(r *http.Request) *string {
	redirect := conf.MustGetConfOrDefault("redirect", "auth", "queryParams", "redirect")
	location, err := url.Parse(r.URL.Query().Get(redirect))

	if err != nil || r.URL.Query().Get(redirect) == "" {
		return nil
	} else {
		if location.Hostname() == "" {
			if origin, err_origin := url.Parse(r.Header.Get("Origin")); err_origin == nil {
				location.Host = origin.Host
			}
		}

		str := location.String()
		return &str
	}
}
