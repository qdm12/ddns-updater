package update

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func ipVersionToIPKind(version ipversion.IPVersion) (kind string) {
	if version == ipversion.IP4or6 {
		return "IP"
	}
	return version.String()
}

func recordToLogString(record records.Record) string {
	return fmt.Sprintf("%s (%s)",
		record.Provider.BuildDomainName(),
		record.Provider.IPVersion())
}

func (s *Service) logDebugNoLookupSkip(hostname, ipKind string, lastIP, ip netip.Addr) {
	s.logger.Debug(fmt.Sprintf("Last %s address stored for %s is %s and your %s address"+
		" is %s, skipping update", ipKind, hostname, lastIP, ipKind, ip))
}

func (s *Service) logInfoNoLookupUpdate(hostname, ipKind string, lastIP, ip netip.Addr) {
	s.logger.Info(fmt.Sprintf("Last %s address stored for %s is %s and your %s address is %s",
		ipKind, hostname, lastIP, ipKind, ip))
}

func (s *Service) logDebugLookupSkip(hostname, ipKind string, recordIP, ip netip.Addr) {
	s.logger.Debug(fmt.Sprintf("%s address of %s is %s and your %s address"+
		" is %s, skipping update", ipKind, hostname, recordIP, ipKind, ip))
}

func (s *Service) logInfoLookupUpdate(hostname, ipKind string, recordIP, ip netip.Addr) {
	s.logger.Info(fmt.Sprintf("%s address of %s is %s and your %s address  is %s",
		ipKind, hostname, recordIP, ipKind, ip))
}

type joinedErrors struct { //nolint:errname
	errs []error
}

func (e *joinedErrors) Error() string {
	errorMessages := make([]string, len(e.errs))
	for i := range e.errs {
		errorMessages[i] = e.errs[i].Error()
	}
	return strings.Join(errorMessages, ", ")
}

func (e *joinedErrors) Unwrap() []error {
	return e.errs
}
