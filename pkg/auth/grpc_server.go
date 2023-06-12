package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
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
			var status uint32 = http.StatusInternalServerError
			if err, ok := err.(*mux.HttpError); ok {
				status = uint32(err.Status)
			}

			return ErrorToValidation(err, &status), err
		}
	case nil:
		status = http.StatusBadRequest
		return Invalid("Did not recieve Credentials.", &status), errors.New("credentials cannot be nil")
	default:
		status = http.StatusBadRequest
		return Invalid("Recieved forbidden on unsupported Credentials.", &status), fmt.Errorf("credentials has unexpected type %T", x)
	}

	//status = http.StatusNoContent
	status = http.StatusCreated
	if id := CredentialsToID(in); id != nil {
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

	//if id := CredentialsToID(in.Credentials); id != nil {
	//	return Valid(fmt.Sprintf("Credentials for `%s` were deleted with success!", *id)), nil
	//} else {
	//	return Valid("Credentials were deleted with success!"), nil
	//}

	str := "TODO"
	var code uint32 = 500
	return Invalid(str, &code), nil
}

func (s *AuthServer) LogIn(ctx context.Context, in *proto.AskPermissions) (*proto.TokenValidation, error) {
	var (
		token       Token
		err         error
		permissions = NIL
		validation  *proto.Validation
		status      uint32
		super       bool
		typ         AuthType
	)

	switch x := in.Credentials.Credentials.(type) {
	case *proto.Credentials_Basic:
		typ = BASIC_EMAIL
		err = s.auther.CheckAccessBasic(ctx, x)
		if err != nil {
			status = http.StatusInternalServerError
			if err, ok := err.(*mux.HttpError); ok {
				status = uint32(err.Status)
			}

			return &proto.TokenValidation{
				Validation: ErrorToValidation(err, &status),
				Token:      nil,
			}, err
		}
		super, err = s.auther.IsSuperBasic(ctx, x)
		if err != nil {
			status = http.StatusInternalServerError
			if err, ok := err.(*mux.HttpError); ok {
				status = uint32(err.Status)
			}

			return &proto.TokenValidation{
				Validation: ErrorToValidation(err, &status),
				Token:      nil,
			}, err
		}
	case nil:
		status = http.StatusBadRequest
		validation = Invalid("Did not recieve Credentials.", &status)
		err = errors.New("credentials cannot be nil")
		return &proto.TokenValidation{
			Validation: validation,
			Token:      nil,
		}, err
	default:
		status = http.StatusBadRequest
		validation = Invalid("Recieved forbidden on unsupported Credentials.", &status)
		err = fmt.Errorf("credentials has unexpected type %T", x)
		return &proto.TokenValidation{
			Validation: validation,
			Token:      nil,
		}, err
	}

	if token, err = s.tokener.GenToken(ctx, &Payload{
		Client:      *CredentialsToID(in.Credentials),
		Permissions: Permission(in.Permissions),
		Super:       super,
		Type:        typ,
	}); err != nil {
		status = http.StatusInternalServerError
		if err, ok := err.(*mux.HttpError); ok {
			status = uint32(err.Status)
		}

		return &proto.TokenValidation{
			Validation: ErrorToValidation(err, &status),
			Token:      nil,
		}, err
	}

	permissions = Permission(in.Permissions)
	status = http.StatusOK
	if id := CredentialsToID(in.Credentials); id != nil {
		validation = Valid(fmt.Sprintf("Credentials for `%s` were created with success!", *id), &status)
	} else {
		validation = Valid("Credentials were created with success!", &status)
	}

	return &proto.TokenValidation{
		Validation: validation,
		Token: &proto.Token{
			Jwt:         string(token),
			Permissions: (*uint64)(&permissions),
		},
	}, err
}

func (s *AuthServer) LogOut(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	err := s.tokener.RevokeToken(ctx, Token(in.Jwt))
	var status uint32 = http.StatusOK
	v := Valid("LogOut", &status)
	if err != nil {
		status = http.StatusInternalServerError
		if err, ok := err.(*mux.HttpError); ok {
			status = uint32(err.Status)
		}

		v = ErrorToValidation(err, &status)
	}

	return v, err
}

func (s *AuthServer) IsSuper(ctx context.Context, in *proto.Credentials) (v *proto.Validation, err error) {
	var (
		status uint32
		super  bool
	)

	switch x := in.Credentials.(type) {
	case *proto.Credentials_Basic:
		err = s.auther.CheckAccessBasic(ctx, x)
		if err != nil {
			status = http.StatusInternalServerError
			if err, ok := err.(*mux.HttpError); ok {
				status = uint32(err.Status)
			}

			return ErrorToValidation(err, &status), err
		}
		super, err = s.auther.IsSuperBasic(ctx, x)
		if err != nil {
			status = http.StatusInternalServerError
			if err, ok := err.(*mux.HttpError); ok {
				status = uint32(err.Status)
			}

			return ErrorToValidation(err, &status), err
		}
	case nil:
		status = http.StatusBadRequest
		v = Invalid("Did not recieve Credentials.", &status)
		err = errors.New("credentials cannot be nil")
		return
	default:
		status = http.StatusBadRequest
		v = Invalid("Recieved forbidden on unsupported Credentials.", &status)
		err = fmt.Errorf("credentials has unexpected type %T", x)
		return
	}

	return &proto.Validation{
		Valid: super,
	}, nil
}

func (s *AuthServer) Validate(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	valid, err := s.tokener.VerifyToken(ctx, Token(in.Jwt), Permission(*in.Permissions))
	if valid {
		return &proto.Validation{
			Valid: valid,
		}, err
	}

	var str strings.Builder
	str.WriteString("The token provided it's invalid")
	if err != nil {
		str.WriteString(":\t")
		str.WriteString(err.Error())
	}
	msg := str.String()

	return &proto.Validation{
			Valid:   valid,
			Message: &msg,
		}, &mux.HttpError{
			Body:   msg,
			Status: http.StatusForbidden,
		}
}
