package rest

import (
	"context"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
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
	db := s.DBH
	db.CreateTables(ctx, h.CreateTable())
	db.CreateIndexes(ctx, h.CreateIndexes())
	db.CreateViews(ctx, h.CreateViews())

	s.mux.Route(route, func(w http.ResponseWriter, r *http.Request, segs []string) {
		var status uint16 = http.StatusInternalServerError
		s.segments = segs
		err := h.Handle(w, r, *s)

		if err != nil {
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
	if len(s.segments) <= i {
		return nil
	}

	return &s.segments[i]
}

func (s RESTServer) SegLen() int {
	return len(s.segments)
}
