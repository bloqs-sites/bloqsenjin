package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	http_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
)

type RESTServer struct {
	mux      *mux.Router
	DBH      db.DataManipulater
	segments []string
}

func NewRESTServer(endpoint string, crud db.DataManipulater) RESTServer {
	return RESTServer{
		mux: mux.NewRouter(endpoint),
		DBH: crud,
	}
}

func (s *RESTServer) AttachHandler(ctx context.Context, route string, h Handler) {
	if err := s.DBH.CreateTables(ctx, h.CreateTable()); err != nil {
		fmt.Printf("%v\n", err)
	}
	if err := s.DBH.CreateIndexes(ctx, h.CreateIndexes()); err != nil {
		fmt.Printf("%v\n", err)
	}
	if err := s.DBH.CreateViews(ctx, h.CreateViews()); err != nil {
		fmt.Printf("%v\n", err)
	}

	s.mux.Route(route, func(w http.ResponseWriter, r *http.Request, segs []string) {
		var status uint16 = http.StatusInternalServerError
		s.segments = segs

		headers := w.Header()
		_, err := helpers.CheckOriginHeader(&headers, r)

		switch r.Method {
		case "":
			fallthrough
		case http.MethodGet:
			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}

			var resources *Resource
			resources, err = h.Read(w, r, *s)

			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}

			if resources == nil {
				err = &mux.HttpError{
					Status: http.StatusNotFound,
				}
				break
			}

			w.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(w)
			ctx := "https://schema.org/"
			typ := resources.Type

			if typ == "" {
				err = &mux.HttpError{
					Status: http.StatusInternalServerError,
				}

				break
			}

			last := s.Seg(s.SegLen() - 1)
			second := s.Seg(1)
			if ((s.SegLen() & 1) == 1) && (last != nil) && (*last != "") && resources.Unique {
				if len(resources.Models) == 0 {
					err = &mux.HttpError{
						Status: http.StatusNotFound,
					}

					break
				} else {
					if second == nil {
						resources.Models[0]["@context"] = ctx
						resources.Models[0]["@type"] = typ
					}
					err = encoder.Encode(resources.Models[0])
				}
			} else {
				if len(resources.Models) > 0 {
					resources.Models = append([]db.JSON{
						{
							"@context": ctx,
							"@type":    typ,
						},
					}, resources.Models...)
				}
				err = encoder.Encode(resources.Models)
			}
		case http.MethodPost:
			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}

			var created *Created
			created, err = h.Create(w, r, *s)

			if err != nil {
				fmt.Printf("%v\n", err)
				break
			}

			if created == nil {
				err = &mux.HttpError{
					Status: http.StatusInternalServerError,
				}
				break
			}

			var id *string = nil

			domain := conf.MustGetConf("REST", "domain").(string)

			if created.LastID != nil {
				id_str := strconv.Itoa(int(*created.LastID))
				id = &id_str
			}

			if id != nil {
				w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", domain, h.Table(), *id))
			}
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/plain")
			}
			w.WriteHeader(int(created.Status))
			w.Write([]byte(created.Message))
		case http.MethodOptions:
			http_helpers.Append(&headers, "Access-Control-Allow-Methods", http.MethodPost)
			http_helpers.Append(&headers, "Access-Control-Allow-Methods", http.MethodOptions)
			headers.Set("Access-Control-Allow-Credentials", "true")
			http_helpers.Append(&headers, "Access-Control-Allow-Headers", "Authorization, Content-Type")
			//bloqs_http.Append(&h, "Access-Control-Expose-Headers", "")
			headers.Set("Access-Control-Max-Age", "0")
		default:
			status = http.StatusMethodNotAllowed
			err = &mux.HttpError{
				Body:   "",
				Status: uint16(status),
			}
		}

		if err != nil {
			fmt.Printf("%v\n", err)
			if err, ok := err.(*mux.HttpError); ok {
				status = err.Status
			}

			w.WriteHeader(int(status))

			if status != http.StatusNoContent {
				w.Write([]byte(err.Error()))

				if w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", "text/plain")
				}
			}
		}
	})
}

func (s *RESTServer) Serve() http.HandlerFunc {
	return s.mux.ServeHTTP
}

func (s RESTServer) Seg(i int) *string {
	if i < 0 || len(s.segments) <= i {
		return nil
	}

	return &s.segments[i]
}

func (s RESTServer) SegLen() int {
	return len(s.segments)
}
