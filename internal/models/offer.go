package models

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

type ItemAvailability = string

const (
	OfferTable        = "offers"
	ItemsOfferedTable = "offers"
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
				"`offer` INT UNSIGNED NOT NULL",
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
		if !valid {
			return nil, &mux.HttpError{}
		}
	}

	if price < 0 {
		return nil, &mux.HttpError{}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	if availabilityStarts.Before(today) {
		return nil, &mux.HttpError{}
	}

	if availabilityEnds.Before(availabilityStarts) {
		return nil, &mux.HttpError{}
	}

	_, _, err := YourProfile(w, r, s, auth.CREATE_OFFER, offeredBy)
	if err != nil {
		return nil, err
	}

	if len(itemsOffered) < 1 {
		return nil, err
	}

	for _, i := range itemsOffered {
		valid, err := isProductCreator(r.Context(), i, offeredBy, s.DBH)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, &mux.HttpError{}
		}
	}

	println(20)
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

	offers := make([]map[string]any, len(itemsOffered))
	for _, i := range itemsOffered {
		offers = append(offers, map[string]any{"offer": id, "item": i})
	}
	_, err = s.DBH.Insert(r.Context(), ItemsOfferedTable, offers)
	if err != nil {
		s.DBH.Delete(r.Context(), "offer", map[string]any{"id": id})

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
	return nil, nil
}

func (Offer) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}

func (Offer) Delete(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, nil
}
