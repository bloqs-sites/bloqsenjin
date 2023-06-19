package helpers

import "net/http"

const (
	BEARER_PREFIX = "Bearer"
)

func Append(h *http.Header, name, value string) {
	if h.Get(name) != "" {
		h.Set(name, h.Get(name)+", "+value)
	} else {
		h.Set(name, value)
	}
}
