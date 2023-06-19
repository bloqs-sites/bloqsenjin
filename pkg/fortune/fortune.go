package fortune

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	http_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
)

const fortune = "fortune"

func Fortune(ctx context.Context) string {
	return FortuneFromDB(ctx, "")
}

func FortuneFromDB(ctx context.Context, db ...string) string {
	cmd := exec.CommandContext(ctx, fortune, "-as", "-e", strings.Join(db, " "))

	return string(getOut(cmd))
}

func getOut(cmd *exec.Cmd) []byte {
	out, _ := cmd.CombinedOutput()
	return out
}

func Server(endpoint string) http.HandlerFunc {
	r := mux.NewRouter(endpoint)
	r.Route("/", func(w http.ResponseWriter, r *http.Request, segs []string) {
		var msg string

		h := w.Header()
		status, err := helpers.CheckOriginHeader(&h, r)

		switch r.Method {
		case "":
			fallthrough
		case http.MethodGet:
			if err != nil {
				break
			}

			msg = FortuneFromDB(r.Context(), r.URL.Query()["databases"]...)
		case http.MethodOptions:
			http_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodGet)
			http_helpers.Append(&h, "Access-Control-Allow-Methods", http.MethodOptions)
			h.Set("Access-Control-Max-Age", fmt.Sprint(time.Hour*24/time.Second))
		default:
			status = http.StatusMethodNotAllowed
		}

		if err != nil {
			msg = err.Error()
		}

		w.WriteHeader(int(status))
		w.Write([]byte(msg))
		w.Header().Add("Content-Type", http_helpers.PLAIN)
	})

	return r.ServeHTTP
}

func Serve(endpoint string, w http.ResponseWriter, r *http.Request) {
	Server(endpoint)(w, r)
}
