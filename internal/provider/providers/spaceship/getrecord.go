package spaceship

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (p *Provider) getRecords(ctx context.Context, client *http.Client) (records []apiRecord, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   "/api/v1/dns/records/" + p.domain,
	}

	values := url.Values{}
	// pagination values, mandatory for the API
	values.Set("take", "100")
	values.Set("skip", "0")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, p.handleAPIError(response)
	}

	var data struct {
		Items []apiRecord `json:"items"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return data.Items, nil
}
