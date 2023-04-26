package db

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
)

type Auth struct {
	basic_user        string
	basic_pass        string
	auth_header       string
	auth_header_value string
}

type D1 struct {
	url    string
	auther Auth
}

func NewD1(url url.URL, auther Auth) D1 {
	return D1{
		url:    url.String(),
		auther: auther,
	}
}

func (dbh *D1) Select(ctx context.Context, table string, columns func() map[string]any) (db.Result, error) {
	r := make([]db.JSON, 0)

	column := columns()

	cl := len(column)
	if cl < 1 {
		return db.Result{
			Rows: r,
		}, nil
	}

	i, keys := 0, make([]string, cl)
	for k := range column {
		keys[i] = k
		i++
	}

	res, err := dbh.pull(
		ctx,
		http.MethodGet,
		strings.Join(append([]string{dbh.url, "DML", table}, keys...), "/"),
		bytes.NewBuffer(make([]byte, 0)),
	)

	if err != nil {
		return db.Result{
			Rows: r,
		}, err
	}

	defer res.Body.Close()

	if res.Header.Get("Content-Type") != "application/json" {
		return db.Result{
			Rows: r,
		}, errors.New("Unexpected response from the database")
	}

	json.NewDecoder(res.Body).Decode(r)

	return db.Result{
		Rows: r,
	}, nil
}

func (dbh *D1) Insert(ctx context.Context, table string, rows []map[string]string) (db.Result, error) {
	if len(rows) < 1 {
		return db.Result{}, errors.New("No rows to be inserted")
	}

	set := make(map[string]bool, len(rows[0]))
	for _, r := range rows {
		for c := range r {
			set[c] = true
		}
	}
	columns, i := make([]string, len(set)), 0
	for c := range set {
		columns[i] = c
		i++
	}

	type body struct {
		columns []string
		rows    []map[string]string
	}
	var buf = bytes.NewBuffer(make([]byte, 0))
	json.NewEncoder(buf).Encode(body{
		columns: columns,
		rows:    rows,
	})
	res, err := dbh.pull(ctx, http.MethodPut, strings.Join([]string{dbh.url, "DML", table}, "/"), buf)
	if err != nil {
		return db.Result{}, err
	}

	if status := res.StatusCode; status < 200 || status >= 300 {
		return db.Result{}, errors.New("Could not create tables.")
	}

	return db.Result{}, nil
}

func (dbh *D1) Update(table string, assignments []map[string]any, conditions []map[string]any) (db.Result, error) {
	return db.Result{}, nil
}

func (dbh *D1) Delete(table string, conditions []map[string]any) (db.Result, error) {
	return db.Result{}, nil
}

func (dbh *D1) CreateTables(ctx context.Context, ts []db.Table) error {
	var buf = bytes.NewBuffer(make([]byte, 0))
	json.NewEncoder(buf).Encode(ts)
	res, err := dbh.pull(ctx, http.MethodPost, strings.Join([]string{dbh.url, "DDL", "table"}, "/"), buf)
	if err != nil {
		return err
	}

	if status := res.StatusCode; status < 200 || status > 299 {
		return errors.New("Could not create tables.")
	}

	return nil
}

func (dbh *D1) CreateIndexes(ts []db.Index) error {
	return nil
}

func (dbh *D1) CreateViews(ts []db.View) error {
	return nil
}

func (dbh *D1) pull(ctx context.Context, url, method string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, url, method, body)
	if err != nil {
		panic(err)
	}

	req.Header.Set(dbh.auther.auth_header, dbh.auther.auth_header_value)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", dbh.auther.basic_user, dbh.auther.basic_pass)))))

	return client.Do(req)
}
