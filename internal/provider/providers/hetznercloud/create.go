package hetznercloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

// createRecord creates a new DNS record using the add_records action.
// It adds the new IP address to the existing RRSet or creates a new RRSet.
// See https://docs.hetzner.cloud/reference/cloud#tag/zone-rrset-actions/add_zone_rrset_records
func (p *Provider) createRecord(ctx context.Context, client *http.Client, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	const urlTemplate = "https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s/actions/add_records"
	url := fmt.Sprintf(urlTemplate, p.domain, p.owner, recordType)

	requestData := struct {
		TTL     uint32   `json:"ttl,omitempty"`
		Records []record `json:"records"`
	}{
		TTL: p.ttl,
		Records: []record{
			{Value: ip.String()},
		},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)
	var responseData actionResponse
	err = decoder.Decode(&responseData)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	return p.handleActionResponse(ctx, client, responseData)
}
