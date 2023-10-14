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

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/ipextract"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	host          string
	ipVersion     ipversion.IPVersion
	token         string
	useProviderIP bool
}

func New(data json.RawMessage, _, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		host:          host,
		ipVersion:     ipVersion,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var tokenRegex = regexp.MustCompile(`^[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}$`)

func (p *Provider) isValid() error {
	if !tokenRegex.MatchString(p.token) {
		return fmt.Errorf("%w: token %q does not match regex %q",
			errors.ErrTokenNotValid, p.token, tokenRegex)
	}
	switch p.host {
	case "@", "*":
		return fmt.Errorf("%w: %q is not valid",
			errors.ErrHostOnlySubdomain, p.host)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString("duckdns.org", p.host, constants.DuckDNS, p.ipVersion)
}

func (p *Provider) Domain() string {
	return "duckdns.org"
}

func (p *Provider) Host() string {
	return p.host
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, "duckdns.org")
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://duckdns.org\">DuckDNS</a>",
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
	values.Set("domains", p.host)
	values.Set("token", p.token)
	if !p.useProviderIP {
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
		if !p.useProviderIP && newIP.Compare(ip) != 0 {
			return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
				errors.ErrIPReceivedMismatch, ip, newIP)
		}
		return newIP, nil
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
