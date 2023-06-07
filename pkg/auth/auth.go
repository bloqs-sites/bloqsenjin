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
	GRANT_SUPER                            = 1 << iota
	REVOKE_SUPER                           = 1 << iota
	NEEDLE_FOR_NEXT_PERMISSION             = iota

	BASIC_EMAIL AuthType = iota - NEEDLE_FOR_NEXT_PERMISSION
)

type Payload struct {
	Client      string
	Permissions Permissions
	Super       bool
}

type Tokener interface {
	GenToken(context.Context, *Payload) (Token, error)
	VerifyToken(context.Context, Token, Permissions) (bool, error)
	RevokeToken(context.Context, Token) error
}

type Auther interface {
	SignInBasic(context.Context, *proto.Credentials_Basic) error
	SignOutBasic(context.Context, *proto.Credentials_Basic) error
	CheckAccessBasic(context.Context, *proto.Credentials_Basic) error
	IsSuperBasic(context.Context, *proto.Credentials_Basic) (bool, error)
	GrantSuper(context.Context, *proto.Credentials) error
	RevokeSuper(context.Context, *proto.Credentials) error
}
