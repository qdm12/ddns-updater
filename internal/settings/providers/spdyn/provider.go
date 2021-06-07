package spdyn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

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
	user          string
	password      string
	token         string
	useProviderIP bool
	logger        log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		User          string `json:"user"`
		Password      string `json:"password"`
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
		user:          extraSettings.User,
		password:      extraSettings.Password,
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
	if len(p.token) > 0 {
		return nil
	}
	switch {
	case len(p.user) == 0:
		return errors.ErrEmptyUsername
	case len(p.password) == 0:
		return errors.ErrEmptyPassword
	case p.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Spdyn]", p.domain, p.host)
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
		Provider:  "<a href=\"https://spdyn.com/\">Spdyn DNS</a>",
		IPVersion: models.HTML(p.ipVersion),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	// see https://wiki.securepoint.de/SPDyn/Variablen
	u := url.URL{
		Scheme: "https",
		Host:   "update.spdyn.de",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", p.BuildDomainName())
	if p.useProviderIP {
		values.Set("myip", "10.0.0.1")
	} else {
		values.Set("myip", ip.String())
	}
	if len(p.token) > 0 {
		values.Set("user", p.BuildDomainName())
		values.Set("pass", p.token)
	} else {
		values.Set("user", p.user)
		values.Set("pass", p.password)
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
	bodyString := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(bodyString))
	}

	switch bodyString {
	case constants.Abuse, "numhost":
		return nil, errors.ErrAbuse
	case constants.Badauth, "!yours":
		return nil, errors.ErrAuth
	case "good":
		return ip, nil
	case constants.Notfqdn:
		return nil, fmt.Errorf("%w: not fqdn", errors.ErrBadRequest)
	case "nochg":
		return ip, nil
	case "nohost", "fatal":
		return nil, errors.ErrHostnameNotExists
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, bodyString)
	}
}
