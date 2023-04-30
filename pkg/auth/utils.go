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

func credentialsToID(c *proto.Credentials) *string {
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
	for _, v := range conf.MustGetConfOrDefault([]any{}, "auth", "supported") {
		if v.(string) == s {
			return true
		}
	}

	return false
}
