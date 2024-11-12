package nowdns

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
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	username   string
	password   string
}

func New(data json.RawMessage, domain string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Username, extraSettings.Password)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		username:   extraSettings.Username,
		password:   extraSettings.Password,
	}, nil
}

func validateSettings(domain, username, password string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case username == "":
		return fmt.Errorf("%w", errors.ErrUsernameNotSet)
	case password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, "@", constants.NowDNS, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return "@"
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
	return p.domain
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.now-dns.com/\">Now-DNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "now-dns.com",
		Path:   "/update",
		User:   url.UserPassword(p.username, p.password),
	}

	values := url.Values{}
	values.Set("hostname", p.domain)
	values.Set("myip", ip.String())
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

	switch response.StatusCode {
	case http.StatusOK:
		switch {
		case strings.Contains(s, "good"):
			newIP, err = netip.ParseAddr(ip.String())
			if err != nil {
				return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
			} else if ip.Compare(newIP) != 0 {
				return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
					errors.ErrIPReceivedMismatch, ip, newIP)
			}
			return ip, nil
		case strings.Contains(s, "nochg"):
			newIP, err = netip.ParseAddr(ip.String())
			if err != nil {
				return netip.Addr{}, fmt.Errorf("%w: in response %q", errors.ErrReceivedNoResult, s)
			} else if ip.Compare(newIP) != 0 {
				return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
					errors.ErrIPReceivedMismatch, ip, newIP)
			}
			return ip, nil
		default:
			return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
		}
	case http.StatusBadRequest:
		switch s {
		case constants.Nohost:
			return netip.Addr{}, fmt.Errorf("%w", errors.ErrHostnameNotExists)
		case constants.Badauth:
			return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
		default:
			return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
		}
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid, response.StatusCode, s)
	}
}
