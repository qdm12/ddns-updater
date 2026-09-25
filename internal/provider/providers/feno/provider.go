package feno

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/ipextract"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		APIKey string `json:"api_key"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, owner, extraSettings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     extraSettings.APIKey,
	}, nil
}

func validateSettings(domain, owner, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case owner == "*":
		return fmt.Errorf("%w", errors.ErrOwnerWildcard)
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case !strings.HasPrefix(apiKey, "feno_live_"):
		return fmt.Errorf("%w: it should start with feno_live_", errors.ErrKeyNotValid)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Feno, p.ipVersion)
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
		Provider:  "<a href=\"https://feno.no/\">FENO</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates the IP address for the provider.
// See https://github.com/mrerikcodes/feno-api/blob/main/docs/DDNS.md
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		// The username is ignored by the server, the API key is the password.
		User: url.UserPassword("ddns-updater", p.apiKey),
		Host: "api.feno.no",
		Path: "/v1/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.owner, p.domain))
	// The server only touches the address family it is sent, so one
	// query parameter per family keeps A and AAAA records independent.
	if ip.Is6() {
		values.Set("myipv6", ip.String())
	} else {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	return parseResponse(response.StatusCode, s, ip)
}

// parseResponse maps the dyndns2 tokens returned by the FENO API to
// errors, and verifies the address echoed back matches the one sent.
func parseResponse(statusCode int, s string, ip netip.Addr) (newIP netip.Addr, err error) {
	switch {
	case s == "":
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	case strings.HasPrefix(s, constants.Badauth):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case strings.HasPrefix(s, constants.Notfqdn):
		return netip.Addr{}, fmt.Errorf("%w: hostname is not a fully qualified domain name",
			errors.ErrHostnameNotExists)
	case strings.HasPrefix(s, constants.Nohost):
		return netip.Addr{}, fmt.Errorf("%w: hostname is not in a zone hosted on FENO nameservers",
			errors.ErrHostnameNotExists)
	case strings.HasPrefix(s, "numhost"):
		return netip.Addr{}, fmt.Errorf("%w: too many hostnames", errors.ErrBadRequest)
	case strings.HasPrefix(s, constants.Badagent):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrIPSentMalformed)
	case strings.HasPrefix(s, constants.Abuse):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrBannedAbuse)
	case strings.HasPrefix(s, "dnserr"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrDNSServerSide)
	case strings.HasPrefix(s, constants.Nineoneone):
		return netip.Addr{}, fmt.Errorf("%w: temporary server side error", errors.ErrDNSServerSide)
	}

	if statusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, statusCode, utils.ToSingleLine(s))
	}

	if !strings.HasPrefix(s, "good") && !strings.HasPrefix(s, "nochg") {
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}

	var ips []netip.Addr
	if ip.Is6() {
		ips = ipextract.IPv6(s)
	} else {
		ips = ipextract.IPv4(s)
	}

	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoIP)
	}

	newIP = ips[0]
	if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
