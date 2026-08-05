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

// createRRSet creates a new RRSet with an A or AAAA record.
// It should only be called if the record type for the owner name does not exist.
// See https://docs.hetzner.cloud/reference/cloud#tag/zone-rrsets/create_zone_rrset
func (p *Provider) createRRSet(ctx context.Context, client *http.Client, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	url := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/%s/rrsets", p.domain)

	requestData := struct {
		Name    string   `json:"name"`
		Type    string   `json:"type"`
		TTL     uint32   `json:"ttl,omitempty"`
		Records []record `json:"records"`
	}{
		Name: p.owner,
		Type: recordType,
		TTL:  p.ttl,
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
