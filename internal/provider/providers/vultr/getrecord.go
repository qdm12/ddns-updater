package vultr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) getRecord(ctx context.Context, client *http.Client) (r Record, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.vultr.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records", p.domain),
	}
	values := url.Values{}
	values.Set("per_page", "500")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Record{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return Record{}, err
	}
	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Error   string
		Status  uint32
		Records []Record `json:"records"`
		Meta    struct {
			Total uint32 `json:"total"`
			Links struct {
				Next     string `json:"next"`
				Previous string `json:"prev"`
			} `json:"links"`
		} `json:"meta"`
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return Record{}, fmt.Errorf("json decoding response body: %w", err)
	}

	if parsedJSON.Error != "" {
		return Record{}, fmt.Errorf("API Error: %s", parsedJSON.Error)
	}

	if response.StatusCode != http.StatusOK {
		return Record{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var existingRecord Record
	for _, rec := range parsedJSON.Records {
		if rec.Name == p.owner && (rec.Type == constants.A || rec.Type == constants.AAAA) {
			existingRecord = rec
			break
		}
	}

	return existingRecord, nil
}
