package duckdns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
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
		return fmt.Errorf("%w", errors.ErrMalformedToken)
	}
	switch p.host {
	case "@", "*":
		return fmt.Errorf("%w", errors.ErrHostOnlySubdomain)
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
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://duckdns.org\">DuckDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.duckdns.org",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("verbose", "true")
	values.Set("domains", p.host)
	values.Set("token", p.token)
	u.RawQuery = values.Encode()
	if !p.useProviderIP {
		if ip.To4() == nil {
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(s))
	}

	const minChars = 2
	switch {
	case len(s) < minChars:
		return nil, fmt.Errorf("%w: response %q is too short", errors.ErrUnmarshalResponse, s)
	case s[0:minChars] == "KO":
		return nil, fmt.Errorf("%w", errors.ErrAuth)
	case s[0:minChars] == "OK":
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, fmt.Errorf("%w", errors.ErrNoResultReceived)
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ips[0])
		}
		if ip != nil && !newIP.Equal(ip) {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
