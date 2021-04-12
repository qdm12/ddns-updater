package info

import (
	"context"
	"encoding/json"
	"errors"
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

var ErrBadHTTPStatus = errors.New("bad HTTP status received")

func (p *ipinfo) get(ctx context.Context, ip net.IP) (
	result Result, err error) {
	const baseURL = "https://ipinfo.io/"
	url := baseURL + ip.String()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, err
	}

	response, err := p.client.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("%w: %s", ErrBadHTTPStatus, response.Status)
	}

	decoder := json.NewDecoder(response.Body)
	var data struct {
		Region  string `json:"region"`
		Country string `json:"country"`
		City    string `json:"city"`
	}
	if err := decoder.Decode(&data); err != nil {
		return result, err
	}

	if len(data.Region) > 0 {
		result.Region = stringPtr(data.Region)
	}
	if len(data.City) > 0 {
		result.City = stringPtr(data.City)
	}
	if len(data.Country) > 0 {
		country := countryCodeToName(data.Country)
		result.Country = stringPtr(country)
	}

	return result, nil
}
