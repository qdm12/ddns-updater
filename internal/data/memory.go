package data

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/records"
)

func (db *database) Insert(record records.Record) (id int) {
	db.Lock()
	defer db.Unlock()
	db.data = append(db.data, record)
	return len(db.data) - 1
}

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

func (db *database) SelectAll() (records []records.Record) {
	db.RLock()
	defer db.RUnlock()
	return db.data
}
