package hetzner

import (
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

// See https://dns.hetzner.com/api-docs#operation/GetZones.
func (p *Provider) getRecordID(ctx context.Context, client *http.Client, ip netip.Addr) (
	identifier string, upToDate bool, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.hetzner.com",
		Path:   "/api/v1/records",
	}

	values := url.Values{}
	values.Set("zone_id", p.zoneIdentifier)
	values.Set("name", p.owner)
	values.Set("type", recordType)
	values.Set("page", "1")
	values.Set("per_page", "1")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", false, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", false, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	default:
		return "", false, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	listRecordsResponse := struct {
		Records []struct {
			ID    string     `json:"id"`
			Value netip.Addr `json:"value"`
		} `json:"records"`
	}{}
	err = decoder.Decode(&listRecordsResponse)
	if err != nil {
		return "", false, fmt.Errorf("json decoding response body: %w", err)
	}

	switch {
	case len(listRecordsResponse.Records) == 0:
		return "", false, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	case len(listRecordsResponse.Records) > 1:
		return "", false, fmt.Errorf("%w: %d instead of 1",
			errors.ErrResultsCountReceived, len(listRecordsResponse.Records))
	}
	identifier = listRecordsResponse.Records[0].ID
	upToDate = listRecordsResponse.Records[0].Value.Compare(ip) == 0
	return identifier, upToDate, nil
}
