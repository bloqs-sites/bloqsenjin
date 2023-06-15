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
	bloqs_auth "github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	"github.com/bloqs-sites/bloqsenjin/proto"
)

type BloqHandler struct {
}

func (BloqHandler) Table() string {
	return "bloq"
}

func (BloqHandler) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: "bloq",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`category` INT UNSIGNED NOT NULL",
				"`hasAdultConsideration` BOOL DEFAULT 0",
				"`description` VARCHAR(140) NOT NULL",
				"`name` VARCHAR(80) NOT NULL",
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
		{
			Name: "bloq_rating",
			Columns: []string{
				"`bloq_id` INT UNSIGNED NOT NULL",
				"`profile_id` INT UNSIGNED NOT NULL",
				"`rating` INT NOT NULL",
				"PRIMARY KEY(`bloq_id`, `profile_id`)",
			},
		},
		{
			Name: "bloq_keyword",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`bloq_id` INT UNSIGNED NOT NULL",
				"`keyword` VARCHAR(182) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (h *BloqHandler) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (h *BloqHandler) CreateViews() []db.View {
	return []db.View{
		//	{
		//		Name:   "bloq_basic",
		//		Select: "SELECT `bloq`.*, `bloq_image`.`image` FROM `bloq` INNER JOIN `bloq_image` ON `bloq`.`id` = `bloq_image`.`bloq_id`;",
		//	},
	}
}

func (m BloqHandler) Handle(w http.ResponseWriter, r *http.Request, s rest.RESTServer) error {
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

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		ctx := "https://schema.org/"
		typ := "Person"
		if ((s.SegLen() & 1) == 1) && (s.Seg(s.SegLen()-1) != nil) && (*s.Seg(s.SegLen() - 1) != "" && (r.URL.Path != "/account/@")) {
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

func (BloqHandler) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		name        string
		description string
		category    string
		adult                = "0"
		image       string   = "NULL"
		keywords    []string = []string{}
	)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, mux.FORM_DATA) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
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
		category = r.FormValue("category")
		adult := r.FormValue("hasAdultConsideration")
		if adult == "yes" || adult == "on" || adult == "1" || adult == "true" {
			adult = "1"
		}
		keywords = r.Form["keywords"]
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
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

	if l := len(description); l > 140 || l <= 0 {
		status = http.StatusUnprocessableEntity
		return &rest.Created{
			Status:  status,
			Message: "`description` body field has to have a length between 1 and 140 characters",
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

	result, err := s.DBH.Insert(r.Context(), "bloq", []map[string]string{
		{
			"name":                  name,
			"description":           description,
			"hasAdultConsideration": adult,
			"category":              category,
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

	_, err = s.DBH.Insert(r.Context(), "bloq_image", []map[string]string{
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
		keywords_inserts := make([]map[string]string, 0, len(keywords))
		for _, keyword := range keywords {
			keywords_inserts = append(keywords_inserts, map[string]string{
				"bloq_id": id,
				"keyword": keyword,
			})
		}

		_, err = s.DBH.Insert(r.Context(), "bloq_keyword", keywords_inserts)

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
		LastID:  result.LastID,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (BloqHandler) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)

	var where map[string]any = nil
	if (id != nil) && (*id != "") {
		where = map[string]any{"id": *id}
	}

	result, err := s.DBH.Select(r.Context(), "bloq", func() map[string]any {
		return map[string]any{
			"id":                    new(int64),
			"category":              new(int64),
			"name":                  new(string),
			"description":           new(string),
			"hasAdultConsideration": new(bool),
		}
	}, where)

	if err != nil {
		return nil, err
	}

	api := conf.MustGetConf("REST", "domain").(string)

	for _, v := range result.Rows {
		result, err := s.DBH.Select(r.Context(), "bloq", func() map[string]any {
			return map[string]any{"image": new(string)}
		}, map[string]any{"bloq_id": v["id"]})

		if err != nil {
			return nil, err
		}

		v["image"] = result.Rows[0]["image"]

		result, err = s.DBH.Select(r.Context(), "bloq_rating", func() map[string]any {
			return map[string]any{
                "profile_id": new(int64),
				"rating": new(string),
			}
		}, map[string]any{"bloq_id": v["id"]})

		if err != nil {
			return nil, err
		}

        for _, r := range result.Rows {
            r["profile"]  = fmt.Sprintf("%s/account/%d", api, *r["profile_id"].(*int64))
            delete(r, "profile_id")
        }

        v["ratings"] = result.Rows
	}

	for _, i := range result.Rows {
		i["url"] = fmt.Sprintf("%s/bloq/%d", api, *i["id"].(*int64))
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

func (BloqHandler) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (BloqHandler) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (h *BloqHandler) MapGenerator() func() map[string]any {
	return func() map[string]any {
		m := make(map[string]any)
		m["id"] = new(int64)
		m["name"] = new(string)
		m["description"] = new(string)
		m["category"] = new(int64)
		m["hasAdultConsideration"] = new(bool)
		m["image"] = new(*string)
		return m
	}
}
