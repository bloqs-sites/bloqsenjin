package models

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	internal_db "github.com/bloqs-sites/bloqsenjin/internal/db"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	bloqs_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/redis/go-redis/v9"
)

type Preference struct {
}

func (Preference) Table() string {
	return "preference"
}

func (Preference) Type() string {
	return "CategoryCode"
}

func (Preference) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "preference",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`name` VARCHAR(80) NOT NULL",
				"`description` VARCHAR(140) NOT NULL",
				"`color` VARCHAR(80) NOT NULL",
				"UNIQUE(`name`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "shares",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`preference1_id` INT UNSIGNED NOT NULL",
				"`preference2_id` INT UNSIGNED NOT NULL",
				"`weight` FLOAT(6,3) DEFAULT 0",
				"UNIQUE(`preference1_id`, `preference2_id`)",
				"PRIMARY KEY(`id`)",
				"CHECK (preference1_id < preference2_id)",
			},
		},
	}
}

func (Preference) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (Preference) CreateViews() []db.View {
	return []db.View{}
}

func (m Preference) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name        string
		description string
		color       string
	)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, bloqs_helpers.X_WWW_FORM_URLENCODED) {
		if err := r.ParseForm(); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.X_WWW_FORM_URLENCODED, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		name = r.FormValue("name")
		description = r.FormValue("description")
		color = r.FormValue("color")
	} else if strings.HasPrefix(ct, bloqs_helpers.FORM_DATA) {
		if err := r.ParseMultipartForm(0x400); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.FORM_DATA, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		name = r.FormValue("name")
		description = r.FormValue("description")
		color = r.FormValue("color")
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
		bloqs_helpers.Append(&h, "Accept", bloqs_helpers.X_WWW_FORM_URLENCODED)
		bloqs_helpers.Append(&h, "Accept", bloqs_helpers.FORM_DATA)
		return &rest.Created{
			Status:  status,
			Message: fmt.Sprintf("request has the usupported media type `%s`", ct),
		}, nil
	}

	if l := len(name); l > 80 || l <= 0 {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: "`name` body field has to have a length between 1 and 80 characters",
		}, nil
	}

	if l := len(description); l > 140 {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: "`description` body field has to have a length with a maximum of 140 characters",
		}, nil
	}

	if l := len(color); l > 80 {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: "`color` body field has to have a length with a maximum of 80 characters",
		}, nil
	}

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	_, err = helpers.ValidateAndGetToken(w, r, a, bloqs_auth.CREATE_PREFERENCE)
	if err != nil {
		return nil, err
	}

	preferences, err := s.DBH.Select(r.Context(), "preference", func() map[string]any {
		return map[string]any{"id": new(int64)}
	}, nil)
	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	var result db.Result
	result, err = s.DBH.Insert(r.Context(), "preference", []map[string]any{
		{
			"name":        name,
			"description": description,
			"color":       color,
		},
	})

	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	shares := make([]map[string]any, 0, len(preferences.Rows))
	res_id := int(*result.LastID)
	for _, p := range preferences.Rows {
		id := int(*p["id"].(*int64))
		var id1, id2 string
		if id < res_id {
			id1 = strconv.Itoa(id)
			id2 = strconv.Itoa(res_id)
		} else {
			id1 = strconv.Itoa(res_id)
			id2 = strconv.Itoa(id)
		}

		shares = append(shares, map[string]any{
			"preference1_id": id1,
			"preference2_id": id2,
			"weight":         "0",
		})
	}

	if len(shares) > 0 {
		if _, err := s.DBH.Insert(r.Context(), "shares", shares); err != nil {
			s.DBH.Delete(r.Context(), "preference", map[string]any{
				"id": strconv.Itoa(res_id),
			})

			status = http.StatusInternalServerError
			return nil, &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
		}
	}

	return &rest.Created{
		LastID:  result.LastID,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (p Preference) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)

	var where []db.Condition = []db.Condition{}
	if (id != nil) && (*id != "") {
        where = append(where, db.Condition{Column: "id", Value: *id})
	}

	result, err := s.DBH.Select(r.Context(), "preference", func() map[string]any {
		return map[string]any{
			"id":          new(int64),
			"name":        new(string),
			"description": new(string),
			"color":       new(string),
		}
	}, where)

	api := conf.MustGetConf("REST", "domain").(string)

	for _, i := range result.Rows {
		i["href"] = fmt.Sprintf("%s/preference/%d", api, *i["id"].(*int64))
	}

	status := http.StatusOK
	msg := ""
	if err != nil {
		status = http.StatusInternalServerError
		msg = err.Error()
	}

	return &rest.Resource{
		Models:  result.Rows,
		Type:    "CategoryCode",
		Status:  uint16(status),
		Unique:  (id != nil) && (*id != ""),
		Message: msg,
	}, err
}

func (p Preference) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (p Preference) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func authSrv(ctx context.Context) (proto.AuthServer, error) {
	// TODO: How can I make it that you can specify which implementation of the interfaces you want to use?
	creds, err := internal_db.NewMySQL(ctx, strings.TrimSpace(os.Getenv("BLOQS_AUTH_MYSQL_DSN")))
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", creds, err)
	}

	opt, err := redis.ParseURL(strings.TrimSpace(os.Getenv("BLOQS_TOKENS_REDIS_DSN")))
	if err != nil {
		return nil, fmt.Errorf("could not parse the `BLOQS_TOKENS_REDIS_DSN` to create the credentials to connect to the DB:\t%s", err)
	}

	a, err := auth.NewBloqsAuther(ctx, creds)
	if err != nil {
		return nil, err
	}

	secrets, err := internal_db.NewKeyDB(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error creating DB instance of type `%T`:\t%s", secrets, err)
	}

	t := auth.NewBloqsTokener(secrets)

	return bloqs_auth.NewAuthServer(a, t), nil
}

func PreferenceExists(ctx context.Context, id int64, s rest.RESTServer) (bool, error) {
	result, err := s.DBH.Select(ctx, "preference", func() map[string]any {
		return map[string]any{"id": new(int64)}
	}, []db.Condition{{Column: "id", Value: id}})
	if err != nil {
		return false, err
	}

	return len(result.Rows) == 1, nil
}
