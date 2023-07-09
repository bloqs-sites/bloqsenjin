package models

import "github.com/bloqs-sites/bloqsenjin/pkg/db"

var (
	profileTable = db.Table{
		Name:    "profilesTODO",
		Columns: []string{},
	}
)

func USE() db.Table {
	return profileTable
}
