package routes

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	bloqs_auth "github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	bloqs_http "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	p "google.golang.org/protobuf/proto"
)

var (
	gRPCPort = flag.Int("gRPCPort", 50051, "The gRPC server port")
)

func SignInRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var err error

		err = helpers.CheckOriginHeader(w, r)

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
			if buf, err := ioutil.ReadAll(r.Body); err != nil {
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

		var v *proto.Validation

		c, cc := createGRPCClient()
		defer cc()

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

			v, err = c.SignIn(r.Context())
		}

		var status uint16

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
	case http.MethodOptions:
		helpers.CheckOriginHeader(w, r)
		w.Header().Add("Access-Control-Allow-Methods", http.MethodPost)
		w.Header().Add("Access-Control-Allow-Methods", http.MethodOptions)
		w.Header().Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func createGRPCClient() (proto.AuthClient, func(), error) {
	conn, err := grpc.Dial(net.JoinHostPort("localhost", fmt.Sprint(*gRPCPort)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return proto.NewAuthClient(conn), func() {
		conn.Close()
	}, nil
}

func a() auth.AuthServer {
	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	creds := db.NewMySQL(os.Getenv("DSN"))
	secrets := db.NewKeyDB(db.NewRedisCreds("localhost", 6379, "", 0))

	a := bloqs_auth.NewBloqsAuther(creds)
	t := bloqs_auth.NewBloqsTokener(secrets)

	return auth.NewAuthServer(a, t)
}
