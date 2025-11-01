package hetznernetworking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// getRecordID fetches the RRSet ID and checks if the IP is up to date.
// It returns the record ID, whether the IP is up to date, and any error.
// If the record doesn't exist, it returns ErrReceivedNoResult.
// See https://docs.hetzner.cloud/reference/cloud#dns
func (p *Provider) getRecordID(ctx context.Context, client *http.Client, ip netip.Addr) (
	identifier string, upToDate bool, err error,
) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	// Extract RR name from domain relative to zone
	rrName, err := p.extractRRName()
	if err != nil {
		return "", false, fmt.Errorf("extracting RR name: %w", err)
	}

	urlString := fmt.Sprintf("https://api.hetzner.cloud/v1/zones/%s/rrsets/%s/%s", p.zoneIdentifier, rrName, recordType)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlString, nil)
	if err != nil {
		return "", false, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", false, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	default:
		return "", false, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var rrSetResponse rrSetResponse
	err = decoder.Decode(&rrSetResponse)
	if err != nil {
		return "", false, fmt.Errorf("json decoding response body: %w", err)
	}

	// Check if any record value matches the current IP
	for _, record := range rrSetResponse.RRSet.Records {
		recordIP, err := netip.ParseAddr(record.Value)
		if err != nil {
			continue // Skip invalid IPs
		}
		if recordIP.Compare(ip) == 0 {
			return rrSetResponse.RRSet.ID, true, nil
		}
	}

	// Record exists but IP doesn't match
	return rrSetResponse.RRSet.ID, false, nil
}

// extractRRName extracts the RR name from the domain relative to the zone
// For example: domain="sub.example.com", zone="example.com" -> "sub"
// For example: domain="example.com", zone="example.com" -> "@"
// For example: domain="*.sub.example.com", zone="example.com" -> "*.sub"
func (p *Provider) extractRRName() (string, error) {
	domain := p.BuildDomainName()
	zone := p.zoneIdentifier

	// Normalize domain and zone to lowercase
	domain = strings.ToLower(domain)
	zone = strings.ToLower(zone)

	// Remove trailing dots if present
	domain = strings.TrimSuffix(domain, ".")
	zone = strings.TrimSuffix(zone, ".")

	// If domain equals zone, this is the apex record
	if domain == zone {
		return "@", nil
	}

	// Check if domain is a subdomain of zone
	if !strings.HasSuffix(domain, "."+zone) {
		return "", fmt.Errorf("domain %s is not a subdomain of zone %s", domain, zone)
	}

	// Extract subdomain part
	subdomain := strings.TrimSuffix(domain, "."+zone)
	if subdomain == "" {
		return "@", nil
	}

	// For wildcard domains, keep the * character
	// For example: "*.sub" should remain "*.sub"
	return subdomain, nil
}
