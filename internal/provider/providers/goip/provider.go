package goip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	domain        string
	ipVersion     ipversion.IPVersion
	ipv6Suffix    netip.Prefix
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"providor_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:        domain,
		ipVersion:     ipVersion,
		ipv6Suffix:    ipv6Suffix,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrUsernameNotSet)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain,"@", constants.GoIP, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
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
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.goip.de/\">GoIP.de</a>",
		IPVersion: p.ipVersion.String(),
	}
}
//https://www.goip.de/setip?username=%5BUser%5D&password=%5BPass%5D&subdomain=%5Bsubdomain.goip.de%5D
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(p.username, p.password),
		Host:   "www.goip.de",
		Path:   "/setip",
	}
	values := url.Values{}
	values.Set("subdomain", p.domain)
  values.Set("username", p.username)
  values.Set("password", p.password)
  values.Set("shortResponse", "true")
	useProviderIP := p.useProviderIP && (ip.Is4() || !p.ipv6Suffix.IsValid())
	if !useProviderIP {
		values.Set("ip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
  fmt.Println(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNoContent:
		return ip, nil
	case http.StatusUnauthorized:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case http.StatusConflict:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrZoneNotFound)
	case http.StatusGone:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAccountInactive)
	case http.StatusLengthRequired:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrIPSentMalformed, ip)
	case http.StatusPreconditionFailed:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrPrivateIPSent, ip)
	case http.StatusServiceUnavailable:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrDNSServerSide)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}
	s := string(b)
  fmt.Println("Start:" + s + ":End")
	switch {
  case strings.HasPrefix(s, p.domain + " (" + ip.String() + ")"):
    fmt.Println("All good")
    return ip, nil
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
