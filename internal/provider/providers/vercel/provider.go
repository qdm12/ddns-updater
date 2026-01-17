package vercel

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
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
	token      string
	teamID     string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Token  string `json:"token"`
		TeamID string `json:"team_id"`
		TTL    uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	// Default TTL to 60 seconds if not set
	ttl := extraSettings.TTL
	if ttl == 0 {
		ttl = 60
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
		teamID:     extraSettings.TeamID,
		ttl:        ttl,
	}, nil
}

func validateSettings(domain, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if token == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Vercel, p.ipVersion)
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
		Provider:  "<a href=\"https://vercel.com/\">Vercel</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
	headers.SetContentType(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) makeURL(path string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "api.vercel.com",
		Path:   path,
	}
	if p.teamID != "" {
		values := url.Values{}
		values.Set("teamId", p.teamID)
		u.RawQuery = values.Encode()
	}
	return u.String()
}

type dnsRecord struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}

func (p *Provider) getRecord(ctx context.Context, client *http.Client, recordType string) (
	record *dnsRecord, err error,
) {
	u := p.makeURL("/v4/domains/" + p.domain + "/records")

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
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

	decoder := json.NewDecoder(response.Body)
	var result struct {
		Records []dnsRecord `json:"records"`
	}
	err = decoder.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("json decoding response body: %w", err)
	}

	// Find the matching record by name and type
	targetName := p.owner
	if targetName == "@" {
		targetName = ""
	}

	for _, r := range result.Records {
		if r.Name == targetName && r.Type == recordType {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("%w", errors.ErrReceivedNoResult)
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client, ip netip.Addr) error {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := p.makeURL("/v4/domains/" + p.domain + "/records")

	name := p.owner
	if name == "@" {
		name = ""
	}

	requestData := struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Value   string `json:"value"`
		TTL     uint32 `json:"ttl,omitempty"`
		Comment string `json:"comment,omitempty"`
	}{
		Name:    name,
		Type:    recordType,
		Value:   ip.String(),
		TTL:     p.ttl,
		Comment: "DDNS Updater automatically manages this record.",
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u, buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return nil
}

func (p *Provider) deleteRecord(ctx context.Context, client *http.Client, recordID string) error {
	u := p.makeURL("/v2/domains/" + p.domain + "/records/" + recordID)

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	record, err := p.getRecord(ctx, client, recordType)
	switch {
	case stderrors.Is(err, errors.ErrReceivedNoResult):
		// Record doesn't exist, create it
		err = p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting record: %w", err)
	}

	// Check if IP is already up to date
	if record.Value == ip.String() {
		return ip, nil
	}

	// Delete the existing record and create a new one with the updated IP
	err = p.deleteRecord(ctx, client, record.ID)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("deleting record: %w", err)
	}

	err = p.createRecord(ctx, client, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating record: %w", err)
	}

	return ip, nil
}

