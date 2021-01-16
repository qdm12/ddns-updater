package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/golibs/network"
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

func (d *ovh) isValid() error {
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

func (d *ovh) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: OVH]", d.domain, d.host)
}

func (d *ovh) Domain() string {
	return d.domain
}

func (d *ovh) Host() string {
	return d.host
}

func (d *ovh) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *ovh) DNSLookup() bool {
	return d.dnsLookup
}

func (d *ovh) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *ovh) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.ovh.com/\">OVH DNS</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *ovh) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.username, d.password),
		Host:   "www.ovh.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("system", "dyndns")
	switch d.host {
	case "@":
		values.Set("hostname", d.domain)
	default:
		values.Set("hostname", fmt.Sprintf("%s.%s", d.host, d.domain))
	}
	if !d.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	}
	s := string(content)
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
