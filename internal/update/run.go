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

func readPersistedIPs(records []librecords.Record) (ip, ipv4, ipv6 net.IP) {
	for _, record := range records {
		switch record.Settings.IPVersion() {
		case constants.IPv4OrIPv6:
			ip = record.History.GetCurrentIP()
			if ip == nil {
				ip = net.IP{127, 0, 0, 1}
			}
		case constants.IPv4:
			ipv4 = record.History.GetCurrentIP()
			if ipv4 == nil {
				ipv4 = net.IP{127, 0, 0, 1}
			}
		case constants.IPv6:
			ipv6 = record.History.GetCurrentIP()
			if ipv6 == nil {
				ipv6 = net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
			}
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

func shouldUpdate(ip, newIP net.IP, force bool) bool {
	ipVersionDisabled := ip == nil
	ipFetchFailed := newIP == nil
	ipChanged := !ip.Equal(newIP)
	switch {
	case ipVersionDisabled, ipFetchFailed:
		return false
	case ipChanged, force:
		return true
	default:
		return false
	}
}

func (r *runner) updateNecessary(records []librecords.Record, ip, ipv4, ipv6 net.IP, force bool) (newIP, newIPv4, newIPv6 net.IP) {
	newIP, newIPv4, newIPv6, errors := r.getNewIPs(ip != nil, ipv4 != nil, ipv6 != nil)
	for _, err := range errors {
		r.logger.Error(err)
	}
	updateIP := shouldUpdate(ip, newIP, force)
	updateIPv4 := shouldUpdate(ipv4, newIPv4, force)
	updateIPv6 := shouldUpdate(ipv6, newIPv6, force)
	if updateIP && !force {
		r.logger.Info("IP address changed from %s to %s", ip, newIP)
	}
	if updateIPv4 && !force {
		r.logger.Info("IPv4 address changed from %s to %s", ipv4, newIPv4)
	}
	if updateIPv6 && !force {
		r.logger.Info("IPv6 address changed from %s to %s", ipv6, newIPv6)
	}
	for id, record := range records {
		now := r.timeNow()
		var err error
		switch {
		case updateIP && record.Settings.IPVersion() == constants.IPv4OrIPv6:
			err = r.updater.Update(id, newIP, now)
		case updateIPv4 && record.Settings.IPVersion() == constants.IPv4:
			err = r.updater.Update(id, newIPv4, now)
		case updateIPv6 && record.Settings.IPVersion() == constants.IPv6:
			err = r.updater.Update(id, newIPv6, now)
		}
		if err != nil {
			r.logger.Error(err)
		}
	}
	return newIP, newIPv4, newIPv6
}

func (r *runner) Run(ctx context.Context, period time.Duration, records []librecords.Record) (forceUpdate func()) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	timer := time.NewTicker(period)
	forceChannel := make(chan struct{})
	go func() {
		ip, ipv4, ipv6 := readPersistedIPs(records)
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
