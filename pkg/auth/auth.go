package auth

import (
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type payload map[string]any

type Tokener interface {
	GenToken(p payload, auths int) string
	VerifyToken(t string, auths int) bool
}

type authManager struct {
	Tokener Tokener
}

func NewAuthManager(t Tokener) authManager {
	return authManager{
		Tokener: t,
	}
}

func (a authManager) AuthDecor(h *rest.Handler, auths int) func(*http.Request, rest.Handler) rest.Handler {
	return func(r *http.Request, h rest.Handler) rest.Handler {
		if a.Tokener.VerifyToken("", auths) {

		}

		return h
	}
}
