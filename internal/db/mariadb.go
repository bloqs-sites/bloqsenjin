package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/rest"
	_ "github.com/go-sql-driver/mysql"
)

type MariaDB struct {
	conn *sql.DB
}

func NewMariaDB(dsn string) MariaDB {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return MariaDB{
		conn: db,
	}
}

func (dbh MariaDB) Select(table string, columns func() map[string]any) (rest.Result, error) {
	r := make([]rest.JSON, 0)

	column := columns()

	cl := len(column)
	if cl < 1 {
		return rest.Result{
			LastID: nil,
			Rows:   r,
		}, nil
	}

	i, keys := 0, make([]string, cl)
	for k := range column {
		keys[i] = k
		i++
	}

	rows, err := dbh.conn.Query(fmt.Sprintf("SELECT %s FROM `%s`;", strings.Join(keys, ", "), table))

	if err != nil {
		return rest.Result{
			LastID: nil,
			Rows:   r,
		}, err
	}

	defer rows.Close()

	if err != nil {
		return rest.Result{
			LastID: nil,
			Rows:   r,
		}, err
	}

	for rows.Next() {
		column := columns()

		vals := make([]any, len(column))

		i := 0
		for _, v := range column {
			vals[i] = v
			i++
		}

		if err := rows.Scan(vals...); err != nil {
			return rest.Result{
				LastID: nil,
				Rows:   r,
			}, err
		}

		row := make(rest.JSON, len(column))

		i = 0
		for k := range column {
			row[k] = vals[i]
			i++
		}

		r = append(r, row)
	}

	return rest.Result{
		LastID: nil,
		Rows:   r,
	}, rows.Err()
}

func (dbh MariaDB) Insert(table string, rows []map[string]any) (rest.Result, error) {
	r := make([]rest.JSON, 0)
	return rest.Result{
		LastID: nil,
		Rows:   r,
	}, nil
}

func (dbh MariaDB) Update(table string, assignments []map[string]any, conditions []map[string]any) (rest.Result, error) {
	r := make([]rest.JSON, 0)
	return rest.Result{
		LastID: nil,
		Rows:   r,
	}, nil
}

func (dbh MariaDB) Delete(table string, conditions []map[string]any) (rest.Result, error) {
	r := make([]rest.JSON, 0)
	return rest.Result{
		LastID: nil,
		Rows:   r,
	}, nil
}

func (dbh MariaDB) CreateTables(ts []rest.Table) error {
	for _, t := range ts {
        _, err := dbh.conn.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s`(%s);",
			t.Name, strings.Join(t.Columns, ", ")))

        if err != nil {
            return err
        }
	}

	return nil
}
