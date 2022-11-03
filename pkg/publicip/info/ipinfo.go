package info

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

func newIpinfo(client *http.Client) *ipinfo {
	return &ipinfo{
		client: client,
	}
}

type ipinfo struct {
	client *http.Client
}

func (p *ipinfo) get(ctx context.Context, ip net.IP) (
	result Result, err error) {
	result.Source = string(Ipinfo)

	url := "https://ipinfo.io/"
	if ip != nil {
		url += ip.String()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, fmt.Errorf("creating request: %w", err)
	}

	response, err := p.client.Do(request)
	if err != nil {
		return result, fmt.Errorf("doing request: %w", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		bodyString := bodyToSingleLine(response.Body)
		_ = response.Body.Close()
		return result, fmt.Errorf("%w (%s)", ErrTooManyRequests, bodyString)
	default:
		bodyString := bodyToSingleLine(response.Body)
		_ = response.Body.Close()
		return result, fmt.Errorf("%w: %d %s (%s)", ErrBadHTTPStatus,
			response.StatusCode, response.Status, bodyString)
	}

	decoder := json.NewDecoder(response.Body)
	var data struct {
		IP      net.IP `json:"ip"`
		Region  string `json:"region"`
		Country string `json:"country"`
		City    string `json:"city"`
	}
	err = decoder.Decode(&data)
	if err != nil {
		return result, fmt.Errorf("decoding JSON response: %w", err)
	}

	result.IP = data.IP
	if data.Region != "" {
		result.Region = stringPtr(data.Region)
	}
	if data.City != "" {
		result.City = stringPtr(data.City)
	}
	if data.Country != "" {
		country := countryCodeToName(data.Country)
		result.Country = stringPtr(country)
	}

	return result, nil
}
