package gigahostno

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
	// email is the account email, used with password for HTTP Basic auth.
	email    string
	password string
	// apiKey is used as a bearer token instead of email and password.
	// It is required for accounts with two-factor authentication enabled.
	apiKey string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		APIKey   string `json:"apikey"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain,
		extraSettings.Email, extraSettings.Password, extraSettings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		email:      extraSettings.Email,
		password:   extraSettings.Password,
		apiKey:     extraSettings.APIKey,
	}, nil
}

func validateSettings(domain, email, password, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	// Authentication is done either with an API key or with the
	// account email and password.
	if apiKey != "" {
		return nil
	}

	switch {
	case email == "" && password == "":
		return fmt.Errorf("%w: API key, or email and password, must be set",
			errors.ErrCredentialsNotSet)
	case email == "":
		return fmt.Errorf("%w", errors.ErrEmailNotSet)
	case password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.GigahostNo, p.ipVersion)
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
		Provider:  "<a href=\"https://gigahost.no/\">Gigahost</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates the IP address for the domain using the Gigahost dynamic DNS API.
// See https://gigahost.no/en/api-dokumentasjon for the API documentation.
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.gigahost.no",
		Path:   "/api/v0/dns/dyndns",
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.owner, p.domain))
	if ip.Is4() {
		values.Set("myip", ip.String())
	} else {
		values.Set("myipv6", ip.String())
	}
	if p.apiKey == "" {
		// HTTP Basic authentication using the account email and password.
		u.User = url.UserPassword(p.email, p.password)
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	if p.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	switch {
	case strings.HasPrefix(s, "good"), strings.HasPrefix(s, "nochg"):
		// success, handled below
	case response.StatusCode == http.StatusUnauthorized,
		strings.HasPrefix(s, constants.Badauth):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case strings.HasPrefix(s, constants.Nohost), strings.HasPrefix(s, constants.Notfqdn):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrHostnameNotExists)
	case strings.HasPrefix(s, constants.Badagent):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrIPSentMalformed)
	case strings.HasPrefix(s, "dnserr"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrDNSServerSide)
	case response.StatusCode != http.StatusOK:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}

	var ips []netip.Addr
	if ip.Is4() {
		ips = ipextract.IPv4(s)
	} else {
		ips = ipextract.IPv6(s)
	}
	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrReceivedNoIP)
	}

	newIP = ips[0]
	if newIP.Compare(ip) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
