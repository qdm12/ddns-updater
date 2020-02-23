package data

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/models"
)

func (db *database) GetEvents(domain, host string) (events []models.HistoryEvent, err error) {
	return db.persistentDB.GetEvents(domain, host)
}

func (db *database) Update(id int, record models.Record) error {
	db.Lock()
	defer db.Unlock()
	if id < 0 {
		return fmt.Errorf("id %d cannot be lower than 0", id)
	}
	if id > len(db.data)-1 {
		return fmt.Errorf("no record config found for id %d", id)
	}
	currentCount := len(db.data[id].History)
	newCount := len(record.History)
	db.data[id] = record
	// new IP address added
	if newCount > currentCount {
		if err := db.persistentDB.StoreNewIP(
			record.Settings.Domain,
			record.Settings.Host,
			record.History.GetCurrentIP(),
			record.History.GetSuccessTime(),
		); err != nil {
			return err
		}
	}
	return nil
}
