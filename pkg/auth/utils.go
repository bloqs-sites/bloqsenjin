package auth

import (
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

func Valid(msg string, status *uint32) *proto.Validation {
	return &proto.Validation{
		Valid:          true,
		Message:        &msg,
		HttpStatusCode: status,
	}
}

func Invalid(msg string, status *uint32) *proto.Validation {
	return &proto.Validation{
		Valid:          false,
		Message:        &msg,
		HttpStatusCode: status,
	}
}

func ErrorToValidation(err error, status *uint32) *proto.Validation {
	return Invalid(err.Error(), status)
}

func CredentialsToID(c *proto.Credentials) *string {
	switch x := c.Credentials.(type) {
	case *proto.Credentials_Basic:
		return &x.Basic.Email
	case nil:
		return nil
	default:
		id := c.String()
		return &id
	}
}

func IsAuthMethodSupported(s string) bool {
	supported, ok := conf.MustGetConfOrDefault(map[string]any{}, "auth", "supported")[s].(bool)
	return ok && supported
}
