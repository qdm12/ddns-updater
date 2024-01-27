package json

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
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

// GetEvents gets all the IP addresses history for a certain domain, host and
// IP version, in the order from oldest to newest.
func (db *Database) GetEvents(domain, host string,
	ipVersion ipversion.IPVersion) (events []models.HistoryEvent, err error) {
	db.RLock()
	defer db.RUnlock()
	for _, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			return filterEvents(record.Events, ipVersion), nil
		}
	}
	return nil, nil
}

func filterEvents(events []models.HistoryEvent, ipVersion ipversion.IPVersion) (filteredEvents []models.HistoryEvent) {
	filteredEvents = make([]models.HistoryEvent, 0, len(events))
	for _, event := range events {
		switch ipVersion {
		case ipversion.IP4:
			if event.IP.Is4() {
				filteredEvents = append(filteredEvents, event)
			}
		case ipversion.IP6:
			if event.IP.Is6() {
				filteredEvents = append(filteredEvents, event)
			}
		case ipversion.IP4or6:
			filteredEvents = append(filteredEvents, event)
		default:
			panic(fmt.Sprintf("IP version %v is not supported", ipVersion))
		}
	}
	return filteredEvents
}
