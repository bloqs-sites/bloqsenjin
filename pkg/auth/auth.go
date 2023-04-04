package auth

import (
	"context"

	"github.com/bloqs-sites/bloqsenjin/proto"
)

type Token string
type Permissions uint64

const NO_PERMISSIONS Permissions = 0

type Payload struct {
	Client      string
	Permissions Permissions
}

type Tokener interface {
	GenToken(context.Context, *Payload) (Token, error)
	VerifyToken(context.Context, Token, Permissions) bool
}

type Auther interface {
	SignInBasic(context.Context, *proto.Credentials_Basic) error
	SignOutBasic(context.Context, *proto.Credentials_Basic, *proto.Token, *Tokener) error
	GrantTokenBasic(context.Context, *proto.CredentialsWantPermissions) (Token, error)
}

