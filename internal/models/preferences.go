package models

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type PreferenceHandler struct {
}

func (p PreferenceHandler) Create(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	if err := r.ParseMultipartForm(1024); err != nil {
		return nil, err
	}

	dbh := *s.GetDB()

	dbh.Insert("preference", []map[string]string{
		{
			"name":        r.FormValue("name"),
			"description": r.FormValue("description"),
		},
	})

	return nil, nil
}

func (p PreferenceHandler) Read(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	dbh := *s.GetDB()

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) > 2 && len(parts[2]) > 0 {
		id, err := strconv.ParseInt(parts[2], 10, 0)

		if err != nil {
			return nil, err
		}

		res, err := dbh.Select("preference", p.MapGenerator())
		if err != nil {
			return nil, err
		}

		rows := res.Rows
		rn := len(rows)

		if rn < 1 {
			return rows, nil
		}

		json := make([]rest.JSON, 1)

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
				return json, nil
			}
		}

		return json, nil
	}

	res, err := dbh.Select("preference", p.MapGenerator())
	if err != nil {
		return nil, err
	}

	rows := res.Rows
	rn := len(rows)

	if rn < 1 {
		return rows, nil
	}

	json, i := make([]rest.JSON, len(rows)+1), 0

	json[i] = rest.JSON{
		"@context": "https://schema.org/",
	}

	for _, v := range rows {
		v["@type"] = "CategoryCode"

		i++
		json[i] = v
	}

	return json, nil
}

func (p PreferenceHandler) Update(*http.Request, rest.Server) ([]rest.JSON, error) {
	return nil, nil
}

func (p PreferenceHandler) Delete(*http.Request, rest.Server) ([]rest.JSON, error) {
	return nil, nil
}

func (p PreferenceHandler) Handle(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	switch r.Method {
	case "":
		fallthrough
	case http.MethodGet:
		return p.Read(r, s)
	case http.MethodPost:
		return p.Create(r, s)
	}

	return nil, errors.New(fmt.Sprint(http.StatusMethodNotAllowed))
}

func (p PreferenceHandler) CreateTable() []rest.Table {
	return []rest.Table{
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
	}
}

func (h *PreferenceHandler) CreateIndexes() []rest.Index {
	return []rest.Index{}
}

func (h *PreferenceHandler) CreateViews() []rest.View {
	return []rest.View{}
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
