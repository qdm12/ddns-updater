package namesilo

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"

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
	key        string
	ttl        *uint32
}

type APIResponse struct {
	Reply struct {
		Code    json.Number `json:"code"`
		Detail  string      `json:"detail"`
		Records []struct {
			ID    string `json:"record_id"`
			Type  string `json:"type"`
			Host  string `json:"host"`
			Value string `json:"value"`
		} `json:"resource_record,omitempty"` // Field only available during list record
	} `json:"reply"`
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		Key string  `json:"key"`
		TTL *uint32 `json:"ttl,omitempty"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, providerSpecificSettings.Key, providerSpecificSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		key:        providerSpecificSettings.Key,
		ttl:        providerSpecificSettings.TTL,
	}, nil
}

func validateSettings(domain, key string, ttl *uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	const (
		minTTL = uint32(3600)
		maxTTL = uint32(2592001)
	)
	switch {
	case key == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case ttl != nil && *ttl < minTTL:
		return fmt.Errorf("%w: %d must be at least %d", errors.ErrTTLTooLow, *ttl, minTTL)
	case ttl != nil && *ttl > maxTTL:
		return fmt.Errorf("%w: %d must be at least %d", errors.ErrTTLTooHigh, *ttl, maxTTL)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.NameSilo, p.ipVersion)
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
		Provider:  "<a href=\"https://www.namesilo.com/\">NameSilo</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update does the following:
// 1. if there's no record, create it.
// 2. if it exists and ip is different, update it.
// 3. if it exists and ip is the same, do nothing.
func (p *Provider) Update(ctx context.Context, client *http.Client, newIP netip.Addr) (netip.Addr, error) {
	recordType := constants.A
	if newIP.Is6() {
		recordType = constants.AAAA
	}

	// retrieve the current DNS record for the given IP type
	recordID, currentIP, err := p.getRecord(ctx, client, recordType)
	if err != nil && !stderrors.Is(err, errors.ErrRecordNotFound) {
		return netip.Addr{}, fmt.Errorf("error retrieving records for %s: %w", p.domain, err)
	}

	// if the current IP is different from the new IP, update the record
	if currentIP != newIP {
		// pass the recordID for updating, or nil for creating a new record
		if err := p.createOrUpdateRecord(ctx, client, recordID, recordType, newIP); err != nil {
			return netip.Addr{}, fmt.Errorf("error setting record for %s: %w", p.BuildDomainName(), err)
		}
	}

	// return the new IP (whether updated or unchanged)
	return newIP, nil
}

// https://www.namesilo.com/api-reference#dns/dns-list-records
func (p *Provider) getRecord(ctx context.Context, client *http.Client, recordType string) (
	id *string, ip netip.Addr, err error,
) {
	queryParams := url.Values{}
	url := p.createRequestURL("/api/dnsListRecords", queryParams)

	response, err := p.sendAPIRequest(ctx, client, url)
	if err != nil {
		return nil, netip.Addr{}, err
	}

	// find matching record
	host := utils.BuildURLQueryHostname(p.owner, p.domain)
	for _, record := range response.Reply.Records {
		if record.Host != host || record.Type != recordType {
			continue
		}
		ip, err = netip.ParseAddr(record.Value)
		if err != nil {
			return nil, netip.Addr{}, fmt.Errorf("parsing existing IP: %w", err)
		}
		return &record.ID, ip, nil
	}

	return nil, netip.Addr{}, fmt.Errorf("%w: no matching records found", errors.ErrRecordNotFound)
}

// https://www.namesilo.com/api-reference#dns/dns-add-record
// https://www.namesilo.com/api-reference#dns/dns-update-record
func (p *Provider) createOrUpdateRecord(
	ctx context.Context,
	client *http.Client,
	recordID *string,
	recordType string,
	ip netip.Addr,
) error {
	name := p.owner
	if name == "@" {
		name = ""
	}

	queryParams := url.Values{
		"rrhost":  {name},
		"rrvalue": {ip.String()},
	}
	if p.ttl != nil {
		queryParams.Set("rrttl", strconv.FormatUint(uint64(*p.ttl), 10))
	}

	var path string
	if recordID == nil {
		// create new record
		path = "/api/dnsAddRecord"
		queryParams.Set("rrtype", recordType)
	} else {
		// update record by id
		path = "/api/dnsUpdateRecord"
		queryParams.Set("rrid", *recordID)
	}

	url := p.createRequestURL(path, queryParams)

	// if the operation was successful, err will be nil
	_, err := p.sendAPIRequest(ctx, client, url)
	return err
}

func (p *Provider) createRequestURL(path string, queryParams url.Values) string {
	baseURL := url.URL{
		Scheme: "https",
		Host:   "www.namesilo.com",
		Path:   path,
	}
	queryParams.Set("version", "1")
	queryParams.Set("type", "json")
	queryParams.Set("key", p.key)
	queryParams.Set("domain", p.domain)
	baseURL.RawQuery = queryParams.Encode()
	return baseURL.String()
}

func (p *Provider) sendAPIRequest(ctx context.Context, client *http.Client, url string) (*APIResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(string(data)))
	}

	var parsedResponse APIResponse
	err = json.Unmarshal(data, &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("json decoding response body: %w", err)
	}

	err = p.validateResponseCode(parsedResponse.Reply.Code, parsedResponse.Reply.Detail)
	if err != nil {
		return nil, err
	}
	return &parsedResponse, nil
}

// https://www.namesilo.com/api-reference (Response Codes)
func (p *Provider) validateResponseCode(code json.Number, detail string) error {
	// The API inconsistently swaps between number and string typing for the code field,
	// but the value should always be an integer.
	parsedCode, err := code.Int64()
	if err != nil {
		return fmt.Errorf("parsing response code: %w", err)
	}

	codeToError := map[int64]error{
		300: nil,                          // Successful API operation
		110: errors.ErrKeyNotValid,        // Invalid API key
		112: errors.ErrFeatureUnavailable, // API not available to Sub-Accounts
		113: errors.ErrBannedAbuse,        // API account cannot be accessed from your IP
		200: errors.ErrDomainDisabled,     // Domain is not active, or does not belong to this user
		201: errors.ErrDNSServerSide,      // Internal system error
		210: errors.ErrUnsuccessful,       // General error (details in response)
		280: errors.ErrBadRequest,         // DNS modification error
	}

	if err, exists := codeToError[parsedCode]; exists {
		if err == nil {
			return nil // Successful operation, no error to return
		}
		return fmt.Errorf("%w: %d: %s", err, parsedCode, detail)
	}

	// Unknown response code
	return fmt.Errorf("%w: %d: %s", errors.ErrUnknownResponse, parsedCode, detail)
}
