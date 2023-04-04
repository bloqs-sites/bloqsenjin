package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/golang-jwt/jwt/v4"
)

type BloqsTokener struct {
    secrets db.KVDBer
}
func NewBloqsTokener(secrets db.KVDBer) *BloqsTokener {
	return &BloqsTokener{secrets}
}

func (t *BloqsTokener) genToken(ctx context.Context, p auth.Payload) string {
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

	key := fmt.Sprintf(jwt_prefix, p.Client)

	secrets, err := t.secrets.Get(ctx, key)

    var secret []byte
    ok := true

	if err != nil {
		ok = false
	}

    if ok {
	    secret, ok = secrets[key]
    }

	if !ok {
		secret := make([]byte, 24)
		_, err := rand.Read(secret)

		if err != nil {
			panic(err)
		}
	}

	tokenstr, err := token.SignedString(secret)

	if err != nil {
		panic(err)
	}

	if !ok {
		puts := make(map[string][]byte, 1)
		puts[key] = secret
		t.secrets.Put(ctx, puts, 7*time.Minute)
	}

	return tokenstr
}

func (t *BloqsTokener) VerifyToken(ctx context.Context, tk auth.Token, p auth.Permissions) bool {
	token, err := jwt.ParseWithClaims(string(tk), &claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("")
		}

		return []byte("secret"), nil
	}, jwt.WithValidMethods([]string{}))

	if err != nil {
		return false
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return (claims.Payload.Permissions & p) == p
	} else if errors.Is(err, jwt.ErrTokenMalformed) {
		return false
	} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return false
	}

	return false
}

