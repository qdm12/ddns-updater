package netlify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

const defaultTTL = 3600

// Provider is Netlify DNS provider.
type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	token      string
	httpClient *http.Client
}

// New creates a new Netlify provider.
func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix,
) (p *Provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}

	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}

	p = &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
		httpClient: &http.Client{},
	}

	err = p.validateSettings()
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return p, nil
}

func (p *Provider) validateSettings() error {
	if p.token == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
} // String returns a string representation of provider.
func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Netlify, p.ipVersion)
}

// Domain returns domain of provider.
func (p *Provider) Domain() string {
	return p.domain
}

// Owner returns owner of provider.
func (p *Provider) Owner() string {
	return p.owner
}

// IPVersion returns IP version of provider.
func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

// IPv6Suffix returns IPv6 suffix of provider.
func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

// BuildDomainName builds the full domain name.
func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

// Proxied returns whether the provider is proxied.
func (p *Provider) Proxied() bool {
	return false
}

// HTML returns HTML representation of provider.
func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.netlify.com\">Netlify</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates DNS record.
func (p *Provider) Update(ctx context.Context, _ *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Find the DNS zone for the domain.
	zone, err := p.findZone(ctx)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to find DNS zone: %w", err)
	}

	// List existing DNS records to find the one to update.
	records, err := p.listDNSRecords(ctx, zone.ID)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to list DNS records: %w", err)
	}

	// Determine the record type and hostname.
	recordType := p.getRecordType()
	hostname := p.getHostname()
	ipStr := ip.String()

	// Find existing record.
	var existingRecord *dnsRecord
	for _, record := range records {
		if record.Type == recordType && record.Hostname == hostname {
			existingRecord = &record
			break
		}
	}

	// If record exists and IP is the same, no update needed.
	if existingRecord != nil && existingRecord.Value == ipStr {
		return ip, nil
	}

	// Create or update the record.
	if existingRecord != nil {
		err = p.updateDNSRecord(ctx, zone.ID, existingRecord.ID, ipStr)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("failed to update DNS record: %w", err)
		}
	} else {
		err = p.createDNSRecord(ctx, zone.ID, hostname, recordType, ipStr)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("failed to create DNS record: %w", err)
		}
	}

	return ip, nil
}

func (p *Provider) getRecordType() string {
	if p.ipVersion == ipversion.IP4 {
		return "A"
	}
	return "AAAA"
}

func (p *Provider) getHostname() string {
	if p.owner == "@" {
		return p.domain
	}
	return p.owner + "." + p.domain
}

// findZone finds the DNS zone for the domain.
func (p *Provider) findZone(ctx context.Context) (*dnsZone, error) {
	url := "https://api.netlify.com/api/v1/dns_zones"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", errors.ErrHTTPStatusNotValid, resp.StatusCode, string(body))
	}

	var zones []dnsZone
	if err := json.Unmarshal(body, &zones); err != nil {
		return nil, err
	}

	// Find the zone that matches our domain.
	for _, zone := range zones {
		if zone.Name == p.domain || strings.HasSuffix(p.domain, "."+zone.Name) {
			return &zone, nil
		}
	}

	return nil, fmt.Errorf("%w: for domain %s", errors.ErrZoneNotFound, p.domain)
}

// listDNSRecords lists all DNS records for a zone.
func (p *Provider) listDNSRecords(ctx context.Context, zoneID string) ([]dnsRecord, error) {
	url := fmt.Sprintf("https://api.netlify.com/api/v1/dns_zones/%s/dns_records", zoneID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", errors.ErrHTTPStatusNotValid, resp.StatusCode, string(body))
	}

	var records []dnsRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, err
	}

	return records, nil
}

// createDNSRecord creates a new DNS record.
func (p *Provider) createDNSRecord(ctx context.Context, zoneID, hostname, recordType, value string) error {
	url := fmt.Sprintf("https://api.netlify.com/api/v1/dns_zones/%s/dns_records", zoneID)

	record := dnsRecordCreate{
		Type:     recordType,
		Hostname: hostname,
		Value:    value,
		TTL:      defaultTTL, // Default TTL
	}

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	p.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", errors.ErrHTTPStatusNotValid, resp.StatusCode, string(body))
	}

	return nil
}

// updateDNSRecord updates an existing DNS record.
func (p *Provider) updateDNSRecord(ctx context.Context, zoneID, recordID, value string) error {
	// Netlify API doesn't have a direct update endpoint for DNS records.
	// We need to delete the existing record and create a new one.

	// First, get the existing record details.
	record, err := p.getDNSRecord(ctx, zoneID, recordID)
	if err != nil {
		return fmt.Errorf("failed to get existing DNS record: %w", err)
	}

	// Delete the existing record.
	if err := p.deleteDNSRecord(ctx, zoneID, recordID); err != nil {
		return fmt.Errorf("failed to delete existing DNS record: %w", err)
	}

	// Create a new record with the updated value.
	return p.createDNSRecord(ctx, zoneID, record.Hostname, record.Type, value)
}

// getDNSRecord gets a single DNS record.
func (p *Provider) getDNSRecord(ctx context.Context, zoneID, recordID string) (*dnsRecord, error) {
	url := fmt.Sprintf("https://api.netlify.com/api/v1/dns_zones/%s/dns_records/%s", zoneID, recordID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", errors.ErrHTTPStatusNotValid, resp.StatusCode, string(body))
	}

	var record dnsRecord
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

// deleteDNSRecord deletes a DNS record.
func (p *Provider) deleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	url := fmt.Sprintf("https://api.netlify.com/api/v1/dns_zones/%s/dns_records/%s", zoneID, recordID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", errors.ErrHTTPStatusNotValid, resp.StatusCode, string(body))
	}

	return nil
}

func (p *Provider) setAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.token)
}

// DNS zone structure.
type dnsZone struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Errors               []string    `json:"errors"`
	SupportedRecordTypes []string    `json:"supported_record_types"`
	UserID               string      `json:"user_id"`
	CreatedAt            string      `json:"created_at"`
	UpdatedAt            string      `json:"updated_at"`
	Records              []dnsRecord `json:"records"`
	DNSServers           []string    `json:"dns_servers"`
	AccountID            string      `json:"account_id"`
	SiteID               string      `json:"site_id"`
	AccountSlug          string      `json:"account_slug"`
	AccountName          string      `json:"account_name"`
	Domain               string      `json:"domain"`
	IPv6Enabled          bool        `json:"ipv6_enabled"`
	Dedicated            bool        `json:"dedicated"`
}

// DNS record structure.
type dnsRecord struct {
	ID        string `json:"id"`
	Hostname  string `json:"hostname"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	TTL       int64  `json:"ttl"`
	Priority  int64  `json:"priority"`
	DNSZoneID string `json:"dns_zone_id"`
	SiteID    string `json:"site_id"`
	Flag      int64  `json:"flag"`
	Tag       string `json:"tag"`
	Managed   bool   `json:"managed"`
}

// DNS record creation structure.
type dnsRecordCreate struct {
	Type     string `json:"type"`
	Hostname string `json:"hostname"`
	Value    string `json:"value"`
	TTL      int64  `json:"ttl"`
	Priority int64  `json:"priority,omitempty"`
	Weight   int64  `json:"weight,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Flag     int64  `json:"flag,omitempty"`
	Tag      string `json:"tag,omitempty"`
}
