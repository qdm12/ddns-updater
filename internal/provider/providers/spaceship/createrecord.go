package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType, address string) error {

	const defaultTTL = 3600

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
			TTL     uint32 `json:"ttl"`
		} `json:"items"`
	}{
		Force: true,
		Items: []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Address string `json:"address"`
			TTL     uint32 `json:"ttl"`
		}{{
			Type:    recordType,
			Name:    p.owner,
			Address: address,
			TTL:     defaultTTL,
		}},
	}

	requestBody := bytes.NewBuffer(nil)
	if err := json.NewEncoder(requestBody).Encode(createData); err != nil {
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
		var apiError APIError
		if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
			return fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		}

		switch response.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: invalid API credentials", errors.ErrAuth)
		case http.StatusNotFound:
			if apiError.Detail == "SOA record for domain "+p.domain+" not found." {
				return fmt.Errorf("%w: domain %s must be configured in Spaceship first",
					errors.ErrDomainNotFound, p.domain)
			}
			return fmt.Errorf("%w: %s", errors.ErrDomainNotFound, apiError.Detail)
		case http.StatusBadRequest:
			var details string
			for _, d := range apiError.Data {
				if d.Field != "" {
					details += fmt.Sprintf(" %s: %s;", d.Field, d.Details)
				} else {
					details += fmt.Sprintf(" %s;", d.Details)
				}
			}
			return fmt.Errorf("%w:%s", errors.ErrBadRequest, details)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: rate limit exceeded", errors.ErrRateLimit)
		default:
			return fmt.Errorf("%w: %d: %s",
				errors.ErrHTTPStatusNotValid, response.StatusCode, apiError.Detail)
		}
	}

	return nil
}
