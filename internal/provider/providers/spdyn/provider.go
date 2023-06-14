package spdyn

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
	host          string
	ipVersion     ipversion.IPVersion
	user          string
	password      string
	token         string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		User          string `json:"user"`
		Password      string `json:"password"`
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		user:          extraSettings.User,
		password:      extraSettings.Password,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	if p.token != "" {
		return nil
	}
	switch {
	case p.user == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	case p.host == "*":
		return fmt.Errorf("%w", errors.ErrHostWildcard)
	}
	return nil
}

func (p *Provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Spdyn]", p.domain, p.host)
}

func (p *Provider) Domain() string {
	return p.domain
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
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://spdyn.com/\">Spdyn DNS</a>",
		IPVersion: models.HTML(p.ipVersion),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// see https://wiki.securepoint.de/SPDyn/Variablen
	u := url.URL{
		Scheme: "https",
		Host:   "update.spdyn.de",
		Path:   "/nic/update",
	}
	hostname := utils.BuildURLQueryHostname(p.host, p.domain)
	values := url.Values{}
	values.Set("hostname", hostname)
	if p.useProviderIP {
		values.Set("myip", "10.0.0.1")
	} else {
		values.Set("myip", ip.String())
	}
	if p.token != "" {
		values.Set("user", hostname)
		values.Set("pass", p.token)
	} else {
		values.Set("user", p.user)
		values.Set("pass", p.password)
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}
	bodyString := string(b)

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(bodyString))
	}

	switch {
	case isAny(bodyString, constants.Abuse, "numhost"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAbuse)
	case isAny(bodyString, constants.Badauth, "!yours"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case strings.HasPrefix(bodyString, "good"):
		return ip, nil
	case bodyString == constants.Notfqdn:
		return netip.Addr{}, fmt.Errorf("%w: not fqdn", errors.ErrBadRequest)
	case strings.HasPrefix(bodyString, "nochg"):
		return ip, nil
	case isAny(bodyString, "nohost", "fatal"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrHostnameNotExists)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, bodyString)
	}
}

func isAny(s string, values ...string) (ok bool) {
	for _, value := range values {
		if s == value {
			return true
		}
	}
	return false
}
