package persistence

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/models"
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
		domain string
		host   string
		events []models.HistoryEvent
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
		events, err := source.GetEvents(domain, host)
		if err != nil {
			return err
		}
		rows = append(rows, row{domain, host, events})
	}

	for _, r := range rows {
		for _, event := range r.events {
			destination.StoreNewIP(r.domain, r.host, event.IP, event.Time)
		}
	}
	return destination.Check()
}
