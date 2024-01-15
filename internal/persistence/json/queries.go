package json

import (
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

// StoreNewIP stores a new IP address for a certain domain and host.
func (db *Database) StoreNewIP(domain, host string, ip netip.Addr, t time.Time) (err error) {
	db.Lock()
	defer db.Unlock()

	targetIndex := -1
	for i, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			targetIndex = i
			break
		}
	}

	recordNotFound := targetIndex == -1
	if recordNotFound {
		db.data.Records = append(db.data.Records, record{
			Domain: domain,
			Host:   host,
		})
		targetIndex = len(db.data.Records) - 1
	}

	event := models.HistoryEvent{
		IP:   ip,
		Time: t,
	}
	db.data.Records[targetIndex].Events = append(db.data.Records[targetIndex].Events, event)
	return db.write()
}

// GetEvents gets all the IP addresses history for a certain domain and host, in the order
// from oldest to newest.
func (db *Database) GetEvents(domain, host string) (events []models.HistoryEvent, err error) {
	db.RLock()
	defer db.RUnlock()
	for _, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			return append(events, record.Events...), nil
		}
	}
	return nil, nil
}
