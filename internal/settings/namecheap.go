package settings

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/golibs/network"
)

//nolint:maligned
type namecheap struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	password      string
	useProviderIP bool
	matcher       regex.Matcher
}

func NewNamecheap(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	if ipVersion == constants.IPv6 {
		return s, fmt.Errorf("IPv6 is not supported by Namecheap API sadly")
	}
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	n := &namecheap{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
		matcher:       matcher,
	}
	if err := n.isValid(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *namecheap) isValid() error {
	if !n.matcher.NamecheapPassword(n.password) {
		return fmt.Errorf("invalid password format")
	}
	return nil
}

func (n *namecheap) String() string {
	return toString(n.domain, n.host, constants.NAMECHEAP, n.ipVersion)
}

func (n *namecheap) Domain() string {
	return n.domain
}

func (n *namecheap) Host() string {
	return n.host
}

func (n *namecheap) IPVersion() models.IPVersion {
	return n.ipVersion
}

func (n *namecheap) DNSLookup() bool {
	return n.dnsLookup
}

func (n *namecheap) BuildDomainName() string {
	return buildDomainName(n.host, n.domain)
}

func (n *namecheap) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", n.BuildDomainName(), n.BuildDomainName())),
		Host:      models.HTML(n.Host()),
		Provider:  "<a href=\"https://namecheap.com\">Namecheap</a>",
		IPVersion: models.HTML(n.ipVersion),
	}
}

func (n *namecheap) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dynamicdns.park-your-domain.com",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("host", n.host)
	values.Set("domain", n.domain)
	values.Set("password", n.password)
	if !n.useProviderIP {
		values.Set("ip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	r = r.WithContext(ctx)
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(status))
	}
	var parsedXML struct {
		Errors struct {
			Error string `xml:"Err1"`
		} `xml:"errors"`
		IP string `xml:"IP"`
	}
	err = xml.Unmarshal(content, &parsedXML)
	if err != nil {
		return nil, err
	} else if parsedXML.Errors.Error != "" {
		return nil, fmt.Errorf(parsedXML.Errors.Error)
	}
	newIP = net.ParseIP(parsedXML.IP)
	if newIP == nil {
		return nil, fmt.Errorf("IP address received %q is malformed", parsedXML.IP)
	}
	if ip != nil && !ip.Equal(newIP) {
		return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
	}
	return newIP, nil
}
