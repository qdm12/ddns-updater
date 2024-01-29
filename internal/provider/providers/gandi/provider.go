package gandi

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
	host       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	ttl        int
	// Authentication, either use the personal access token
	// or the deprecated API key.
	// See https://api.gandi.net/docs/authentication/
	personalAccessToken string
	// apiKey is deprecated so personalAccessToken should be used
	// instead.
	apiKey string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		PersonalAccessToken string `json:"personal_access_token"`
		APIKey              string `json:"key"`
		TTL                 int    `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:              domain,
		host:                host,
		ipVersion:           ipVersion,
		ipv6Suffix:          ipv6Suffix,
		personalAccessToken: extraSettings.PersonalAccessToken,
		apiKey:              extraSettings.APIKey,
		ttl:                 extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	if p.apiKey == "" && p.personalAccessToken == "" {
		return fmt.Errorf("%w: API key and personal access token not set", errors.ErrKeyNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Gandi, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
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
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.gandi.net/\">gandi</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.api.gandi.net",
		Path:   fmt.Sprintf("/api/v5/domains/%s/records/%s/%s", p.domain, p.host, recordType),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	const defaultTTL = 3600
	ttl := defaultTTL
	if p.ttl != 0 {
		ttl = p.ttl
	}
	requestData := struct {
		Values [1]string `json:"rrset_values"`
		TTL    int       `json:"rrset_ttl"`
	}{
		Values: [1]string{ip.Unmap().String()},
		TTL:    ttl,
	}
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	if p.personalAccessToken != "" {
		request.Header.Set("Authorization", "Bearer "+p.personalAccessToken)
	} else {
		// Note the API key is deprecated.
		request.Header.Set("X-Api-Key", p.apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return ip, nil
}
