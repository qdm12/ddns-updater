package powerdns

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
	serverURL  string
	apiKey     string
	serverID   string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		ServerURL string `json:"server_url"`
		APIKey    string `json:"api_key"`
		ServerID  string `json:"server_id"`
		TTL       uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	if extraSettings.ServerID == "" {
		extraSettings.ServerID = "localhost"
	}
	if extraSettings.TTL == 0 {
		extraSettings.TTL = 300
	}

	err = validateSettings(domain, extraSettings.ServerURL, extraSettings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		serverURL:  extraSettings.ServerURL,
		apiKey:     extraSettings.APIKey,
		serverID:   extraSettings.ServerID,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, serverURL, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case serverURL == "":
		return fmt.Errorf("%w", errors.ErrURLNotSet)
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	}

	_, err = url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("server URL is not valid: %w", err)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.PowerDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://doc.powerdns.com/authoritative/http-api/\">PowerDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	request.Header.Set("X-API-Key", p.apiKey)
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordName := utils.BuildURLQueryHostname(p.owner, p.domain) + "."
	zoneName := p.domain + "."

	u := fmt.Sprintf("%s/api/v1/servers/%s/zones/%s", p.serverURL, p.serverID, zoneName)

	type record struct {
		Content  string `json:"content"`
		Disabled bool   `json:"disabled"`
	}

	type rrSet struct {
		Name       string   `json:"name"`
		Type       string   `json:"type"`
		TTL        uint32   `json:"ttl"`
		ChangeType string   `json:"changetype"`
		Records    []record `json:"records"`
	}

	requestData := struct {
		RRSets []rrSet `json:"rrsets"`
	}{
		RRSets: []rrSet{
			{
				Name:       recordName,
				Type:       recordType,
				TTL:        p.ttl,
				ChangeType: "REPLACE",
				Records: []record{
					{
						Content:  ip.String(),
						Disabled: false,
					},
				},
			},
		},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return ip, nil
}
