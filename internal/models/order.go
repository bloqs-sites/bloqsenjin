package models

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	internal_helpers "github.com/bloqs-sites/bloqsenjin/internal/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	mux "github.com/bloqs-sites/bloqsenjin/pkg/http"
	"github.com/bloqs-sites/bloqsenjin/pkg/http/helpers"
	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
)

const (
	OrderTable = "order"
	OrderType  = "Order"
)

type Order struct{}

func (Order) Table() string {
	return OrderTable
}

func (Order) CreateTable() []db.Table {
	return []db.Table{
		{
			Name: OrderTable,
			Columns: []string{
				"`id` INT UNSIGNED AUTO_INCREMENT",
				"`acceptedOffer` INT UNSIGNED NOT NULL",
				"`customer` VARCHAR(320) NOT NULL",
				"PRIMARY KEY(`id`)",
			},
		},
	}
}

func (Order) CreateIndexes() []db.Index {
	return []db.Index{}
}

func (Order) CreateViews() []db.View {
	return []db.View{}
}

func (Order) Create(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Created, error) {
	var (
		status uint16 = http.StatusInternalServerError

		acceptedOffer int64
		quantity      = 1
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

		var err error
		acceptedOfferInt, err := strconv.Atoi(r.FormValue("acceptedOffer"))
		if err != nil {
			return nil, err
		}
		acceptedOffer = int64(acceptedOfferInt)
		qstr := r.FormValue("quantity")
		if qstr != "" {
			q, err := strconv.Atoi(qstr)
			if err == nil {
				quantity = q
			}
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

		var err error
		acceptedOfferInt, err := strconv.Atoi(r.FormValue("acceptedOffer"))
		if err != nil {
			return nil, err
		}
		acceptedOffer = int64(acceptedOfferInt)
		qstr := r.FormValue("quantity")
		if qstr != "" {
			q, err := strconv.Atoi(qstr)
			if err == nil {
				quantity = q
			}
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

	a, err := authSrv(r.Context())

	if err != nil {
		return nil, err
	}

	claims, err := internal_helpers.ValidateAndGetClaims(w, r, a, auth.CREATE_ORDER)
	if err != nil {
		return nil, err
	}

	for i := 0; i < quantity; i++ {
		_, err := s.DBH.Insert(r.Context(), OrderTable, []map[string]any{
			{
				"customer":      claims.Payload.Client,
				"acceptedOffer": acceptedOffer,
			},
		})
		if err != nil {
			status = http.StatusInternalServerError
			return nil, &mux.HttpError{
				Body:   err.Error(),
				Status: status,
			}
		}
	}

	return &rest.Created{
		Message: "",
		Status:  http.StatusCreated,
	}, nil
}

func (Order) Read(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	id := s.Seg(0)
	var where []db.Condition = make([]db.Condition, 0)
	if (id != nil) && (*id != "") {
		where = append(where, db.Condition{Column: "id", Value: *id})
	}

	if helpers.FormValueTrue(r.URL.Query().Get("myself")) {
		a, err := authSrv(r.Context())
		if err != nil {
			return nil, err
		}

		claims, err := internal_helpers.ValidateAndGetClaims(w, r, a, auth.DELETE_ORDER)
		if err != nil {
			return nil, err
		}

		where = append(where, db.Condition{
			Column: "customer",
			Value:  claims.Payload.Client,
		})
	}

	res, err := s.DBH.Select(r.Context(), OrderTable, func() map[string]any {
		return map[string]any{
            "id": new(int64),
            "acceptedOffer": new(int64),
        }
	}, where)

	status := http.StatusInternalServerError
	if err == nil {
		status = http.StatusOK
	}

	return &rest.Resource{
		Models: res.Rows,
		Type:   OrderType,
		Status: uint16(status),
		Unique: (id != nil) && (*id != ""),
	}, err
}

func (Order) Update(http.ResponseWriter, *http.Request, rest.RESTServer) (*rest.Resource, error) {
	return nil, &mux.HttpError{Status: http.StatusMethodNotAllowed}
}

func (Order) Delete(w http.ResponseWriter, r *http.Request, s rest.RESTServer) (*rest.Resource, error) {
	a, err := authSrv(r.Context())
	if err != nil {
		return nil, err
	}

	claims, err := internal_helpers.ValidateAndGetClaims(w, r, a, auth.DELETE_ORDER)
	if err != nil {
		return nil, err
	}

	where := map[string]any{"customer": claims.Payload.Client}
	id := s.Seg(0)
	if (id != nil) && (*id != "") {
		where["id"] = *id
	}

	err = s.DBH.Delete(r.Context(), OrderTable, where)

	return nil, err
}
