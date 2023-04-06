package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	basic_email_prefix = "creds:basic:email:%s"
	jwt_prefix         = "token:jwt:%s"
)

type claims struct {
	auth.Payload
	jwt.RegisteredClaims
}

type BloqsAuther struct {
	creds db.KVDBer
}

func NewBloqsAuther(creds db.KVDBer) *BloqsAuther {
	return &BloqsAuther{creds}
}

func (a *BloqsAuther) SignInBasic(ctx context.Context, c *proto.Credentials_Basic) *auth.AuthError {
	if err := verifyEmail(c.Basic.Email); err != nil {
		return auth.NewAuthError(err.Error(), http.StatusBadRequest)
	}

	pass := c.Basic.Password

	if len(pass) > 72 { // bcrypt says that "GenerateFromPassword does not accept passwords longer than 72 bytes"
		return auth.NewAuthError("The password provided it's too long (bigger than 72 bytes)", http.StatusBadRequest)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return auth.NewAuthError(err.Error(), http.StatusInternalServerError)
	}

	entries := make(map[string][]byte, 1)
	entries[fmt.Sprintf(basic_email_prefix, c.Basic.Email)] = hash

	if err != a.creds.Put(ctx, entries, 0) {
		return auth.NewAuthError(err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *BloqsAuther) SignOutBasic(ctx context.Context, c *proto.Credentials_Basic, tk *proto.Token, t auth.Tokener) *auth.AuthError {
	if err := a.CheckAccessBasic(ctx, c); err != nil {
		return err
	}

	if err := a.creds.Delete(ctx, fmt.Sprintf(basic_email_prefix, c.Basic.GetEmail())); err != nil {
		return err
	}

	return nil
}

func (a *BloqsAuther) GrantTokenBasic(ctx context.Context, c *proto.Credentials_Basic, p auth.Permissions, t auth.Tokener) (auth.Token, *auth.AuthError) {
	if err := a.CheckAccessBasic(ctx, c); err != nil {
		return "", err
	}

	return t.GenToken(ctx, &auth.Payload{
		Client:      c.Basic.Email,
		Permissions: p,
	})
}

func (a *BloqsAuther) CheckAccessBasic(ctx context.Context, c *proto.Credentials_Basic) *auth.AuthError {
	hashes, err := a.creds.Get(ctx, fmt.Sprintf(basic_email_prefix, c.Basic.GetEmail()))

	if err != nil {
		return err
	}

	hash := hashes[fmt.Sprintf(basic_email_prefix, c.Basic.GetEmail())]

	if err := bcrypt.CompareHashAndPassword(hash, []byte(c.Basic.GetPassword())); err != nil {
		return err
	}

	return nil
}
