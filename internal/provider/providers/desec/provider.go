package desec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
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
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	if owner == "" {
		owner = "@" // default
	}

	err = validateSettings(domain, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
	}, nil
}

func validateSettings(domain, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("domain is not valid: %w", err)
	}

	if token == "" {
		return errors.New("token is not set")
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.DeSEC, p.ipVersion)
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
		Provider:  "<a href=\"https://desec.io/\">deSEC</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(p.BuildDomainName(), p.token),
		Host:   "update.dedyn.io",
		Path:   "",
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.owner, p.domain))
	if ip.Is6() {
		values.Set("myipv6", ip.String())
		values.Set("myipv4", "preserve")
	} else {
		values.Set("myipv4", ip.String())
		values.Set("myipv6", "preserve")
	}
	values.Set("myip", ip.String())
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

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return netip.Addr{}, fmt.Errorf("bad authentication: %s", utils.ToSingleLine(s))
	case http.StatusNotFound:
		return netip.Addr{}, fmt.Errorf("hostname does not exist: %s", utils.ToSingleLine(s))
	default:
		return netip.Addr{}, fmt.Errorf("HTTP status is not valid: %d: %s",
			response.StatusCode, utils.ToSingleLine(s))
	}

	switch {
	case strings.HasPrefix(s, constants.Notfqdn):
		return netip.Addr{}, errors.New("hostname does not exist")
	case strings.HasPrefix(s, "badrequest"):
		return netip.Addr{}, errors.New("bad request sent")
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return netip.Addr{}, fmt.Errorf("unknown response received: %s", utils.ToSingleLine(s))
	}
}
