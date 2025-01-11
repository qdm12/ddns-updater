package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
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

	var existingRecord *Record
	for _, record := range records {
		if record.Type == recordType && record.Name == p.owner {
			recordCopy := record
			existingRecord = &recordCopy
			break
		}
	}

	if existingRecord == nil {
		if err := p.createRecord(ctx, client, recordType, ip.String()); err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	currentIP, err := netip.ParseAddr(existingRecord.Address)
	if err == nil && currentIP.Compare(ip) == 0 {
		return ip, nil // IP is already up to date
	}

	if err := p.deleteRecord(ctx, client, existingRecord); err != nil {
		return netip.Addr{}, fmt.Errorf("deleting record: %w", err)
	}

	if err := p.createRecord(ctx, client, recordType, ip.String()); err != nil {
		return netip.Addr{}, fmt.Errorf("creating record: %w", err)
	}

	return ip, nil
}

func (p *Provider) deleteRecord(ctx context.Context, client *http.Client, record *Record) error {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   fmt.Sprintf("/api/v1/dns/records/%s", p.domain),
	}

	deleteData := []Record{{
		Type:    record.Type,
		Name:    record.Name,
		Address: record.Address,
	}}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(deleteData); err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), &requestBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	return nil
}
