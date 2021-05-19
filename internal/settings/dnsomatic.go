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

type dnsomatic struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
	matcher       regex.Matcher `json:"-"`
}

func NewDNSOMatic(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (s Settings, err error) {
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
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
		matcher:       matcher,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dnsomatic) isValid() error {
	switch {
	case !d.matcher.DNSOMaticUsername(d.username):
		return fmt.Errorf("%w: %s", ErrMalformedUsername, d.username)
	case !d.matcher.DNSOMaticPassword(d.password):
		return ErrMalformedPassword
	case len(d.username) == 0:
		return ErrEmptyUsername
	case len(d.password) == 0:
		return ErrEmptyPassword
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

func (d *dnsomatic) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *dnsomatic) Proxied() bool {
	return false
}

func (d *dnsomatic) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dnsomatic) MarshalJSON() (b []byte, err error) {
	return json.Marshal(d)
}

func (d *dnsomatic) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dnsomatic.com/\">dnsomatic</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *dnsomatic) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	// Multiple hosts can be updated in one query, see https://www.dnsomatic.com/docs/api
	u := url.URL{
		Scheme: "https",
		Host:   "updates.dnsomatic.com",
		Path:   "/nic/update",
		User:   url.UserPassword(d.username, d.password),
	}
	values := url.Values{}
	values.Set("hostname", d.BuildDomainName())
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

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, s)
	}

	switch s {
	case nohost, notfqdn:
		return nil, ErrHostnameNotExists
	case badauth:
		return nil, ErrAuth
	case badagent:
		return nil, ErrBannedUserAgent
	case abuse:
		return nil, ErrAbuse
	case "dnserr", nineoneone:
		return nil, fmt.Errorf("%w: %s", ErrDNSServerSide, s)
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ipsV4 := verification.NewVerifier().SearchIPv4(s)
		ipsV6 := verification.NewVerifier().SearchIPv6(s)
		ips := append(ipsV4, ipsV6...) //nolint:gocritic
		if ips == nil {
			return nil, fmt.Errorf("no IP address in response")
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, ips[0])
		}
		if !d.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
}
