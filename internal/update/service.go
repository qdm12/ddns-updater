package update

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Service struct {
	period    time.Duration
	db        Database
	updater   UpdaterInterface
	cooldown  time.Duration
	resolver  LookupIPer
	ipGetter  PublicIPFetcher
	logger    Logger
	timeNow   func() time.Time
	hioClient HealthchecksIOClient

	// Service lifecycle
	runCancel   context.CancelFunc
	done        <-chan struct{}
	force       chan struct{}
	forceResult chan []error
}

func NewService(db Database, updater UpdaterInterface, ipGetter PublicIPFetcher,
	period time.Duration, cooldown time.Duration, logger Logger, resolver LookupIPer,
	timeNow func() time.Time, hioClient HealthchecksIOClient,
) *Service {
	return &Service{
		period:      period,
		db:          db,
		updater:     updater,
		force:       make(chan struct{}),
		forceResult: make(chan []error),
		cooldown:    cooldown,
		resolver:    resolver,
		ipGetter:    ipGetter,
		logger:      logger,
		timeNow:     timeNow,
		hioClient:   hioClient,
	}
}

func (s *Service) lookupIPsResilient(ctx context.Context, hostname string, tries uint) (
	ipv4, ipv6 []netip.Addr, err error,
) {
	type result struct {
		network string
		ips     []netip.Addr
		err     error
	}
	results := make(chan result)
	networks := []string{"ip4", "ip6"}
	lookupCtx, cancel := context.WithCancel(ctx)
	for _, network := range networks {
		go func(ctx context.Context, network string, results chan<- result) {
			errs := make([]error, 0, tries)
			for range tries {
				ips, err := s.resolver.LookupNetIP(ctx, network, hostname)
				if err != nil {
					if strings.HasSuffix(err.Error(), "no such host") {
						results <- result{network: network} // no IP address for this network
						return
					}
					errs = append(errs, err)
					continue // retry
				}
				results <- result{network: network, ips: ips, err: err}
				return
			}
			results <- result{
				network: network,
				err:     fmt.Errorf("ip look up failed after %d tries: %w", tries, errors.Join(errs...)),
			}
		}(lookupCtx, network, results)
	}

	for range networks {
		result := <-results
		if result.err != nil {
			if err == nil {
				cancel()
				err = fmt.Errorf("looking up %s addresses: %w", result.network, result.err)
			}
			continue
		}
		switch result.network {
		case "ip4":
			ipv4 = result.ips
		case "ip6":
			ipv6 = result.ips
		}
	}
	cancel()

	return ipv4, ipv6, err
}

func doIPVersion(records []librecords.Record) (doIP, doIPv4, doIPv6 bool) {
	for _, record := range records {
		switch record.Provider.IPVersion() {
		case ipversion.IP4or6:
			doIP = true
		case ipversion.IP4:
			doIPv4 = true
		case ipversion.IP6:
			doIPv6 = true
		}
		if doIP && doIPv4 && doIPv6 {
			return true, true, true
		}
	}
	return doIP, doIPv4, doIPv6
}

func (s *Service) getNewIPs(ctx context.Context, doIP, doIPv4, doIPv6 bool) (
	ip, ipv4, ipv6 netip.Addr, errors []error,
) {
	var err error
	if doIP {
		ip, err = tryAndRepeatGettingIP(ctx, s.ipGetter.IP, s.logger, ipversion.IP4or6)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv4 {
		ipv4, err = tryAndRepeatGettingIP(ctx, s.ipGetter.IP4, s.logger, ipversion.IP4)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv6 {
		ipv6, err = tryAndRepeatGettingIP(ctx, s.ipGetter.IP6, s.logger, ipversion.IP6)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return ip, ipv4, ipv6, errors
}

// recordVerdict tells why a record is updated, or is not updated, during an
// update cycle.
type recordVerdict uint8

const (
	// updateNeeded means the record must be updated to the public IP address
	// of the program.
	updateNeeded recordVerdict = iota
	// updateMatchesPublicIP means the record was checked against the public IP
	// address of the program and already uses it.
	updateMatchesPublicIP
	// updateNotChecked means the record was not checked against the public IP
	// address of the program, because it is within its cooldown or ban period,
	// or because the update cycle was canceled.
	updateNotChecked
	// updatePublicIPUnknown means the record could not be checked because no
	// public IP address matching its IP version was found.
	updatePublicIPUnknown
)

// getRecordVerdicts returns the verdict of every record, in the same order as
// the records given.
func (s *Service) getRecordVerdicts(ctx context.Context, records []librecords.Record,
	ip, ipv4, ipv6 netip.Addr,
) (verdicts []recordVerdict) {
	verdicts = make([]recordVerdict, len(records))
	for i, record := range records {
		verdicts[i] = s.shouldUpdateRecord(ctx, record, ip, ipv4, ipv6)
	}
	return verdicts
}

func (s *Service) shouldUpdateRecord(ctx context.Context, record librecords.Record,
	ip, ipv4, ipv6 netip.Addr,
) (verdict recordVerdict) {
	now := s.timeNow()

	isWithinCooldown := now.Sub(record.History.GetSuccessTime()) < s.cooldown
	if isWithinCooldown {
		s.logger.Debug(fmt.Sprintf(
			"record %s is within cooldown period of %s, skipping update",
			recordToLogString(record), s.cooldown))
		return updateNotChecked
	}

	const banPeriod = time.Hour
	isWithinBanPeriod := record.LastBan != nil && now.Sub(*record.LastBan) < banPeriod
	if isWithinBanPeriod {
		s.logger.Info(fmt.Sprintf(
			"record %s is within ban period of %s started at %s, skipping update",
			recordToLogString(record), banPeriod, *record.LastBan))
		return updateNotChecked
	}

	hostname := record.Provider.BuildDomainName()
	ipVersion := record.Provider.IPVersion()
	publicIP := getIPMatchingVersion(ip, ipv4, ipv6, ipVersion)

	if !publicIP.IsValid() {
		message := fmt.Sprintf("Skipping update for %s because %s address was not found",
			hostname, ipVersionToIPKind(ipVersion))
		if record.Status == constants.PUBLICIPUNKNOWN {
			// The record is already known to have no public IP address, warning
			// again at every cycle would spam the logs during a network outage.
			s.logger.Debug(message)
		} else {
			s.logger.Warn(message)
		}
		return updatePublicIPUnknown
	} else if publicIP.Is6() {
		publicIP = ipv6WithSuffix(publicIP, record.Provider.IPv6Suffix())
	}

	if record.Provider.Proxied() {
		lastIP := record.History.GetCurrentIP() // can be nil
		return s.shouldUpdateRecordNoLookup(hostname, ipVersion, lastIP, publicIP)
	}
	return s.shouldUpdateRecordWithLookup(ctx, hostname, ipVersion, publicIP)
}

func (s *Service) shouldUpdateRecordNoLookup(hostname string, ipVersion ipversion.IPVersion,
	lastIP, publicIP netip.Addr,
) (verdict recordVerdict) {
	ipKind := ipVersionToIPKind(ipVersion)
	if publicIP.IsValid() && publicIP.Compare(lastIP) != 0 {
		s.logInfoNoLookupUpdate(hostname, ipKind, lastIP, publicIP)
		return updateNeeded
	}
	s.logDebugNoLookupSkip(hostname, ipKind, lastIP, publicIP)
	return updateMatchesPublicIP
}

func (s *Service) shouldUpdateRecordWithLookup(ctx context.Context, hostname string,
	ipVersion ipversion.IPVersion, publicIP netip.Addr,
) (verdict recordVerdict) {
	const tries = 5
	recordIPv4s, recordIPv6s, err := s.lookupIPsResilient(ctx, hostname, tries)
	if err != nil {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			s.logger.Warn("DNS resolution of " + hostname + ": " + ctxErr.Error())
			return updateNotChecked
		}
		s.logger.Warn("cannot DNS resolve " + hostname + " after " +
			strconv.Itoa(tries) + " tries: " + err.Error()) // update anyway
	}

	ipKind := ipVersionToIPKind(ipVersion)
	recordIPs := recordIPv4s
	if publicIP.Is6() {
		recordIPs = recordIPv6s
	}
	recordIPs = getIPsMatchingVersion(recordIPs, recordIPv4s, recordIPv6s, ipVersion)

	if publicIP.IsValid() && !ipsContainsIP(recordIPs, publicIP) {
		// Note if the recordIP is not valid (not found), we want to update.
		s.logInfoLookupUpdate(hostname, ipKind, recordIPs, publicIP)
		return updateNeeded
	}
	s.logDebugLookupSkip(hostname, ipKind, recordIPs, publicIP)
	return updateMatchesPublicIP
}

func ipsContainsIP(ips []netip.Addr, ip netip.Addr) bool {
	for _, ip2 := range ips {
		if ip.Compare(ip2) == 0 {
			return true
		}
	}
	return false
}

func getIPMatchingVersion(ip, ipv4, ipv6 netip.Addr, ipVersion ipversion.IPVersion) netip.Addr {
	switch ipVersion {
	case ipversion.IP4or6:
		return ip
	case ipversion.IP4:
		return ipv4
	case ipversion.IP6:
		return ipv6
	default:
		panic(fmt.Sprintf("invalid IP version %s", ipVersion))
	}
}

func getIPsMatchingVersion(ip, ipv4, ipv6 []netip.Addr, ipVersion ipversion.IPVersion) []netip.Addr {
	switch ipVersion {
	case ipversion.IP4or6:
		return ip
	case ipversion.IP4:
		return ipv4
	case ipversion.IP6:
		return ipv6
	default:
		panic(fmt.Sprintf("invalid IP version %s", ipVersion))
	}
}

func setUpToDateStatus(db Database, id uint, updateIP netip.Addr, now time.Time) error {
	record, err := db.Select(id)
	if err != nil {
		return err
	}
	record.Status = constants.UPTODATE
	// The message of the previous status does not apply anymore, for example it
	// could be the reason of a previous update failure.
	record.Message = ""
	record.Time = now
	if !record.History.GetCurrentIP().IsValid() {
		record.History = append(record.History, models.HistoryEvent{
			IP:   updateIP,
			Time: now,
		})
	}
	return db.Update(id, record)
}

func setPublicIPUnknownStatus(db Database, id uint, now time.Time) error {
	record, err := db.Select(id)
	if err != nil {
		return err
	}
	record.Status = constants.PUBLICIPUNKNOWN
	// The message of the previous status does not apply anymore, for example
	// it could be a no IP change message from the up to date status.
	record.Message = ""
	record.Time = now
	return db.Update(id, record)
}

func (s *Service) updateNecessary(ctx context.Context) (errors []error) {
	records := s.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	s.logger.Debug(fmt.Sprintf("configured to fetch IP: v4 or v6: %t, v4: %t, v6: %t", doIP, doIPv4, doIPv6))
	ip, ipv4, ipv6, errors := s.getNewIPs(ctx, doIP, doIPv4, doIPv6)
	s.logger.Debug(fmt.Sprintf("your public IP address are: v4 or v6: %s, v4: %s, v6: %s", ip, ipv4, ipv6))
	for _, err := range errors {
		s.logger.Error(err.Error())
	}

	verdicts := s.getRecordVerdicts(ctx, records, ip, ipv4, ipv6)

	// Current time is used to set the status of records whose public IP address
	// is missing, or which are already up to date. No need to have it queried
	// within the next for loop since each iteration is fast and has no IO
	// involved.
	now := s.timeNow()

	errors = append(errors, s.setStatuses(records, verdicts, ip, ipv4, ipv6, now)...)

	for i, verdict := range verdicts {
		if verdict != updateNeeded {
			continue
		}
		id := uint(i)
		record := records[id]
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Provider.IPVersion())
		// Note: each record id has a matching valid public IP address.
		if updateIP.Is6() {
			updateIP = ipv6WithSuffix(updateIP, record.Provider.IPv6Suffix())
		}
		s.logger.Info("Updating record " + record.Provider.String() + " to use " + updateIP.String())
		err := s.updater.Update(ctx, id, updateIP)
		if err != nil {
			errors = append(errors, err)
			s.logger.Error(err.Error())
		}
	}

	healthchecksIOState := healthchecksio.Ok
	if len(errors) > 0 {
		healthchecksIOState = healthchecksio.Fail
	}

	err := s.hioClient.Ping(ctx, healthchecksIOState)
	if err != nil {
		s.logger.Error("pinging healthchecks.io failed: " + err.Error())
	}

	return errors
}

// setStatuses writes the status of every record that is not updated during this
// cycle: the ones already using the public IP address of the program, and the
// ones that cannot be verified because their public IP address is missing.
func (s *Service) setStatuses(records []librecords.Record, verdicts []recordVerdict,
	ip, ipv4, ipv6 netip.Addr, now time.Time,
) (errors []error) {
	for i, record := range records {
		id := uint(i)
		verdict := verdicts[i]
		if verdict == updateNeeded {
			continue
		}
		ipVersion := record.Provider.IPVersion()
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, ipVersion)
		if !updateIP.IsValid() {
			// warning was already logged in [Service.getRecordVerdicts]
			// The record cannot be verified against a public IP address, for example
			// because the network is down. A failure is kept because it tells why the
			// record is not up to date, and the public IP unknown state keeps the
			// time of the first cycle where the address was missing.
			if record.Status == constants.FAIL || record.Status == constants.PUBLICIPUNKNOWN {
				continue
			}
			err := setPublicIPUnknownStatus(s.db, id, now)
			if err != nil {
				err = fmt.Errorf("setting public IP unknown status: %w", err)
				errors = append(errors, err)
				s.logger.Error(err.Error())
			}
			continue
		}

		// The unset and public IP unknown statuses are transient ones, waiting for
		// the status of a real update. They are re-evaluated at every cycle,
		// otherwise the health check would report the program as unhealthy forever,
		// although its records are correct.
		transientStatus := record.Status == constants.UNSET ||
			record.Status == constants.PUBLICIPUNKNOWN
		// A failure is only cleared when the record has just been checked against
		// the public IP address and found to match it. Otherwise the reason of the
		// failure would disappear although the record is still not up to date.
		repairedFailure := record.Status == constants.FAIL && verdict == updateMatchesPublicIP
		if !transientStatus && !repairedFailure {
			continue
		}
		if updateIP.Is6() {
			updateIP = ipv6WithSuffix(updateIP, record.Provider.IPv6Suffix())
		}

		err := setUpToDateStatus(s.db, id, updateIP, now)
		if err != nil {
			err = fmt.Errorf("setting up to date status: %w", err)
			errors = append(errors, err)
			s.logger.Error(err.Error())
		}
	}

	return errors
}

func (s *Service) String() string {
	return "updater"
}

func (s *Service) Start(ctx context.Context) (runError <-chan error, startErr error) {
	ready := make(chan struct{})
	runCtx, runCancel := context.WithCancel(context.Background())
	s.runCancel = runCancel
	done := make(chan struct{})
	s.done = done
	go s.run(runCtx, ready, done) //nolint:contextcheck
	select {
	case <-ready:
	case <-ctx.Done():
		return nil, s.Stop()
	}
	return nil, nil //nolint:nilnil
}

func (s *Service) run(ctx context.Context, ready chan<- struct{},
	done chan<- struct{},
) {
	defer close(done)
	ticker := time.NewTicker(s.period)
	close(ready)
	for {
		select {
		case <-ticker.C:
			s.updateNecessary(ctx)
		case <-s.force:
			s.forceResult <- s.updateNecessary(ctx)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (s *Service) Stop() (err error) {
	s.runCancel()
	<-s.done
	return nil
}

func (s *Service) ForceUpdate(ctx context.Context) (errs []error) {
	s.force <- struct{}{}

	select {
	case errs = <-s.forceResult:
	case <-ctx.Done():
		errs = []error{ctx.Err()}
	}
	return errs
}
