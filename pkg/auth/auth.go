package auth

import "context"

type Payload struct {
	Client      string
	Permissions uint64
}

type Tokener interface {
	GenToken(ctx context.Context, p Payload) string
	VerifyToken(ctx context.Context, t string, auths uint64) bool
}
