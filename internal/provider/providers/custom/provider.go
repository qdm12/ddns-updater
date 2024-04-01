package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain       string
	host         string
	ipVersion    ipversion.IPVersion
	ipv6Suffix   netip.Prefix
	url          *url.URL
	ipv4Key      string
	ipv6Key      string
	successRegex regexp.Regexp
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		URL          string        `json:"url"`
		IPv4Key      string        `json:"ipv4key"`
		IPv6Key      string        `json:"ipv6key"`
		SuccessRegex regexp.Regexp `json:"success_regex"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding provider specific settings: %w", err)
	}

	parsedURL, err := url.Parse(extraSettings.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	p = &Provider{
		domain:       domain,
		host:         host,
		ipVersion:    ipVersion,
		ipv6Suffix:   ipv6Suffix,
		url:          parsedURL,
		ipv4Key:      extraSettings.IPv4Key,
		ipv6Key:      extraSettings.IPv6Key,
		successRegex: extraSettings.SuccessRegex,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.url.String() == "":
		return fmt.Errorf("%w", errors.ErrURLNotSet)
	case p.url.Scheme != "https":
		return fmt.Errorf("%w: %s", errors.ErrURLNotHTTPS, p.url.Scheme)
	case p.ipv4Key == "":
		return fmt.Errorf("%w", errors.ErrIPv4KeyNotSet)
	case p.ipv6Key == "":
		return fmt.Errorf("%w", errors.ErrIPv6KeyNotSet)
	case p.successRegex.String() == "":
		return fmt.Errorf("%w", errors.ErrSuccessRegexNotSet)
	default:
		return nil
	}
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Custom, p.ipVersion)
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
	updateHostname := p.url.Hostname()
	return models.HTMLRow{
		Domain: fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:   p.Host(),
		Provider: fmt.Sprintf("<a href=\"https://%s/\">%s: %s</a>",
			updateHostname, constants.Custom, updateHostname),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	values, err := url.ParseQuery(p.url.RawQuery)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parsing URL query: %w", err)
	}
	ipKey := p.ipv4Key
	if ip.Is6() {
		ipKey = p.ipv6Key
	}
	values.Set(ipKey, ip.String())
	p.url.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	}

	if p.successRegex.MatchString(s) {
		return ip, nil
	}

	return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse,
		utils.ToSingleLine(s))
}
