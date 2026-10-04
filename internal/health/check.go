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

var (
	ErrRecordUpdateFailed = errors.New("record update failed")
	ErrRecordIPNotSet     = errors.New("record IP not set")
	ErrLookupMismatch     = errors.New("lookup IP addresses do not match")
)

// isHealthy checks all the records were updated successfully and returns an
// error if not. Within a record TTL after its last update, previous IP
// addresses are also accepted since resolvers may not have propagated the
// update yet.
func isHealthy(ctx context.Context, db AllSelecter, resolver LookupIPer,
	now time.Time,
) (err error) {
	records := db.SelectAll()
	for _, record := range records {
		if record.Status == constants.FAIL {
			return fmt.Errorf("%w: %s", ErrRecordUpdateFailed, record.String())
		} else if record.Provider.Proxied() {
			continue
		}

		hostname := record.Provider.BuildDomainName()

		currentIP := record.History.GetCurrentIP()
		if !currentIP.IsValid() {
			return fmt.Errorf("%w: for hostname %s", ErrRecordIPNotSet, hostname)
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

		found := false
		lookedUpIPsString := make([]string, len(lookedUpNetIPs))
		for i, netIP := range lookedUpNetIPs {
			var ip netip.Addr
			switch {
			case netIP == nil:
			case netIP.To4() != nil:
				ip = netip.AddrFrom4([4]byte(netIP.To4()))
			default: // IPv6
				ip = netip.AddrFrom16([16]byte(netIP.To16()))
			}
			if slices.Contains(acceptedIPs, ip) {
				found = true
				break
			}
			lookedUpIPsString[i] = ip.String()
		}
		if !found {
			return fmt.Errorf("%w: %s instead of %s for %s",
				ErrLookupMismatch, strings.Join(lookedUpIPsString, ","), currentIP, hostname)
		}
	}
	return nil
}

// recordTTL returns the record TTL in seconds if the provider knows it.
func recordTTL(record records.Record) (ttl *uint32) {
	ttlProvider, ok := record.Provider.(ttlProvider)
	if !ok {
		return nil
	}
	return ttlProvider.TTL()
}
