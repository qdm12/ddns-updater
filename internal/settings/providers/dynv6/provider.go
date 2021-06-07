package dynv6

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	token         string
	useProviderIP bool
	logger        log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
		logger:        logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case len(p.token) == 0:
		return errors.ErrEmptyToken
	case p.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: DynV6]", p.domain, p.host)
}

func (p *provider) Domain() string {
	return p.domain
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
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://dynv6.com/\">DynV6 DNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	isIPv4 := ip.To4() != nil
	host := "dynv6.com"
	if isIPv4 {
		host = "ipv4." + host
	} else {
		host = "ipv6." + host
	}
	u := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/api/update",
	}
	values := url.Values{}
	values.Set("token", p.token)
	values.Set("zone", p.BuildDomainName())
	if !p.useProviderIP {
		if isIPv4 {
			values.Set("ipv4", ip.String())
		} else {
			values.Set("ipv6", ip.String())
		}
	}
	u.RawQuery = values.Encode()

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

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}
	return nil, fmt.Errorf("%w: %d: %s",
		errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
}
