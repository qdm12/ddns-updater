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
	for i, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			db.data.Records[i].Events = append(db.data.Records[i].Events, models.HistoryEvent{
				IP:   ip,
				Time: t,
			})
			return db.write()
		}
	}
	db.data.Records = append(db.data.Records, record{
		Domain: domain,
		Host:   host,
		Events: []models.HistoryEvent{{
			IP:   ip,
			Time: t,
		}},
	})
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
