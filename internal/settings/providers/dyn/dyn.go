package dyn

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
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type dyn struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (d *dyn, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d = &dyn{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dyn) isValid() error {
	switch {
	case len(d.username) == 0:
		return errors.ErrEmptyUsername
	case len(d.password) == 0:
		return errors.ErrEmptyPassword
	case d.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (d *dyn) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Dyn]", d.domain, d.host)
}

func (d *dyn) Domain() string {
	return d.domain
}

func (d *dyn) Host() string {
	return d.host
}

func (d *dyn) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *dyn) Proxied() bool {
	return false
}

func (d *dyn) BuildDomainName() string {
	return utils.BuildDomainName(d.host, d.domain)
}

func (d *dyn) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://dyn.com/\">Dyn DNS</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *dyn) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.username, d.password),
		Host:   "members.dyndns.org",
		Path:   "/v3/update",
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
