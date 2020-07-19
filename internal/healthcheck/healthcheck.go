package healthcheck

import (
	"fmt"
	"net"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/golibs/logging"
)

type lookupIPFunc func(host string) ([]net.IP, error)

// IsHealthy checks all the records were updated successfully and returns an error if not
func IsHealthy(db data.Database, lookupIP lookupIPFunc, logger logging.Logger) (err error) {
	defer func() {
		if err != nil {
			logger.Warn("unhealthy: %s", err)
		}
	}()
	records := db.SelectAll()
	for _, record := range records {
		if record.Status == constants.FAIL {
			return fmt.Errorf("%s", record.String())
		} else if !record.Settings.DNSLookup() {
			continue
		}
		hostname := record.Settings.BuildDomainName()
		lookedUpIPs, err := lookupIP(hostname)
		if err != nil {
			return err
		}
		currentIP := record.History.GetCurrentIP()
		if currentIP == nil {
			return fmt.Errorf("no database set IP address found for %s", hostname)
		}
		found := false
		lookedUpIPsString := make([]string, len(lookedUpIPs))
		for i, lookedUpIP := range lookedUpIPs {
			lookedUpIPsString[i] = lookedUpIP.String()
			if lookedUpIP.Equal(currentIP) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("lookup IP addresses for %s are %s instead of %s", hostname, strings.Join(lookedUpIPsString, ","), currentIP)
		}
	}
	return nil
}
