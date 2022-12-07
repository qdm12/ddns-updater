package health

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
)

func MakeIsHealthy(db AllSelecter, resolver LookupIPer) func() error {
	return func() (err error) {
		return isHealthy(db, resolver)
	}
}

var (
	ErrRecordUpdateFailed = errors.New("record update failed")
	ErrRecordIPNotSet     = errors.New("record IP not set")
	ErrLookupMismatch     = errors.New("lookup IP addresses do not match")
)

// isHealthy checks all the records were updated successfully and returns an error if not.
func isHealthy(db AllSelecter, resolver LookupIPer) (err error) {
	records := db.SelectAll()
	for _, record := range records {
		if record.Status == constants.FAIL {
			return fmt.Errorf("%w: %s", ErrRecordUpdateFailed, record.String())
		} else if record.Settings.Proxied() {
			continue
		}
		hostname := record.Settings.BuildDomainName()
		lookedUpIPs, err := resolver.LookupIP(context.Background(), "ip", hostname)
		if err != nil {
			return err
		}
		currentIP := record.History.GetCurrentIP()
		if currentIP == nil {
			return fmt.Errorf("%w: for hostname %s", ErrRecordIPNotSet, hostname)
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
			return fmt.Errorf("%w: %s instead of %s for %s",
				ErrLookupMismatch, strings.Join(lookedUpIPsString, ","), currentIP, hostname)
		}
	}
	return nil
}
