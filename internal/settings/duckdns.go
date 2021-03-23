package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type duckdns struct {
	host          string
	ipVersion     ipversion.IPVersion
	token         string
	useProviderIP bool
	matcher       regex.Matcher
}

func NewDuckdns(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &duckdns{
		host:          host,
		ipVersion:     ipVersion,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
		matcher:       matcher,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *duckdns) isValid() error {
	if !d.matcher.DuckDNSToken(d.token) {
		return ErrMalformedToken
	}
	switch d.host {
	case "@", "*":
		return ErrHostOnlySubdomain
	}
	return nil
}

func (d *duckdns) String() string {
	return toString("duckdns.org", d.host, constants.DUCKDNS, d.ipVersion)
}

func (d *duckdns) Domain() string {
	return "duckdns.org"
}

func (d *duckdns) Host() string {
	return d.host
}

func (d *duckdns) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *duckdns) Proxied() bool {
	return false
}

func (d *duckdns) BuildDomainName() string {
	return buildDomainName(d.host, "duckdns.org")
}

func (d *duckdns) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://duckdns.org\">DuckDNS</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *duckdns) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "www.duckdns.org",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("verbose", "true")
	values.Set("domains", d.host)
	values.Set("token", d.token)
	u.RawQuery = values.Encode()
	if !d.useProviderIP {
		if ip.To4() == nil {
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}

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
			ErrBadHTTPStatus, response.StatusCode, bodyDataToSingleLine(s))
	}

	const minChars = 2
	switch {
	case len(s) < minChars:
		return nil, fmt.Errorf("%w: response %q is too short", ErrUnmarshalResponse, s)
	case s[0:minChars] == "KO":
		return nil, ErrAuth
	case s[0:minChars] == "OK":
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, ErrNoResultReceived
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, ips[0])
		}
		if ip != nil && !newIP.Equal(ip) {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
}
