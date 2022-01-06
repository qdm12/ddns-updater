package dynu

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
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain        string
	host          string
	group         string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
		Group         string `json:"group"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		group:         extraSettings.Group,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case p.username == "":
		return errors.ErrEmptyUsername
	case p.password == "":
		return errors.ErrEmptyPassword
	case p.host == "":
		return errors.ErrEmptyHost
	case p.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Dynu, p.ipVersion)
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
		Provider:  "<a href=\"https://dynu.com/\">Dynu</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dynu.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("username", p.username)
	values.Set("password", p.password)
	values.Set("location", p.group)
	if p.host != "@" {
		values.Set("alias", p.host)
		values.Set("hostname", p.domain)
	} else {
		values.Set("hostname", utils.BuildURLQueryHostname(p.host, p.domain))
	}
	if !p.useProviderIP {
		if ip.To4() == nil {
			values.Set("myipv6", ip.String())
		} else {
			values.Set("myip", ip.String())
		}
	}
	u.RawQuery = values.Encode()

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

	switch {
	case strings.Contains(s, constants.Badauth):
		return nil, errors.ErrAuth
	case strings.Contains(s, constants.Notfqdn):
		return nil, errors.ErrHostnameNotExists
	case strings.Contains(s, constants.Abuse):
		return nil, errors.ErrAbuse
	case strings.Contains(s, "good"):
		return ip, nil
	case strings.Contains(s, "nochg"): // Updated but not changed
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}
}
