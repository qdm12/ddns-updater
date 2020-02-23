package json

import (
	"fmt"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

// StoreNewIP stores a new IP address for a certain domain and host.
func (db *database) StoreNewIP(domain, host string, ip net.IP, t time.Time) (err error) {
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
// from oldest to newest
func (db *database) GetEvents(domain, host string) (events []models.HistoryEvent, err error) {
	db.RLock()
	defer db.RUnlock()
	for _, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			for _, event := range record.Events {
				events = append(events, event)
			}
			return events, nil
		}
	}
	return nil, fmt.Errorf("no record found for domain %q and host %q", domain, host)
}

// GetAllDomainsHosts gets all the domains and hosts from the database
func (db *database) GetAllDomainsHosts() (domainshosts []models.DomainHost, err error) {
	db.RLock()
	defer db.RUnlock()
	for _, record := range db.data.Records {
		domainshosts = append(domainshosts, models.DomainHost{
			Domain: record.Domain,
			Host:   record.Host,
		})
	}
	return domainshosts, nil
}
