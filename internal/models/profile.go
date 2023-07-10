package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	uri "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	bloqs_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type Profile struct {
}

func (Profile) Table() string {
	return "profile"
}

func (Profile) Type() string {
	return "Person"
}

func (Profile) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "profile",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`name` VARCHAR(80) NOT NULL",
				"`description` VARCHAR(140) DEFAULT NULL",
				"`honorificPrefix` VARCHAR(80) DEFAULT NULL",
				"`honorificSuffix` VARCHAR(80) DEFAULT NULL",
				"`image` VARCHAR(254) DEFAULT NULL",
				"`url` VARCHAR(255) DEFAULT NULL",
				"`hasAdultConsideration` BOOL DEFAULT 0",
				"`level` INT UNSIGNED DEFAULT 0",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "credential_profiles",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`credential_id` VARCHAR(320) NOT NULL",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`birthDate` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"UNIQUE (`credential_id`, `profile_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "profile_languages",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`language` VARCHAR(255) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "profile_likes",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`preference_id` INT UNSIGNED NOT NULL",
				"`weight` FLOAT(7, 3) UNSIGNED NOT NULL",
				"UNIQUE (`profile_id`, `preference_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "profile_follows",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`follower_id` INT UNSIGNED NOT NULL",
				"UNIQUE (`profile_id`, `follower_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (Profile) CreateIndexes() []db.Index {
	return nil
}

func (Profile) CreateViews() []db.View {
	var sql strings.Builder

	sql.WriteString("SELECT")
	sql.WriteString(" `profile`.*, COUNT(`profile_follows`.`profile_id`) AS `followers`, COUNT(`profile_follows`.`follower_id`) AS `following`")
	sql.WriteString(" FROM `profile` INNER JOIN `profile_follows` ON `profile`.`id` = `profile_follows`.`profile_id`")
	sql.WriteString(" GROUP BY `profile`.`id`")

	return []db.View{
		{
			Name:   "profile_view",
			Select: sql.String(),
		},
	}
}

func (Profile) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name                  string
		description           *string = nil
		image                 multipart.File
		image_header          *multipart.FileHeader
		url                   *string  = nil
		hasAdultConsideration          = false
		likes                 []string = []string{}
	)

	nsfw := conf.MustGetConfOrDefault(false, "REST", "NSFW")

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, bloqs_helpers.FORM_DATA) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
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
		description_value := r.FormValue("description")
		if description_value != "" {
			description = &description_value
		}
		url_value := r.FormValue("url")
		if url_value != "" {
			url = &url_value
		}

		var err error
		image, image_header, err = r.FormFile("image")

		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", bloqs_helpers.FORM_DATA, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		if image != nil {
			defer image.Close()
		}

		if nsfw {
			hasAdultConsideration = bloqs_helpers.FormValueTrue(r.FormValue("hasAdultConsideration"))
		}

		likes = r.Form["likes"]
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
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

	if description != nil {
		if l := len(*description); l > 140 || l < 0 {
			status = http.StatusUnprocessableEntity
			return &rest.Created{
				Status:  status,
				Message: "`description` body field has to have a length between 0 and 140 characters",
			}, nil
		}
	}

	if url != nil {
		if _, err := uri.ParseRequestURI(*url); err != nil {
			status = http.StatusUnprocessableEntity
			return &rest.Created{
					Status:  status,
					Message: err.Error(),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}
	}

	if image_header != nil {
		if ct := image_header.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
			status = http.StatusUnprocessableEntity
			return &rest.Created{
				Status:  status,
				Message: "`image` it's not really a `image/*`",
			}, nil
		}
	}

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	claims, err := helpers.ValidateAndGetClaims(w, r, a, bloqs_auth.CREATE_PROFILE)
	if err != nil {
		return nil, err
	}

	var result db.Result
	result, err = s.DBH.Select(r.Context(), "credential_profiles", func() map[string]any {
		return nil
	}, []db.Condition{{Column: "credential_id", Value: claims.Payload.Client}})
	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	max := conf.MustGetConfOrDefault[float64](1, "REST", "profiles", "max")
	if len(result.Rows) >= int(max) {
		status = http.StatusForbidden
		return nil, &mux.HttpError{
			Body:   fmt.Sprintf("the maximum limit of this resource (%d) has reached.", int(max)),
			Status: status,
		}
	}

	insert := map[string]any{
		"name":                  name,
		"description":           description,
		"url":                   url,
		"hasAdultConsideration": hasAdultConsideration,
	}
	if image_header != nil {
		insert["image"] = image_header.Filename
	}

	result, err = s.DBH.Insert(r.Context(), "profile", []map[string]any{insert})

	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	id := strconv.Itoa(int(*result.LastID))

	_, err = s.DBH.Insert(r.Context(), "credential_profiles", []map[string]any{
		{
			"credential_id": claims.Payload.Client,
			"profile_id":    id,
		},
	})

	if err != nil {
		s.DBH.Delete(r.Context(), "profile", map[string]any{"id": id})

		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	if len(likes) != 0 {
		likes_inserts := make([]map[string]any, 0, len(likes))
		weight := 1000 / len(likes)
		for _, like := range likes {
			likes_inserts = append(likes_inserts, map[string]any{
				"profile_id":    id,
				"preference_id": like,
				"weight":        weight,
			})
		}

		_, err = s.DBH.Insert(r.Context(), "profile_likes", likes_inserts)

		if err != nil {
			s.DBH.Delete(r.Context(), "profile", map[string]any{"id": id})
			s.DBH.Delete(r.Context(), "credential_profiles", map[string]any{
				"credential_id": claims.Payload.Client,
				"account_id":    id,
			})

			status = http.StatusInternalServerError
			return nil, &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
		}

		for n := 0; n < len(likes); n++ {
			for m := n + 1; m < len(likes); m++ {
				i, err := strconv.Atoi(likes[n])
				if err != nil {
					status = http.StatusInternalServerError
					return nil, &mux.HttpError{
						Body:   err.Error(),
						Status: status,
					}
				}
				j, err := strconv.Atoi(likes[m])
				if err != nil {
					status = http.StatusInternalServerError
					return nil, &mux.HttpError{
						Body:   err.Error(),
						Status: status,
					}
				}

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
					w := res.Rows[0]["weight"].(*float32)

					s.DBH.Update(r.Context(), "shares", map[string]any{
						"weight": *w + 1.0,
					}, map[string]any{
						"id": res.Rows[0]["id"],
					})
				} else {
					s.DBH.Insert(r.Context(), "shares", []map[string]any{
						{
							"preference1_id": min,
							"preference2_id": max,
							"weight":         1,
						},
					})
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

func (Profile) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)
	second := s.Seg(1)

	you := conf.MustGetConfOrDefault("@", "REST", "myself")
	api := conf.MustGetConf("REST", "domain").(string)
	// nsfw := conf.MustGetConfOrDefault(false, "REST", "nsfw")

	var (
		result db.Result
		err    error
	)

	if second == nil {
		if id != nil && *id == you {
			a, err := authSrv(r.Context())
			if err != nil {
				return nil, err
			}

			claims, err := helpers.ValidateAndGetClaims(w, r, a, bloqs_auth.READ_PROFILE)
			if err != nil {
				return nil, err
			}

			client := claims.Payload.Client

			res, err := s.DBH.Select(r.Context(), "credential_profiles", func() map[string]any {
				return map[string]any{
					"profile_id": new(int64),
					"birthDate":  new(string),
				}
			}, []db.Condition{{Column: "credential_id", Value: client}})
			if err != nil {
				return nil, err
			}

			var wait sync.WaitGroup
			wait.Add(len(res.Rows))
			accs := make([]db.JSON, 0, len(res.Rows))
			search := func(id int64, birthDate string) {
				defer wait.Done()

				var res db.Result
				//res, err = s.DBH.Select(r.Context(), "profile_view", func() map[string]any {
				res, err = s.DBH.Select(r.Context(), "profile", func() map[string]any {
					return map[string]any{
						"id":                    new(int64),
						"name":                  new(string),
						"description":           new(sql.NullString),
						"image":                 new(sql.NullString),
						"url":                   new(sql.NullString),
						"hasAdultConsideration": new(bool),
						"level":                 new(uint8),
						//					"followers":             new(uint64),
						//					"following":             new(uint64),
					}
				}, []db.Condition{{Column: "id", Value: id}})
				if err != nil {
					return
				}

				if len(res.Rows) > 0 {
					accs = append(accs, personalAccount(r.Context(), res.Rows[0], birthDate, s))
				}
			}

			for _, i := range res.Rows {
				go search(*i["profile_id"].(*int64), *i["birthDate"].(*string))
			}

			wait.Wait()

			result = db.Result{Rows: accs}
		} else {
			var where []db.Condition = nil
			cols := map[string]any{
				"id":          new(int64),
				"name":        new(string),
				"description": new(sql.NullString),
				"image":       new(sql.NullString),
				"url":         new(sql.NullString),
				"level":       new(uint8),
			}

			var birthDate *string = nil

			a, err := authSrv(r.Context())
			if err != nil {
				return nil, err
			}

			if (id != nil) && (*id != "") {
				where = append(where, db.Condition{Column: "id", Value: *id})
			}

			claims, err := helpers.ValidateAndGetClaims(w, r, a, bloqs_auth.NIL)
			if err == nil {
				res, err := s.DBH.Select(r.Context(), "credential_profiles", func() map[string]any {
					return map[string]any{
						"profile_id": new(int64),
						"birthDate":  new(string),
					}
				}, []db.Condition{
					{Column: "credential_id", Value: claims.Payload.Client},
					{Column: "profile_id", Value: id},
				})

				if err == nil && res.Rows[0] != nil {
					birthDate = res.Rows[0]["birthDate"].(*string)
					cols["hasAdultConsideration"] = new(bool)
					//				cols["followers"] = new(uint64)
					//				cols["following"] = new(uint64)
				}
			}

			//result, err = s.DBH.Select(r.Context(), "profile_view", func() map[string]any {
			result, err = s.DBH.Select(r.Context(), "profile", func() map[string]any {
				return cols
			}, where)

			if err == nil && birthDate != nil && result.Rows[0] != nil {
				result.Rows[0] = personalAccount(r.Context(), result.Rows[0], *birthDate, s)
			}
		}

		for _, i := range result.Rows {
			i["href"] = fmt.Sprintf("%s/profile/%d", api, *i["id"].(*int64))

			if v := i["description"].(*sql.NullString); v.Valid {
				i["description"] = v.String
			} else {
				i["description"] = nil
			}

			if v := i["image"].(*sql.NullString); v.Valid {
				i["image"] = v.String
			} else {
				i["image"] = nil
			}

			if v := i["url"].(*sql.NullString); v.Valid {
				i["url"] = v.String
			} else {
				i["url"] = nil
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
			Type:    "Person",
			Unique:  id != nil && *id != you,
			Status:  uint16(status),
			Message: msg,
		}, err
	} else if *second == "makesOffer" {
		id, err := strconv.Atoi(*id)
		if err != nil {
			return nil, err
		}

		_, acc, err := YourProfile(w, r, s, bloqs_auth.NIL, int64(id))
		if err != nil {
			return nil, err
		}

		return personMakesOffer(r.Context(), acc, s.DBH)
	}

	return nil, nil
}

func (Profile) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Profile) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func calcProfileLvL(creation_date time.Time) uint8 {
	now := time.Now()

	diff := now.Sub(creation_date).Hours()

	x1 := conf.MustGetConfOrDefault[float64](3, "REST", "profiles", "level", "levelUp", "first", "value")
	switch conf.MustGetConfOrDefault("month", "REST", "profiles", "level", "levelUp", "first", "type") {
	case "day":
		x1 *= 24
	case "month":
		x1 *= 24 * 30
	case "year":
		x1 *= 24 * 30 * 12
	}
	var y1 float64 = 1

	x2 := conf.MustGetConfOrDefault[float64](6, "REST", "profiles", "level", "levelUp", "another", "value")
	switch conf.MustGetConfOrDefault("month", "REST", "profiles", "level", "levelUp", "another", "type") {
	case "day":
		x1 *= 24
	case "month":
		x1 *= 24 * 30
	case "year":
		x1 *= 24 * 30 * 12
	}
	y2 := conf.MustGetConfOrDefault[float64](2, "REST", "profiles", "level", "levelUp", "another", "level")

	a := (y1 - y2) / (math.Log2(x1) - math.Log2(x2))
	k := math.Exp2(y1/a) / x1

	lvl := uint8(math.Floor(a*math.Log2(k*diff))) + 1
	max := conf.MustGetConfOrDefault[uint8](8, "REST", "profiles", "level", "max")

	if lvl > max {
		return max
	}

	return lvl
}

func calcProfileLvLByString(creation_date_str string) uint8 {
	date, err := time.Parse(time.DateTime, creation_date_str)

	if err != nil {
		return 0
	}

	return calcProfileLvL(date)
}

func personalAccount(ctx context.Context, acc db.JSON, birthDate string, s rest.RESTServer) db.JSON {
	api := conf.MustGetConf("REST", "domain").(string)

	id := *acc["id"].(*int64)

	if lvl := calcProfileLvLByString(birthDate); lvl > *acc["level"].(*uint8) {
		if err := s.DBH.Update(ctx, "profile", map[string]any{
			"level": lvl,
		}, map[string]any{"id": id}); err != nil {
			fmt.Printf("%v\n", err.Error())
		} else {
			acc["level"] = lvl
		}
	}

	res, err := s.DBH.Select(ctx, "profile_likes", func() map[string]any {
		return map[string]any{
			"preference_id": new(int64),
			"weight":        new(float32),
		}
	}, []db.Condition{{Column: "profile_id", Value: id}})
	if err != nil {
		fmt.Printf("%v\n", err)
		return acc
	}

	for _, i := range res.Rows {
		i["id"] = i["preference_id"]
		i["url"] = fmt.Sprintf("%s/preference/%d", api, *i["preference_id"].(*int64))
		i["@type"] = "CategoryCode"
		delete(i, "preference_id")
	}

	acc["likes"] = res.Rows

	following := make(map[string]any, 2)
	following["size"] = acc["following"]
	following["url"] = fmt.Sprintf("%s/profile/%d/following", api, id)
	acc["following"] = following

	followers := make(map[string]any, 2)
	followers["size"] = acc["followers"]
	followers["url"] = fmt.Sprintf("%s/profile/%d/followers", api, id)
	acc["followers"] = followers

	return acc
}

func YourProfile(w http.ResponseWriter, r *http.Request, s rest.RESTServer, p bloqs_auth.Permission, you int64) (*bloqs_auth.Claims, db.JSON, error) {
	a, err := authSrv(r.Context())

	if err != nil {
		return nil, nil, err
	}

	claims, err := helpers.ValidateAndGetClaims(w, r, a, p)
	if err != nil {
		return nil, nil, err
	}

	res, err := s.DBH.Select(r.Context(), "credential_profiles", func() map[string]any {
		return map[string]any{
			"profile_id": new(int64),
			"birthDate":  new(string),
		}
	}, []db.Condition{
		{Column: "credential_id", Value: claims.Payload.Client},
		{Column: "profile_id", Value: you},
	})

	if err != nil || res.Rows[0] == nil {
		return nil, nil, &mux.HttpError{
			Body:   "Can't create a bloq for a profile that isn't yours.",
			Status: http.StatusUnauthorized,
		}
	}

	birthDate := *res.Rows[0]["birthDate"].(*string)

	res, err = s.DBH.Select(r.Context(), "profile", func() map[string]any {
		return map[string]any{
			"id":                    new(int64),
			"name":                  new(string),
			"description":           new(sql.NullString),
			"image":                 new(sql.NullString),
			"url":                   new(sql.NullString),
			"hasAdultConsideration": new(bool),
			"level":                 new(uint8),
		}
	}, []db.Condition{{Column: "id", Value: you}})
	if err != nil {
		return claims, nil, err
	}

	if len(res.Rows) != 1 {
		return claims, nil, nil
	}

	return claims, personalAccount(r.Context(), res.Rows[0], birthDate, s), nil
}
