package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	netlib "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

//nolint:maligned
type google struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewGoogle(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	g := &google{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *google) isValid() error {
	switch {
	case len(g.username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(g.password) == 0:
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

func (g *google) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Google]", g.domain, g.host)
}

func (g *google) Domain() string {
	return g.domain
}

func (g *google) Host() string {
	return g.host
}

func (g *google) DNSLookup() bool {
	return g.dnsLookup
}

func (g *google) IPVersion() models.IPVersion {
	return g.ipVersion
}

func (g *google) BuildDomainName() string {
	return buildDomainName(g.host, g.domain)
}

func (g *google) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://domains.google.com/m/registrar\">Google</a>",
		IPVersion: models.HTML(g.ipVersion),
	}
}

func (g *google) Update(client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "domains.google.com",
		Path:   "/nic/update",
		User:   url.UserPassword(g.username, g.password),
	}
	values := url.Values{}
	fqdn := g.BuildDomainName()
	values.Set("hostname", fqdn)
	if !g.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentig.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	switch s {
	case "":
		return nil, fmt.Errorf("HTTP status %d", status)
	case nohost:
		return nil, fmt.Errorf("hostname does not exist")
	case badauth:
		return nil, fmt.Errorf("invalid username password combination")
	case "notfqdn":
		return nil, fmt.Errorf("hostname %q is not a valid fully qualified domain name", fqdn)
	case "badagent":
		return nil, fmt.Errorf("user agent is banned")
	case "abuse":
		return nil, fmt.Errorf("username is banned due to abuse")
	case "911":
		return nil, fmt.Errorf("Google's internal server error 911")
	case "conflict A":
		return nil, fmt.Errorf("custom A record conflicts with the update")
	case "conflict AAAA":
		return nil, fmt.Errorf("custom AAAA record conflicts with the update")
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ipsV4 := verification.NewVerifier().SearchIPv4(s)
		ipsV6 := verification.NewVerifier().SearchIPv6(s)
		ips := append(ipsV4, ipsV6...)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if !g.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("invalid response %q", s)
}
