package dd24

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
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (
	p *Provider, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
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
	if p.password == "" {
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Dd24, p.ipVersion)
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
		Provider:  "<a href=\"https://www.domaindiscount24.com/\">DD24</a>",
		IPVersion: models.HTML(p.ipVersion),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// see https://www.domaindiscount24.com/faq/en/dynamic-dns
	u := url.URL{
		Scheme: "https",
		Host:   "dynamicdns.key-systems.net",
		Path:   "/update.php",
	}
	values := url.Values{}
	values.Set("hostname", p.BuildDomainName())
	values.Set("password", p.password)
	if p.useProviderIP {
		values.Set("ip", "auto")
	} else {
		values.Set("ip", ip.String())
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
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(s))
	}

	s = strings.ToLower(s)

	switch {
	case strings.Contains(s, "authorization failed"):
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case s == "", strings.Contains(s, "success"):
		return ip, nil
	// TODO missing cases
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
