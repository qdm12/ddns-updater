package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type variomedia struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	email         string
	password      string
	useProviderIP bool
}

func NewVariomedia(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &variomedia{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		email:         extraSettings.Email,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *variomedia) isValid() error {
	switch {
	case len(d.email) == 0:
		return errors.ErrEmptyEmail
	case len(d.password) == 0:
		return errors.ErrEmptyPassword
	case d.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (d *variomedia) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Variomedia]", d.domain, d.host)
}

func (d *variomedia) Domain() string {
	return d.domain
}

func (d *variomedia) Host() string {
	return d.host
}

func (d *variomedia) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *variomedia) Proxied() bool {
	return false
}

func (d *variomedia) BuildDomainName() string {
	return utils.BuildDomainName(d.host, d.domain)
}

func (d *variomedia) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://variomedia.de/\">Variomedia</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *variomedia) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	host := "dyndns.variomedia.de"
	if d.useProviderIP {
		if ip.To4() == nil {
			host = "dyndns6.variomedia.de"
		} else {
			host = "dyndns4.variomedia.de"
		}
	}

	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.email, d.password),
		Host:   host,
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", d.BuildDomainName())
	if !d.useProviderIP {
		values.Set("myip", ip.String())
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

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(s))
	}

	switch {
	case strings.HasPrefix(s, constants.Notfqdn):
		return nil, errors.ErrHostnameNotExists
	case strings.HasPrefix(s, "badrequest"):
		return nil, errors.ErrBadRequest
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
