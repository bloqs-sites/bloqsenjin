package models

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type ItemAvailability = string

const (
	OfferTable        = "offers"
	ItemsOfferedTable = "offersItems"
	OfferType         = "Offer"

	BackOrder           ItemAvailability = "BackOrder"
	Discontinued        ItemAvailability = "Discontinued"
	InStock             ItemAvailability = "InStock"
	InStoreOnly         ItemAvailability = "InStoreOnly"
	LimitedAvailability ItemAvailability = "LimitedAvailability"
	OnlineOnly          ItemAvailability = "OnlineOnly"
	OutOfStock          ItemAvailability = "OutOfStock"
	PreOrder            ItemAvailability = "PreOrder"
	PreSale             ItemAvailability = "PreSale"
	SoldOute            ItemAvailability = "SoldOute"
)

var (
	ItemAvailabilities = []ItemAvailability{
		BackOrder,
		Discontinued,
		InStock,
		InStoreOnly,
		LimitedAvailability,
		OnlineOnly,
		OutOfStock,
		PreOrder,
		PreSale,
		SoldOute,
	}
)

type Offer struct{}

func (Offer) Table() string {
	return OfferTable
}

func (Offer) Type() string {
	return OfferType
}

func (Offer) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: OfferTable,
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				fmt.Sprintf("`availability` ENUM('%s') NOT NULL", strings.Join(ItemAvailabilities, "','")),
				"`availabilityStarts` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`availabilityEnds` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
				"`offeredBy` INT UNSIGNED NOT NULL",
				"`price` DOUBLE NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
		{
			Name: ItemsOfferedTable,
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`offers` INT UNSIGNED NOT NULL",
				"`item` INT UNSIGNED NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (Offer) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (Offer) CreateViews() []db.View {
	return []db.View{}
}

func (Offer) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		availability       *string
		availabilityStarts time.Time
		availabilityEnds   time.Time
		offeredBy          int64
		price              float32
		itemsOffered       []int64
	)

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, helpers.X_WWW_FORM_URLENCODED) {
		if err := r.ParseForm(); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", helpers.X_WWW_FORM_URLENCODED, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		availabilityStr := r.FormValue("availability")
		if availabilityStr != "" {
			availability = &availabilityStr
		}
		var err error
		availabilityStarts, err = time.Parse(time.RFC3339, r.FormValue("availabilityStarts"))
		if err != nil {
			return nil, err
		}
		availabilityEnds, err = time.Parse(time.RFC3339, r.FormValue("availabilityEnds"))
		if err != nil {
			return nil, err
		}
		offeredByInt, err := strconv.Atoi(r.FormValue("creator"))
		if err != nil {
			return nil, err
		}
		offeredBy = int64(offeredByInt)
		price64, err := strconv.ParseFloat(r.FormValue("price"), 32)
		if err != nil {
			return nil, err
		}
		price = float32(price64)
		itemsOffered = make([]int64, 0, len(r.Form["itemsOffered"]))
		for _, i := range r.Form["itemsOffered"] {
			int, err := strconv.Atoi(i)
			if err != nil {
				return nil, err
			}
			itemsOffered = append(itemsOffered, int64(int))
		}
	} else if strings.HasPrefix(ct, helpers.FORM_DATA) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			status = http.StatusBadRequest
			return &rest.Created{
					Status:  status,
					Message: fmt.Sprintf("the HTTP request body could not be parsed as `%s`:\t%s", helpers.FORM_DATA, err),
				}, &mux.HttpError{
					Body:   err.Error(),
					Status: status,
				}
		}

		availabilityStr := r.FormValue("availability")
		if availabilityStr != "" {
			availability = &availabilityStr
		}
		var err error
		availabilityStarts, err = time.Parse(time.RFC3339, r.FormValue("availabilityStarts"))
		if err != nil {
			return nil, err
		}
		availabilityEnds, err = time.Parse(time.RFC3339, r.FormValue("availabilityEnds"))
		if err != nil {
			return nil, err
		}
		offeredByInt, err := strconv.Atoi(r.FormValue("creator"))
		if err != nil {
			return nil, err
		}
		offeredBy = int64(offeredByInt)
		price64, err := strconv.ParseFloat(r.FormValue("price"), 32)
		if err != nil {
			return nil, err
		}
		price = float32(price64)
		itemsOffered = make([]int64, 0, len(r.Form["itemsOffered"]))
		for _, i := range r.Form["itemsOffered"] {
			int, err := strconv.Atoi(i)
			if err != nil {
				return nil, err
			}
			itemsOffered = append(itemsOffered, int64(int))
		}
	} else {
		status = http.StatusUnsupportedMediaType
		h := w.Header()
		helpers.Append(&h, "Accept", helpers.X_WWW_FORM_URLENCODED)
		helpers.Append(&h, "Accept", helpers.FORM_DATA)
		return &rest.Created{
			Status:  status,
			Message: fmt.Sprintf("request has the usupported media type `%s`", ct),
		}, nil
	}

	if availability != nil {
		valid := false
		for _, i := range ItemAvailabilities {
			if i == *availability {
				valid = true
				break
			}
		}
		if !valid && *availability != "" {
			return nil, &mux.HttpError{
				Body:   "invalid availability value",
				Status: http.StatusBadRequest,
			}
		}
	} else {
		defaultv := ""
		availability = &defaultv
	}

	if price < 0 {
		return nil, &mux.HttpError{
			Body:   "price lower that 0",
			Status: http.StatusBadRequest,
		}
	}

	today := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)

	if availabilityStarts.Before(today) {
		return nil, &mux.HttpError{
			Body:   "availability start date already passed",
			Status: http.StatusBadRequest,
		}
	}

	if availabilityEnds.Before(availabilityStarts) {
		return nil, &mux.HttpError{
			Body:   "availability end date happend before availability start date",
			Status: http.StatusBadRequest,
		}
	}

	_, _, err := YourProfile(w, r, s, auth.CREATE_OFFER, offeredBy)
	if err != nil {
		return nil, err
	}

	if len(itemsOffered) < 1 {
		return nil, &mux.HttpError{
			Body:   "No items offered",
			Status: http.StatusUnprocessableEntity,
		}
	}

	for _, i := range itemsOffered {
		valid, err := isProductCreator(r.Context(), i, offeredBy, s.DBH)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, &mux.HttpError{
				Body:   fmt.Sprintf("Item with id `%d` it's not yours", i),
				Status: http.StatusBadRequest,
			}
		}
	}

	result, err := s.DBH.Insert(r.Context(), OfferTable, []map[string]any{
		{
			"availability":       availability,
			"availabilityStarts": availabilityStarts,
			"availabilityEnds":   availabilityEnds,
			"offeredBy":          offeredBy,
			"price":              price,
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

	offers := make([]map[string]any, 0, len(itemsOffered))
	for _, i := range itemsOffered {
		offers = append(offers, map[string]any{"offers": id, "item": i})
	}
	_, err = s.DBH.Insert(r.Context(), ItemsOfferedTable, offers)
	if err != nil {
		s.DBH.Delete(r.Context(), OfferTable, map[string]any{"id": id})

		status = http.StatusInternalServerError
		return nil, &mux.HttpError{
			Body:   err.Error(),
			Status: status,
		}
	}

	return &rest.Created{
		LastID:  &id,
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Offer) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	product := r.URL.Query().Get("product")

	where := []db.Condition{}
	id := s.Seg(0)
	if (id != nil) && (*id != "") {
		where = append(where, db.Condition{Column: "id", Value: *id})
	}

	if product != "" {
		product_id, err := strconv.ParseInt(product, 10, 64)
		if err != nil {
			return nil, err
		}

		where = append(where, db.Condition{Column: "item", Value: product_id})
	}

	res, err := s.DBH.Select(r.Context(), ItemsOfferedTable,
		func() map[string]any {
			return map[string]any{"offers": new(int64)}
		}, where)

	results := make([]db.JSON, 0, len(res.Rows))
	api := conf.MustGetConf("REST", "domain").(string)
	for _, i := range res.Rows {
		id := *i["offers"].(*int64)
		now := time.Now()
		res, err := s.DBH.Select(r.Context(), OfferTable,
			func() map[string]any {
				return map[string]any{
					"id":                 new(int64),
					"availability":       new(ItemAvailability),
					"availabilityStarts": new(string),
					"availabilityEnds":   new(string),
					"offeredBy":          new(int64),
					"price":              new(float32),
				}
			}, []db.Condition{
				{Column: "id", Value: id},
				{Column: "availabilityStarts", Op: db.LE, Value: now},
				{Column: "availabilityEnds", Op: db.GE, Value: now},
			})
		if err != nil {
			return nil, err
		}
		for _, o := range res.Rows {
			res, err = s.DBH.Select(r.Context(), ItemsOfferedTable,
				func() map[string]any {
					return map[string]any{"offers": new(int64)}
				}, []db.Condition{{Column: "item", Value: *o["id"].(*int64)}})

			if err != nil {
				return nil, err
			}

			for _, i := range res.Rows {
				i["href"] = fmt.Sprintf("%s/bloq/%d", api, i["offers"])
			}

			o["itemsOffered"] = append([]db.JSON{{
				"@context": "https://schema.org/",
				"@type":    "Product",
			}}, res.Rows...)
			results = append(results, o)
		}
	}

	return &rest.Resource{
		Models: results,
		Type:   OfferType,
		Status: http.StatusOK,
	}, err
}

func (Offer) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Offer) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}
