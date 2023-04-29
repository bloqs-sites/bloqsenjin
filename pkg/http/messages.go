package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
)

const (
	PLAIN                 = "text/plain"
	X_WWW_FORM_URLENCODED = "application/x-www-form-urlencoded"
	FORM_DATA             = "multipart/form-data"
	GRPC                  = "application/grpc"

	JWT_COOKIE = "__Host-bloqs-auth"

	BEARER_PREFIX = "Bearer "
)

var (
	Query     = conf.MustGetConfOrDefault("type", "auth", "signInTypeQueryParam")
	Token_exp = conf.MustGetConfOrDefault(900000, "auth", "token", "exp")
)

func ExtractToken(w http.ResponseWriter, r *http.Request) (jwt []byte, revoke bool) {
	revoke = false

	cookie, err := r.Cookie(JWT_COOKIE)
	if err == http.ErrNoCookie {
		header := r.Header.Get("Authorization")

		if !strings.HasPrefix(header, BEARER_PREFIX) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		jwt = []byte(header[len(BEARER_PREFIX):])
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jwt = []byte(cookie.Value)

	if err = cookie.Valid(); err != nil {
		goto revocation
	}

	if i := cookie.MaxAge; i <= 0 || i > Token_exp {
		goto revocation
	}

	if cookie.Expires.IsZero() {
		goto revocation
	}

	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		goto revocation
	}

	return

revocation:
	revoke = true

	revokeCookie(cookie, w)

	w.WriteHeader(http.StatusUnauthorized)
	return
}

func revokeCookie(c *http.Cookie, w http.ResponseWriter) {
	c.Value = ""
	c.Expires = time.Unix(0, 0)
	c.HttpOnly = true
	c.Secure = true
	c.SameSite = http.SameSiteStrictMode
	http.SetCookie(w, c)
}
