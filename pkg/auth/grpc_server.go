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

func NewAuthServer(a Auther, t Tokener) *AuthServer {
	return &AuthServer{
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
			return ErrorToValidation(err, &status), err
		}
	case nil:
		return Invalid("Did not recieve Credentials.", nil), errors.New("credentials cannot be nil")
	default:
		return Invalid("Recieved forbidden on unsupported Credentials.", nil), fmt.Errorf("credentials has unexpected type %T", x)
	}

	status = http.StatusNoContent
	if id := credentialsToID(in); id != nil {
		return Valid(fmt.Sprintf("Credentials for `%s` were created with success!", *id), &status), nil
	} else {
		return Valid("Credentials were created with success!", &status), nil
	}
}

func (s *AuthServer) SignOut(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	//switch x := in.Credentials.Credentials.(type) {
	//case *proto.Credentials_Basic:
	//	if err := s.auther.SignOutBasic(ctx, x, in.Token, s.tokener); err != nil {
	//		return ErrorToValidation(err), err
	//	}
	//case nil:
	//	return Invalid("Did not recieve Credentials."), errors.New("Credentials cannot be nil")
	//default:
	//	return Invalid("Recieved forbidden on unsupported Credentials."), fmt.Errorf("Credentials has unexpected type %T", x)
	//}

	//if id := credentialsToID(in.Credentials); id != nil {
	//	return Valid(fmt.Sprintf("Credentials for `%s` were deleted with success!", *id)), nil
	//} else {
	//	return Valid("Credentials were deleted with success!"), nil
	//}

	str := "TODO"
	var code uint32 = 500
	return Invalid(str, &code), nil
}

func (s *AuthServer) LogIn(ctx context.Context, in *proto.AskPermissions) (*proto.TokenValidation, error) {
	//var (
	//	token       Token
	//	err         error
	//	permissions = NIL
	//	validation  *proto.Validation
	//)

	//switch x := in.Credentials.Credentials.(type) {
	//case *proto.Credentials_Basic:
	//	token, err = s.auther.GrantTokenBasic(ctx, x, Permissions(in.Permissions), s.tokener)
	//case nil:
	//	err = errors.New("Credentials cannot be nil")
	//default:
	//	err = fmt.Errorf("Credentials.Creds has unexpected type %T", x)
	//}

	//if err == nil {
	//	permissions = Permissions(in.Permissions)
	//}

	//if err == nil {
	//	if id := credentialsToID(in.Credentials); id != nil {
	//		validation = Valid(fmt.Sprintf("The Credentials for `%s` are Valid, here is your token!", *id))
	//	} else {
	//		validation = Valid("The Credentials are Valid, here is your token!")
	//	}
	//} else {
	//	validation = ErrorToValidation(err)
	//}

	//return &proto.TokenValidation{
	//	Validation: Validation,
	//	Token: &proto.Token{
	//		Jwt:         []byte(token),
	//		Permissions: (*uint64)(&permissions),
	//	},
	//}, err

	return nil, nil
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

func (s *AuthServer) Revoke(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	return &proto.Validation{
		Valid: s.tokener.VerifyToken(ctx, Token(in.Jwt), Permissions(*in.Permissions)),
	}, nil
}
