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

type strato struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	password      string
	useProviderIP bool
}

func NewStrato(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	ss := &strato{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := ss.isValid(); err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *strato) isValid() error {
	switch {
	case len(s.password) == 0:
		return fmt.Errorf("password cannot be empty")
	case s.host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func (s *strato) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Strato]", s.domain, s.host)
}

func (s *strato) Domain() string {
	return s.domain
}

func (s *strato) Host() string {
	return s.host
}

func (s *strato) IPVersion() models.IPVersion {
	return s.ipVersion
}

func (s *strato) DNSLookup() bool {
	return s.dnsLookup
}

func (s *strato) BuildDomainName() string {
	return buildDomainName(s.host, s.domain)
}

func (s *strato) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", s.BuildDomainName(), s.BuildDomainName())),
		Host:      models.HTML(s.Host()),
		Provider:  "<a href=\"https://strato.com/\">Strato DNS</a>",
		IPVersion: models.HTML(s.ipVersion),
	}
}

func (s *strato) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(s.domain, s.password),
		Host:   "dyndns.strato.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	switch s.host {
	case "@":
		values.Set("hostname", s.domain)
	default:
		values.Set("hostname", fmt.Sprintf("%s.%s", s.host, s.domain))
	}
	if !s.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(status))
	}
	str := string(content)
	switch {
	case strings.HasPrefix(str, notfqdn):
		return nil, fmt.Errorf("fully qualified domain name is not valid")
	case strings.HasPrefix(str, abuse):
		return nil, ErrAbuse
	case strings.HasPrefix(str, "badrequest"):
		return nil, fmt.Errorf("bad request")
	case strings.HasPrefix(str, "badauth"):
		return nil, ErrAuth
	case strings.HasPrefix(str, "good"), strings.HasPrefix(str, "nochg"):
		return ip, nil
	default:
		return nil, fmt.Errorf("unknown response: %s", str)
	}
}
