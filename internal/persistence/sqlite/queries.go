package sqlite

import (
	"fmt"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
)

/* Access to SQLite is NOT thread safe so we use a mutex */

// StoreNewIP stores a new IP address for a certain
// domain and host.
func (db *database) StoreNewIP(domain, host string, ip net.IP) (err error) {
	db.Lock()
	defer db.Unlock()
	// Inserts new IP
	_, err = db.sqlite.Exec(
		`INSERT INTO updates_ips(domain,host,ip,t_new,t_last)
		VALUES(?, ?, ?, ?, ?, ?);`,
		domain,
		host,
		ip.String(),
		time.Now(),
		time.Now(), // unneeded but it's hard to modify tables in sqlite
	)
	return err
}

// GetIPs gets all the IP addresses history for a certain domain and host, in the order
// from oldest to newest
func (db *database) GetIPs(domain, host string) (ips []net.IP, successTime time.Time, err error) {
	db.Lock()
	defer db.Unlock()
	rows, err := db.sqlite.Query(
		`SELECT ip, t_new
		FROM updates_ips
		WHERE domain = ? AND host = ?
		ORDER BY t_new ASC`,
		domain,
		host,
	)
	if err != nil {
		return nil, successTime, err
	}
	defer func() {
		closeErr := rows.Close()
		if err != nil {
			err = fmt.Errorf("%s, %s", err, closeErr)
		} else {
			err = closeErr
		}
	}()
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip, &successTime); err != nil {
			return nil, successTime, err
		}
		ips = append(ips, net.ParseIP(ip))
	}
	if err := rows.Err(); err != nil {
		return nil, successTime, err
	}
	return ips, successTime, nil
}

// GetAllDomainsHosts gets all domains and hosts from the database
func (db *database) GetAllDomainsHosts() (domainshosts []models.DomainHost, err error) {
	db.Lock()
	defer db.Unlock()
	rows, err := db.sqlite.Query(`SELECT DISTINCT domain, host FROM updates_ips`)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err != nil {
			err = fmt.Errorf("%s, %s", err, closeErr)
		} else {
			err = closeErr
		}
	}()
	for rows.Next() {
		domainHost := models.DomainHost{}
		if err := rows.Scan(&domainHost.Domain, &domainHost.Host); err != nil {
			return nil, err
		}
		domainshosts = append(domainshosts, domainHost)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return domainshosts, nil
}

// SetSuccessTime sets the latest successful update time for a particular domain, host.
func (db *database) SetSuccessTime(domain, host string, successTime time.Time) error {
	return fmt.Errorf("not implemented") // no plan to migrate back to sqlite
}
