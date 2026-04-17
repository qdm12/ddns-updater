package hetznernetworking

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
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
		return false, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	default:
		return false, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var rrSetResponse rrSetResponse
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

var errDomainNotSubOfZone = stderrors.New("domain is not a subdomain of zone")
