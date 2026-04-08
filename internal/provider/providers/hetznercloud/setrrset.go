package hetznercloud

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

// setRRSet legt einen RRset an oder aktualisiert ihn (upsert).
// Siehe https://docs.hetzner.cloud/reference/cloud#tag/zones/POST/zones/{zone_id}/rrsets
func (p *Provider) setRRSet(ctx context.Context, client *http.Client, zoneID string, ip netip.Addr) (
	newIP netip.Addr, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.hetzner.cloud",
		Path:   "/v1/zones/" + zoneID + "/rrsets",
	}

	requestData := struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		TTL     uint32 `json:"ttl"`
		Records []struct {
			Value string `json:"value"`
		} `json:"records"`
	}{
		Name:  p.owner,
		Type:  recordType,
		TTL:   p.ttl,
		Records: []struct {
			Value string `json:"value"`
		}{{Value: ip.String()}},
	}

	buffer := bytes.NewBuffer(nil)
	if err = json.NewEncoder(buffer).Encode(requestData); err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var result struct {
		RRSet struct {
			Records []struct {
				Value string `json:"value"`
			} `json:"records"`
		} `json:"rrset"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	if len(result.RRSet.Records) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	}

	newIP, err = netip.ParseAddr(result.RRSet.Records[0].Value)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	}

	if newIP.Compare(ip) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent %s but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
