package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	bloqs_http "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"
	p "google.golang.org/protobuf/proto"

	_ "github.com/joho/godotenv/autoload"
)

func logRoute(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		v      *proto.TokenValidation
		status uint32

		a proto.AuthServer
	)

	types_route := conf.MustGetConfOrDefault("/types", "auth", "typesPath")

	h := w.Header()
	status, err = helpers.CheckOriginHeader(&h, r)

	switch r.Method {
	case http.MethodPost:
		if err != nil {
			v = &proto.TokenValidation{
				Validation: bloqs_auth.ErrorToValidation(err, &status),
			}
			goto respond
		}

		var ask *proto.AskPermissions

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_http.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_http.X_WWW_FORM_URLENCODED, err), &status),
				}
				goto respond
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_http.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = &proto.TokenValidation{
					Validation: bloqs_auth.Invalid(fmt.Sprintf("could not read the HTTP request body:\t %s", err), &status),
				}
				goto respond
			} else {
				ask = new(proto.AskPermissions)
				if err := p.Unmarshal(buf, ask); err != nil {
					status = http.StatusBadRequest
					v = &proto.TokenValidation{
						Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_http.GRPC, err), &status),
					}
					goto respond
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			status = http.StatusUnsupportedMediaType
			bloqs_http.Append(&h, "Accept", bloqs_http.X_WWW_FORM_URLENCODED)
			bloqs_http.Append(&h, "Accept", bloqs_http.GRPC)
			v = &proto.TokenValidation{
				Validation: bloqs_auth.Invalid(fmt.Sprintf("request has the usupported media type `%s`", ct), &status),
			}
			goto respond
		}

		t := bloqs_http.GetQuery()
		if !r.URL.Query().Has(t) {
			status = http.StatusBadRequest
			v = &proto.TokenValidation{
				Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` that specifies the method to use for authentication/authorization was not defined. Define it with one of the supported values (.%s).\n", t, types_route), &status),
			}
			goto respond
		}
		perm := conf.MustGetConfOrDefault("permissions", "auth", "permissionsQueryParam")

		method := r.URL.Query().Get(t)
		permissions := auth.DEFAULT_PERMISSIONS
		if r.URL.Query().Has(perm) {
			ps, ok := r.URL.Query()[perm]
			if ok {
				permissions = bloqs_auth.NIL
				for _, i := range ps {
					p, ok := auth.Permissions[i]
					if !ok {
						status = http.StatusBadRequest
						v = &proto.TokenValidation{
							Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` that specifies the permissions for the token to have was has an invalid value. Check which values are supported (.%s).\n", perm, "TODO"), &status),
						}
						goto respond
					}
					permissions |= p
				}
			}
		}

		switch method {
		case "basic":
			if !bloqs_auth.IsAuthMethodSupported(method) {
				status = http.StatusUnprocessableEntity
				v = &proto.TokenValidation{
					Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` value `%s` it's unsupported. Define it with one of the supported values (.%s).\n", t, method, types_route), &status),
				}
				goto respond
			}

			if ask == nil {
				email := r.FormValue("email")

				if email == "" {
					status = http.StatusUnprocessableEntity
					v = &proto.TokenValidation{
						Validation: bloqs_auth.Invalid("`email` body field is empty and needs to be defined to proceed.\n", &status),
					}
					goto respond
				}

				pass := r.FormValue("pass")

				if pass == "" {
					status = http.StatusUnprocessableEntity
					v = &proto.TokenValidation{
						Validation: bloqs_auth.Invalid("`pass` body field is empty and needs to be defined to proceed.\n", &status),
					}
					goto respond
				}

				ask = &proto.AskPermissions{
					Credentials: &proto.Credentials{
						Credentials: &proto.Credentials_Basic{
							Basic: &proto.Credentials_BasicCredentials{
								Email:    email,
								Password: pass,
							},
						},
					},
					Permissions: uint64(permissions),
				}
			}

			a, err = authSrv(r.Context())
			if err != nil {
				status = http.StatusInternalServerError
				v = &proto.TokenValidation{
					Validation: bloqs_auth.ErrorToValidation(err, &status),
				}
				goto respond
			}

			v, err = a.LogIn(r.Context(), ask)
			goto respond
		default:
			status = http.StatusBadRequest
			v = &proto.TokenValidation{
				Validation: bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` has an unsupported value. Define it with one of the supported values (.%s).\n", t, types_route), &status),
			}
			goto respond
		}
	case http.MethodDelete:
		if err != nil {
			v = &proto.TokenValidation{
				Validation: bloqs_auth.Invalid("", &status),
			}
			goto respond
		}
	case http.MethodOptions:
		bloqs_http.Append(&h, "Access-Control-Allow-Methods", http.MethodPost)
		bloqs_http.Append(&h, "Access-Control-Allow-Methods", http.MethodDelete)
		bloqs_http.Append(&h, "Access-Control-Allow-Methods", http.MethodOptions)
		h.Set("Access-Control-Allow-Credentials", "true")
		//bloqs_http.Append(&h, "Access-Control-Allow-Headers", "")
		//bloqs_http.Append(&h, "Access-Control-Expose-Headers", "")
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
			Validation: bloqs_auth.Invalid("", &status),
		}
		goto respond
	}

respond:
	if v := v.Validation; v != nil {
		if code := v.HttpStatusCode; code != nil {
			status = *code
			v.HttpStatusCode = nil
		} else {
			if err != nil {
				status = http.StatusInternalServerError
			} else {
				if v.Valid {
					status = http.StatusOK
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
