package spaceship

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Record struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (p *Provider) getRecords(ctx context.Context, client *http.Client) (
	records []Record, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   fmt.Sprintf("/api/v1/dns/records/%s", p.domain),
	}

	values := url.Values{}
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

	var recordsResponse struct {
		Items []Record `json:"items"`
	}

	if err := json.NewDecoder(response.Body).Decode(&recordsResponse); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return recordsResponse.Items, nil
}
