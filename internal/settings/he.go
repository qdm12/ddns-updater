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

type he struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	password      string
	useProviderIP bool
}

func NewHe(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	h := &he{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := h.isValid(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *he) isValid() error {
	if len(h.password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

func (h *he) String() string {
	return toString(h.domain, h.host, constants.HE, h.ipVersion)
}

func (h *he) Domain() string {
	return h.domain
}

func (h *he) Host() string {
	return h.host
}

func (h *he) DNSLookup() bool {
	return h.dnsLookup
}

func (h *he) IPVersion() models.IPVersion {
	return h.ipVersion
}

func (h *he) BuildDomainName() string {
	return buildDomainName(h.host, h.domain)
}

func (h *he) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", h.BuildDomainName(), h.BuildDomainName())),
		Host:      models.HTML(h.Host()),
		Provider:  "<a href=\"https://dns.he.net/\">he.net</a>",
		IPVersion: models.HTML(h.ipVersion),
	}
}

func (h *he) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	fqdn := h.BuildDomainName()
	u := url.URL{
		Scheme: "https",
		Host:   "dyn.dns.he.net",
		Path:   "/nic/update",
		User:   url.UserPassword(fqdn, h.password),
	}
	values := url.Values{}
	values.Set("hostname", fqdn)
	if !h.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentih.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	switch s {
	case "":
		return nil, fmt.Errorf(http.StatusText(status))
	case badauth:
		return nil, fmt.Errorf("invalid username password combination")
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		verifier := verification.NewVerifier()
		ipsV4 := verifier.SearchIPv4(s)
		ipsV6 := verifier.SearchIPv6(s)
		ips := append(ipsV4, ipsV6...)
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("IP address received %q is malformed", ips[0])
		}
		if !h.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("invalid response %q", s)
}
