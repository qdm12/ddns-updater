package persistence

import (
	"fmt"
	"net"
	"time"

	"github.com/qdm12/golibs/logging"
)

func Migrate(source, destination Database, logger logging.Logger) (err error) {
	defer func() {
		closeErr := source.Close()
		if err != nil {
			err = fmt.Errorf("%s, %s", err, closeErr)
		} else {
			err = closeErr
		}
	}()

	type row struct {
		domain      string
		host        string
		ips         []net.IP
		successTime time.Time
	}
	var rows []row

	domainshosts, err := source.GetAllDomainsHosts()
	if err != nil {
		return err
	}
	logger.Info("Migrating %d domain-host tuples", len(domainshosts))

	for i := range domainshosts {
		domain := domainshosts[i].Domain
		host := domainshosts[i].Host
		ips, successTime, err := source.GetIPs(domain, host)
		if err != nil {
			return err
		}
		rows = append(rows, row{domain, host, ips, successTime})
	}

	for _, r := range rows {
		for _, ip := range r.ips {
			destination.StoreNewIP(r.domain, r.host, ip)
		}
		destination.SetSuccessTime(r.domain, r.host, r.successTime)
	}
	return nil
}
