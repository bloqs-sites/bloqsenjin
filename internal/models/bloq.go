package models

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type BloqHandler struct {
}

func (h *BloqHandler) Create(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}

    defer r.MultipartForm.RemoveAll()

    bloqrow := make(map[string]string)

    bloqrow["name"] = r.MultipartForm.Value["name"][0]

    if len(bloqrow["name"]) > 80 {
        return nil, fmt.Errorf("Bloq name provided is too big")
    }

    bloqrow["description"] = r.MultipartForm.Value["description"][0]

    if len(bloqrow["description"]) > 140 {
        return nil, fmt.Errorf("Bloq description provided is too big")
    }

    bloqrow["category"] = r.MultipartForm.Value["category"][0]

    if _, err := strconv.ParseUint(bloqrow["category"], 10, 0); err != nil  {
        return nil, fmt.Errorf("Category ID `%s` is invalid", bloqrow["category"])
    }

    bloqrow["hasAdultConsideration"] = r.MultipartForm.Value["hasAdultConsideration"][0]

    p18, err := strconv.ParseBool(bloqrow["hasAdultConsideration"]);

    if err != nil  {
        p18 = false
    }

    if p18 {
        bloqrow["hasAdultConsideration"] = "1"
    } else {
        bloqrow["hasAdultConsideration"] = "0"
    }

	dbh := *s.GetDB()

    res, err := dbh.Insert("bloq", []map[string]string{ bloqrow })

    if err != nil || res.LastID == nil {
        return nil, fmt.Errorf("Internal error")
    }

    bloqirow := make(map[string]string)

    bloqirow["bloq"] = strconv.FormatInt(*res.LastID, 10)

    img := r.MultipartForm.File["image"][0]

    if !strings.HasPrefix(img.Header.Get("Content-Type"), "image/") {
        return nil, fmt.Errorf("File provided has not a Content-Type image/*")
    }

    if img.Size > (32 << 20) {
        return nil, fmt.Errorf("Image to big")
    }

    bloqirow["image"] = img.Filename

	bloqirow["changeTimestamp"] = strconv.FormatInt(time.Now().Unix(), 10)

	dbh.Insert("bloq_image", []map[string]string{ bloqirow })

    bloqkrow := make(map[string]string)

    bloqkrow["bloq"] = strconv.FormatInt(*res.LastID, 10)

    for _, k := range r.MultipartForm.Value["keyword"] {
        bloqkrow["keyword"] = k
	    dbh.Insert("bloq_keyword", []map[string]string{ bloqkrow })
    }

    /* Saves image somewhere like Cloudflare R2, IPFS, idk */

	return res.Rows, nil
}

func (h *BloqHandler) Read(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	dbh := *s.GetDB()

	res, err := dbh.Select("preference", h.MapGenerator())
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

func (h *BloqHandler) Update(*http.Request, rest.Server) ([]rest.JSON, error) {
	return nil, nil
}

func (h *BloqHandler) Delete(*http.Request, rest.Server) ([]rest.JSON, error) {
	return nil, nil
}

func (h *BloqHandler) Handle(r *http.Request, s rest.Server) ([]rest.JSON, error) {
	switch r.Method {
	case "":
		fallthrough
	case http.MethodGet:
		return h.Read(r, s)
	case http.MethodPost:
		return h.Create(r, s)
	}

	return nil, errors.New(fmt.Sprint(http.StatusMethodNotAllowed))
}

func (h *BloqHandler) CreateTable() []rest.Table {
	return []rest.Table{
		{
			Name: "bloq",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`category` INT UNSIGNED NOT NULL",
				"`hasAdultConsideration` BOOL DEFAULT 0",
                "`description` VARCHAR(140) NOT NULL",
                "`name` VARCHAR(80) NOT NULL",
				"UNIQUE(`name`)",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: "bloq_image",
			Columns: []string{
				"`bloq` INT UNSIGNED NOT NULL",
                "`image` VARCHAR(254)",
				"`changeTimestamp` INT NOT NULL",
				"PRIMARY KEY(`bloq`)",
			},
		},
		{
			Name: "bloq_rating",
			Columns: []string{
				"`bloq` INT UNSIGNED NOT NULL",
				"`client` INT UNSIGNED NOT NULL",
				"`rating` INT NOT NULL",
				"PRIMARY KEY(`bloq`, `client`)",
			},
		},
		{
			Name: "bloq_keyword",
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`bloq` INT UNSIGNED NOT NULL",
				"`keyword` VARCHAR(182) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (h *BloqHandler) CreateIndexes() []rest.Index {
    return []rest.Index{}
}

func (h *BloqHandler) CreateViews() []rest.View {
    return []rest.View{}
}

func (h *BloqHandler) MapGenerator() func() map[string]any {
	return func() map[string]any {
		m := make(map[string]any)
		m["id"] = new(int64)
		m["description"] = new(string)
		m["name"] = new(string)
		return m
	}
}
