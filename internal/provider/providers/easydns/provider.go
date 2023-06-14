package easydns

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
	username      string
	token         string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
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
		username:      extraSettings.Username,
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
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.token == "":
		return fmt.Errorf("%w", errors.ErrEmptyToken)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.EasyDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://easydns.com\">EasyDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (
	newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.cp.easydns.com",
		Path:   "dyn/generic.php",
		User:   url.UserPassword(p.username, p.token),
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.host, p.domain))
	if !p.useProviderIP {
		values.Set("myip", ip.String())
	}
	if p.host == "*" {
		values.Set("wildcard", "ON")
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
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus,
			response.StatusCode, utils.ToSingleLine(s))
	}

	switch {
	case s == "":
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrNoResultReceived)
	case strings.Contains(s, "NO_SERVICE"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrNoService)
	case strings.Contains(s, "NO_ACCESS"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case strings.Contains(s, "ILLEGAL_INPUT"), strings.Contains(s, "TOO_SOON"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAbuse)
	case strings.Contains(s, "NO_ERROR"), strings.Contains(s, "OK"):
		return ip, nil
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}
}
