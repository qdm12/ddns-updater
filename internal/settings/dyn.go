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

type dyn struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewDyn(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &dyn{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
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
		return fmt.Errorf("username cannot be empty")
	case len(d.password) == 0:
		return fmt.Errorf("password cannot be empty")
	case d.host == "*":
		return fmt.Errorf(`host cannot be "*"`)
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

func (d *dyn) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *dyn) DNSLookup() bool {
	return d.dnsLookup
}

func (d *dyn) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dyn) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://dyn.com/\">Dyn DNS</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *dyn) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.username, d.password),
		Host:   "members.dyndns.org",
		Path:   "/v3/update",
	}
	values := url.Values{}
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
		return nil, fmt.Errorf(http.StatusText(status))
	}
	s := string(content)
	switch {
	case strings.HasPrefix(s, notfqdn):
		return nil, fmt.Errorf("fully qualified domain name is not valid")
	case strings.HasPrefix(s, "badrequest"):
		return nil, fmt.Errorf("bad request")
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return nil, fmt.Errorf("unknown response: %s", s)
	}
}
