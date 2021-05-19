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
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type namecheap struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	password      string
	useProviderIP bool
	matcher       regex.Matcher `json:"-"`
}

func NewNamecheap(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (s Settings, err error) {
	if ipVersion == ipversion.IP6 {
		return s, ErrIPv6NotSupported
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
		return ErrMalformedPassword
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

func (n *namecheap) IPVersion() ipversion.IPVersion {
	return n.ipVersion
}

func (n *namecheap) Proxied() bool {
	return false
}

func (n *namecheap) BuildDomainName() string {
	return buildDomainName(n.host, n.domain)
}

func (n *namecheap) MarshalJSON() (b []byte, err error) {
	return json.Marshal(n)
}

func (n *namecheap) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", n.BuildDomainName(), n.BuildDomainName())),
		Host:      models.HTML(n.Host()),
		Provider:  "<a href=\"https://namecheap.com\">Namecheap</a>",
		IPVersion: models.HTML(n.ipVersion.String()),
	}
}

func (n *namecheap) setHeaders(request *http.Request) {
	setUserAgent(request)
	setAccept(request, "application/xml")
}

func (n *namecheap) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	n.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	decoder := xml.NewDecoder(response.Body)
	var parsedXML struct {
		Errors struct {
			Error string `xml:"Err1"`
		} `xml:"errors"`
		IP string `xml:"IP"`
	}
	if err := decoder.Decode(&parsedXML); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	if parsedXML.Errors.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrUnsuccessfulResponse, parsedXML.Errors.Error)
	}
	newIP = net.ParseIP(parsedXML.IP)
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, parsedXML.IP)
	}
	if ip != nil && !ip.Equal(newIP) {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, newIP.String())
	}
	return newIP, nil
}
