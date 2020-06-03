package update

import (
	"context"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/golibs/logging"
)

type Runner interface {
	Run(ctx context.Context, period time.Duration) (forceUpdate func())
}

type runner struct {
	db          data.Database
	updater     Updater
	netLookupIP func(hostname string) ([]net.IP, error)
	ipGetter    IPGetter
	logger      logging.Logger
	timeNow     func() time.Time
}

func NewRunner(db data.Database, updater Updater, ipGetter IPGetter, logger logging.Logger, timeNow func() time.Time) Runner {
	return &runner{
		db:          db,
		updater:     updater,
		netLookupIP: net.LookupIP,
		ipGetter:    ipGetter,
		logger:      logger,
		timeNow:     timeNow,
	}
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

func (r *runner) getNewIPs(doIP, doIPv4, doIPv6 bool) (ip, ipv4, ipv6 net.IP, errors []error) {
	var err error
	if doIP {
		ip, err = r.ipGetter.IP()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv4 {
		ipv4, err = r.ipGetter.IPv4()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if doIPv6 {
		ipv6, err = r.ipGetter.IPv6()
		if err != nil {
			errors = append(errors, err)
		}
	}
	return ip, ipv4, ipv6, errors
}

func (r *runner) getRecordIDsToUpdate(records []librecords.Record, ip, ipv4, ipv6 net.IP) (recordIDs map[int]struct{}) {
	recordIDs = make(map[int]struct{})
	for id, record := range records {
		hostname := record.Settings.BuildDomainName()
		recordIPv4, recordIPv6, err := r.lookupIPs(hostname)
		if err != nil {
			r.logger.Warn(err) // update anyway
		}
		switch record.Settings.IPVersion() {
		case constants.IPv4OrIPv6:
			if ip != nil && !ip.Equal(recordIPv4) && !ip.Equal(recordIPv6) {
				recordIP := recordIPv4
				if ip.To4() == nil {
					recordIP = recordIPv6
				}
				r.logger.Info("IP address of %s is %s and your IP address is %s", hostname, recordIP, ip)
				recordIDs[id] = struct{}{}
			}
		case constants.IPv4:
			if ipv4 != nil && !ipv4.Equal(recordIPv4) {
				r.logger.Info("IPv4 address of %s is %s and your IPv4 address is %s", hostname, recordIPv4, ipv4)
				recordIDs[id] = struct{}{}
			}
		case constants.IPv6:
			if ipv6 != nil && !ipv6.Equal(recordIPv6) {
				r.logger.Info("IPv6 address of %s is %s and your IPv6 address is %s", hostname, recordIPv6, ipv6)
				recordIDs[id] = struct{}{}
			}
		}
	}
	return recordIDs
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

func (r *runner) setEmptyUpToDateRecord(id int, record librecords.Record, ip, ipv4, ipv6 net.IP, now time.Time) error {
	updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
	record.History = append(record.History, models.HistoryEvent{
		IP:   updateIP,
		Time: now,
	})
	record.Status = constants.UPTODATE
	record.Time = now
	return r.db.Update(id, record)
}

func (r *runner) updateNecessary() {
	records := r.db.SelectAll()
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	ip, ipv4, ipv6, errors := r.getNewIPs(doIP, doIPv4, doIPv6)
	for _, err := range errors {
		r.logger.Error(err)
	}
	recordIDs := r.getRecordIDsToUpdate(records, ip, ipv4, ipv6)
	now := r.timeNow()
	for id, record := range records {
		_, requireUpdate := recordIDs[id]
		unset := record.History.GetCurrentIP() == nil && record.Status == constants.UNSET
		if !requireUpdate && unset {
			err := r.setEmptyUpToDateRecord(id, record, ip, ipv4, ipv6, now)
			if err != nil {
				r.logger.Error(err)
			}
		}
	}
	for id := range recordIDs {
		record := records[id]
		updateIP := getIPMatchingVersion(ip, ipv4, ipv6, record.Settings.IPVersion())
		r.logger.Info("Updating record %s", record.Settings)
		if err := r.updater.Update(id, updateIP, r.timeNow()); err != nil {
			r.logger.Error(err)
		}
	}
}

func (r *runner) Run(ctx context.Context, period time.Duration) (forceUpdate func()) {
	timer := time.NewTicker(period)
	forceChannel := make(chan struct{})
	go func() {
		for {
			select {
			case <-timer.C:
				r.updateNecessary()
			case <-forceChannel:
				r.updateNecessary()
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
	return func() {
		forceChannel <- struct{}{}
	}
}
