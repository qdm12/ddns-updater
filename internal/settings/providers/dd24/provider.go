package dd24

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
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
	password      string
	useProviderIP bool
	logger        log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (
	p *provider, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if len(p.password) == 0 {
		return errors.ErrEmptyPassword
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Dd24, p.ipVersion)
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
		Provider:  "<a href=\"https://www.domaindiscount24.com/\">DD24</a>",
		IPVersion: models.HTML(p.ipVersion),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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
		values.Set("hostname", "auto")
	} else {
		values.Set("hostname", ip.String())
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

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(s))
	}

	s = strings.ToLower(s)

	switch {
	case strings.Contains(s, "authorization failed"):
		return nil, errors.ErrAuth
	case s == "":
		return ip, nil
	// TODO missing cases
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
