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

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type google struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func NewGoogle(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
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
		return ErrEmptyUsername
	case len(g.password) == 0:
		return ErrEmptyPassword
	}
	return nil
}

func (g *google) String() string {
	return toString(g.domain, g.host, constants.GOOGLE, g.ipVersion)
}

func (g *google) Domain() string {
	return g.domain
}

func (g *google) Host() string {
	return g.host
}

func (g *google) IPVersion() ipversion.IPVersion {
	return g.ipVersion
}

func (g *google) Proxied() bool {
	return false
}

func (g *google) BuildDomainName() string {
	return buildDomainName(g.host, g.domain)
}

func (g *google) MarshalJSON() (b []byte, err error) {
	return json.Marshal(g)
}

func (g *google) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://domains.google.com/m/registrar\">Google</a>",
		IPVersion: models.HTML(g.ipVersion.String()),
	}
}

func (g *google) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	setUserAgent(request)

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

	switch s {
	case "":
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, s)
	case nohost, notfqdn:
		return nil, ErrHostnameNotExists
	case badauth:
		return nil, ErrAuth
	case badagent:
		return nil, ErrBannedUserAgent
	case abuse:
		return nil, ErrAbuse
	case nineoneone:
		return nil, ErrDNSServerSide
	case "conflict A", "conflict AAAA":
		return nil, ErrConflictingRecord
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ipsV4 := verification.NewVerifier().SearchIPv4(s)
		ipsV6 := verification.NewVerifier().SearchIPv6(s)
		ips := append(ipsV4, ipsV6...)
		if len(ips) == 0 {
			return nil, ErrNoResultReceived
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, ips[0])
		} else if !g.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
}
