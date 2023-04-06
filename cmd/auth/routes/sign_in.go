package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"google.golang.org/grpc"
)

func SignInRoute(s *grpc.Server, ch chan error, client_creator func(chan error) (proto.AuthClient, func())) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var err error

			err = helpers.CheckOriginHeader(w, r)

			if err != nil {
				return
			}

			ct := r.Header.Get("Content-Type")
			if strings.HasPrefix(ct, X_WWW_FORM_URLENCODED) {
				if err = r.ParseForm(); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				//} else if !strings.HasPrefix(ct, FORM_DATA) {
				//	if err = r.ParseMultipartForm(64 << 20); err != nil {
				//	    w.WriteHeader(http.StatusBadRequest)
				//		return
				//	}
			} else {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				w.Header().Add("Accept", X_WWW_FORM_URLENCODED)
				//w.Header().Add("Accept", FORM_DATA)
				return
			}

			var v *proto.Validation

			c, cc := client_creator(ch)
			defer cc()

			switch r.URL.Query().Get(query) {
			case "basic":
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

				v, err = c.SignIn(r.Context(), &proto.Credentials{
					Credentials: &proto.Credentials_Basic{
						Basic: &proto.Credentials_BasicCredentials{
							Email:    email,
							Password: pass,
						},
					},
				})
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
			//w.Header().Add("Access-Control-Allow-Headers", "")
			w.Header().Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
