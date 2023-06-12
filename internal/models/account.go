package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/bloqs-sites/bloqsenjin/internal/auth"
	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
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
				"`weight` FLOAT(6, 3) UNSIGNED NOT NULL",
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
	case "":
		fallthrough
	case http.MethodGet:
		if err != nil {
			return err
		}

		resources, err := m.Read(w, r, s)

		if err != nil {
			return err
		}

		if resources == nil {
			return &mux.HttpError{
				Status: http.StatusNotFound,
			}
		}

		encoder := json.NewEncoder(w)
		ctx := "https://schema.org/"
		typ := "Person"
		if ((s.SegLen() & 1) == 1) && (s.Seg(s.SegLen()-1) != nil) && (*s.Seg(s.SegLen() - 1) != "") {
			if len(resources.Models) == 0 {
				return &mux.HttpError{
					Status: http.StatusNotFound,
				}
			} else {
				resources.Models[0]["@context"] = ctx
				resources.Models[0]["@type"] = typ
				return encoder.Encode(resources.Models[0])
			}
		} else {
			resources.Models = append([]db.JSON{
				{
					"@context": ctx,
					"@type":    typ,
				},
			}, resources.Models...)
			return encoder.Encode(resources.Models)
		}
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

func (Account) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name                  string
		hasAdultConsideration          = "0"
		image                 string   = "NULL"
		likes                 []string = []string{}
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
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			hasAdultConsideration = "1"
		}
		likes = r.Form["likes"]
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

	tk, err := mux.ExtractToken(w, r)

	if err != nil {
		return nil, err
	}

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	permission := bloqs_auth.CREATE_ACCOUNT
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
	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	max := conf.MustGetConfOrDefault(1, "REST", "max")
	if len(result.Rows) >= max {
		status = http.StatusForbidden
		return nil, &mux.HttpError{
			Body:   fmt.Sprintf("the maximum limit of this resource (%d) has reached.", max),
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

	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	id := strconv.Itoa(int(*result.LastID))

	_, err = s.DBH.Insert(r.Context(), "credential_accounts", []map[string]string{
		{
			"credential_id": claims.Payload.Client,
			"account_id":    id,
		},
	})

	if err != nil {
		s.DBH.Delete(r.Context(), "account", map[string]any{"id": id})

		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	likes_inserts := make([]map[string]string, 0, len(likes))
	weight := strconv.Itoa(int(float64(100 / len(likes))))
	for _, like := range likes {
		likes_inserts = append(likes_inserts, map[string]string{
			"account_id":    id,
			"preference_id": like,
			"weight":        weight,
		})
	}

	_, err = s.DBH.Insert(r.Context(), "account_likes", likes_inserts)

	if err != nil {
		s.DBH.Delete(r.Context(), "account", map[string]any{"id": id})

		s.DBH.Delete(r.Context(), "credential_accounts", map[string]any{
			"credential_id": claims.Payload.Client,
			"account_id":    id,
		})

		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	for i := 0; i < len(likes); i++ {
		for j := i + 1; j < len(likes); j++ {
			var max, min int
			if i < j {
				min = i
				max = j
			} else {
				min = j
				max = i
			}

			res, err := s.DBH.Select(r.Context(), "shares", func() map[string]any {
				return map[string]any{
					"id":     new(int64),
					"weight": new(float32),
				}
			}, map[string]any{
				"preference1_id": min,
				"preference2_id": max,
			})

			if err != nil {
				continue
			}

			if len(res.Rows) > 0 {
				id := res.Rows[0]["id"].(*int64)
				w := res.Rows[0]["weight"].(*float32)

				if err := s.DBH.Update(r.Context(), "shares", map[string]any{
					"weight": *w + 1.0,
				}, map[string]any{
					"id": *id,
				}); err != nil {
					fmt.Printf("%v\n", err)
				}
			} else {
				if _, err := s.DBH.Insert(r.Context(), "shares", []map[string]string{
					{
						"preference1_id": strconv.Itoa(min),
						"preference2_id": strconv.Itoa(max),
						"weight":         "1",
					},
				}); err != nil {
					fmt.Printf("%v\n", err)
				}
			}
		}
	}

	return &rest.Created{
		LastID:  result.LastID,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Account) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)

	you := conf.MustGetConfOrDefault("@", "REST", "myself")
	api := conf.MustGetConf("REST", "domain").(string)

	var (
		result db.Result
		err    error
	)

	if id != nil && *id == you {
		a, err := authSrv(r.Context())

		if err != nil {
			return nil, err
		}

		tk, err := mux.ExtractToken(w, r)

		if err != nil {
			return nil, err
		}

		p := bloqs_auth.READ_ACCOUNT
		v, err := a.Validate(r.Context(), &proto.Token{
			Jwt:         string(tk),
			Permissions: (*uint64)(&p),
		})

		if err != nil {
			return nil, err
		}

		if !v.Valid {
			msg := ""
			if v.Message != nil {
				msg = *v.Message
			}

			return nil, &mux.HttpError{
				Body:   msg,
				Status: http.StatusUnauthorized,
			}
		}

		claims := &auth.Claims{}
		claims_str, err := base64.RawStdEncoding.DecodeString(strings.Split(string(tk), ".")[1])
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(claims_str, claims)
		if err != nil {
			return nil, err
		}
		*id = claims.Payload.Client

		res, err := s.DBH.Select(r.Context(), "credential_accounts", func() map[string]any {
			return map[string]any{
				"account_id": new(int64),
			}
		}, map[string]any{"credential_id": *id})

		if err != nil {
			return nil, err
		}

		var wait sync.WaitGroup

		wait.Add(len(res.Rows))

		accs := make([]db.JSON, 0, len(res.Rows))

		search := func(id any) {
			defer wait.Done()

			var res db.Result
			res, err = s.DBH.Select(r.Context(), "account", func() map[string]any {
				return map[string]any{
					"id":    new(int64),
					"name":  new(string),
					"image": new(string),
				}
			}, map[string]any{"id": id})

			if err != nil {
				return
			}

			acc := res.Rows[0]

			res, err = s.DBH.Select(r.Context(), "account_likes", func() map[string]any {
				return map[string]any{
					"preference_id": new(int64),
					"weight":        new(float32),
				}
			}, map[string]any{"account_id": acc["id"]})

			if err != nil {
				return
			}

			likes := make([]db.JSON, 0, len(res.Rows))
			for _, i := range res.Rows {
				i["url"] = fmt.Sprintf("%s/preference/%d", api, *i["preference_id"].(*int64))
				i["@type"] = "Category"
				delete(i, "preference_id")
				likes = append(likes, i)
			}

			acc["likes"] = likes

			accs = append(accs, acc)
		}

		for _, i := range res.Rows {
			go search(i["account_id"])
		}

		wait.Wait()

		result = db.Result{Rows: accs}
	} else {
		var where map[string]any = nil
		if (id != nil) && (*id != "") {
			where = map[string]any{"id": *id}
		}

		result, err = s.DBH.Select(r.Context(), "account", func() map[string]any {
			return map[string]any{
				"id":    new(int64),
				"name":  new(string),
				"image": new(string),
			}
		}, where)

		for _, i := range result.Rows {
			i["url"] = fmt.Sprintf("%s/account/%d", api, *i["id"].(*int64))
		}
	}

	status := http.StatusOK
	msg := ""
	if err != nil {
		status = http.StatusInternalServerError
		msg = err.Error()
	}

	return &rest.Resource{
		Models:  result.Rows,
		Status:  uint16(status),
		Message: msg,
	}, err
}

func (Account) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Account) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}
