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

	var found bool
	var existingRecord apiRecord

	for _, record := range records {
		if record.Type == recordType && record.Name == p.owner {
			existingRecord = record
			found = true
			break
		}
	}

	if found {
		currentIP, err := netip.ParseAddr(existingRecord.Address)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("parsing existing IP address: %w", err)
		}
		if currentIP == ip {
			return ip, nil
		}
		err = p.deleteRecord(ctx, client, existingRecord)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("deleting existing record: %w", err)
		}
	}

	newRecord := apiRecord{
		Type:    recordType,
		Name:    p.owner,
		Address: ip.String(),
		TTL:     p.ttl,
	}
	err = p.createRecord(ctx, client, newRecord)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating record: %w", err)
	}

	return ip, nil
}
