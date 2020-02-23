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
			db.data.Records[i].IPs = append(db.data.Records[i].IPs, ipData{
				IP:   ip,
				Time: t,
			})
			return db.write()
		}
	}
	db.data.Records = append(db.data.Records, record{
		Domain: domain,
		Host:   host,
		IPs: []ipData{{
			IP:   ip,
			Time: t,
		}},
	})
	return db.write()
}

// GetIPs gets all the IP addresses history for a certain domain and host, in the order
// from oldest to newest
func (db *database) GetIPs(domain, host string) (ips []net.IP, successTime time.Time, err error) {
	db.RLock()
	defer db.RUnlock()
	for _, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			for _, ipData := range record.IPs {
				ips = append(ips, ipData.IP)
				successTime = ipData.Time // latest is the right one
			}
			return ips, successTime, nil
		}
	}
	return ips, successTime, fmt.Errorf("no record found for domain %q and host %q", domain, host)
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

// SetSuccessTime sets the latest successful update time for a particular domain, host.
func (db *database) SetSuccessTime(domain, host string, successTime time.Time) error {
	db.Lock()
	defer db.Unlock()
	for i, record := range db.data.Records {
		if record.Domain == domain && record.Host == host {
			L := len(db.data.Records[i].IPs)
			db.data.Records[i].IPs[L-1].Time = successTime
			return db.write()
		}
	}
	return fmt.Errorf("no record found for domain %q and host %q", domain, host)
}
