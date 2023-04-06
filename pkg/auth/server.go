package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/proto"
)

type AuthServer struct {
	proto.UnimplementedAuthServer

	auther  Auther
	tokener Tokener
}

func NewAuthServer(a Auther, t Tokener) AuthServer {
	return AuthServer{
		auther:  a,
		tokener: t,
	}
}

func (s *AuthServer) SignIn(ctx context.Context, in *proto.Credentials) (*proto.Validation, error) {
	var status uint32
	switch x := in.Credentials.(type) {
	case *proto.Credentials_Basic:
		if err := s.auther.SignInBasic(ctx, x); err != nil {
			status = uint32(err.http_status_code)
			return errorToValidation(err, &status), err
		}
	case nil:
		return invalid("Did not recieve Credentials.", nil), errors.New("Credentials cannot be nil")
	default:
		return invalid("Recieved forbidden on unsupported Credentials.", nil), fmt.Errorf("Credentials has unexpected type %T", x)
	}

	status = http.StatusNoContent
	if id := credentialsToID(in); id != nil {
		return valid(fmt.Sprintf("Credentials for `%s` were created with success!", *id), &status), nil
	} else {
		return valid("Credentials were created with success!", &status), nil
	}
}

func (s *AuthServer) SignOut(ctx context.Context, in *proto.CredentialsWithToken) (*proto.Validation, error) {
	switch x := in.Credentials.Credentials.(type) {
	case *proto.Credentials_Basic:
		if err := s.auther.SignOutBasic(ctx, x, in.Token, s.tokener); err != nil {
			return errorToValidation(err), err
		}
	case nil:
		return invalid("Did not recieve Credentials."), errors.New("Credentials cannot be nil")
	default:
		return invalid("Recieved forbidden on unsupported Credentials."), fmt.Errorf("Credentials has unexpected type %T", x)
	}

	if id := credentialsToID(in.Credentials); id != nil {
		return valid(fmt.Sprintf("Credentials for `%s` were deleted with success!", *id)), nil
	} else {
		return valid("Credentials were deleted with success!"), nil
	}
}

func (s *AuthServer) LogIn(ctx context.Context, in *proto.CredentialsWantPermissions) (*proto.TokenValidation, error) {
	var (
		token       Token
		err         error
		permissions = NO_PERMISSIONS
		validation  *proto.Validation
	)

	switch x := in.Credentials.Credentials.(type) {
	case *proto.Credentials_Basic:
		token, err = s.auther.GrantTokenBasic(ctx, x, Permissions(in.Permissions), s.tokener)
	case nil:
		err = errors.New("Credentials cannot be nil")
	default:
		err = fmt.Errorf("Credentials.Creds has unexpected type %T", x)
	}

	if err == nil {
		permissions = Permissions(in.Permissions)
	}

	if err == nil {
		if id := credentialsToID(in.Credentials); id != nil {
			validation = valid(fmt.Sprintf("The Credentials for `%s` are valid, here is your token!", *id))
		} else {
			validation = valid("The Credentials are valid, here is your token!")
		}
	} else {
		validation = errorToValidation(err)
	}

	return &proto.TokenValidation{
		Validation: validation,
		Token: &proto.Token{
			Jwt:         []byte(token),
			Permissions: (*uint64)(&permissions),
		},
	}, err
}

func (s *AuthServer) LogOut(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	return &proto.Validation{
		Valid: true,
	}, nil
}

func (s *AuthServer) Validate(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	return &proto.Validation{
		Valid: s.tokener.VerifyToken(ctx, Token(in.Jwt), Permissions(*in.Permissions)),
	}, nil
}
