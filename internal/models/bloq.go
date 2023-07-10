package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	bloqs_helpers "github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type Bloq struct {
}

func (Bloq) Table() string {
	return "bloq"
}

const BLOQ_TYPE = "Product"

// aggregateRating
// * category
// * hasAdultConsideration (review schema)
// * isRelatedTo
// * keywords
// * releaseDate
// * review
// * description
// * identifier
// * image
// * name
// * url

/*
/bloq/
/bloq/:id/
/bloq/:id/related
/bloq/:id/reviews
*/

func (Bloq) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "bloq",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`creator` INT UNSIGNED NOT NULL",
				"`category` INT UNSIGNED NOT NULL",
				"`hasAdultConsideration` BOOL DEFAULT 0",
				"`releaseDate` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`description` VARCHAR(140) NOT NULL",
				"`name` VARCHAR(80) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "bloq_related",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`bloq_id` INT UNSIGNED NOT NULL",
				"`related_id` INT UNSIGNED NOT NULL",
				"UNIQUE(`bloq_id`, `related_id`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "bloq_keywords",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`bloq_id` INT UNSIGNED NOT NULL",
				"`keyword` VARCHAR(182) NOT NULL",
				"UNIQUE(`bloq_id`, `keyword`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "bloq_review",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`itemReviewed` INT UNSIGNED NOT NULL",
				"`author` INT UNSIGNED NOT NULL",
				"`associatedReview` INT UNSIGNED DEFAULT NULL",
				"`reviewBody` TEXT DEFAULT NULL",
				"`reviewRating` INT NOT NULL",
				"`dateCreated` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`dateModified` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`inLanguage` TEXT NOT NULL",
				"UNIQUE(`itemReviewed`, `author`, `associatedReview`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "bloq_image",
			Columns: []string{
				"`bloq_id` INT UNSIGNED NOT NULL",
				"`image` VARCHAR(254)",
				"`changeTimestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"PRIMARY KEY(`bloq_id`)",
			},
		},
	}
}

func (h *Bloq) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (h *Bloq) CreateViews() []db.View {
	return []db.View{
		//	{
		//		Name:   "bloq_basic",
		//		Select: "SELECT `bloq`.*, `bloq_image`.`image` FROM `bloq` INNER JOIN `bloq_image` ON `bloq`.`id` = `bloq_image`.`bloq_id`;",
		//	},
	}
}

func (Bloq) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name                  string
		description           string
		category              int
		hasAdultConsideration = false
		image                 multipart.File
		image_header          *multipart.FileHeader
		keywords              []string = []string{}
		creator               int
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
		description = r.FormValue("description")
		var err error
		creator, err = strconv.Atoi(r.FormValue("creator"))
		if err != nil {
			return nil, err
		}
		category, err = strconv.Atoi(r.FormValue("category"))
		if err != nil {
			return nil, err
		}
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

		keywords = r.Form["keywords"]
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

	if l := len(description); l > 140 || l <= 0 {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: "`description` body field has to have a length between 1 and 140 characters",
		}, nil
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

	for _, v := range keywords {
		if l := len(v); l > 182 || l <= 0 {
			status = http.StatusUnprocessableEntity
			return &rest.Created{
				Status:  status,
				Message: fmt.Sprintf("`keyword` `%s` body field has to have a length between 1 and 182 characters", v),
			}, nil
		}
	}

	exists, err := PreferenceExists(r.Context(), int64(category), s)
	if err != nil {
		return nil, err
	}
	if !exists {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: fmt.Sprintf("`category` with id `%d` does not exist", category),
		}, nil
	}

	_, _, err = YourProfile(w, r, s, bloqs_auth.CREATE_BLOQ, int64(creator))
	if err != nil {
		return nil, err
	}

	result, err := s.DBH.Insert(r.Context(), "bloq", []map[string]any{
		{
			"name":                  name,
			"description":           description,
			"hasAdultConsideration": hasAdultConsideration,
			"category":              category,
			"creator":               creator,
		},
	})

	if err != nil {
		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	id := *result.LastID

	_, err = s.DBH.Insert(r.Context(), "bloq_image", []map[string]any{
		{
			"bloq_id": id,
			"image":   image,
		},
	})

	if err != nil {
		s.DBH.Delete(r.Context(), "bloq", map[string]any{"id": id})

		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	if len(keywords) != 0 {
		keywords_inserts := make([]map[string]any, 0, len(keywords))
		for _, keyword := range keywords {
			keywords_inserts = append(keywords_inserts, map[string]any{
				"bloq_id": id,
				"keyword": keyword,
			})
		}

		_, err = s.DBH.Insert(r.Context(), "bloq_keywords", keywords_inserts)

		if err != nil {
			s.DBH.Delete(r.Context(), "bloq", map[string]any{"id": id})
			s.DBH.Delete(r.Context(), "bloq_image", map[string]any{"bloq_id": id})

			status = http.StatusInternalServerError
			return nil, &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
		}
	}

	return &rest.Created{
		LastID:  &id,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Bloq) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)
	second := s.Seg(1)

	api := conf.MustGetConf("REST", "domain").(string)

	var (
		acc db.JSON
		err error
	)

	myself := conf.MustGetConfOrDefault("@", "REST", "myself")
	profile, err := strconv.Atoi(r.URL.Query().Get(myself))
	if err == nil {
		_, acc, err = YourProfile(w, r, s, bloqs_auth.NIL, int64(profile))
		if err != nil {
			return nil, err
		}
	}

	var where []db.Condition = make([]db.Condition, 0)
	if (id != nil) && (*id != "") {
		where = append(where, db.Condition{Column: "id", Value: *id})
	}

	if conf.MustGetConfOrDefault(false, "REST", "NSFW") {
		if acc != nil {
			if bloqs_helpers.FormValueTrue(r.URL.Query().Get("NSFW")) {
				where = append(where, db.Condition{
					Column: "hasAdultConsideration",
					Value:  true,
				})
			} else {
				where = append(where, db.Condition{
					Column: "hasAdultConsideration",
					Value:  *acc["hasAdultConsideration"].(*bool),
				})
			}
		}
	} else {
		where = append(where, db.Condition{
			Column: "hasAdultConsideration",
			Value:  false,
		})
	}

	category := r.URL.Query().Get("category")
	if category != "" {
		if v, err := strconv.Atoi(category); err != nil {
			where = append(where, db.Condition{Column: "category", Value: v})
		}
	}

	result, err := s.DBH.Select(r.Context(), "bloq", func() map[string]any {
		return map[string]any{
			"id":                    new(int64),
			"creator":               new(int64),
			"category":              new(int64),
			"name":                  new(string),
			"description":           new(string),
			"hasAdultConsideration": new(bool),
			"releaseDate":           new(string),
		}
	}, where)

	if err != nil || len(result.Rows) == 0 {
		return nil, err
	}

	if second == nil {
		for _, v := range result.Rows {
			id := *v["id"].(*int64)

			result, err := s.DBH.Select(r.Context(), "bloq_related", func() map[string]any {
				return map[string]any{"related_id": new(int64)}
			}, []db.Condition{{Column: "bloq_id", Value: id}})
			if err != nil {
				return nil, err
			}

			related := make([]db.JSON, 0, len(result.Rows)+1)
			related = append(related, db.JSON{"@type": "Product"})
			for _, i := range result.Rows {
				url := fmt.Sprintf("%s/bloq/%d", api, *i["related_id"].(*int64))
				related = append(related, db.JSON{"url": url})
			}
			v["related"] = related

			result, err = s.DBH.Select(r.Context(), "bloq_keywords", func() map[string]any {
				return map[string]any{"keyword": new(string)}
			}, []db.Condition{{Column: "bloq_id", Value: id}})
			if err != nil {
				return nil, err
			}

			keywords := make([]string, 0, len(result.Rows))
			for _, i := range result.Rows {
				keywords = append(keywords, *i["keyword"].(*string))
			}
			v["keywords"] = keywords

			v["reviews"] = fmt.Sprintf("%s/bloq/%d/reviews/", api, id)

			result, err = s.DBH.Select(r.Context(), "bloq_image", func() map[string]any {
				return map[string]any{"image": new(sql.NullString)}
			}, []db.Condition{{Column: "bloq_id", Value: id}})
			if err != nil {
				return nil, err
			}

			image := result.Rows[0]["image"].(*sql.NullString)
			if image.Valid {
				v["image"] = image.String
			}

			v["url"] = fmt.Sprintf("%s/bloq/%d", api, id)
		}

		status := http.StatusOK
		msg := ""
		if err != nil {
			status = http.StatusInternalServerError
			msg = err.Error()
		}

		return &rest.Resource{
			Models:  result.Rows,
			Type:    BLOQ_TYPE,
			Unique:  id != nil,
			Status:  uint16(status),
			Message: msg,
		}, err
	} else if *second == "related" {
		second_id := s.Seg(2)

		if second_id != nil {
			return nil, &mux.HttpError{}
		}

		result, err := s.DBH.Select(r.Context(), "bloq_related", func() map[string]any {
			return map[string]any{"related_id": new(int64)}
		}, []db.Condition{{Column: "bloq_id", Value: id}})
		if err != nil {
			return nil, err
		}

		related := make([]db.JSON, 0, len(result.Rows)+1)
		related = append(related, db.JSON{"@type": "Product"})
		for _, i := range result.Rows {
			url := fmt.Sprintf("%s/bloq/%d", api, *i["related_id"].(*int64))
			related = append(related, db.JSON{"url": url})
		}

		return &rest.Resource{
			Models: related,
			Type:   BLOQ_TYPE,
			Status: http.StatusOK,
			Unique: false,
		}, nil
	} else if *second == "reviews" {
		second_id := s.Seg(2)

		cols := map[string]any{
			"id":           new(int64),
			"author":       new(int64),
			"reviewBody":   new(string),
			"reviewRating": new(int8),
			"dateCreated":  new(string),
			"dateModified": new(string),
			"inLanguage":   new(string),
		}
		where := []db.Condition{{Column: "itemReviewed", Value: *id}}
		if second_id != nil {
			where = append(where, db.Condition{Column: "id", Value: *second_id})
			cols["associatedReview"] = new(int64)
			if third := s.Seg(3); third != nil && *third == "associated" {
				if s.Seg(4) != nil {
					return nil, &mux.HttpError{
						Status: http.StatusNotFound,
					}
				}
				cols = map[string]any{"id": new(int64)}
				where = append(where, db.Condition{Column: "associatedReview", Value: *id})
			}
		}

		result, err := s.DBH.Select(r.Context(), "bloq_review", func() map[string]any {
			return cols
		}, where)
		if err != nil {
			return nil, err
		}

		unique := second_id != nil && s.Seg(3) == nil

		related := make([]db.JSON, 0, len(result.Rows))
		if !unique {
			related = append(related, db.JSON{"@type": "Product"})
			for _, i := range result.Rows {
				i["url"] = fmt.Sprintf("%s/bloq/%d/reviews/%d", api, id, *i["related_id"].(*int64))
				i["associated"] = fmt.Sprintf("%s/bloq/%d/associated", api, id)
				related = append(related, i)
			}
		} else {
			if len(result.Rows) == 1 {
				i := result.Rows[0]
				i["url"] = fmt.Sprintf("%s/bloq/%d/reviews/%d", api, id, *i["related_id"].(*int64))
				i["associated"] = fmt.Sprintf("%s/bloq/%d/associated", api, id)
				related = append(related, i)
			}
		}

		return &rest.Resource{
			Models: related,
			Type:   "Review",
			Status: http.StatusOK,
			Unique: unique,
		}, nil
	}

	return nil, nil
}

func (Bloq) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Bloq) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func isProductCreator(ctx context.Context, product int64, creator int64, dbh db.DataManipulater) (bool, error) {
	res, err := dbh.Select(ctx, "bloq", func() map[string]any {
		return map[string]any{"creator": new(int64)}
	}, []db.Condition{
		{Column: "id", Value: product},
		{Column: "creator", Value: creator},
	})

	return len(res.Rows) == 1, err
}

func personMakesOffer(ctx context.Context, person db.JSON, dbh db.DataManipulater) (*rest.Resource, error) {
	if person == nil {
		return nil, errors.New("no person received passed")
	}

	var where []db.Condition = make([]db.Condition, 0, 2)
	where = append(where, db.Condition{
		Column: "creator",
		Value:  *person["id"].(*int64),
	})
	if !conf.MustGetConfOrDefault(false, "REST", "NSFW") {
		where = append(where, db.Condition{
			Column: "hasAdultConsideration",
			Value:  false,
		})
	}

	result, err := dbh.Select(ctx, "bloq", func() map[string]any {
		return map[string]any{
			"id":                    new(int64),
			"creator":               new(int64),
			"category":              new(int64),
			"name":                  new(string),
			"description":           new(string),
			"hasAdultConsideration": new(bool),
			"releaseDate":           new(string),
		}
	}, where)

	if err != nil {
		return nil, err
	}

	if len(result.Rows) < 1 {
		return &rest.Resource{
			Models: []db.JSON{},
			Status: http.StatusNotFound,
			Type:   BLOQ_TYPE,
		}, nil
	}

	api := conf.MustGetConf("REST", "domain").(string)
	for _, v := range result.Rows {
		id := *v["id"].(*int64)

		result, err := dbh.Select(ctx, "bloq_related", func() map[string]any {
			return map[string]any{"related_id": new(int64)}
		}, []db.Condition{{Column: "bloq_id", Value: id}})
		if err != nil {
			return nil, err
		}

		related := make([]db.JSON, 0, len(result.Rows)+1)
		related = append(related, db.JSON{"@type": "Product"})
		for _, i := range result.Rows {
			url := fmt.Sprintf("%s/bloq/%d", api, *i["related_id"].(*int64))
			related = append(related, db.JSON{"url": url})
		}
		v["related"] = related

		result, err = dbh.Select(ctx, "bloq_keywords", func() map[string]any {
			return map[string]any{"keyword": new(string)}
		}, []db.Condition{{Column: "bloq_id", Value: id}})
		if err != nil {
			return nil, err
		}

		keywords := make([]string, 0, len(result.Rows))
		for _, i := range result.Rows {
			keywords = append(keywords, *i["keyword"].(*string))
		}
		v["keywords"] = keywords

		v["reviews"] = fmt.Sprintf("%s/bloq/%d/reviews/", api, id)

		result, err = dbh.Select(ctx, "bloq_image", func() map[string]any {
			return map[string]any{"image": new(sql.NullString)}
		}, []db.Condition{{Column: "bloq_id", Value: id}})
		if err != nil {
			return nil, err
		}

		image := result.Rows[0]["image"].(*sql.NullString)
		if image.Valid {
			v["image"] = image.String
		}

		v["url"] = fmt.Sprintf("%s/bloq/%d", api, id)
	}

	status := http.StatusOK
	msg := ""
	if err != nil {
		status = http.StatusInternalServerError
		msg = err.Error()
	}

	return &rest.Resource{
		Models:  result.Rows,
		Type:    BLOQ_TYPE,
		Unique:  false,
		Status:  uint16(status),
		Message: msg,
	}, err
}
