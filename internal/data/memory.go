package data

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/records"
)

func (db *database) Select(id int) (record records.Record, err error) {
	db.RLock()
	defer db.RUnlock()
	if id < 0 {
		return record, fmt.Errorf("id %d cannot be lower than 0", id)
	}
	if id > len(db.data)-1 {
		return record, fmt.Errorf("no record config found for id %d", id)
	}
	return db.data[id], nil
}

func (db *database) SelectAll() []records.Record {
	db.RLock()
	defer db.RUnlock()
	if db.data == nil {
		db.data = make([]records.Record, 0)
	}
	return db.data
}
