package sitenl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	var settings struct {
		APIKey string `json:"api_key"`
		TTL    uint32 `json:"ttl"`
	}
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	ttl := uint32(3600)
	if settings.TTL > 0 {
		ttl = settings.TTL
	}

	err = validateSettings(domain, settings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     settings.APIKey,
		ttl:        ttl,
	}, nil
}

func validateSettings(domain, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	const apiKeyLength = 256
	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case len(apiKey) != apiKeyLength:
		return fmt.Errorf("%w: must be %d characters, got %d",
			errors.ErrKeyNotValid, apiKeyLength, len(apiKey))
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.SiteNl, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.site.nl/\">Site.nl</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	request.Header.Set("X-API-Key", p.apiKey)
}

type dnsRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Value string `json:"value"`
}

// recordFQDN builds the FQDN with trailing dot as required by the site.nl API.
// For owner "@" the root domain itself is used; otherwise owner is prepended.
func recordFQDN(owner, domain string) string {
	if owner == "@" {
		return domain + "."
	}
	return owner + "." + domain + "."
}

// getDomainID finds the integer ID for p.domain using GET /v2/domain_names.
// See https://backend.site.nl/v2
func (p *Provider) getDomainID(ctx context.Context, client *http.Client) (id uint64, err error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "backend.site.nl",
		Path:     "/v2/domain_names",
		RawQuery: url.Values{"search": {p.domain}}.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var domains []struct {
		ID                uint64 `json:"id"`
		Name              string `json:"name"`
		DNSControlEnabled int    `json:"dns_control_enabled"`
		DomainTopLevel    struct {
			TLD string `json:"top_level_domain"`
		} `json:"domain_top_level"`
	}
	if err = json.NewDecoder(response.Body).Decode(&domains); err != nil {
		return 0, fmt.Errorf("json decoding response: %w", err)
	}

	for _, d := range domains {
		fullName := d.Name + "." + d.DomainTopLevel.TLD
		if fullName == p.domain {
			if d.DNSControlEnabled == 0 {
				return 0, fmt.Errorf("%w: DNS control is not enabled for %s",
					errors.ErrFeatureUnavailable, p.domain)
			}
			return d.ID, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", errors.ErrDomainNotFound, p.domain)
}

// getRecords fetches the current DNS records for the given domain ID.
// See https://backend.site.nl/v2
func (p *Provider) getRecords(ctx context.Context, client *http.Client, domainID uint64) (
	records []dnsRecord, err error,
) {
	u := url.URL{
		Scheme: "https",
		Host:   "backend.site.nl",
		Path:   fmt.Sprintf("/v2/domain_names/%d", domainID),
	}

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
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var domain struct {
		DNSRecords []dnsRecord `json:"dns_records"`
	}
	if err = json.NewDecoder(response.Body).Decode(&domain); err != nil {
		return nil, fmt.Errorf("json decoding response: %w", err)
	}
	return domain.DNSRecords, nil
}

// patchRecords does a complete replacement of DNS records for the given domain ID.
// See https://backend.site.nl/v2
func (p *Provider) patchRecords(ctx context.Context, client *http.Client,
	domainID uint64, records []dnsRecord,
) (err error) {
	requestBody := struct {
		DNSRecords []dnsRecord `json:"dns_records"`
	}{DNSRecords: records}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("json encoding request: %w", err)
	}

	u := url.URL{
		Scheme: "https",
		Host:   "backend.site.nl",
		Path:   fmt.Sprintf("/v2/domain_names/%d/dns_records", domainID),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return nil
	}

	// Parse error body for a more descriptive message.
	var apiError struct {
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	}
	_ = json.NewDecoder(response.Body).Decode(&apiError)

	switch apiError.Error {
	case "AUTH_002", "AUTH_003", "AUTH_004", "AUTH_006":
		return fmt.Errorf("%w: %s", errors.ErrAuth, apiError.Message)
	case "DNS_001":
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, apiError.Message)
	case "DNS_002":
		return fmt.Errorf("%w: DNS control is not enabled for %s", errors.ErrFeatureUnavailable, p.domain)
	case "DOMAIN_NOT_FOUND":
		return fmt.Errorf("%w: %s", errors.ErrDomainNotFound, p.domain)
	case "EXT_001":
		return fmt.Errorf("%w: %s", errors.ErrDNSServerSide, apiError.Message)
	}

	return fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
}

// Update updates the IP address for the owner record of p.domain.
// It fetches all existing DNS records, updates or inserts the A/AAAA record
// for the configured owner, then does a complete replacement via PATCH.
// See https://backend.site.nl/v2
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	domainID, err := p.getDomainID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("finding domain ID: %w", err)
	}

	records, err := p.getRecords(ctx, client, domainID)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting DNS records: %w", err)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	targetName := recordFQDN(p.owner, p.domain)
	updated := false
	for i, r := range records {
		if r.Name == targetName && r.Type == recordType {
			if r.Value == ip.String() {
				return ip, nil // already up to date
			}
			records[i].Value = ip.String()
			updated = true
			break
		}
	}
	if !updated {
		records = append(records, dnsRecord{
			Name:  targetName,
			Type:  recordType,
			TTL:   p.ttl,
			Value: ip.String(),
		})
	}

	if err = p.patchRecords(ctx, client, domainID, records); err != nil {
		return netip.Addr{}, fmt.Errorf("updating DNS records: %w", err)
	}
	return ip, nil
}
