package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/bloqs-sites/bloqsenjin/pkg/email"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"golang.org/x/crypto/bcrypt"
)

const (
	jwt_prefix    = "token:jwt:%s"
	table         = "credentials"
	id_type_table = "id-type"
	failed_table  = "failed"
)

type BloqsAuther struct {
	creds db.DataManipulater
}

func NewBloqsAuther(ctx context.Context, creds db.DataManipulater) (*BloqsAuther, error) {
	err := creds.CreateTables(ctx, []db.Table{
		{
			Name: table,
			Columns: []string{
				"`id` INTEGER PRIMARY KEY AUTO_INCREMENT",
				"`identifier` VARCHAR(320) NOT NULL",
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
				"`id` INTEGER PRIMARY KEY AUTO_INCREMENT",
				"`credential` INTEGER NOT NULL",
				"`timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				//fmt.Sprintf("FOREIGN KEY (`credential`) REFERENCES `%s`(`id`)", table),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &BloqsAuther{creds}, nil
}

func (a *BloqsAuther) SignInBasic(ctx context.Context, c *proto.Credentials_Basic) error {
	u := time.Now()
	if err := email.VerifyEmail(ctx, c.Basic.Email); err != nil {
		status := uint16(http.StatusInternalServerError)

		switch err := err.(type) {
		case *email.InvalidEmailError:
			status = err.Status
		case *email.ServerError:
			status = uint16(http.StatusInternalServerError)
		}

		return &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	log.Printf("%s took %v", "VerifyEmail", time.Since(u))

	pass := c.Basic.Password

	if len(pass) > 72 { // bcrypt says that "GenerateFromPassword does not accept passwords longer than 72 bytes"
		return errors.New("the password provided it's too long (bigger than 72 bytes)")
	}

	// TODO: test password entropy

	u = time.Now()
	exists, err := a.creds.Select(ctx, table, func() map[string]any {
		return map[string]any{
			"identifier": new(string),
			"type":       new(int),
		}
	}, map[string]any{
		"identifier": c.Basic.Email,
		"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
	})
	log.Printf("%s took %v", "Select", time.Since(u))

	if err != nil {
		return err
	}

	if len(exists.Rows) > 0 {
		return &mux.HttpError{
			Body:   "credentials already in use",
			Status: http.StatusConflict,
		}
	}

	u = time.Now()
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	log.Printf("%s took %v", "GenerateFromPassword", time.Since(u))

	u = time.Now()
	if _, err := a.creds.Insert(ctx, table, []map[string]any{
		{
			"identifier": c.Basic.Email,
			"type":       auth.BASIC_EMAIL,
			"secret":     hash,
		},
	}); err != nil {
		return err
	}
	log.Printf("%s took %v", "Insert", time.Since(u))

	return nil
}

func (a *BloqsAuther) SignOutBasic(ctx context.Context, c *proto.Credentials_Basic) error {
	if err := a.CheckAccessBasic(ctx, c); err != nil {
		return err
	}

	if err := a.creds.Delete(ctx, table, map[string]any{
		"identifier": c.Basic.Email,
		"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
	}); err != nil {
		return err
	}

	return nil
}

func (a *BloqsAuther) GrantSuper(ctx context.Context, creds *proto.Credentials) error {
	return a.super(ctx, creds, true)
}

func (a *BloqsAuther) RevokeSuper(ctx context.Context, creds *proto.Credentials) error {
	return a.super(ctx, creds, false)
}

func (a *BloqsAuther) super(ctx context.Context, creds *proto.Credentials, super bool) (err error) {
	switch x := creds.Credentials.(type) {
	case *proto.Credentials_Basic:
		if err = a.CheckAccessBasic(ctx, x); err != nil {
			if err, ok := err.(*mux.HttpError); ok {
				return err
			}

			var status uint32 = http.StatusInternalServerError
			return &mux.HttpError{
				Body:   "",
				Status: uint16(status),
			}
		}

		err = a.creds.Update(ctx, table, map[string]any{
			"is_super": super,
		}, map[string]any{
			"identifier": x.Basic.Email,
			"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
		})
	case nil:
		status := http.StatusBadRequest
		return &mux.HttpError{
			Body:   "did not recieve Credentials.",
			Status: uint16(status),
		}
	default:
		status := http.StatusBadRequest
		return &mux.HttpError{
			Body:   "recieved forbidden on unsupported Credentials.",
			Status: uint16(status),
		}
	}

	if err != nil {
		var status uint32 = http.StatusInternalServerError
		return &mux.HttpError{
			Body:   err.Error(),
			Status: uint16(status),
		}
	}

	return nil
}

func (a *BloqsAuther) IsSuperBasic(ctx context.Context, creds *proto.Credentials_Basic) (super bool, err error) {
	var ok bool
	super = false

	if err = a.CheckAccessBasic(ctx, creds); err != nil {
		if err, ok = err.(*mux.HttpError); ok {
			return
		}

		var status uint32 = http.StatusInternalServerError
		err = &mux.HttpError{
			Body:   "",
			Status: uint16(status),
		}
		return
	}

	res, err := a.creds.Select(ctx, table, func() map[string]any {
		return map[string]any{
			"is_super": new(bool),
		}
	}, map[string]any{
		"identifier": creds.Basic.Email,
		"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
	})

	if err != nil {
		err = &mux.HttpError{
			Body:   err.Error(),
			Status: http.StatusInternalServerError,
		}
		return
	}

	if len(res.Rows) != 1 {
		err = &mux.HttpError{
			Body:   "wrong credentials",
			Status: http.StatusUnauthorized,
		}
		return
	}

	super = *res.Rows[0]["is_super"].(*bool)

	return
}

func (a *BloqsAuther) CheckAccessBasic(ctx context.Context, c *proto.Credentials_Basic) error {
	res, err := a.creds.Select(ctx, table, func() map[string]any {
		return map[string]any{
			"secret": new([]byte),
		}
	}, map[string]any{
		"identifier": c.Basic.Email,
		"type":       strconv.Itoa(int(auth.BASIC_EMAIL)),
	})

	if err != nil {
		return &mux.HttpError{
			Body:   err.Error(),
			Status: http.StatusInternalServerError,
		}
	}

	if len(res.Rows) != 1 {
		return &mux.HttpError{
			Body:   "wrong credentials",
			Status: http.StatusUnauthorized,
		}
	}

	secret := res.Rows[0]["secret"].(*[]byte)
	if err := bcrypt.CompareHashAndPassword(*secret, []byte(c.Basic.GetPassword())); err != nil {
		return &mux.HttpError{
			Body:   "wrong credentials",
			Status: http.StatusUnauthorized,
		}
	}

	return nil
}
