package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/golibs/network"
)

//nolint:maligned
type ddnss struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewDdnss(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
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

func (d *ddnss) isValid() error {
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

func (d *ddnss) String() string {
	return toString(d.domain, d.host, constants.DDNSSDE, d.ipVersion)
}

func (d *ddnss) Domain() string {
	return d.domain
}

func (d *ddnss) Host() string {
	return d.host
}

func (d *ddnss) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *ddnss) DNSLookup() bool {
	return d.dnsLookup
}

func (d *ddnss) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *ddnss) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://ddnss.de/\">DDNSS.de</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *ddnss) Update(client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.ddnss.de",
		Path:   "/upd.php",
	}
	values := url.Values{}
	values.Set("user", d.username)
	values.Set("pwd", d.password)
	fqdn := d.domain
	if d.host != "@" {
		fqdn = d.host + "." + d.domain
	}
	values.Set("host", fqdn)
	if !d.useProviderIP {
		if ip.To4() == nil { // ipv6
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	if status != http.StatusOK {
		return nil, fmt.Errorf("received status %d with message: %s", status, s)
	}
	switch {
	case strings.Contains(s, "badysys"):
		return nil, fmt.Errorf("ddnss.de: invalid system parameter")
	case strings.Contains(s, badauth):
		return nil, fmt.Errorf("ddnss.de: bad authentication")
	case strings.Contains(s, "notfqdn"):
		return nil, fmt.Errorf("ddnss.de: hostname %q does not exist", fqdn)
	case strings.Contains(s, "Updated 1 hostname"):
		return ip, nil
	default:
		return nil, fmt.Errorf("unknown response received from ddnss.de: %s", s)
	}
}
