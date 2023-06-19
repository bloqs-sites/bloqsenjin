package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

type AuthClient struct {
}

func (s *AuthClient) Validate(ctx context.Context, in *proto.Token) (*proto.Validation, error) {
	domain, err := conf.GetConf("auth", "domain")
	if err != nil {
		return nil, err
	}
	path, err := conf.GetConf("auth", "paths", "verify")
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, fmt.Sprint(domain, path), "POST", nil)
	if err != nil {
		return nil, err
	}

	client.Do(req)

	return nil, nil
}
