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
	if id > uint(len(db.data))-1 {
		return record, fmt.Errorf("%w: for id %d", ErrRecordNotFound, id)
	}
	return db.data[id], nil
}

func (db *Database) SelectAll() (records []records.Record) {
	db.RLock()
	defer db.RUnlock()
	return db.data
}

// ReplaceAll atomically replaces all records in the database with the provided records.
// This holds a write lock during the entire operation to ensure atomicity.
func (db *Database) ReplaceAll(records []records.Record) {
	db.Lock()
	defer db.Unlock()
	db.data = records
}
