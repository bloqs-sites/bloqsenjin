package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type BloqsTokener struct {
	secrets db.KVDBer
}

func NewBloqsTokener(secrets db.KVDBer) *BloqsTokener {
	return &BloqsTokener{secrets}
}

type NoPermissionsError struct {
	permission auth.Permission
}

func (err NoPermissionsError) Error() string {
	format := "The token provided does not have the `%s` permission."
	hash := auth.GetPermissionsHash(err.permission)
	return fmt.Sprintf(format, hash)
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
		secret = make([]byte, 32)

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
		str      string
		auth_api = conf.MustGetConfOrDefault("", "auth", "domain")
		rest_api = conf.MustGetConfOrDefault("", "REST", "domain")
		token    = jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
			*p,
			jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
				Issuer:    auth_api,
				Subject:   p.Client,
				Audience:  []string{auth_api, rest_api},
				ID:        uuid.NewString(),
			},
		})
	)

	str, err = token.SignedString(secret)
	tokenstr = auth.Token(str)

	return
}

func (t *BloqsTokener) VerifyToken(ctx context.Context, tk auth.Token, p auth.Permission) (bool, error) {
	token, err := t.ParseToken(ctx, tk)

	if err != nil {
		return false, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if (claims.Payload.Permissions & p) == p {
			return true, nil
		} else {
			return false, NoPermissionsError{p}
		}
	} else {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return false, fmt.Errorf("that's not even a token:\t%v", err)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return false, fmt.Errorf("invalid signature:\t%v", err)
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return false, fmt.Errorf("timing is everything:\t%v", err)
		} else {
			return false, fmt.Errorf("couldn't handle this token:\t%v", err)
		}
	}
}

func (t *BloqsTokener) RevokeToken(ctx context.Context, tk auth.Token) error {
	token, err := t.ParseToken(ctx, tk)

	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		sub := claims.Subject
		key := fmt.Sprintf(jwt_prefix, sub)
		return t.secrets.Delete(ctx, key)
	} else {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return fmt.Errorf("that's not even a token:\t%v", err)
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return fmt.Errorf("invalid signature:\t%v", err)
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return fmt.Errorf("timing is everything:\t%v", err)
		} else {
			return fmt.Errorf("couldn't handle this token:\t%v", err)
		}
	}
}

func (t *BloqsTokener) ParseToken(ctx context.Context, tk auth.Token) (*jwt.Token, error) {
	auth_api := conf.MustGetConf("auth", "domain").(string)
	rest_api := conf.MustGetConf("REST", "domain").(string)

	token, err := jwt.ParseWithClaims(string(tk), &Claims{}, t.keyfunc(ctx), jwt.WithValidMethods([]string{
		"HS256",
		"HS384",
		"HS512",
	}), jwt.WithJSONNumber(), jwt.WithLeeway(5*time.Second))

	if err != nil {
		return nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return nil, err
	}

	if issuer != auth_api && issuer != rest_api {
		return nil, errors.New("token has invalid claims: token has invalid issuer")
	}

	return token, err
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
