package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

//nolint:maligned
type duckdns struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	token         string
	useProviderIP bool
}

func NewDuckdns(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &duckdns{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *duckdns) isValid() error {
	switch {
	case !constants.MatchDuckDNSToken(d.token):
		return fmt.Errorf("invalid token format")
	case d.host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func (d *duckdns) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Duckdns]", d.domain, d.host)
}

func (d *duckdns) Domain() string {
	return d.domain
}

func (d *duckdns) Host() string {
	return d.host
}

func (d *duckdns) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *duckdns) DNSLookup() bool {
	return d.dnsLookup
}

func (d *duckdns) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *duckdns) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://duckdns.org\">DuckDNS</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *duckdns) Update(client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.duckdns.org",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("verbose", "true")
	values.Set("domains", d.domain)
	values.Set("token", d.token)
	u.RawQuery = values.Encode()
	if !d.useProviderIP {
		values.Set("ip", ip.String())
	}
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	s := string(content)
	switch {
	case len(s) < 2:
		return nil, fmt.Errorf("response %q is too short", s)
	case s[0:2] == "KO":
		return nil, fmt.Errorf("invalid domain token combination")
	case s[0:2] == "OK":
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if ip != nil && !newIP.Equal(ip) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	default:
		return nil, fmt.Errorf("invalid response %q", s)
	}
}
