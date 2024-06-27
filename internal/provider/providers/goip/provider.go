package goip

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
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain        string
	owner         string
	ipVersion     ipversion.IPVersion
	ipv6Suffix    netip.Prefix
	username      string
	password      string
	useProviderIP bool
}

const defaultDomain = "goip.de"

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	// Note domain is of the form:
	// - for retro-compatibility: "", "goip.de" or "goip.it"
	// - domain.goip.de or domain.goip.it since goip.de and goip.it are eTLDs.
	if domain == "" { // retro-compatibility
		domain = defaultDomain
	}
	if domain == defaultDomain || domain == "goip.it" { // retro-compatibility
		ownerParts := strings.Split(owner, ".")
		lastOwnerPart := ownerParts[len(ownerParts)-1]
		domain = lastOwnerPart + "." + domain // form domain.goip.de or domain.goip.it
		if len(ownerParts) > 1 {
			owner = strings.Join(ownerParts[:len(ownerParts)-1], ".")
		} else {
			owner = "@" // root domain
		}
	}

	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, owner, extraSettings.Username, extraSettings.Password)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:        domain,
		owner:         owner,
		ipVersion:     ipVersion,
		ipv6Suffix:    ipv6Suffix,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}, nil
}

var regexDomain = regexp.MustCompile(`^.+\.(goip\.de|goip\.it)$`)

func validateSettings(domain, owner, username, password string) (err error) {
	const maxDomainLabels = 3 // domain.goip.de
	switch {
	case username == "":
		return fmt.Errorf("%w", errors.ErrUsernameNotSet)
	case password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	case !regexDomain.MatchString(domain):
		return fmt.Errorf(`%w: %q must match have the effective TLD "goip.de" or "goip.it"`,
			errors.ErrDomainNotValid, domain)
	case strings.Count(owner, ".") > maxDomainLabels-1:
		return fmt.Errorf("%w: %q has more than %d labels",
			errors.ErrDomainNotValid, domain, maxDomainLabels)
	case owner == "*":
		return fmt.Errorf("%w: %q", errors.ErrOwnerWildcard, owner)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.owner, p.domain, constants.GoIP, p.ipVersion)
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
		Provider:  "<a href=\"https://www." + p.domain + "/\">" + p.domain + "</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// See https://www.goip.de/update-url.html
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(p.username, p.password),
		Host:   "www.goip.de",
		Path:   "/setip",
	}
	values := url.Values{}
	values.Set("subdomain", p.BuildDomainName())
	values.Set("username", p.username)
	values.Set("password", p.password)
	values.Set("shortResponse", "true")
	if ip.Is4() {
		if !p.useProviderIP {
			values.Set("ip", ip.String())
		}
	} else {
		// IPv6 cannot be automatically detected
		values.Set("ip6", ip.String())
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

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}
	s := string(b)
	switch {
	case strings.HasPrefix(s, p.BuildDomainName()+" ("+ip.String()+")"):
		return ip, nil
	case strings.HasPrefix(strings.ToLower(s), "zugriff verweigert"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}
}
