package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	bloqs_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

type Org struct {
}

const ORG_TYPE = "Organization"

func (Org) Table() string {
	return "org"
}

func (Org) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "org",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`name` VARCHAR(80) NOT NULL",
				"`description` VARCHAR(80) NOT NULL",
				"`url` VARCHAR(255) NOT NULL",
				"`logo` VARCHAR(255) NOT NULL",
				"`email` VARCHAR(320) DEFAULT NULL",
				"`founder` INT UNSIGNED NOT NULL",
				"`foudingDate` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "org_members",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`org_id` INT UNSIGNED NOT NULL",
				"`profile_id` INT UNSIGNED NOT NULL",
				"UNIQUE(`org_id`, `profile_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "org_languages",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`org_id` INT UNSIGNED NOT NULL",
				"`language` VARCHAR(255) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "org_ratings",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`org_id` INT UNSIGNED NOT NULL",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`ratingValue` INT NOT NULL",
				"`ratingExplanation` TEXT DEFAULT NULL",
				"UNIQUE(`org_id`, `profile_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (Org) CreateIndexes() []db.Index {
	return nil
}

func (Org) CreateViews() []db.View {
	return nil
}

func (Org) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name                  string
		hasAdultConsideration          = "0"
		image                 string   = "NULL"
		likes                 []string = []string{}
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
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			hasAdultConsideration = "1"
		}
		likes = r.Form["likes"]
	} else if strings.HasPrefix(ct, bloqs_helpers.FORM_DATA) {
		if err := r.ParseMultipartForm(0x400); err != nil {
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
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			hasAdultConsideration = "1"
		}
		likes = r.Form["likes"]
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

	tk, err := bloqs_helpers.ExtractToken(w, r)

	if err != nil {
		return nil, err
	}

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	permission := bloqs_auth.CREATE_PROFILE
	v, err := a.Validate(r.Context(), &proto.Token{
		Jwt:         string(tk),
		Permissions: (*uint64)(&permission),
	})

	if err != nil {
		return nil, err
	}

	claims := &bloqs_auth.Claims{}
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
	}, []db.Condition{{Column: "credential_id", Value: claims.Payload.Client}})
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

	result, err = s.DBH.Insert(r.Context(), "account", []map[string]any{
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

	_, err = s.DBH.Insert(r.Context(), "credential_accounts", []map[string]any{
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

	if len(likes) != 0 {
		likes_inserts := make([]map[string]any, 0, len(likes))
		weight := strconv.Itoa(int(float64(100 / len(likes))))
		for _, like := range likes {
			likes_inserts = append(likes_inserts, map[string]any{
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
				}, []db.Condition{
					{Column: "preference1_id", Value: min},
					{Column: "preference2_id", Value: max},
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
					if _, err := s.DBH.Insert(r.Context(), "shares", []map[string]any{
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
	}

	return &rest.Created{
		LastID:  result.LastID,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Org) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
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

		tk, err := bloqs_helpers.ExtractToken(w, r)

		if err != nil {
			return nil, err
		}

		p := bloqs_auth.READ_PROFILE
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

		claims := &bloqs_auth.Claims{}
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
		}, []db.Condition{{Column: "credential_id", Value: *id}})

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
			}, []db.Condition{{Column: "id", Value: id}})

			if err != nil {
				return
			}

			acc := res.Rows[0]

			res, err = s.DBH.Select(r.Context(), "account_likes", func() map[string]any {
				return map[string]any{
					"preference_id": new(int64),
					"weight":        new(float32),
				}
			}, []db.Condition{{Column: "account_id", Value: acc["id"]}})

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
		var where []db.Condition = []db.Condition{}
		if (id != nil) && (*id != "") {
            where = append(where, db.Condition{Column: "id", Value: *id})
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
		Type:    ORG_TYPE,
		Status:  uint16(status),
		Message: msg,
	}, err
}

func (Org) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Org) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}
