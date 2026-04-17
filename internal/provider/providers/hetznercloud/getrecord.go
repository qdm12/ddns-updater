package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// getRecordID fetches the RRSet ID and checks if the IP is up to date.
// It returns the record ID, whether the IP is up to date, and any error.
// If the record doesn't exist, it returns ErrReceivedNoResult.
// See https://docs.hetzner.cloud/reference/cloud#dns
func (p *Provider) getRecordID(ctx context.Context, client *http.Client, ip netip.Addr) (
	upToDate bool, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	urlString := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s", p.domain, p.owner, recordType)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlString, nil)
	if err != nil {
		return false, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, fmt.Errorf("%w", errors.ErrRecordNotFound)
	default:
		return false, handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)
	var rrSetResponse struct {
		RRSet struct {
			ID      string `json:"id"`
			Records []struct {
				Value string `json:"value"`
			} `json:"records"`
		} `json:"rrset"`
	}
	err = decoder.Decode(&rrSetResponse)
	if err != nil {
		return false, fmt.Errorf("json decoding response body: %w", err)
	}

	// Check if any record value matches the current IP
	for _, record := range rrSetResponse.RRSet.Records {
		recordIP, err := netip.ParseAddr(record.Value)
		if err != nil {
			continue // Skip invalid IPs
		}
		if recordIP.Compare(ip) == 0 {
			return true, nil
		}
	}

	// Record exists but IP doesn't match
	return false, nil
}
