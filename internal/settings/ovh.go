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

type ovh struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewOVH(data json.RawMessage, _, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	o := &ovh{
		domain:        "mypersonaldomain.ovh",
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := o.isValid(); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *ovh) isValid() error {
	switch {
	case len(o.username) == 0:
		return ErrEmptyUsername
	case len(o.password) == 0:
		return ErrEmptyPassword
	case o.host == "*":
		return ErrHostWildcard
	}
	return nil
}

func (o *ovh) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: OVH]", o.domain, o.host)
}

func (o *ovh) Domain() string {
	return o.domain
}

func (o *ovh) Host() string {
	return o.host
}

func (o *ovh) IPVersion() models.IPVersion {
	return o.ipVersion
}

func (o *ovh) DNSLookup() bool {
	return o.dnsLookup
}

func (o *ovh) BuildDomainName() string {
	return buildDomainName(o.host, o.domain)
}

func (o *ovh) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", o.BuildDomainName(), o.BuildDomainName())),
		Host:      models.HTML(o.Host()),
		Provider:  "<a href=\"https://www.ovh.com/\">OVH DNS</a>",
		IPVersion: models.HTML(o.ipVersion),
	}
}

func (o *ovh) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(o.username, o.password),
		Host:   "www.ovh.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("system", "dyndns")
	values.Set("hostname", o.BuildDomainName())
	if !o.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

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
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, s)
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
