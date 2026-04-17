package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

// checkRecord checks if the record exists and if it is already up to date
// regarding its IP address.
// See https://docs.hetzner.cloud/reference/cloud#tag/zone-rrsets/get_zone_rrset
func (p *Provider) checkRecord(ctx context.Context, client *http.Client, ip netip.Addr) (
	exists, upToDate bool, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	url := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s",
		p.domain, p.owner, recordType)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, false, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return false, false, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return false, false, nil
	default:
		return false, false, handleErrorResponse(response)
	}

	decoder := json.NewDecoder(response.Body)
	var responseData struct {
		RRSet struct {
			ID      string `json:"id"`
			Records []struct {
				Value string `json:"value"`
			} `json:"records"`
		} `json:"rrset"`
	}
	err = decoder.Decode(&responseData)
	if err != nil {
		return true, false, fmt.Errorf("json decoding response body: %w", err)
	}

	for _, record := range responseData.RRSet.Records {
		recordIP, err := netip.ParseAddr(record.Value)
		if err == nil && recordIP == ip {
			return true, true, nil
		}
	}

	return true, false, nil
}
