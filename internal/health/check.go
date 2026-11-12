package health

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/records"
)

// MakeIsHealthy returns a function checking all the records were updated
// successfully, returning an error if not.
func MakeIsHealthy(db AllSelecter, resolver LookupIPer,
	timeNow func() time.Time,
) func(ctx context.Context) error {
	return func(ctx context.Context) (err error) {
		return isHealthy(ctx, db, resolver, timeNow())
	}
}

// isHealthy checks all the records were updated successfully and returns an
// error listing every offending record if not. This tells how many records are
// affected, rather than just the first one.
func isHealthy(ctx context.Context, db AllSelecter, resolver LookupIPer,
	now time.Time,
) error {
	var errs []error
	for _, record := range db.SelectAll() {
		err := checkRecord(ctx, record, resolver, now)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// checkRecord returns an error telling why the record is not healthy, or nil if
// it is. Within a record TTL after its last update, previous IP addresses are
// also accepted since resolvers may not have propagated the update yet.
func checkRecord(ctx context.Context, record records.Record, resolver LookupIPer,
	now time.Time,
) error {
	switch record.Status {
	case constants.FAIL:
		return fmt.Errorf("record update failed: %s", record.String())
	case constants.PUBLICIPUNKNOWN:
		return fmt.Errorf("public IP address not found: %s", record.String())
	}

	if record.Provider.Proxied() {
		return nil
	}

	hostname := record.Provider.BuildDomainName()

	currentIP := record.History.GetCurrentIP()
	if !currentIP.IsValid() {
		return fmt.Errorf("record IP not set: for hostname %s", hostname)
	}

	acceptedIPs := []netip.Addr{currentIP}
	ttl := recordTTL(record)
	if ttl != nil && now.Before(record.History.GetSuccessTime().Add(time.Duration(*ttl)*time.Second)) {
		acceptedIPs = append(acceptedIPs, record.History.GetPreviousIPs()...)
	}

	lookedUpNetIPs, err := resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return err
	}

	lookedUpIPsString := make([]string, 0, len(lookedUpNetIPs))
	for _, netIP := range lookedUpNetIPs {
		if netIP == nil {
			continue
		}
		var ip netip.Addr
		if netIP.To4() != nil {
			ip = netip.AddrFrom4([4]byte(netIP.To4()))
		} else { // IPv6
			ip = netip.AddrFrom16([16]byte(netIP.To16()))
		}
		if slices.Contains(acceptedIPs, ip) {
			return nil
		}
		lookedUpIPsString = append(lookedUpIPsString, ip.String())
	}

	return fmt.Errorf("lookup IP addresses do not match: %s instead of %s for %s",
		strings.Join(lookedUpIPsString, ","), currentIP, hostname)
}

// recordTTL returns the record TTL in seconds if the provider knows it.
func recordTTL(record records.Record) (ttl *uint32) {
	ttlProvider, ok := record.Provider.(ttlProvider)
	if !ok {
		return nil
	}
	return ttlProvider.TTL()
}
