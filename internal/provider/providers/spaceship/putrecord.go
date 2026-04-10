package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (p *Provider) putRecord(ctx context.Context, client *http.Client, record apiRecord) error {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   "/api/v1/dns/records/" + p.domain,
	}

	requestData := struct {
		Force bool        `json:"force"`
		Items []apiRecord `json:"items"`
	}{
		Force: true,
		Items: []apiRecord{record},
	}

	requestBody := bytes.NewBuffer(nil)
	if err := json.NewEncoder(requestBody).Encode(requestData); err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), requestBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return p.handleAPIError(response)
	}

	return nil
}
