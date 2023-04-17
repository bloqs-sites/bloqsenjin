package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwt_prefix    = "token:jwt:%s"
	table         = "credentials"
	id_type_table = "id-type"
	failed_table  = "failed"
)

type claims struct {
	auth.Payload
	jwt.RegisteredClaims
}

type BloqsAuther struct {
	creds db.DataManipulater
}

func NewBloqsAuther(ctx context.Context, creds db.DataManipulater) *BloqsAuther {
	creds.CreateTables(ctx, []db.Table{
		{
			Name: table,
			Columns: []string{
				"`id` INTEGER PRIMARY KEY AUTOINCREMENT",
				"`identifier` TEXT NOT NULL",
				"`type` INT NOT NULL",
				"`secret` TEXT NOT NULL",
				"`is_super` BOOLEAN NOT NULL DEFAULT 0",
				"`created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`modified_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`last_log_in` TIMESTAMP",
				"UNIQUE (`identifier`, `type`)",
			},
		},
		{
			Name: "failed",
			Columns: []string{
				"`id` INTEGER PRIMARY KEY AUTOINCREMENT",
				"`credential` INTEGER NOT NULL",
				"`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				fmt.Sprintf("FOREIGN KEY (`credential`) REFERENCES `%s`(`id`)", table),
			},
		},
	})
	//creds.CreateViews(ctx, []db.View{})
	return &BloqsAuther{creds}
}

func (a *BloqsAuther) SignInBasic(ctx context.Context, c *proto.Credentials_Basic) *auth.AuthError {
	if err := verifyEmail(c.Basic.Email); err != nil {
		return auth.NewAuthError(err.Error(), http.StatusBadRequest)
	}

	pass := c.Basic.Password

	if len(pass) > 72 { // bcrypt says that "GenerateFromPassword does not accept passwords longer than 72 bytes"
		return auth.NewAuthError("The password provided it's too long (bigger than 72 bytes)", http.StatusBadRequest)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return auth.NewAuthError(err.Error(), http.StatusInternalServerError)
	}

	if _, err := a.creds.Insert(ctx, table, []map[string]string{
		map[string]string{
			"identifier": c.Basic.Email,
			"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
			"secret":     string(hash),
		},
	}); err != nil {
		return auth.NewAuthError(err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *BloqsAuther) SignOutBasic(ctx context.Context, c *proto.Credentials_Basic, tk *proto.Token, t auth.Tokener) *auth.AuthError {
	if err := a.CheckAccessBasic(ctx, c); err != nil {
		return err
	}

	if err := a.creds.Delete(ctx, fmt.Sprintf(basic_email_prefix, c.Basic.GetEmail())); err != nil {
		return err
	}

	return nil
}

func (a *BloqsAuther) GrantTokenBasic(ctx context.Context, c *proto.Credentials_Basic, p auth.Permissions, t auth.Tokener) (auth.Token, *auth.AuthError) {
	if err := a.CheckAccessBasic(ctx, c); err != nil {
		return "", err
	}

	return t.GenToken(ctx, &auth.Payload{
		Client:      c.Basic.Email,
		Permissions: p,
	})
}

func (a *BloqsAuther) CheckAccessBasic(ctx context.Context, c *proto.Credentials_Basic) *auth.AuthError {
	hashes, err := a.creds.Select(ctx, table, func() map[string]any {
        return map[string]any{
            "secret": new(string),
        }
    })

	if err != nil {
		return err
	}

	hash := hashes[fmt.Sprintf(basic_email_prefix, c.Basic.GetEmail())]

	if err := bcrypt.CompareHashAndPassword(hash, []byte(c.Basic.GetPassword())); err != nil {
		return err
	}

	return nil
}
