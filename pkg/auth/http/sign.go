package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	bloqs_http "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/redis/go-redis/v9"
	p "google.golang.org/protobuf/proto"

	_ "github.com/joho/godotenv/autoload"
)

//var (
//	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")
//)

func signRoute(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		v      *proto.Validation
		status uint32

		a proto.AuthServer
	)

	types_route := conf.MustGetConfOrDefault("/types", "auth", "typesPath")

	h := w.Header()
	status, err = helpers.CheckOriginHeader(&h, r)

	switch r.Method {
	case http.MethodPost:
		if err != nil {
			v = bloqs_auth.ErrorToValidation(err, &status)
			goto respond
		}

		var credentials *proto.Credentials

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_http.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				status = http.StatusBadRequest
				v = bloqs_auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_http.X_WWW_FORM_URLENCODED, err), &status)
				goto respond
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_http.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = bloqs_auth.Invalid(fmt.Sprintf("could not read the HTTP request body:\t %s", err), &status)
				goto respond
			} else {
				credentials = new(proto.Credentials)
				if err := p.Unmarshal(buf, credentials); err != nil {
					status = http.StatusBadRequest
					v = bloqs_auth.Invalid(fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_http.GRPC, err), &status)
					goto respond
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			status = http.StatusUnsupportedMediaType
			bloqs_http.Append(&h, "Accept", bloqs_http.X_WWW_FORM_URLENCODED)
			bloqs_http.Append(&h, "Accept", bloqs_http.GRPC)
			v = bloqs_auth.Invalid(fmt.Sprintf("request has the usupported media type `%s`", ct), &status)
			goto respond
		}

		t := bloqs_http.GetQuery()
		if !r.URL.Query().Has(t) {
			status = http.StatusBadRequest
			v = bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` that specifies the method to use for authentication/authorization was not defined. Define it with one of the supported values (.%s).\n", t, types_route), &status)
			goto respond
		}

		method := r.URL.Query().Get(t)
		switch method {
		case "basic":
			if !bloqs_auth.IsAuthMethodSupported(method) {
				status = http.StatusUnprocessableEntity
				v = bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` value `%s` it's unsupported. Define it with one of the supported values (.%s).\n", t, method, types_route), &status)
				goto respond
			}

			if credentials == nil {
				email := r.FormValue("email")

				if email == "" {
					status = http.StatusUnprocessableEntity
					v = bloqs_auth.Invalid("`email` body field is empty and needs to be defined to proceed.\n", &status)
					goto respond
				}

				pass := r.FormValue("pass")

				if pass == "" {
					status = http.StatusUnprocessableEntity
					v = bloqs_auth.Invalid("`pass` body field is empty and needs to be defined to proceed.\n", &status)
					goto respond
				}

				credentials = &proto.Credentials{
					Credentials: &proto.Credentials_Basic{
						Basic: &proto.Credentials_BasicCredentials{
							Email:    email,
							Password: pass,
						},
					},
				}
			}

			a, err = authSrv(r.Context())
			if err != nil {
				status = http.StatusInternalServerError
				v = bloqs_auth.ErrorToValidation(err, &status)
				goto respond
			}

			v, err = a.SignIn(r.Context(), credentials)
			goto respond
		default:
			status = http.StatusBadRequest
			v = bloqs_auth.Invalid(fmt.Sprintf("the HTTP query parameter `%s` has an unsupported value. Define it with one of the supported values (.%s).\n", t, types_route), &status)
			goto respond
		}
	case http.MethodDelete:
		if err != nil {
			v = bloqs_auth.Invalid("", &status)
			goto respond
		}

		var token *proto.Token

		a, err = authSrv(r.Context())
		if err != nil {
			status = http.StatusInternalServerError
			v = bloqs_auth.ErrorToValidation(err, &status)
			goto respond
		}

		jwt, revoke := bloqs_http.ExtractToken(w, r)
		token = &proto.Token{
			Jwt: string(jwt),
		}

		if revoke {
			v, err = a.Revoke(r.Context(), token)
			status = http.StatusUnauthorized
			if v.HttpStatusCode == nil {
				v.HttpStatusCode = &status
			}
			goto respond
		}

		if jwt != nil {
			status = http.StatusUnauthorized
			v = bloqs_auth.Invalid("", &status)
			goto respond
		}

		switch r.URL.Query().Get(bloqs_http.GetQuery()) {
		case "basic":
			v, err = a.SignOut(r.Context(), token)
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
		v = &proto.Validation{
			Valid:          err == nil,
			Message:        &msg,
			HttpStatusCode: &status,
		}
		goto respond
	default:
		status = http.StatusMethodNotAllowed
		v = bloqs_auth.Invalid("", &status)
		goto respond
	}

respond:
	if v != nil {
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

//func createGRPCClient() (proto.AuthClient, func(), error) {
//	conn, err := grpc.Dial(net.JoinHostPort("localhost", fmt.Sprint(*gRPCPort)), grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		return nil, nil, err
//	}
//	return proto.NewAuthClient(conn), func() {
//		conn.Close()
//	}, nil
//}

func authSrv(ctx context.Context) (proto.AuthServer, error) {
	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	creds, err := db.NewMySQL(ctx, os.Getenv("BLOQS_AUTH_MYSQL_DSN"))
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", creds, err)
	}

	opt, err := redis.ParseURL(os.Getenv("BLOQS_TOKENS_REDIS_DSN"))
	if err != nil {
		return nil, fmt.Errorf("could not parse the `BLOQS_TOKENS_REDIS_DSN` to create the credentials to connect to the DB:\t%s", err)
	}

	a, err := auth.NewBloqsAuther(ctx, creds)
	if err != nil {
		return nil, err
	}

	secrets, err := db.NewKeyDB(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", secrets, err)
	}

	t := auth.NewBloqsTokener(secrets)

	return bloqs_auth.NewAuthServer(a, t), nil
}
