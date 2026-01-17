package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (p *Provider) deleteRecord(ctx context.Context, client *http.Client, record Record) error {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   "/api/v1/dns/records/" + p.domain,
	}

	deleteData := []Record{record}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(deleteData); err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), &requestBody)
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
