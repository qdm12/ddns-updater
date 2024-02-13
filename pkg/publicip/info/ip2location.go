package info

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
)

func newIP2Location(client *http.Client) *ip2Location {
	return &ip2Location{
		client: client,
	}
}

type ip2Location struct {
	client *http.Client
}

func (p *ip2Location) get(ctx context.Context, ip netip.Addr) (
	result Result, err error) {
	result.Source = string(Ipinfo)

	url := "https://api.ip2location.io/"
	if ip.IsValid() {
		url += "?ip=" + ip.String()
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
		IP          netip.Addr `json:"ip"`
		RegionName  string     `json:"region_name"`
		CountryName string     `json:"country_name"`
		CityName    string     `json:"city_name"`
		// More fields available see https://www.ip2location.io/ip2location-documentation
	}
	err = decoder.Decode(&data)
	if err != nil {
		return result, fmt.Errorf("decoding JSON response: %w", err)
	}

	result.IP = data.IP
	if data.RegionName != "" {
		result.Region = stringPtr(data.RegionName)
	}
	if data.CityName != "" {
		result.City = stringPtr(data.CityName)
	}
	if data.CountryName != "" {
		country := countryCodeToName(data.CountryName)
		result.Country = stringPtr(country)
	}

	return result, nil
}
