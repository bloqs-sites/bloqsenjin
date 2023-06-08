package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

type Account struct {
}

func (Account) Table() string {
	return "account"
}

func (Account) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "account",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`name` VARCHAR(80) NOT NULL",
				"`hasAdultConsideration` BOOL DEFAULT 0",
				"`image` VARCHAR(254) DEFAULT NULL",
				"UNIQUE(`name`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "credential_accounts",
			Columns: []string{
				"`credential_id` VARCHAR(320) NOT NULL",
				"`account_id` INT UNSIGNED NOT NULL",
				"UNIQUE (`credential_id`, `account_id`)",
			},
		},
		{
			Name: "account_likes",
			Columns: []string{
				"`account_id` INT UNSIGNED NOT NULL",
				"`preference_id` INT UNSIGNED NOT NULL",
				"UNIQUE (`account_id`, `preference_id`)",
			},
		},
	}
}

func (Account) CreateIndexes() []db.Index {
	return nil
}

func (Account) CreateViews() []db.View {
	return nil
}

func (m Account) Handle(w http.ResponseWriter, r *http.Request, s rest.RESTServer) error {
	var (
		status uint32

		err error
	)

	h := w.Header()
	_, err = helpers.CheckOriginHeader(&h, r)

	switch r.Method {
	case http.MethodPost:
		if err != nil {
			return err
		}

		created, err := m.Create(w, r, s)

		if err != nil {
			return err
		}

		if created == nil {
			return &mux.HttpError{
				Status: http.StatusInternalServerError,
			}
		}

		var id *string = nil

		domain := conf.MustGetConf("REST", "domain").(string)

		if created.LastID != nil {
			id_str := strconv.Itoa(int(*created.LastID))
			id = &id_str
		}

		if id != nil {
			w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", domain, m.Table(), *id))
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/plain")
		}
		w.WriteHeader(int(created.Status))
		w.Write([]byte(created.Message))

		return nil
	case http.MethodOptions:
		mux.Append(&h, "Access-Control-Allow-Methods", http.MethodPost)
		mux.Append(&h, "Access-Control-Allow-Methods", http.MethodOptions)
		h.Set("Access-Control-Allow-Credentials", "true")
		mux.Append(&h, "Access-Control-Allow-Headers", "Authorization")
		//bloqs_http.Append(&h, "Access-Control-Expose-Headers", "")
		h.Set("Access-Control-Max-Age", "0")
		return err
	default:
		status = http.StatusMethodNotAllowed
		return &mux.HttpError{
			Body:   "",
			Status: uint16(status),
		}
	}
}

func (Account) MapGenerator() func() map[string]any {
	return func() map[string]any {
		m := make(map[string]any)
		m["id"] = new(int64)
		m["name"] = new(string)
		m["hasAdultConsideration"] = new(bool)
		m["image"] = new(string)
		return m
	}
}

func (Account) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (res *rest.Created, err error) {
	var (
		status uint16 = http.StatusInternalServerError

		name                  string
		hasAdultConsideration          = "0"
		image                 string   = "NULL"
		likes                 []string = []string{}
	)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, mux.X_WWW_FORM_URLENCODED) {
		if err = r.ParseForm(); err != nil {
			status = http.StatusBadRequest
			res = &rest.Created{
				Status:  status,
				Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", mux.X_WWW_FORM_URLENCODED, err),
			}
			err = &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
			return
		}

		name = r.FormValue("name")
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			hasAdultConsideration = "1"
		}
		likes = r.Form["likes"]
	} else if strings.HasPrefix(ct, mux.FORM_DATA) {
		if err = r.ParseMultipartForm(0x400); err != nil {
			status = http.StatusBadRequest
			res = &rest.Created{
				Status:  status,
				Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", mux.X_WWW_FORM_URLENCODED, err),
			}
			err = &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
			return
		}

		name = r.FormValue("name")
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			hasAdultConsideration = "1"
		}
		likes = r.Form["likes"]
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
		mux.Append(&h, "Accept", mux.X_WWW_FORM_URLENCODED)
		mux.Append(&h, "Accept", mux.FORM_DATA)
		res = &rest.Created{
			Status:  status,
			Message: fmt.Sprintf("request has the usupported media type `%s`", ct),
		}
		if err != nil {
			err = &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
		}
		return
	}

	if l := len(name); l > 80 || l <= 0 {
		status = http.StatusUnprocessableEntity
		res = &rest.Created{
			Status:  status,
			Message: "`name` body field has to have a length between 1 and 80 characters",
		}
		return
	}

	tk, err := mux.ExtractToken(w, r)

	if err != nil {
		return nil, err
	}

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	permission := auth.CREATE_ACCOUNT
	v, err := a.Validate(r.Context(), &proto.Token{
		Jwt:         string(tk),
		Permissions: (*uint64)(&permission),
	})

	if err != nil {
		return nil, err
	}

	claims := &auth.Claims{}
    claims_str, err := base64.RawStdEncoding.DecodeString(strings.Split(string(tk), ".")[1])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(claims_str, claims)

    println(1)
    fmt.Printf("%v\n", claims_str)
    fmt.Printf("%v\n", claims)
    fmt.Printf("%v\n", err)
	if err != nil {
		return nil, err
	}

	if !v.Valid {
		msg := ""

		if v.Message != nil {
			msg = *v.Message
		}

		if v.HttpStatusCode != nil {
			status = uint16(*v.HttpStatusCode)
		}

		return nil, &mux.HttpError{
			Body:   msg,
			Status: status,
		}
	}

	var result db.Result
	result, err = s.DBH.Select(r.Context(), "credential_accounts", func() map[string]any {
		return nil
	}, map[string]any{
		"credential_id": claims.Payload.Client,
	})
    println(2)
	if err != nil {
		status = http.StatusInternalServerError
		err = &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
		return
	}

    max := conf.MustGetConfOrDefault(1, "REST", "max")
    if len(result.Rows) >= max {
        status = http.StatusForbidden
        return nil, &mux.HttpError{
            Body: fmt.Sprintf("the maximum limit of this resource (%d) has reached.", max),
            Status: status,
        }
    }

	result, err = s.DBH.Insert(r.Context(), "account", []map[string]string{
		{
			"name":                  name,
			"hasAdultConsideration": hasAdultConsideration,
			"image":                 image,
		},
	})

    println(3)
	if err != nil {
		status = http.StatusInternalServerError
		err = &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
		return
	}

	id := strconv.Itoa(int(*res.LastID))

	_, err = s.DBH.Insert(r.Context(), "credential_accounts", []map[string]string{
		{
			"credential_id": claims.Payload.Client,
			"account_id":    id,
		},
	})

    println(4)
	if err != nil {
		s.DBH.Delete(r.Context(), "account", map[string]any{"id": id})

		status = http.StatusInternalServerError
		err = &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
		return
	}

	likes_inserts := make([]map[string]string, 0, len(likes))
	for _, like := range likes {
		likes_inserts = append(likes_inserts, map[string]string{
			"account_id":    id,
			"preference_id": like,
		})
	}

	_, err = s.DBH.Insert(r.Context(), "credential_accounts", likes_inserts)

    println(5)
	if err != nil {
		s.DBH.Delete(r.Context(), "account", map[string]any{
			"id": strconv.Itoa(int(*res.LastID)),
		})

		s.DBH.Delete(r.Context(), "credential_accounts", map[string]any{
			"credential_id": claims.Payload.Client,
			"account_id":    strconv.Itoa(int(*res.LastID)),
		})

		status = http.StatusInternalServerError
		err = &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
		return
	}

	return &rest.Created{
		LastID:  result.LastID,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Account) Read(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Account) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Account) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}
