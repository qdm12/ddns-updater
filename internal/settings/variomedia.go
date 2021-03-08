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
)

type variomedia struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func NewVariomedia(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
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
		username:      extraSettings.Username,
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
	case len(d.username) == 0:
		return ErrEmptyUsername
	case len(d.password) == 0:
		return ErrEmptyPassword
	case d.host == "*":
		return ErrHostWildcard
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

func (d *variomedia) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *variomedia) Proxied() bool {
	return false
}

func (d *variomedia) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
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
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.username, d.password),
		Host:   "dyndns.variomedia.de",
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
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyDataToSingleLine(s))
	}

	switch {
	case strings.HasPrefix(s, notfqdn):
		return nil, ErrHostnameNotExists
	case strings.HasPrefix(s, "badrequest"):
		return nil, ErrBadRequest
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
}
