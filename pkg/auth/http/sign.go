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
		status uint16
	)

	err = helpers.CheckOriginHeader(w, r)

	switch r.Method {
	case http.MethodPost:
		if err != nil {
			return
		}

		var credentials *proto.Credentials

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, bloqs_http.X_WWW_FORM_URLENCODED) {
			if err = r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else if r.ProtoMajor == 2 && strings.HasPrefix(ct, bloqs_http.GRPC) {
			if buf, err := io.ReadAll(r.Body); err != nil {
				return
			} else {
				if err := p.Unmarshal(buf, credentials); err != nil {
					return
				}
				//s.ServeHTTP(w, r)
			}
		} else {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Header().Add("Accept", bloqs_http.X_WWW_FORM_URLENCODED)
			w.Header().Add("Accept", bloqs_http.GRPC)
			return
		}

		switch r.URL.Query().Get(bloqs_http.Query) {
		case "basic":
			if credentials == nil {
				email := r.FormValue("email")

				if email == "" {
					w.WriteHeader(http.StatusUnprocessableEntity)
					return
				}

				pass := r.FormValue("pass")

				if pass == "" {
					w.WriteHeader(http.StatusUnprocessableEntity)
					return
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
		}
	case http.MethodDelete:
		if err != nil {
			return
		}

		var token *proto.Token

		a := authSrv(r.Context())

		jwt, revoke := bloqs_http.ExtractToken(w, r)
		token = &proto.Token{
			Jwt: jwt,
		}

		if revoke {
			v, err = a.Revoke(r.Context(), token)
            println(v, err)
			return
		}

		if jwt != nil {
			return
		}

		switch r.URL.Query().Get(bloqs_http.Query) {
		case "basic":
            v, err = a.SignOut(r.Context(), token)
		}
	case http.MethodOptions:
		w.Header().Add("Access-Control-Allow-Methods", http.MethodPost)
		w.Header().Add("Access-Control-Allow-Methods", http.MethodDelete)
		w.Header().Add("Access-Control-Allow-Methods", http.MethodOptions)
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		//w.Header().Add("Access-Control-Allow-Headers", "")
		//w.Header().Add("Access-Control-Expose-Headers", "")
		//w.Header().Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		w.Header().Set("Access-Control-Max-Age", "0")
		w.WriteHeader(http.StatusOK)
        return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
        return
	}

	if v != nil {
		if code := v.HttpStatusCode; code != nil {
			status = uint16(*code)
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

		if status != http.StatusNoContent {
			json.NewEncoder(w).Encode(v)
			w.Header().Set("Content-Type", "application/json")
		}

		w.WriteHeader(int(status))
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
