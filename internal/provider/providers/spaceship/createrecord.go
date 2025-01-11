package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType, address string) error {
	u := url.URL{
		Scheme: "https",
		Host:   "spaceship.dev",
		Path:   fmt.Sprintf("/api/v1/dns/records/%s", p.domain),
	}

	createData := struct {
		Force bool `json:"force"`
		Items []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Address string `json:"address"`
			TTL     int    `json:"ttl"`
		} `json:"items"`
	}{
		Force: true,
		Items: []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Address string `json:"address"`
			TTL     int    `json:"ttl"`
		}{{
			Type:    recordType,
			Name:    p.owner,
			Address: address,
			TTL:     3600,
		}},
	}

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(createData); err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), &requestBody)
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
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	return nil
}
