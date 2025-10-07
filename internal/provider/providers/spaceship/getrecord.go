package spaceship

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) getRecords(ctx context.Context, client *http.Client) (records []Record, err error) {
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
		var apiError APIError
		if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
			return nil, fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		}

		switch response.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: invalid API credentials", errors.ErrAuth)
		case http.StatusNotFound:
			if apiError.Detail == "SOA record for domain "+p.domain+" not found." {
				return nil, fmt.Errorf("%w: domain %s must be configured in Spaceship first",
					errors.ErrDomainNotFound, p.domain)
			}
			return nil, fmt.Errorf("%w: %s", errors.ErrRecordResourceSetNotFound, apiError.Detail)
		case http.StatusBadRequest:
			var details string
			for _, d := range apiError.Data {
				if d.Field != "" {
					details += fmt.Sprintf(" %s: %s;", d.Field, d.Details)
				} else {
					details += fmt.Sprintf(" %s;", d.Details)
				}
			}
			return nil, fmt.Errorf("%w:%s", errors.ErrBadRequest, details)
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("%w: rate limit exceeded", errors.ErrRateLimit)
		default:
			return nil, fmt.Errorf("%w: %d: %s",
				errors.ErrHTTPStatusNotValid, response.StatusCode, apiError.Detail)
		}
	}

	var recordsResponse struct {
		Items []Record `json:"items"`
	}

	if err := json.NewDecoder(response.Body).Decode(&recordsResponse); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return recordsResponse.Items, nil
}
