package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	bloqs_http "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"
	p "google.golang.org/protobuf/proto"
)

//var (
//	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")
//)

func signRoute(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		v      *proto.Validation
		status uint32
	)

	h := w.Header()
	status, err = helpers.CheckOriginHeader(&h, r)

	switch r.Method {
	case http.MethodPost:
		if err != nil {
			v = bloqs_auth.ErrorToValidation(err, &status)
			break
		}

		var credentials *proto.Credentials

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_http.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				status = http.StatusBadRequest
				v = bloqs_auth.Invalid("", &status)
				break
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_http.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				status = http.StatusBadRequest
				v = bloqs_auth.Invalid("", &status)
				break
			} else {
				if err := p.Unmarshal(buf, credentials); err != nil {
					status = http.StatusBadRequest
					v = bloqs_auth.Invalid("", &status)
					break
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			status = http.StatusUnsupportedMediaType
            bloqs_http.Append(&h, "Accept", bloqs_http.X_WWW_FORM_URLENCODED)
            bloqs_http.Append(&h, "Accept", bloqs_http.GRPC)
			v = bloqs_auth.Invalid("", &status)
			break
		}

		if !r.URL.Query().Has(bloqs_http.GetQuery()) {
			status = http.StatusBadRequest
			v = bloqs_auth.Invalid("", &status)
			break
		}

		method := r.URL.Query().Get(bloqs_http.GetQuery())
		switch method {
		case "basic":
			if !bloqs_auth.IsAuthMethodSupported(method) {
				status = http.StatusUnprocessableEntity
				v = bloqs_auth.Invalid("", &status)
				goto respond
			}

			if credentials == nil {
				email := r.FormValue("email")

				if email == "" {
					status = http.StatusUnprocessableEntity
					v = bloqs_auth.Invalid("", &status)
					goto respond
				}

				pass := r.FormValue("pass")

				if pass == "" {
					status = http.StatusUnprocessableEntity
					v = bloqs_auth.Invalid("", &status)
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

			a := authSrv(r.Context())
			v, err = a.SignIn(r.Context(), credentials)
		default:
			status = http.StatusBadRequest
			v = bloqs_auth.Invalid("", &status)
			goto respond
		}
	case http.MethodDelete:
		if err != nil {
			v = bloqs_auth.Invalid("", &status)
			goto respond
		}

		var token *proto.Token

		a := authSrv(r.Context())

		jwt, revoke := bloqs_http.ExtractToken(w, r)
		token = &proto.Token{
			Jwt: jwt,
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
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		//bloqs_http.Append(&h, "Access-Control-Allow-Headers", "")
		//bloqs_http.Append(&h, "Access-Control-Expose-Headers", "")
		//w.Header().Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		w.Header().Set("Access-Control-Max-Age", "0")
		if err != nil {
			w.Write([]byte(err.Error()))
		}
		v = &proto.Validation{
			Valid:          err == nil,
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
			w.Header().Set("Content-Type", "application/json")
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

func authSrv(ctx context.Context) bloqs_auth.AuthServer {
	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	creds := db.NewMySQL(os.Getenv("BLOQS_AUTH_MYSQL_DSN"))
	secrets := db.NewKeyDB(db.NewRedisCreds("localhost", 6379, "", 0))

	a := auth.NewBloqsAuther(ctx, &creds)
	t := auth.NewBloqsTokener(secrets)

	return bloqs_auth.NewAuthServer(a, t)
}
