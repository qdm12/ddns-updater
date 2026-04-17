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

// updateRecord updates an existing DNS record using the set_records action.
// It replaces all existing records with the new IP address.
func (p *Provider) updateRecord(ctx context.Context, client *http.Client, ip netip.Addr) (
	newIP netip.Addr, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	const urlTemplate = "https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s/actions/set_records"
	urlString := fmt.Sprintf(urlTemplate, p.domain, p.owner, recordType)

	requestData := recordsRequest{
		Records: []recordValue{
			{Value: ip.String()},
		},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, urlString, buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return netip.Addr{}, handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)
	var actionResp actionResponse
	err = decoder.Decode(&actionResp)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	err = p.handleActionResponse(ctx, client, actionResp)
	if err != nil {
		return netip.Addr{}, err
	}

	return ip, nil
}
