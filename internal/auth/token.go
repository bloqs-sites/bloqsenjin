package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/golang-jwt/jwt/v5"
)

type BloqsTokener struct {
	secrets db.KVDBer
}

func NewBloqsTokener(secrets db.KVDBer) *BloqsTokener {
	return &BloqsTokener{secrets}
}

func (t *BloqsTokener) GenToken(ctx context.Context, p *auth.Payload) (tokenstr auth.Token, err error) {
	tokenstr = ""

	key := fmt.Sprintf(jwt_prefix, p.Client)

	var secrets map[string][]byte
	secrets, err = t.secrets.Get(ctx, key)
	if err != nil {
		return
	}

	secret, ok := secrets[key]

	if !ok || (len(secret) == 0) { // create a new secret
		secret := make([]byte, 32)

		if _, err = rand.Read(secret); err != nil {
			return
		}

		puts := make(map[string][]byte, 1)
		puts[key] = secret
		if err = t.secrets.Put(ctx, puts, 7*time.Minute); err != nil {
			return
		}
	}

	var (
		str   string
		token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims{
			*p,
			jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
				Issuer:    "bloqsenjin",
				Subject:   p.Client,
			},
		})
	)
	str, err = token.SignedString(secret)
	tokenstr = auth.Token(str)

	return
}

func (t *BloqsTokener) VerifyToken(ctx context.Context, tk auth.Token, p auth.Permissions) (bool, error) {
	token, err := t.parseToken(ctx, tk)

	if err != nil {
		return false, err
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return (claims.Payload.Permissions & p) == p, nil
	} else {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return false, errors.Join(errors.New("that's not even a token"), err)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return false, errors.Join(errors.New("invalid signature"), err)
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return false, errors.Join(errors.New("timing is everything"), err)
		} else {
			return false, errors.Join(errors.New("couldn't handle this token"), err)
		}
	}
}

func (t *BloqsTokener) RevokeToken(ctx context.Context, tk auth.Token) error {
	token, err := t.parseToken(ctx, tk)

	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		sub := claims.Subject
		key := fmt.Sprintf(jwt_prefix, sub)
		return t.secrets.Delete(ctx, key)
	} else {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return errors.Join(errors.New("that's not even a token"), err)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return errors.Join(errors.New("invalid signature"), err)
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return errors.Join(errors.New("timing is everything"), err)
		} else {
			return errors.Join(errors.New("couldn't handle this token"), err)
		}
	}
}

func (t *BloqsTokener) parseToken(ctx context.Context, tk auth.Token) (*jwt.Token, error) {
	return jwt.ParseWithClaims(string(tk), &claims{}, t.keyfunc(ctx), jwt.WithValidMethods([]string{
		"HS256",
		"HS384",
		"HS512",
	}), jwt.WithJSONNumber(), jwt.WithIssuer("bloqsenjin"), jwt.WithLeeway(5*time.Second))
}

func (t *BloqsTokener) keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(tk *jwt.Token) (any, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tk.Header["alg"])
		}

		sub, err := tk.Claims.GetSubject()
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf(jwt_prefix, sub)

		var secrets map[string][]byte
		secrets, err = t.secrets.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		secret, ok := secrets[key]
		if !ok {
			return nil, fmt.Errorf("")
		}

		return secret, nil
	}
}
