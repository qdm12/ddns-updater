package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordID string, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.hetzner.com",
		Path:   "/api/v1/records/" + recordID,
	}

	requestData := struct {
		Type           string `json:"type"`
		Name           string `json:"name"`
		Value          string `json:"value"`
		ZoneIdentifier string `json:"zone_id"`
		TTL            uint32 `json:"ttl"`
	}{
		Type:           recordType,
		Name:           p.owner,
		Value:          ip.String(),
		ZoneIdentifier: p.zoneIdentifier,
		TTL:            p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return ip, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Record struct {
			Value netip.Addr `json:"value"`
		} `json:"record"`
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	newIP = parsedJSON.Record.Value
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
