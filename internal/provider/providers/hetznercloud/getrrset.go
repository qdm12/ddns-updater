package hetznercloud

import (
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

// getRRSet liefert die aktuelle IP des passenden RRsets (A oder AAAA) zurück.
// Gibt errors.ErrReceivedNoResult wenn kein passender RRset vorhanden ist.
// Siehe https://docs.hetzner.cloud/reference/cloud#tag/zones/GET/zones/{zone_id}/rrsets
func (p *Provider) getRRSet(ctx context.Context, client *http.Client, zoneID string, ip netip.Addr) (
	currentIP netip.Addr, err error,
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

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var result struct {
		RRSets []struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Records []struct {
				Value string `json:"value"`
			} `json:"records"`
		} `json:"rrsets"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	// Passenden RRset nach Name (owner) und Typ filtern
	for _, rrset := range result.RRSets {
		if rrset.Name != p.owner || rrset.Type != recordType {
			continue
		}
		if len(rrset.Records) == 0 {
			return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoResult)
		}
		currentIP, err = netip.ParseAddr(rrset.Records[0].Value)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
		}
		return currentIP, nil
	}
	return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoResult)
}
