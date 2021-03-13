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
	"github.com/qdm12/golibs/logging"
)

type Runner interface {
	Run(ctx context.Context, period time.Duration)
	ForceUpdate(ctx context.Context) []error
}

type runner struct {
	db          data.Database
	updater     Updater
	force       chan struct{}
	forceResult chan []error
	cooldown    time.Duration
	netLookupIP func(hostname string) ([]net.IP, error)
	ipGetter    publicip.Fetcher
	logger      logging.Logger
	timeNow     func() time.Time
}

func NewRunner(db data.Database, updater Updater, ipGetter publicip.Fetcher,
	cooldown time.Duration, logger logging.Logger, timeNow func() time.Time) Runner {
	return &runner{
		db:          db,
		updater:     updater,
		force:       make(chan struct{}),
		forceResult: make(chan []error),
		cooldown:    cooldown,
		netLookupIP: net.LookupIP,
		ipGetter:    ipGetter,
		logger:      logger,
		timeNow:     timeNow,
	}
}

func (r *runner) lookupIPsResilient(hostname string, tries int) (ipv4 net.IP, ipv6 net.IP, err error) {
	for i := 0; i < tries; i++ {
		ipv4, ipv6, err = r.lookupIPs(hostname)
		if err == nil {
			return ipv4, ipv6, nil
		}
	}
	return nil, nil, err
}

func (r *runner) lookupIPs(hostname string) (ipv4 net.IP, ipv6 net.IP, err error) {
	ips, err := r.netLookupIP(hostname)
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
		case constants.IPv4OrIPv6:
			doIP = true
		case constants.IPv4:
			doIPv4 = true
		case constants.IPv6:
			doIPv6 = true
		}
		if doIP && doIPv4 && doIPv6 {
			return true, true, true
		}
	}
	return doIP, doIPv4, doIPv6
}

func (r *runner) getNewIPs(ctx context.Context, doIP, doIPv4, doIPv6 bool) (ip, ipv4, ipv6 net.IP, errors []error) {
	var err error
	if doIP {
		ip, err = r.ipGetter.IP(ctx)
		if err != nil {
			errors = append(errors, err)
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
	}
	return ip, ipv4, ipv6, errors
}

func (r *runner) getRecordIDsToUpdate(records []librecords.Record, ip, ipv4, ipv6 net.IP,
	now time.Time) (recordIDs map[int]struct{}) {
	recordIDs = make(map[int]struct{})
	for id, record := range records {
		if shouldUpdate := r.shouldUpdateRecord(record, ip, ipv4, ipv6, now); shouldUpdate {
			recordIDs[id] = struct{}{}
		}
	}
	return recordIDs
}

func (r *runner) shouldUpdateRecord(record librecords.Record, ip, ipv4, ipv6 net.IP, now time.Time) (update bool) {
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
	return r.shouldUpdateRecordWithLookup(hostname, ipVersion, ip, ipv4, ipv6)
}

func (r *runner) shouldUpdateRecordNoLookup(hostname string, ipVersion models.IPVersion,
	lastIP, ip, ipv4, ipv6 net.IP) (update bool) {
	switch ipVersion {
	case constants.IPv4OrIPv6:
		if ip != nil && !ip.Equal(lastIP) {
			r.logger.Info("Last IP address stored for %s is %s and your IP address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IP address stored for %s is %s and your IP address is %s, skipping update", hostname, lastIP, ip)
	case constants.IPv4:
		if ipv4 != nil && !ipv4.Equal(lastIP) {
			r.logger.Info("Last IPv4 address stored for %s is %s and your IPv4 address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IPv4 address stored for %s is %s and your IP address is %s, skipping update",
			hostname, lastIP, ip)
	case constants.IPv6:
		if ipv6 != nil && !ipv6.Equal(lastIP) {
			r.logger.Info("Last IPv6 address stored for %s is %s and your IPv6 address is %s", hostname, lastIP, ip)
			return true
		}
		r.logger.Debug("Last IPv6 address stored for %s is %s and your IP address is %s, skipping update",
			hostname, lastIP, ip)
	}
	return false
}

func (r *runner) shouldUpdateRecordWithLookup(hostname string, ipVersion models.IPVersion,
	ip, ipv4, ipv6 net.IP) (update bool) {
	const tries = 5
	recordIPv4, recordIPv6, err := r.lookupIPsResilient(hostname, tries)
	if err != nil {
		r.logger.Warn("cannot DNS resolve %s after %d tries: %s", hostname, tries, err) // update anyway
	}
	switch ipVersion {
	case constants.IPv4OrIPv6:
		recordIP := recordIPv4
		if ip.To4() == nil {
			recordIP = recordIPv6
		}
		if ip != nil && !ip.Equal(recordIPv4) && !ip.Equal(recordIPv6) {
			r.logger.Info("IP address of %s is %s and your IP address is %s", hostname, recordIP, ip)
			return true
		}
		r.logger.Debug("IP address of %s is %s and your IP address is %s, skipping update", hostname, recordIP, ip)
	case constants.IPv4:
		if ipv4 != nil && !ipv4.Equal(recordIPv4) {
			r.logger.Info("IPv4 address of %s is %s and your IPv4 address is %s", hostname, recordIPv4, ipv4)
			return true
		}
		r.logger.Debug("IPv4 address of %s is %s and your IPv4 address is %s, skipping update", hostname, recordIPv4, ipv4)
	case constants.IPv6:
		if ipv6 != nil && !ipv6.Equal(recordIPv6) {
			r.logger.Info("IPv6 address of %s is %s and your IPv6 address is %s", hostname, recordIPv6, ipv6)
			return true
		}
		r.logger.Debug("IPv6 address of %s is %s and your IPv6 address is %s, skipping update", hostname, recordIPv6, ipv6)
	}
	return false
}

func getIPMatchingVersion(ip, ipv4, ipv6 net.IP, ipVersion models.IPVersion) net.IP {
	switch ipVersion {
	case constants.IPv4OrIPv6:
		return ip
	case constants.IPv4:
		return ipv4
	case constants.IPv6:
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

func (r *runner) updateNecessary(ctx context.Context) (errors []error) {
	records := r.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	r.logger.Debug("configured to fetch IP: v4 or v6: %t, v4: %t, v6: %t", doIP, doIPv4, doIPv6)
	ip, ipv4, ipv6, errors := r.getNewIPs(ctx, doIP, doIPv4, doIPv6)
	r.logger.Debug("your public IP address are: v4 or v6: %s, v4: %s, v6: %s", ip, ipv4, ipv6)
	for _, err := range errors {
		r.logger.Error(err)
	}

	now := r.timeNow()
	recordIDs := r.getRecordIDsToUpdate(records, ip, ipv4, ipv6, now)

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

func (r *runner) Run(ctx context.Context, period time.Duration) {
	ticker := time.NewTicker(period)
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

func (r *runner) ForceUpdate(ctx context.Context) (errs []error) {
	r.force <- struct{}{}

	select {
	case errs = <-r.forceResult:
	case <-ctx.Done():
		errs = []error{ctx.Err()}
	}
	return errs
}
