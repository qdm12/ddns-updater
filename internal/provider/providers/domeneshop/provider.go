package domeneshop

import (
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
	token      string
	secret     string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		Token  string `json:"token"`
		Secret string `json:"secret"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, owner,
		providerSpecificSettings.Token, providerSpecificSettings.Secret)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		token:      providerSpecificSettings.Token,
		secret:     providerSpecificSettings.Secret,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
	}, nil
}

func validateSettings(domain, owner, token, secret string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case owner == "*":
		return fmt.Errorf("%w", errors.ErrOwnerWildcard)
	case token == "":
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	case secret == "":
		return fmt.Errorf("%w", errors.ErrSecretNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Domeneshop, p.ipVersion)
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
		Provider:  "<a href=\"https://domene.shop/\">Domeneshop</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Link to documentation:
// https://api.domeneshop.no/docs/#tag/ddns/paths/~1dyndns~1update/get
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.domeneshop.no",
		Path:   "/v0/dyndns/update",
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.owner, p.domain))
	values.Set("myip", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	request.SetBasicAuth(p.token, p.secret)
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNoContent:
		return ip, nil
	case http.StatusNotFound:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrHostnameNotExists, utils.BodyToSingleLine(response.Body))
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
}
