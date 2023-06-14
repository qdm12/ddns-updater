package data

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/records"
)

func (db *Database) GetEvents(domain, host string) (events []models.HistoryEvent, err error) {
	return db.persistentDB.GetEvents(domain, host)
}

func (db *Database) Update(id uint, record records.Record) (err error) {
	db.Lock()
	defer db.Unlock()
	if int(id) > len(db.data)-1 {
		return fmt.Errorf("%w: for id %d", ErrRecordNotFound, id)
	}
	currentCount := len(db.data[id].History)
	newCount := len(record.History)
	db.data[id] = record
	// new IP address added
	if newCount > currentCount {
		if err := db.persistentDB.StoreNewIP(
			record.Provider.Domain(),
			record.Provider.Host(),
			record.History.GetCurrentIP(),
			record.History.GetSuccessTime(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) Close() (err error) {
	db.Lock() // ensure write operation finishes
	defer db.Unlock()
	return db.persistentDB.Close()
}
