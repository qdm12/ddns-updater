package data

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/records"
)

func (db *Database) Select(id uint) (record records.Record, err error) {
	db.RLock()
	defer db.RUnlock()
	if id > uint(len(db.data))-1 {
		return record, fmt.Errorf("record not found: for id %d", id)
	}
	return db.data[id], nil
}

func (db *Database) SelectAll() (records []records.Record) {
	db.RLock()
	defer db.RUnlock()
	return db.data
}
