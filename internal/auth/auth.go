package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/golang-jwt/jwt/v4"
)

type claims struct {
	auth.Payload
	jwt.RegisteredClaims
}

type Auther struct{}

func (a Auther) GenToken(p auth.Payload) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims{
		p,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "bloqsenjin",
			Subject:   p.Client,
		},
	})

	tokenstr, err := token.SignedString([]byte("secret"))

	if err != nil {
		panic(err)
	}

	return tokenstr
}

func (a Auther) VerifyToken(t string, auths uint) bool {
	token, err := jwt.ParseWithClaims(t, &claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("")
		}

		return []byte("secret"), nil
	}, jwt.WithValidMethods([]string{}))

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return (claims.Payload.Permissions & auths) == auths
	} else if errors.Is(err, jwt.ErrTokenMalformed) {
		return false
	} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return false
	}

	return false
}
