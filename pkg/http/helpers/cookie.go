package helpers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
)

const (
	//JWT_COOKIE = "__Host-bloqs-auth"
	JWT_COOKIE = "_Secure-bloqs-auth"
)

func SetToken(w http.ResponseWriter, r *http.Request, jwt string) error {
	exp := conf.MustGetConfOrDefault[float64](900000, "auth", "token", "exp")

	http.SetCookie(w, &http.Cookie{
		Name:     JWT_COOKIE,
		Value:    jwt,
		Expires:  time.Now().Add(time.Duration(exp)),
		Path:     "/",
		Domain:   r.URL.Host,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	return nil
}

func ExtractToken(w http.ResponseWriter, r *http.Request) (jwt []byte, err error) {
	var cookie *http.Cookie
	cookie, err = r.Cookie(JWT_COOKIE)
	if err == http.ErrNoCookie {
		err = nil
		header := r.Header.Get("Authorization")

		if header == "" {
			return nil, &mux.HttpError{
				Body:   fmt.Sprintf("HTTP Cookie `%s` and/or `Authorization` HTTP Header is missing", JWT_COOKIE),
				Status: http.StatusUnauthorized,
			}
		}

		bearerToken := strings.Split(header, " ")

		if len(bearerToken) != 2 || bearerToken[0] != BEARER_PREFIX {
			return nil, &mux.HttpError{
				Body:   "`Authorization` HTTP Header does not have a Bearer token",
				Status: http.StatusUnauthorized,
			}
		}

		return []byte(bearerToken[1]), nil
	} else if err != nil {
		err = &mux.HttpError{Body: "", Status: http.StatusInternalServerError}
		return
	}

	jwt = []byte(cookie.Value)

	exp := conf.MustGetConfOrDefault(900000, "auth", "token", "exp")

	// TODO: needs to look if the status codes used are the best for the situations.
	if err = cookie.Valid(); err != nil {
		err = &mux.HttpError{Body: fmt.Sprintf("invalid HTTP Cookie:\t %v", err), Status: http.StatusBadRequest}
		goto revocation
	}

	if i := cookie.MaxAge; i <= 0 || i > exp {
		err = &mux.HttpError{Body: "the HTTP Cookie is expired", Status: http.StatusBadRequest}
		goto revocation
	}

	if cookie.Expires.IsZero() {
		err = &mux.HttpError{Body: "the HTTP Cookie is expired", Status: http.StatusBadRequest}
		goto revocation
	}

	if !cookie.Secure || !cookie.HttpOnly {
		// err = &mux.HttpError{Body: "", Status: http.}
		goto revocation
	}

	return

revocation:
	revokeCookie(cookie, w)

	if err == nil {
		err = &mux.HttpError{Body: "", Status: http.StatusUnauthorized}
	}
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
