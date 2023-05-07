package auth

import (
	"context"

	"github.com/bloqs-sites/bloqsenjin/proto"
)

type Token string
type Permissions uint64
type AuthType uint8

const (
	NIL                        Permissions = 0
	SIGN_OUT                               = 1 << iota
	NEEDLE_FOR_NEXT_PERMISSION             = iota

	BASIC_EMAIL AuthType = iota - NEEDLE_FOR_NEXT_PERMISSION
)

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
	SignOutBasic(context.Context, *proto.Credentials_Basic, *proto.Token, Tokener) error
	GrantTokenBasic(context.Context, *proto.Credentials_Basic, Permissions, Tokener) (Token, error)
}
