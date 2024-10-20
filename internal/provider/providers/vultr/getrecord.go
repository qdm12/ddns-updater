package vultr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// https://www.vultr.com/api/#tag/dns/operation/list-dns-domain-records
func (p *Provider) getRecord(ctx context.Context, client *http.Client,
	recordType string) (recordID string, existingIP netip.Addr, err error,
) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.vultr.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records", p.domain),
	}

	// max return of get records is 500 records
	values := url.Values{}
	values.Set("per_page", "500")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", netip.Addr{}, err
	}
	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)

	// todo: implement pagination
	var parsedJSON struct {
		Error   string
		Records []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			Data string `json:"data"`
		} `json:"records"`
		Meta struct {
			Total uint32 `json:"total"`
			Links struct {
				Next     string `json:"next"`
				Previous string `json:"prev"`
			} `json:"links"`
		} `json:"meta"`
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return "", netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	if parsedJSON.Error != "" {
		return "", netip.Addr{}, fmt.Errorf("API Error: %s", parsedJSON.Error)
	}

	if response.StatusCode != http.StatusOK {
		return "", netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	for _, rec := range parsedJSON.Records {
		if rec.Name != p.owner || rec.Type != recordType {
			continue
		}
		existingIP, err = netip.ParseAddr(rec.Data)
		if err != nil {
			return "", netip.Addr{}, fmt.Errorf("parsing existing IP: %w", err)
		}
		return rec.ID, existingIP, nil
	}

	return "", netip.Addr{}, fmt.Errorf("%w: in %d records", errors.ErrRecordNotFound, len(parsedJSON.Records))
}
