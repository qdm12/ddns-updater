package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

type dnsomatic struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
	matcher       regex.Matcher
}

func NewDNSOMatic(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &dnsomatic{
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

func (d *dnsomatic) isValid() error {
	switch {
	case !d.matcher.DNSOMaticUsername(d.username):
		return fmt.Errorf("username %q does not match DNS-O-Matic username format", d.username)
	case !d.matcher.DNSOMaticPassword(d.password):
		return fmt.Errorf("password does not match DNS-O-Matic password format")
	case len(d.username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(d.password) == 0:
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

func (d *dnsomatic) String() string {
	return toString(d.domain, d.host, constants.DNSOMATIC, d.ipVersion)
}

func (d *dnsomatic) Domain() string {
	return d.domain
}

func (d *dnsomatic) Host() string {
	return d.host
}

func (d *dnsomatic) DNSLookup() bool {
	return d.dnsLookup
}

func (d *dnsomatic) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *dnsomatic) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dnsomatic) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dnsomatic.com/\">dnsomatic</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *dnsomatic) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	// Multiple hosts can be updated in one query, see https://www.dnsomatic.com/docs/api
	u := url.URL{
		Scheme: "https",
		Host:   "updates.dnsomatic.com",
		Path:   "/nic/update",
		User:   url.UserPassword(d.username, d.password),
	}
	values := url.Values{}
	fqdn := d.BuildDomainName()
	values.Set("hostname", fqdn)
	if !d.useProviderIP {
		values.Set("myip", ip.String())
	}
	values.Set("wildcard", "NOCHG")
	if d.host == "*" {
		values.Set("wildcard", "ON")
	}
	values.Set("mx", "NOCHG")
	values.Set("backmx", "NOCHG")
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentid.mcgaw@gmail.com")
	r = r.WithContext(ctx)
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(status))
	}
	s := string(content)
	switch s {
	case nohost:
		return nil, fmt.Errorf("hostname does not exist")
	case badauth:
		return nil, fmt.Errorf("invalid username password combination")
	case notfqdn:
		return nil, fmt.Errorf("hostname %q is not a valid fully qualified domain name", fqdn)
	case badagent:
		return nil, fmt.Errorf("user agent is banned")
	case abuse:
		return nil, fmt.Errorf("username is banned due to abuse")
	case "dnserr":
		return nil, fmt.Errorf("DNS error encountered, please contact DNS-O-Matic")
	case nineoneone:
		return nil, fmt.Errorf("dnsomatic's internal server error 911")
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
		if !d.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("invalid response %q", s)
}
