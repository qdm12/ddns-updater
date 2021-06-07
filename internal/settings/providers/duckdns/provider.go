package duckdns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type provider struct {
	host          string
	ipVersion     ipversion.IPVersion
	token         string
	useProviderIP bool
	matcher       regex.Matcher
	logger        log.Logger
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		host:          host,
		ipVersion:     ipVersion,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
		matcher:       matcher,
		logger:        logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if !p.matcher.DuckDNSToken(p.token) {
		return errors.ErrMalformedToken
	}
	switch p.host {
	case "@", "*":
		return errors.ErrHostOnlySubdomain
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString("duckdns.org", p.host, constants.DuckDNS, p.ipVersion)
}

func (p *provider) Domain() string {
	return "duckdns.org"
}

func (p *provider) Host() string {
	return p.host
}

func (p *provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *provider) Proxied() bool {
	return false
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, "duckdns.org")
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://duckdns.org\">DuckDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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

	p.logger.Debug("HTTP GET: " + u.String())

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
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
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
		return nil, errors.ErrAuth
	case s[0:minChars] == "OK":
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, errors.ErrNoResultReceived
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
