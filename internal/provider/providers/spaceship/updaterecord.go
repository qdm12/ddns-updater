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
	var targetRecord apiRecord

	for _, record := range records {
		if record.Type == recordType && record.Name == p.owner {
			targetRecord = record
			found = true
			break
		}
	}

	if found {
		currentIP, err := netip.ParseAddr(targetRecord.Address)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("parsing existing IP address: %w", err)
		} else if currentIP == ip {
			return ip, nil
		}
	} else {
		targetRecord.Type = recordType
		targetRecord.Name = p.owner
	}

	targetRecord.TTL = p.ttl
	targetRecord.Address = ip.String()

	err = p.putRecord(ctx, client, targetRecord)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("putting record: %w", err)
	}

	return ip, nil
}
