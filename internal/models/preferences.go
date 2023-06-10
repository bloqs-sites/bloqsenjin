package models

import (
	"context"
	"errors"
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
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/redis/go-redis/v9"
)

type PreferenceHandler struct {
}

func (p PreferenceHandler) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name        string
		description string
	)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, mux.X_WWW_FORM_URLENCODED) {
		if err := r.ParseForm(); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", mux.X_WWW_FORM_URLENCODED, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		name = r.FormValue("name")
		description = r.FormValue("description")
	} else if strings.HasPrefix(ct, mux.FORM_DATA) {
		if err := r.ParseMultipartForm(0x400); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", mux.X_WWW_FORM_URLENCODED, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		name = r.FormValue("name")
		description = r.FormValue("description")
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
		mux.Append(&h, "Accept", mux.X_WWW_FORM_URLENCODED)
		mux.Append(&h, "Accept", mux.FORM_DATA)
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

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	tk, err := mux.ExtractToken(w, r)

	if err != nil {
		return nil, err
	}

	permission := auth.CREATE_PREFERENCE
	v, err := a.Validate(r.Context(), &proto.Token{
		Jwt:         string(tk),
		Permissions: (*uint64)(&permission),
	})

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
	result, err = s.DBH.Insert(r.Context(), "preference", []map[string]string{
		{
			"name":        name,
			"description": description,
		},
	})

	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	shares := make([]map[string]string, 0, len(preferences.Rows))
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

		shares = append(shares, map[string]string{
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

func (p PreferenceHandler) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	dbh := s.DBH

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) > 2 && len(parts[2]) > 0 {
		id, err := strconv.ParseInt(parts[2], 10, 0)

		if err != nil {
			return nil, err
		}

		res, err := dbh.Select(r.Context(), "preference", p.MapGenerator(), nil)
		if err != nil {
			return nil, err
		}

		rows := res.Rows
		rn := len(rows)

		if rn < 1 {
			//return rows, nil
			return nil, nil
		}

		json := make([]db.JSON, 1)

		for _, v := range rows {
			i, ok := v["id"]

			if !ok {
				continue
			}

			j, ok := i.(*int64)

			if ok && *j == id {
				v["@context"] = "https://schema.org/"
				v["@type"] = "CategoryCode"
				json[0] = v
				//return json, nil
				return nil, nil
			}
		}

		//return json, nil
		return nil, nil
	}

	res, err := dbh.Select(r.Context(), "preference", p.MapGenerator(), nil)
	if err != nil {
		return nil, err
	}

	rows := res.Rows
	rn := len(rows)

	if rn < 1 {
		//return rows, nil
		return nil, nil
	}

	json, i := make([]db.JSON, len(rows)+1), 0

	json[i] = db.JSON{
		"@context": "https://schema.org/",
	}

	for _, v := range rows {
		v["@type"] = "CategoryCode"

		i++
		json[i] = v
	}

	//return json, nil
	return nil, nil
}

func (p PreferenceHandler) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (p PreferenceHandler) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (p PreferenceHandler) Handle(w http.ResponseWriter, r *http.Request, s rest.RESTServer) error {
	var (
		status uint32

		err error
	)

	h := w.Header()
	_, err = helpers.CheckOriginHeader(&h, r)

	switch r.Method {
	case "":
		fallthrough
	case http.MethodGet:
		if err != nil {
			return err
		}

		redirect, err := p.Read(w, r, s)

		if err != nil {
			return err
		}

		fmt.Printf("redirect.Message: %v\n", redirect.Message)

		//w.Header().Set("Content-Type", "application/json")

		//if models := res.Models; len(models) == 0 {
		//	_, err = w.Write([]byte("{}"))
		//} else if len(models) == 1 {
		//	err = json.NewEncoder(w).Encode(models[0])
		//} else {
		//	err = json.NewEncoder(w).Encode(models)
		//}

		//if err != nil {
		//	w.WriteHeader(http.StatusInternalServerError)
		//	fmt.Fprintf(w, "%s", err.Error())
		//}
	case http.MethodPost:
		if err != nil {
			return err
		}

		created, err := p.Create(w, r, s)

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
			w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", domain, p.Table(), *id))
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

	return errors.New("")
}

func (p PreferenceHandler) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "preference",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`name` VARCHAR(80)",
				"`description` VARCHAR(140)",
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

func (h *PreferenceHandler) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (h *PreferenceHandler) CreateViews() []db.View {
	return []db.View{}
}

func (p PreferenceHandler) MapGenerator() func() map[string]any {
	return func() map[string]any {
		m := make(map[string]any)
		m["id"] = new(int64)
		m["description"] = new(string)
		m["name"] = new(string)
		return m
	}
}

func (p PreferenceHandler) Table() string {
	return "preference"
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
