package update

import (
	"context"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/golibs/logging"
)

type Runner interface {
	Run(ctx context.Context, period time.Duration, records []librecords.Record) (forceUpdate func())
}

type runner struct {
	updater  Updater
	ipGetter IPGetter
	logger   logging.Logger
	timeNow  func() time.Time
}

func NewRunner(updater Updater, ipGetter IPGetter, logger logging.Logger, timeNow func() time.Time) Runner {
	return &runner{
		updater:  updater,
		ipGetter: ipGetter,
		logger:   logger,
		timeNow:  timeNow,
	}
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

func readPersistedIPs(records []librecords.Record) (ip, ipv4, ipv6 net.IP) {
	for _, record := range records {
		switch record.Settings.IPVersion() {
		case constants.IPv4OrIPv6:
			ip = record.History.GetCurrentIP()
		case constants.IPv4:
			ipv4 = record.History.GetCurrentIP()
		case constants.IPv6:
			ipv6 = record.History.GetCurrentIP()
		}
		if ip != nil && ipv4 != nil && ipv6 != nil {
			return ip, ipv4, ipv6
		}
	}
	return ip, ipv4, ipv6
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

func shouldUpdate(doIP bool, ip, newIP net.IP, force bool) bool {
	ipFetchFailed := newIP == nil
	ipChanged := ip == nil || !ip.Equal(newIP)
	switch {
	case !doIP, ipFetchFailed:
		return false
	case ipChanged, force:
		return true
	default:
		return false
	}
}

func (r *runner) logIPChanges(force bool, doIP, doIPv4, doIPv6 bool, ip, ipv4, ipv6, newIP, newIPv4, newIPv6 net.IP) {
	updateIP := shouldUpdate(doIP, ip, newIP, force)
	updateIPv4 := shouldUpdate(doIPv4, ipv4, newIPv4, force)
	updateIPv6 := shouldUpdate(doIPv6, ipv6, newIPv6, force)
	if updateIP {
		if force {
			r.logger.Info("Fetched IP adddress %s", newIP)
		} else {
			r.logger.Info("IP address changed from %s to %s", ip, newIP)
		}
	}
	if updateIPv4 {
		if force {
			r.logger.Info("Fetched IPv4 adddress %s", newIPv4)
		} else {
			r.logger.Info("IPv4 address changed from %s to %s", ipv4, newIPv4)
		}
	}
	if updateIPv6 {
		if force {
			r.logger.Info("Fetched IPv6 adddress %s", newIPv6)
		} else {
			r.logger.Info("IPv6 address changed from %s to %s", ipv6, newIPv6)
		}
	}
}

func (r *runner) updateNecessary(records []librecords.Record, ip, ipv4, ipv6 net.IP, force bool) (newIP, newIPv4, newIPv6 net.IP) {
	doIP, doIPv4, doIPv6 := doIPVersion(records)
	newIP, newIPv4, newIPv6, errors := r.getNewIPs(doIP, doIPv4, doIPv6)
	for _, err := range errors {
		r.logger.Error(err)
	}
	updateIP := shouldUpdate(doIP, ip, newIP, force)
	updateIPv4 := shouldUpdate(doIPv4, ipv4, newIPv4, force)
	updateIPv6 := shouldUpdate(doIPv6, ipv6, newIPv6, force)
	r.logIPChanges(force, doIP, doIPv4, doIPv6, ip, ipv4, ipv6, newIP, newIPv4, newIPv6)
	for id, record := range records {
		now := r.timeNow()
		var err error
		switch {
		case updateIP && record.Settings.IPVersion() == constants.IPv4OrIPv6:
			r.logger.Info("Updating record %s for ipv4 or ipv6", record.Settings)
			err = r.updater.Update(id, newIP, now)
		case updateIPv4 && record.Settings.IPVersion() == constants.IPv4:
			r.logger.Info("Updating record %s for ipv4 only", record.Settings)
			err = r.updater.Update(id, newIPv4, now)
		case updateIPv6 && record.Settings.IPVersion() == constants.IPv6:
			r.logger.Info("Updating record %s for ipv6 only", record.Settings)
			err = r.updater.Update(id, newIPv6, now)
		}
		if err != nil {
			r.logger.Error(err)
		}
	}
	return newIP, newIPv4, newIPv6
}

func (r *runner) Run(ctx context.Context, period time.Duration, records []librecords.Record) (forceUpdate func()) {
	timer := time.NewTicker(period)
	forceChannel := make(chan struct{})
	ip, ipv4, ipv6 := readPersistedIPs(records)
	if ip != nil {
		r.logger.Info("Found last IP address %s in database", ip)
	}
	if ipv4 != nil {
		r.logger.Info("Found last IPv4 address %s in database", ipv4)
	}
	if ipv6 != nil {
		r.logger.Info("Found last IPv6 address %s in database", ipv6)
	}
	go func() {
		for {
			select {
			case <-timer.C:
				ip, ipv4, ipv6 = r.updateNecessary(records, ip, ipv4, ipv6, false)
			case <-forceChannel:
				ip, ipv4, ipv6 = r.updateNecessary(records, ip, ipv4, ipv6, true)
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
