package auth

import "github.com/bloqs-sites/bloqsenjin/proto"

func valid(msg string, status *uint32) *proto.Validation {
	return &proto.Validation{
		Valid:          true,
		Message:        &msg,
		HttpStatusCode: status,
	}
}

func invalid(msg string, status *uint32) *proto.Validation {
	return &proto.Validation{
		Valid:          false,
		Message:        &msg,
		HttpStatusCode: status,
	}
}

func errorToValidation(err error, status *uint32) *proto.Validation {
	return invalid(err.Error(), status)
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
