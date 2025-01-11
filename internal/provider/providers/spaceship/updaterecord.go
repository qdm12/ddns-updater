package spaceship

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	records, err := p.getRecords(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting records: %w", err)
	}

	var existingRecord Record

	// Check exact matches for both type and name
	for _, record := range records {
		if record.Type == recordType && record.Name == p.owner {
			existingRecord = record
			break
		}
	}

	if existingRecord.Name == "" {
		err := p.createRecord(ctx, client, recordType, ip.String())
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	currentIP, err := netip.ParseAddr(existingRecord.Address)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parsing existing IP address: %w", err)
	}

	if currentIP.Compare(ip) == 0 {
		return ip, nil // IP is already up to date
	}

	err = p.deleteRecord(ctx, client, existingRecord)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("deleting record: %w", err)
	}

	err = p.createRecord(ctx, client, recordType, ip.String())
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating record: %w", err)
	}

	return ip, nil
}
