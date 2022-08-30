package data

import (
	"errors"
	"fmt"

	"github.com/qdm12/ddns-updater/internal/records"
)

var ErrRecordNotFound = errors.New("record not found")

func (db *Database) Select(id uint) (record records.Record, err error) {
	db.RLock()
	defer db.RUnlock()
	if int(id) > len(db.data)-1 {
		return record, fmt.Errorf("%w: for id %d", ErrRecordNotFound, id)
	}
	return db.data[id], nil
}

func (db *Database) SelectAll() (records []records.Record) {
	db.RLock()
	defer db.RUnlock()
	return db.data
}
