package update

import (
	"context"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/logging"
)

type Runner interface {
	Run(ctx context.Context, done chan<- struct{})
	ForceUpdate(ctx context.Context) []error
}

type runner struct {
	period      time.Duration
	db          data.Database
	updater     Updater
	force       chan struct{}
	forceResult chan []error
	ipv6Mask    net.IPMask
	cooldown    time.Duration
	resolver    *net.Resolver
	ipGetter    publicip.Fetcher
	logger      logging.Logger
	timeNow     func() time.Time
}

func NewRunner(db data.Database, updater Updater, ipGetter publicip.Fetcher,
	period time.Duration, ipv6Mask net.IPMask, cooldown time.Duration,
	logger logging.Logger, timeNow func() time.Time) Runner {
	return &runner{
		period:      period,
		db:          db,
		updater:     updater,
		force:       make(chan struct{}),
		forceResult: make(chan []error),
		ipv6Mask:    ipv6Mask,
		cooldown:    cooldown,
		resolver:    net.DefaultResolver,
		ipGetter:    ipGetter,
		logger:      logger,
		timeNow:     timeNow,
	}
}

func (r *runner) lookupIPsResilient(ctx context.Context, hostname string, tries int) (
	ipv4 net.IP, ipv6 net.IP, err error) {
	for i := 0; i < tries; i++ {
		ipv4, ipv6, err = r.lookupIPs(ctx, hostname)
		if err == nil {
			return ipv4, ipv6, nil
		}
	}
	return nil, nil, err
}

func (r *runner) lookupIPs(ctx context.Context, hostname string) (ipv4 net.IP, ipv6 net.IP, err error) {
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

func (r *runner) getNewIPs(ctx context.Context, doIP, doIPv4, doIPv6 bool, ipv6Mask net.IPMask) (
	ip, ipv4, ipv6 net.IP, errors []error) {
	var err error
	if doIP {
		ip, err = r.ipGetter.IP(ctx)
		if err != nil {
			errors = append(errors, err)
		}
		if ip.To4() == nil {
			ip = ip.Mask(ipv6Mask)
		}
	}
	if doIPv4 {
		ipv4, err = r.ipGetter.IP4(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv6 {
		ipv6, err = r.ipGetter.IP6(ctx)
		if err != nil {
			errors = append(errors, err)
		}
		ipv6 = ipv6.Mask(ipv6Mask)
	}
	return ip, ipv4, ipv6, errors
}

func (r *runner) getRecordIDsToUpdate(ctx context.Context, records []librecords.Record,
	ip, ipv4, ipv6 net.IP, now time.Time, ipv6Mask net.IPMask) (recordIDs map[int]struct{}) {
	recordIDs = make(map[int]struct{})
	for id, record := range records {
		if shouldUpdate := r.shouldUpdateRecord(ctx, record, ip, ipv4, ipv6, now, ipv6Mask); shouldUpdate {
			recordIDs[id] = struct{}{}
		}
	}
	return recordIDs
}

func (r *runner) shouldUpdateRecord(ctx context.Context, record librecords.Record,
	ip, ipv4, ipv6 net.IP, now time.Time, ipv6Mask net.IPMask) (update bool) {
	isWithinBanPeriod := record.LastBan != nil && now.Sub(*record.LastBan) < time.Hour
	isWithinCooldown := now.Sub(record.History.GetSuccessTime()) < r.cooldown
	if isWithinBanPeriod || isWithinCooldown {
		r.logger.Debug("record %s is within ban period or cooldown period, skipping update",
			record.Settings.BuildDomainName())
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

func (r *runner) shouldUpdateRecordNoLookup(hostname string, ipVersion ipversion.IPVersion,
	lastIP, ip, ipv4, ipv6 net.IP) (update bool) {
	switch ipVersion {
	case ipversion.IP4or6:
		if ip != nil && !ip.Equal(lastIP) {
			r.logger.Info("Last IP address stored for %s is %s and your IP address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IP address stored for %s is %s and your IP address is %s, skipping update", hostname, lastIP, ip)
	case ipversion.IP4:
		if ipv4 != nil && !ipv4.Equal(lastIP) {
			r.logger.Info("Last IPv4 address stored for %s is %s and your IPv4 address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IPv4 address stored for %s is %s and your IP address is %s, skipping update",
			hostname, lastIP, ip)
	case ipversion.IP6:
		if ipv6 != nil && !ipv6.Equal(lastIP) {
			r.logger.Info("Last IPv6 address stored for %s is %s and your IPv6 address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IPv6 address stored for %s is %s and your IP address is %s, skipping update",
			hostname, lastIP, ip)
	}
	return false
}

func (r *runner) shouldUpdateRecordWithLookup(ctx context.Context, hostname string, ipVersion ipversion.IPVersion,
	ip, ipv4, ipv6 net.IP, ipv6Mask net.IPMask) (update bool) {
	const tries = 5
	recordIPv4, recordIPv6, err := r.lookupIPsResilient(ctx, hostname, tries)
	if err != nil {
		if err := ctx.Err(); err != nil {
			r.logger.Warn("DNS resolution of " + hostname + ": " + err.Error())
			return false
		}
		r.logger.Warn("cannot DNS resolve %s after %d tries: %s", hostname, tries, err) // update anyway
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
			r.logger.Info("IP address of %s is %s and your IP address is %s", hostname, recordIP, ip)
			return true
		}
		r.logger.Debug("IP address of %s is %s and your IP address is %s, skipping update", hostname, recordIP, ip)
	case ipversion.IP4:
		if ipv4 != nil && !ipv4.Equal(recordIPv4) {
			r.logger.Info("IPv4 address of %s is %s and your IPv4 address is %s", hostname, recordIPv4, ipv4)
			return true
		}
		r.logger.Debug("IPv4 address of %s is %s and your IPv4 address is %s, skipping update", hostname, recordIPv4, ipv4)
	case ipversion.IP6:
		if ipv6 != nil && !ipv6.Equal(recordIPv6) {
			r.logger.Info("IPv6 address of %s is %s and your IPv6 address is %s", hostname, recordIPv6, ipv6)
			return true
		}
		r.logger.Debug("IPv6 address of %s is %s and your IPv6 address is %s, skipping update", hostname, recordIPv6, ipv6)
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

func setInitialUpToDateStatus(db data.Database, id int, updateIP net.IP, now time.Time) error {
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

func (r *runner) updateNecessary(ctx context.Context, ipv6Mask net.IPMask) (errors []error) {
	records := r.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	r.logger.Debug("configured to fetch IP: v4 or v6: %t, v4: %t, v6: %t", doIP, doIPv4, doIPv6)
	ip, ipv4, ipv6, errors := r.getNewIPs(ctx, doIP, doIPv4, doIPv6, ipv6Mask)
	r.logger.Debug("your public IP address are: v4 or v6: %s, v4: %s, v6: %s", ip, ipv4, ipv6)
	for _, err := range errors {
		r.logger.Error(err)
	}

	now := r.timeNow()
	recordIDs := r.getRecordIDsToUpdate(ctx, records, ip, ipv4, ipv6, now, ipv6Mask)

	for id, record := range records {
		_, requireUpdate := recordIDs[id]
		if requireUpdate || record.Status != constants.UNSET {
			continue
		}
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
		if err := setInitialUpToDateStatus(r.db, id, updateIP, now); err != nil {
			errors = append(errors, err)
			r.logger.Error(err)
		}
	}
	for id := range recordIDs {
		record := records[id]
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
		r.logger.Info("Updating record %s to use %s", record.Settings, updateIP)
		if err := r.updater.Update(ctx, id, updateIP, r.timeNow()); err != nil {
			errors = append(errors, err)
			r.logger.Error(err)
		}
	}

	return errors
}

func (r *runner) Run(ctx context.Context, done chan<- struct{}) {
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

func (r *runner) ForceUpdate(ctx context.Context) (errs []error) {
	r.force <- struct{}{}

	select {
	case errs = <-r.forceResult:
	case <-ctx.Done():
		errs = []error{ctx.Err()}
	}
	return errs
}
