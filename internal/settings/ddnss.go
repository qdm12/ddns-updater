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
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type ddnss struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func NewDdnss(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &ddnss{
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

func (d *ddnss) isValid() error {
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

func (d *ddnss) String() string {
	return utils.ToString(d.domain, d.host, constants.DdnssDe, d.ipVersion)
}

func (d *ddnss) Domain() string {
	return d.domain
}

func (d *ddnss) Host() string {
	return d.host
}

func (d *ddnss) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *ddnss) Proxied() bool {
	return false
}

func (d *ddnss) BuildDomainName() string {
	return utils.BuildDomainName(d.host, d.domain)
}

func (d *ddnss) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://ddnss.de/\">DDNSS.de</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *ddnss) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.ddnss.de",
		Path:   "/upd.php",
	}
	values := url.Values{}
	values.Set("user", d.username)
	values.Set("pwd", d.password)
	values.Set("host", d.BuildDomainName())
	if !d.useProviderIP {
		if ip.To4() == nil { // ipv6
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
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
	case strings.Contains(s, "badysys"):
		return nil, errors.ErrInvalidSystemParam
	case strings.Contains(s, constants.Badauth):
		return nil, errors.ErrAuth
	case strings.Contains(s, constants.Notfqdn):
		return nil, errors.ErrHostnameNotExists
	case strings.Contains(s, "Updated 1 hostname"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}
