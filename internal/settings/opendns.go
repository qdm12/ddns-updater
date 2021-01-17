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

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
)

type opendns struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewOpendns(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	o := &opendns{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := o.isValid(); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *opendns) isValid() error {
	switch {
	case len(o.username) == 0:
		return ErrEmptyUsername
	case len(o.password) == 0:
		return ErrEmptyPassword
	case o.host == "*":
		return ErrHostWildcard
	}
	return nil
}

func (o *opendns) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Opendns]", o.domain, o.host)
}

func (o *opendns) Domain() string {
	return o.domain
}

func (o *opendns) Host() string {
	return o.host
}

func (o *opendns) IPVersion() models.IPVersion {
	return o.ipVersion
}

func (o *opendns) DNSLookup() bool {
	return o.dnsLookup
}

func (o *opendns) BuildDomainName() string {
	return buildDomainName(o.host, o.domain)
}

func (o *opendns) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", o.BuildDomainName(), o.BuildDomainName())),
		Host:      models.HTML(o.Host()),
		Provider:  "<a href=\"https://opendns.com/\">Opendns DNS</a>",
		IPVersion: models.HTML(o.ipVersion),
	}
}

func (o *opendns) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(o.username, o.password),
		Host:   "updates.opendns.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", o.BuildDomainName())
	if !o.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

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
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, s)
	}

	if !strings.HasPrefix(s, "good ") {
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
	responseIPString := strings.TrimPrefix(s, "good ")
	responseIP := net.ParseIP(responseIPString)
	if responseIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, responseIPString)
	} else if !newIP.Equal(ip) {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, responseIP)
	}
	return ip, nil
}
