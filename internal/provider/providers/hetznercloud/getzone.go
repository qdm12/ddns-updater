package hetznercloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// getZoneID ermittelt die Zone-ID anhand des Domain-Namens.
// Siehe https://docs.hetzner.cloud/reference/cloud#tag/zones/GET/zones
func (p *Provider) getZoneID(ctx context.Context, client *http.Client) (zoneID string, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.hetzner.cloud",
		Path:   "/v1/zones",
	}
	values := url.Values{}
	values.Set("name", p.domain)
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var result struct {
		Zones []struct {
			ID string `json:"id"`
		} `json:"zones"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("json decoding response body: %w", err)
	}

	if len(result.Zones) == 0 {
		return "", fmt.Errorf("%w: %s", errors.ErrZoneNotFound, p.domain)
	}
	return result.Zones[0].ID, nil
}
