package update

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
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
	ipv6Mask    net.IPMask
	cooldown    time.Duration
	resolver    LookupIPer
	ipGetter    PublicIPFetcher
	logger      Logger
	timeNow     func() time.Time
}

func NewRunner(db Database, updater UpdaterInterface, ipGetter PublicIPFetcher,
	period time.Duration, ipv6Mask net.IPMask, cooldown time.Duration,
	logger Logger, resolver LookupIPer, timeNow func() time.Time) *Runner {
	return &Runner{
		period:      period,
		db:          db,
		updater:     updater,
		force:       make(chan struct{}),
		forceResult: make(chan []error),
		ipv6Mask:    ipv6Mask,
		cooldown:    cooldown,
		resolver:    resolver,
		ipGetter:    ipGetter,
		logger:      logger,
		timeNow:     timeNow,
	}
}

func (r *Runner) lookupIPsResilient(ctx context.Context, hostname string, tries int) (
	ipv4 net.IP, ipv6 net.IP, err error) {
	for i := 0; i < tries; i++ {
		ipv4, ipv6, err = r.lookupIPs(ctx, hostname)
		if err == nil {
			return ipv4, ipv6, nil
		}
	}
	return nil, nil, err
}

func (r *Runner) lookupIPs(ctx context.Context, hostname string) (ipv4 net.IP, ipv6 net.IP, err error) {
	ips, err := r.resolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, nil, err
	}
	for _, ip := range ips {
		if ip.To4() == nil {
			ipv6 = ip
		} else {
			ipv4 = ip
		}
	}
	return ipv4, ipv6, nil
}

func doIPVersion(records []librecords.Record) (doIP, doIPv4, doIPv6 bool) {
	for _, record := range records {
		switch record.Settings.IPVersion() {
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

func (r *Runner) getNewIPs(ctx context.Context, doIP, doIPv4, doIPv6 bool, ipv6Mask net.IPMask) (
	ip, ipv4, ipv6 net.IP, errors []error) {
	var err error
	if doIP {
		ip, err = tryAndRepeatGettingIP(ctx, r.ipGetter.IP, r.logger, ipversion.IP4or6)
		if err != nil {
			errors = append(errors, err)
		}
		if ip.To4() == nil {
			ip = ip.Mask(ipv6Mask)
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
		ipv6 = ipv6.Mask(ipv6Mask)
	}
	return ip, ipv4, ipv6, errors
}

func (r *Runner) getRecordIDsToUpdate(ctx context.Context, records []librecords.Record,
	ip, ipv4, ipv6 net.IP, now time.Time, ipv6Mask net.IPMask) (recordIDs map[uint]struct{}) {
	recordIDs = make(map[uint]struct{})
	for i, record := range records {
		if shouldUpdate := r.shouldUpdateRecord(ctx, record, ip, ipv4, ipv6, now, ipv6Mask); shouldUpdate {
			id := uint(i)
			recordIDs[id] = struct{}{}
		}
	}
	return recordIDs
}

func (r *Runner) shouldUpdateRecord(ctx context.Context, record librecords.Record,
	ip, ipv4, ipv6 net.IP, now time.Time, ipv6Mask net.IPMask) (update bool) {
	isWithinBanPeriod := record.LastBan != nil && now.Sub(*record.LastBan) < time.Hour
	isWithinCooldown := now.Sub(record.History.GetSuccessTime()) < r.cooldown
	if isWithinBanPeriod || isWithinCooldown {
		domain := record.Settings.BuildDomainName()
		r.logger.Debug("record " + domain + " is within ban period or cooldown period, skipping update")
		return false
	}

	hostname := record.Settings.BuildDomainName()
	ipVersion := record.Settings.IPVersion()
	if record.Settings.Proxied() {
		lastIP := record.History.GetCurrentIP() // can be nil
		return r.shouldUpdateRecordNoLookup(hostname, ipVersion, lastIP, ip, ipv4, ipv6)
	}
	return r.shouldUpdateRecordWithLookup(ctx, hostname, ipVersion, ip, ipv4, ipv6, ipv6Mask)
}

func (r *Runner) shouldUpdateRecordNoLookup(hostname string, ipVersion ipversion.IPVersion,
	lastIP, ip, ipv4, ipv6 net.IP) (update bool) {
	switch ipVersion {
	case ipversion.IP4or6:
		if ip != nil && !ip.Equal(lastIP) {
			r.logger.Info("Last IP address stored for " + hostname +
				" is " + lastIP.String() + " and your IP address is " + ip.String())
			return true
		}
		r.logger.Debug("Last IP address stored for " + hostname + " is " +
			lastIP.String() + " and your IP address is " + ip.String() + ", skipping update")
	case ipversion.IP4:
		if ipv4 != nil && !ipv4.Equal(lastIP) {
			r.logger.Info("Last IPv4 address stored for " + hostname +
				" is " + lastIP.String() + " and your IPv4 address is " + ip.String())
			return true
		}
		r.logger.Debug("Last IPv4 address stored for " + hostname + " is " +
			lastIP.String() + " and your IPv4 address is " + ip.String() + ", skipping update")
	case ipversion.IP6:
		if ipv6 != nil && !ipv6.Equal(lastIP) {
			r.logger.Info("Last IPv6 address stored for " + hostname +
				" is " + lastIP.String() + " and your IPv6 address is " + ip.String())
			return true
		}
		r.logger.Debug("Last IPv6 address stored for " + hostname + " is " +
			lastIP.String() + " and your IPv6 address is " + ip.String() + ", skipping update")
	}
	return false
}

func (r *Runner) shouldUpdateRecordWithLookup(ctx context.Context, hostname string, ipVersion ipversion.IPVersion,
	ip, ipv4, ipv6 net.IP, ipv6Mask net.IPMask) (update bool) {
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

	if recordIPv6 != nil {
		recordIPv6 = recordIPv6.Mask(ipv6Mask)
	}

	switch ipVersion {
	case ipversion.IP4or6:
		recordIP := recordIPv4
		if ip.To4() == nil {
			recordIP = recordIPv6
		}
		if ip != nil && !ip.Equal(recordIPv4) && !ip.Equal(recordIPv6) {
			r.logger.Info("IP address of " + hostname + " is " + recordIP.String() +
				" and your IP address is " + ip.String())
			return true
		}
		r.logger.Debug("IP address of " + hostname + " is " + recordIP.String() +
			" and your IP address is " + ip.String() + ", skipping update")
	case ipversion.IP4:
		if ipv4 != nil && !ipv4.Equal(recordIPv4) {
			r.logger.Info("IPv4 address of " + hostname + " is " + recordIPv4.String() +
				" and your IPv4 address is " + ipv4.String())
			return true
		}
		r.logger.Debug("IPv4 address of " + hostname + " is " + recordIPv4.String() +
			" and your IPv4 address is " + ipv4.String() + ", skipping update")
	case ipversion.IP6:
		if ipv6 != nil && !ipv6.Equal(recordIPv6) {
			r.logger.Info("IPv6 address of " + hostname + " is " + recordIPv6.String() +
				" and your IPv6 address is " + ipv6.String())
			return true
		}
		r.logger.Debug("IPv6 address of " + hostname + " is " + recordIPv6.String() +
			" and your IPv6 address is " + ipv6.String() + ", skipping update")
	}
	return false
}

func getIPMatchingVersion(ip, ipv4, ipv6 net.IP, ipVersion ipversion.IPVersion) net.IP {
	switch ipVersion {
	case ipversion.IP4or6:
		return ip
	case ipversion.IP4:
		return ipv4
	case ipversion.IP6:
		return ipv6
	}
	return nil
}

func setInitialUpToDateStatus(db Database, id uint, updateIP net.IP, now time.Time) error {
	record, err := db.Select(id)
	if err != nil {
		return err
	}
	record.Status = constants.UPTODATE
	record.Time = now
	if record.History.GetCurrentIP() == nil {
		record.History = append(record.History, models.HistoryEvent{
			IP:   updateIP,
			Time: now,
		})
	}
	return db.Update(id, record)
}

func (r *Runner) updateNecessary(ctx context.Context, ipv6Mask net.IPMask) (errors []error) {
	records := r.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	r.logger.Debug(fmt.Sprintf("configured to fetch IP: v4 or v6: %t, v4: %t, v6: %t", doIP, doIPv4, doIPv6))
	ip, ipv4, ipv6, errors := r.getNewIPs(ctx, doIP, doIPv4, doIPv6, ipv6Mask)
	r.logger.Debug(fmt.Sprintf("your public IP address are: v4 or v6: %s, v4: %s, v6: %s", ip, ipv4, ipv6))
	for _, err := range errors {
		r.logger.Error(err.Error())
	}

	now := r.timeNow()
	recordIDs := r.getRecordIDsToUpdate(ctx, records, ip, ipv4, ipv6, now, ipv6Mask)

	for i, record := range records {
		id := uint(i)
		_, requireUpdate := recordIDs[id]
		if requireUpdate || record.Status != constants.UNSET {
			continue
		}
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
		err := setInitialUpToDateStatus(r.db, id, updateIP, now)
		if err != nil {
			errors = append(errors, err)
			r.logger.Error(err.Error())
		}
	}
	for id := range recordIDs {
		record := records[id]
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
		r.logger.Info("Updating record " + record.Settings.String() + " to use " + updateIP.String())
		err := r.updater.Update(ctx, id, updateIP, r.timeNow())
		if err != nil {
			errors = append(errors, err)
			r.logger.Error(err.Error())
		}
	}

	return errors
}

func (r *Runner) Run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(r.period)
	for {
		select {
		case <-ticker.C:
			r.updateNecessary(ctx, r.ipv6Mask)
		case <-r.force:
			r.forceResult <- r.updateNecessary(ctx, r.ipv6Mask)
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
