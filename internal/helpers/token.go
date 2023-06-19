package helpers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

func ValidateAndGetToken(w http.ResponseWriter, r *http.Request, a proto.AuthServer, p auth.Permission) ([]byte, error) {
	tk, err := helpers.ExtractToken(w, r)
	if err != nil {
		return nil, err
	}

	if v, err := a.Validate(r.Context(), &proto.Token{
		Jwt:         string(tk),
		Permissions: (*uint64)(&p),
	}); err != nil {
		return nil, err
	} else if !v.Valid {
		var msg string
		if v.Message != nil {
			msg = *v.Message
		}

		return nil, &mux.HttpError{
			Body:   msg,
			Status: http.StatusUnauthorized,
		}
	}

	return tk, nil
}
func ValidateAndGetClaims(w http.ResponseWriter, r *http.Request, a proto.AuthServer, p auth.Permission) (*auth.Claims, error) {
	if tk, err := ValidateAndGetToken(w, r, a, p); err != nil {
		return nil, err
	} else {
		return ExtractClaims(string(tk))
	}
}

func ExtractClaims(tk string) (*auth.Claims, error) {
	claims := &auth.Claims{}
	claims_str, err := base64.RawStdEncoding.DecodeString(strings.Split(string(tk), ".")[1])
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(claims_str, claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}
