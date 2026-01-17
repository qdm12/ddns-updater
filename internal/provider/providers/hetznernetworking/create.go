package hetznernetworking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// createRecord creates a new DNS record using the add_records action.
// It adds the new IP address to the existing RRSet or creates a new RRSet.
func (p *Provider) createRecord(ctx context.Context, client *http.Client, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	// Extract RR name from domain relative to zone
	rrName, err := p.extractRRName()
	if err != nil {
		return fmt.Errorf("extracting RR name: %w", err)
	}

	urlString := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s/actions/add_records", p.zoneIdentifier, rrName, recordType)

	requestData := recordsRequest{
		TTL: p.ttl,
		Records: []recordValue{
			{Value: ip.String()},
		},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, urlString, buffer)
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
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var actionResp actionResponse
	err = decoder.Decode(&actionResp)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	// Verify the action was created successfully
	if actionResp.Action.ID == 0 {
		return fmt.Errorf("%w", errors.ErrReceivedNoResult)
	}

	// Check if action status indicates success or is still running
	if actionResp.Action.Status != "running" && actionResp.Action.Status != "success" {
		return fmt.Errorf("%w: action status %s", errors.ErrUnsuccessful, actionResp.Action.Status)
	}

	return nil
}
