package vultr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// https://www.vultr.com/api/#tag/dns/operation/list-dns-domain-records
func (p *Provider) getRecord(ctx context.Context, client *http.Client,
	recordType string) (recordID string, recordIP netip.Addr, err error,
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

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return "", netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return "", netip.Addr{}, fmt.Errorf("closing response body: %w", err)
	}

	// todo: implement pagination
	var parsedJSON struct {
		Error   string `json:"error"`
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
	err = json.Unmarshal(bodyBytes, &parsedJSON)
	switch {
	case err != nil:
		return "", netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	case parsedJSON.Error != "":
		return "", netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnsuccessful, parsedJSON.Error)
	case response.StatusCode != http.StatusOK:
		return "", netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, parseJSONErrorOrFullBody(bodyBytes))
	}

	for _, record := range parsedJSON.Records {
		if record.Name != p.owner || record.Type != recordType {
			continue
		}
		recordIP, err = netip.ParseAddr(record.Data)
		if err != nil {
			return "", netip.Addr{}, fmt.Errorf("parsing existing IP: %w", err)
		}
		return record.ID, recordIP, nil
	}

	return "", netip.Addr{}, fmt.Errorf("%w: in %d records", errors.ErrRecordNotFound, len(parsedJSON.Records))
}
