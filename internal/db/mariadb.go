package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	_ "github.com/go-sql-driver/mysql"
)

type MySQL struct {
	conn *sql.DB
}

func NewMySQL(ctx context.Context, dsn string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	dbh := &MySQL{
		conn: db,
	}

	if err != nil {
		return dbh, err
	}

	if err := db.PingContext(ctx); err != nil {
		return dbh, fmt.Errorf("the DSN specified might be invalid. Could not connect to the DB:\t%s", err)
	}

	//db.SetConnMaxLifetime(time.Minute * 3)
	//db.SetMaxOpenConns(10)
	//db.SetMaxIdleConns(10)

	return dbh, nil
}

func (dbh *MySQL) Select(ctx context.Context, table string, columns func() map[string]any, where []db.Condition) (res db.Result, err error) {
	r := make([]db.JSON, 0)

	res.Rows = r

	column := columns()
	cl := len(column)
	if cl < 1 {
		return
	}

	keys := make([]string, 0, cl)
	for k := range column {
		keys = append(keys, k)
	}

	conditions := make([]string, 0, len(where))
	vals := make([]any, 0, len(conditions))
	for _, v := range where {
		k := v.Column
		switch v.Op {
		case db.EQ:
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", k))
		case db.NE:
			conditions = append(conditions, fmt.Sprintf("`%s` != ?", k))
		case db.GT:
			conditions = append(conditions, fmt.Sprintf("`%s` > ?", k))
		case db.GE:
			conditions = append(conditions, fmt.Sprintf("`%s` >= ?", k))
		case db.LT:
			conditions = append(conditions, fmt.Sprintf("`%s` < ?", k))
		case db.LE:
			conditions = append(conditions, fmt.Sprintf("`%s` <= ?", k))
		}
		vals = append(vals, v.Value)
	}

	var rows *sql.Rows

	if len(where) > 0 {
		var stmt *sql.Stmt
		stmt, err = dbh.conn.PrepareContext(ctx, fmt.Sprintf("SELECT %s FROM `%s` WHERE %s;", strings.Join(keys, ", "), table, strings.Join(conditions, " AND ")))
		if err != nil {
			return
		}
		defer stmt.Close()

		rows, err = stmt.QueryContext(ctx, vals...)
	} else {
		rows, err = dbh.conn.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM `%s`;", strings.Join(keys, ", "), table))
	}

	defer rows.Close()
	if err != nil {
		fmt.Printf("%#v\t%#v\n", res, err)
		return
	}

	for rows.Next() {
		loopc := columns()

		vals := make([]any, 0, len(column))

		for _, v := range keys {
			vals = append(vals, loopc[v])
		}

		if err = rows.Scan(vals...); err != nil {
			return
		}

		row := make(db.JSON, len(column))

		i := 0
		for _, v := range keys {
			row[v] = vals[i]
			i++
		}

		r = append(r, row)
	}

	res.Rows = r

	err = rows.Err()

	return
}

func (dbh *MySQL) Insert(ctx context.Context, table string, rows []map[string]any) (db.Result, error) {
	if len(rows) < 1 {
		return db.Result{
			LastID: nil,
			Rows:   nil,
		}, errors.New("no rows to be inserted")
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

	rowsvals, i := make([][]any, len(rows)), 0
	for _, r := range rows {
		rowsvals[i] = make([]any, len(columns))
		for j, c := range columns {
			v, ok := r[c]

			if !ok {
				//rowsvals[i][j] = "DEFAULT"
				//rowsvals[i][j] = "NULL"
				//continue
				return db.Result{
					LastID: nil,
					Rows:   nil,
				}, errors.New("cannot find value for column")
			}

			rowsvals[i][j] = v
		}
		i++
	}

	rowsstr := make([]string, len(rowsvals))
	vals, i := make([]any, len(rowsvals)*len(columns)), 0
	for j, r := range rowsvals {
		var rowstr strings.Builder
		rowstr.WriteString("(")
		first := true
		for _, v := range r {
			vals[i] = v
			i++
			if first {
				rowstr.WriteString("?")
				first = false
				continue
			}
			rowstr.WriteString(", ?")
		}
		rowstr.WriteString(")")
		rowsstr[j] = rowstr.String()
	}

	stmt := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES %s", table, strings.Join(columns, "`, `"), strings.Join(rowsstr, ", "))

	res, err := dbh.conn.ExecContext(ctx, stmt, vals...)

	if err == nil {
		last, lasterr := res.LastInsertId()

		if lasterr != nil {
			return db.Result{
				LastID: nil,
				Rows:   nil,
			}, err
		}

		return db.Result{
			LastID: &last,
			Rows:   nil,
		}, err
	}

	return db.Result{
		LastID: nil,
		Rows:   nil,
	}, err
}

func (dbh *MySQL) Update(ctx context.Context, table string, assignments map[string]any, conditions map[string]any) error {
	if assignments == nil || len(assignments) < 1 {
		return errors.New("no assignments")
	}

	var stmt strings.Builder
	stmt.WriteString("UPDATE `")
	stmt.WriteString(table)

	vals := make([]any, 0, len(assignments)+len(conditions))

	set := make([]string, 0, len(assignments))
	for k, v := range assignments {
		set = append(set, fmt.Sprintf("`%s`=?", k))
		vals = append(vals, v)
	}
	stmt.WriteString("` SET ")
	stmt.WriteString(strings.Join(set, ", "))

	if len(conditions) > 0 {
		where := make([]string, 0, len(conditions))
		for k, v := range conditions {
			where = append(where, fmt.Sprintf("`%s`=?", k))
			vals = append(vals, v)
		}
		stmt.WriteString(" WHERE ")
		stmt.WriteString(strings.Join(where, " AND "))
	}

	stmt.WriteString(";")
	_, err := dbh.conn.ExecContext(ctx, stmt.String(), vals...)
	return err
}

func (dbh *MySQL) Delete(ctx context.Context, table string, conditions map[string]any) error {
	var stmt strings.Builder
	stmt.WriteString("DELETE FROM `")
	stmt.WriteString(table)

	vals := make([]any, 0, len(conditions))
	if len(conditions) > 0 {
		where := make([]string, 0, len(conditions))
		for k, v := range conditions {
			where = append(where, fmt.Sprintf("`%s`=?", k))
			vals = append(vals, v)
		}
		stmt.WriteString("` WHERE ")
		stmt.WriteString(strings.Join(where, " AND "))
	}

	stmt.WriteString(";")
	_, err := dbh.conn.ExecContext(ctx, stmt.String(), vals...)
	return err
}

func (dbh *MySQL) CreateTables(ctx context.Context, ts []db.Table) error {
	for _, t := range ts {
		if _, err := dbh.conn.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s`(%s);", t.Name, strings.Join(t.Columns, ", "))); err != nil {
			return err
		}
	}

	return nil
}

func (dbh *MySQL) CreateIndexes(context.Context, []db.Index) error {
	return nil
}

func (dbh *MySQL) CreateViews(ctx context.Context, ts []db.View) error {
	for _, t := range ts {
		_, err := dbh.conn.ExecContext(ctx, fmt.Sprintf("CREATE OR REPLACE VIEW `%s` AS %s;",
			t.Name, t.Select))

		if err != nil {
			return err
		}
	}

	return nil
}

func (dbh MySQL) DropTables(ctx context.Context, tables []db.Table) error {
	for _, i := range tables {
		if _, err := dbh.conn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", i.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (dbh *MySQL) Close() error {
	return dbh.conn.Close()
}
