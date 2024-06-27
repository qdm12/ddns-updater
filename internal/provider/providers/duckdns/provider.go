package duckdns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
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
	domain        string
	owner         string
	ipVersion     ipversion.IPVersion
	ipv6Suffix    netip.Prefix
	token         string
	useProviderIP bool
}

const eTLD = "duckdns.org"

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	// Note domain is of the form:
	// - for retro-compatibility: "", "duckdns.org"
	// - domain.duckdns.org since duckdns.org is an eTLD.
	if domain == "" { // retro-compatibility
		domain = eTLD
	}
	if domain == eTLD { // retro-compatibility
		ownerParts := strings.Split(owner, ".")
		lastOwnerPart := ownerParts[len(ownerParts)-1]
		domain = lastOwnerPart + "." + domain // form domain.duckdns.org
		if len(ownerParts) > 1 {
			owner = strings.Join(ownerParts[:len(ownerParts)-1], ".")
		} else {
			owner = "@" // root domain
		}
	}

	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, owner, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:        domain,
		owner:         owner,
		ipVersion:     ipVersion,
		ipv6Suffix:    ipv6Suffix,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}, nil
}

var (
	regexDomain = regexp.MustCompile(`^.+\.(duckdns\.org)$`)
	tokenRegex  = regexp.MustCompile(`^[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}$`)
)

func validateSettings(domain, owner, token string) (err error) {
	const maxDomainLabels = 3 // domain.duckdns.org
	switch {
	case !regexDomain.MatchString(domain):
		return fmt.Errorf(`%w: %q must have the effective TLD "duckdns.org"`,
			errors.ErrDomainNotValid, domain)
	case strings.Count(owner, ".") > maxDomainLabels-1:
		return fmt.Errorf("%w: %q has more than %d labels",
			errors.ErrDomainNotValid, domain, maxDomainLabels)
	case owner == "*":
		return fmt.Errorf("%w: %s", errors.ErrOwnerWildcard, owner)
	case !tokenRegex.MatchString(token):
		return fmt.Errorf("%w: token %q does not match regex %q",
			errors.ErrTokenNotValid, token, tokenRegex)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.DuckDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://www.duckdns.org/\">DuckDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.duckdns.org",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("verbose", "true")
	values.Set("domains", p.BuildDomainName())
	values.Set("token", p.token)
	useProviderIP := p.useProviderIP && (ip.Is4() || !p.ipv6Suffix.IsValid())
	if !useProviderIP {
		if ip.Is6() {
			values.Set("ipv6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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

	const minChars = 2
	switch {
	case len(s) < minChars:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrResponseTooShort, s)
	case s[0:minChars] == "KO":
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case s[0:minChars] == "OK":
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
		if !useProviderIP && newIP.Compare(ip) != 0 {
			return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
				errors.ErrIPReceivedMismatch, ip, newIP)
		}
		return newIP, nil
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
