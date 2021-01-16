package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/golibs/network"
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
	d := &opendns{
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

func (d *opendns) isValid() error {
	switch {
	case len(d.username) == 0:
		return ErrEmptyUsername
	case len(d.password) == 0:
		return ErrEmptyPassword
	case d.host == "*":
		return ErrHostWildcard
	}
	return nil
}

func (d *opendns) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Opendns]", d.domain, d.host)
}

func (d *opendns) Domain() string {
	return d.domain
}

func (d *opendns) Host() string {
	return d.host
}

func (d *opendns) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *opendns) DNSLookup() bool {
	return d.dnsLookup
}

func (d *opendns) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *opendns) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://opendns.com/\">Opendns DNS</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *opendns) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(d.username, d.password),
		Host:   "updates.opendns.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	switch d.host {
	case "@":
		values.Set("hostname", d.domain)
	default:
		values.Set("hostname", fmt.Sprintf("%s.%s", d.host, d.domain))
	}
	if !d.useProviderIP {
		values.Set("myip", ip.String())
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
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, status, string(content))
	}
	s := string(content)
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
