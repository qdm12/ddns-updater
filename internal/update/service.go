package update

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Runner struct {
	period      time.Duration
	db          Database
	updater     UpdaterInterface
	force       chan struct{}
	forceResult chan []error
	cooldown    time.Duration
	resolver    LookupIPer
	ipGetter    PublicIPFetcher
	logger      Logger
	timeNow     func() time.Time
	hioClient   HealthchecksIOClient
}

func NewRunner(db Database, updater UpdaterInterface, ipGetter PublicIPFetcher,
	period time.Duration, cooldown time.Duration, logger Logger, resolver LookupIPer,
	timeNow func() time.Time, hioClient HealthchecksIOClient) *Runner {
	return &Runner{
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

func (r *Runner) lookupIPsResilient(ctx context.Context, hostname string, tries int) (
	ipv4 netip.Addr, ipv6 netip.Addr, err error) {
	for i := 0; i < tries; i++ {
		ipv4, ipv6, err = r.lookupIPs(ctx, hostname)
		if err == nil {
			return ipv4, ipv6, nil
		}
	}
	return netip.Addr{}, netip.Addr{}, err
}

func (r *Runner) lookupIPs(ctx context.Context, hostname string) (
	ipv4 netip.Addr, ipv6 netip.Addr, err error) {
	netIPs, err := r.resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return netip.Addr{}, netip.Addr{}, err
	}
	ips := make([]netip.Addr, len(netIPs))
	for i, netIP := range netIPs {
		switch {
		case netIP == nil:
		case netIP.To4() != nil:
			ips[i] = netip.AddrFrom4([4]byte(netIP.To4()))
		default: // IPv6
			ips[i] = netip.AddrFrom16([16]byte(netIP.To16()))
		}
	}

	for _, ip := range ips {
		if ip.Is6() {
			ipv6 = ip
		} else {
			ipv4 = ip
		}
	}
	return ipv4, ipv6, nil
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

func (r *Runner) getNewIPs(ctx context.Context, doIP, doIPv4, doIPv6 bool) (
	ip, ipv4, ipv6 netip.Addr, errors []error) {
	var err error
	if doIP {
		ip, err = tryAndRepeatGettingIP(ctx, r.ipGetter.IP, r.logger, ipversion.IP4or6)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv4 {
		ipv4, err = tryAndRepeatGettingIP(ctx, r.ipGetter.IP4, r.logger, ipversion.IP4)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv6 {
		ipv6, err = tryAndRepeatGettingIP(ctx, r.ipGetter.IP6, r.logger, ipversion.IP6)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return ip, ipv4, ipv6, errors
}

func (r *Runner) getRecordIDsToUpdate(ctx context.Context, records []librecords.Record,
	ip, ipv4, ipv6 netip.Addr) (recordIDs map[uint]struct{}) {
	recordIDs = make(map[uint]struct{})
	for i, record := range records {
		shouldUpdate := r.shouldUpdateRecord(ctx, record, ip, ipv4, ipv6)
		if shouldUpdate {
			id := uint(i)
			recordIDs[id] = struct{}{}
		}
	}
	return recordIDs
}

func (r *Runner) shouldUpdateRecord(ctx context.Context, record librecords.Record,
	ip, ipv4, ipv6 netip.Addr) (update bool) {
	now := r.timeNow()

	isWithinCooldown := now.Sub(record.History.GetSuccessTime()) < r.cooldown
	if isWithinCooldown {
		r.logger.Debug(fmt.Sprintf(
			"record %s is within cooldown period of %s, skipping update",
			recordToLogString(record), r.cooldown))
		return false
	}

	const banPeriod = time.Hour
	isWithinBanPeriod := record.LastBan != nil && now.Sub(*record.LastBan) < banPeriod
	if isWithinBanPeriod {
		r.logger.Info(fmt.Sprintf(
			"record %s is within ban period of %s started at %s, skipping update",
			recordToLogString(record), banPeriod, *record.LastBan))
		return false
	}

	hostname := record.Provider.BuildDomainName()
	ipVersion := record.Provider.IPVersion()
	publicIP := getIPMatchingVersion(ip, ipv4, ipv6, ipVersion)

	if !publicIP.IsValid() {
		r.logger.Warn(fmt.Sprintf("Skipping update for %s because %s address was not found",
			hostname, ipVersionToIPKind(ipVersion)))
		return false
	} else if publicIP.Is6() {
		publicIP = ipv6WithSuffix(publicIP, record.Provider.IPv6Suffix())
	}

	if record.Provider.Proxied() {
		lastIP := record.History.GetCurrentIP() // can be nil
		return r.shouldUpdateRecordNoLookup(hostname, ipVersion, lastIP, publicIP)
	}
	return r.shouldUpdateRecordWithLookup(ctx, hostname, ipVersion, publicIP)
}

func (r *Runner) shouldUpdateRecordNoLookup(hostname string, ipVersion ipversion.IPVersion,
	lastIP, publicIP netip.Addr) (update bool) {
	ipKind := ipVersionToIPKind(ipVersion)
	if publicIP.IsValid() && publicIP.Compare(lastIP) != 0 {
		r.logInfoNoLookupUpdate(hostname, ipKind, lastIP, publicIP)
		return true
	}
	r.logDebugNoLookupSkip(hostname, ipKind, lastIP, publicIP)
	return false
}

func (r *Runner) shouldUpdateRecordWithLookup(ctx context.Context, hostname string,
	ipVersion ipversion.IPVersion, publicIP netip.Addr) (update bool) {
	const tries = 5
	recordIPv4, recordIPv6, err := r.lookupIPsResilient(ctx, hostname, tries)
	if err != nil {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			r.logger.Warn("DNS resolution of " + hostname + ": " + ctxErr.Error())
			return false
		}
		r.logger.Warn("cannot DNS resolve " + hostname + " after " +
			fmt.Sprint(tries) + " tries: " + err.Error()) // update anyway
	}

	ipKind := ipVersionToIPKind(ipVersion)
	recordIP := recordIPv4
	if publicIP.Is6() {
		recordIP = recordIPv6
	}
	recordIP = getIPMatchingVersion(recordIP, recordIPv4, recordIPv6, ipVersion)

	if publicIP.IsValid() && publicIP.Compare(recordIP) != 0 {
		// Note if the recordIP is not valid (not found), we want to update.
		r.logInfoLookupUpdate(hostname, ipKind, recordIP, publicIP)
		return true
	}
	r.logDebugLookupSkip(hostname, ipKind, recordIP, publicIP)
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
	}
	return netip.Addr{}
}

func setInitialUpToDateStatus(db Database, id uint, updateIP netip.Addr, now time.Time) error {
	record, err := db.Select(id)
	if err != nil {
		return err
	}
	record.Status = constants.UPTODATE
	record.Time = now
	if !record.History.GetCurrentIP().IsValid() {
		record.History = append(record.History, models.HistoryEvent{
			IP:   updateIP,
			Time: now,
		})
	}
	return db.Update(id, record)
}

func setInitialPublicIPFailStatus(db Database, id uint, now time.Time) error {
	record, err := db.Select(id)
	if err != nil {
		return err
	}
	record.Status = constants.FAIL
	record.Message = "public IP address not found"
	record.Time = now
	return db.Update(id, record)
}

func (r *Runner) updateNecessary(ctx context.Context) (errors []error) {
	records := r.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	r.logger.Debug(fmt.Sprintf("configured to fetch IP: v4 or v6: %t, v4: %t, v6: %t", doIP, doIPv4, doIPv6))
	ip, ipv4, ipv6, errors := r.getNewIPs(ctx, doIP, doIPv4, doIPv6)
	r.logger.Debug(fmt.Sprintf("your public IP address are: v4 or v6: %s, v4: %s, v6: %s", ip, ipv4, ipv6))
	for _, err := range errors {
		r.logger.Error(err.Error())
	}

	recordIDs := r.getRecordIDsToUpdate(ctx, records, ip, ipv4, ipv6)

	// Current time is used to set initial states for records already
	// up to date or in the fail state due to the public IP not found.
	// No need to have it queried within the next for loop since each
	// iteration is fast and has no IO involved.
	now := r.timeNow()

	for i, record := range records {
		id := uint(i)
		_, requireUpdate := recordIDs[id]
		if requireUpdate || record.Status != constants.UNSET {
			continue
		}

		ipVersion := record.Provider.IPVersion()
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, ipVersion)
		if !updateIP.IsValid() {
			// warning was already logged in getRecordIDsToUpdate
			err := setInitialPublicIPFailStatus(r.db, id, now)
			if err != nil {
				err = fmt.Errorf("setting initial public IP fail status: %w", err)
				errors = append(errors, err)
				r.logger.Error(err.Error())
			}
			continue
		} else if updateIP.Is6() {
			updateIP = ipv6WithSuffix(updateIP, record.Provider.IPv6Suffix())
		}

		err := setInitialUpToDateStatus(r.db, id, updateIP, now)
		if err != nil {
			err = fmt.Errorf("setting initial up to date status: %w", err)
			errors = append(errors, err)
			r.logger.Error(err.Error())
		}
	}
	for id := range recordIDs {
		record := records[id]
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Provider.IPVersion())
		// Note: each record id has a matching valid public IP address.
		if updateIP.Is6() {
			updateIP = ipv6WithSuffix(updateIP, record.Provider.IPv6Suffix())
		}
		r.logger.Info("Updating record " + record.Provider.String() + " to use " + updateIP.String())
		err := r.updater.Update(ctx, id, updateIP)
		if err != nil {
			errors = append(errors, err)
			r.logger.Error(err.Error())
		}
	}

	healthchecksIOState := healthchecksio.Ok
	if len(errors) > 0 {
		healthchecksIOState = healthchecksio.Fail
	}

	err := r.hioClient.Ping(ctx, healthchecksIOState)
	if err != nil {
		r.logger.Error("pinging healthchecks.io failed: " + err.Error())
	}

	return errors
}

func (r *Runner) Run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(r.period)
	for {
		select {
		case <-ticker.C:
			r.updateNecessary(ctx)
		case <-r.force:
			r.forceResult <- r.updateNecessary(ctx)
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (r *Runner) ForceUpdate(ctx context.Context) (errs []error) {
	r.force <- struct{}{}

	select {
	case errs = <-r.forceResult:
	case <-ctx.Done():
		errs = []error{ctx.Err()}
	}
	return errs
}
