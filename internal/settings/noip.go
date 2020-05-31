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
type noip struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewNoip(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	n := &noip{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := n.isValid(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *noip) isValid() error {
	switch {
	case len(n.username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(n.username) > 50:
		return fmt.Errorf("username cannot be longer than 50 characters")
	case len(n.password) == 0:
		return fmt.Errorf("password cannot be empty")
	case n.host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func (n *noip) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Noip]", n.domain, n.host)
}

func (n *noip) Domain() string {
	return n.domain
}

func (n *noip) Host() string {
	return n.host
}

func (n *noip) DNSLookup() bool {
	return n.dnsLookup
}

func (n *noip) IPVersion() models.IPVersion {
	return n.ipVersion
}

func (n *noip) BuildDomainName() string {
	return buildDomainName(n.host, n.domain)
}

func (n *noip) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", n.BuildDomainName(), n.BuildDomainName())),
		Host:      models.HTML(n.Host()),
		Provider:  "<a href=\"https://www.noip.com/\">NoIP</a>",
		IPVersion: models.HTML(n.ipVersion),
	}
}

func (n *noip) Update(client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dynupdate.no-ip.com",
		Path:   "/nic/update",
		User:   url.UserPassword(n.username, n.password),
	}
	values := url.Values{}
	values.Set("hostname", n.BuildDomainName())
	if !n.useProviderIP {
		if ip.To4() == nil {
			values.Set("myipv6", ip.String())
		} else {
			values.Set("myip", ip.String())
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
	switch s {
	case "":
		return nil, fmt.Errorf("HTTP status %d", status)
	case "911":
		return nil, fmt.Errorf("NoIP's internal server error 911")
	case "abuse":
		return nil, fmt.Errorf("username is banned due to abuse")
	case "!donator":
		return nil, fmt.Errorf("user has not this extra feature")
	case "badagent":
		return nil, fmt.Errorf("user agent is banned")
	case badauth:
		return nil, fmt.Errorf("invalid username password combination")
	case nohost:
		return nil, fmt.Errorf("hostname does not exist")
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if !n.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("invalid response %q", s)
}
