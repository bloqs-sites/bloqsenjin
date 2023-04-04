package auth

import "github.com/bloqs-sites/bloqsenjin/proto"

func valid(msg string) *proto.Validation {
    return &proto.Validation{
        Valid: true,
        Message: &msg,
    }
}

func invalid(msg string) *proto.Validation {
    return &proto.Validation{
        Valid: false,
        Message: &msg,
    }
}

func errorToValidation(err error) *proto.Validation {
    return invalid(err.Error())
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
