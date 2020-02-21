package healthcheck

import (
	"fmt"
	"net"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
)

// IsHealthy checks all the records were updated successfully and returns an error if not
func IsHealthy(db data.Database, lookupIP func(host string) ([]net.IP, error)) error {
	records := db.SelectAll()
	for _, record := range records {
		if record.Status == constants.FAIL {
			return fmt.Errorf("%s", record.String())
		} else if record.Settings.NoDNSLookup {
			continue
		}
		lookedUpIPs, err := lookupIP(record.Settings.BuildDomainName())
		if err != nil {
			return err
		}
		currentIP := record.History.GetCurrentIP()
		if currentIP == nil {
			return fmt.Errorf("no set IP address found")
		}
		for _, lookedUpIP := range lookedUpIPs {
			if lookedUpIP.Equal(currentIP) {
				return fmt.Errorf(
					"lookup IP address of %s is %s instead of %s",
					record.Settings.BuildDomainName(), lookedUpIP, currentIP)
			}
		}
	}
	return nil
}
