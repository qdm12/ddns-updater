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

//nolint:maligned
type selfhostde struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	username      string
	password      string
	useProviderIP bool
}

func NewSelfhostde(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	sd := &selfhostde{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := sd.isValid(); err != nil {
		return nil, err
	}
	return sd, nil
}

func (sd *selfhostde) isValid() error {
	switch {
	case len(sd.username) == 0:
		return fmt.Errorf("username cannot be empty")
	case len(sd.password) == 0:
		return fmt.Errorf("password cannot be empty")
	case sd.host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func (sd *selfhostde) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Selfhost.de]", sd.domain, sd.host)
}

func (sd *selfhostde) Domain() string {
	return sd.domain
}

func (sd *selfhostde) Host() string {
	return sd.host
}

func (sd *selfhostde) IPVersion() models.IPVersion {
	return sd.ipVersion
}

func (sd *selfhostde) DNSLookup() bool {
	return sd.dnsLookup
}

func (sd *selfhostde) BuildDomainName() string {
	return buildDomainName(sd.host, sd.domain)
}

func (sd *selfhostde) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", sd.BuildDomainName(), sd.BuildDomainName())),
		Host:      models.HTML(sd.Host()),
		Provider:  "<a href=\"https://selfhost.de/\">Selfhost.de</a>",
		IPVersion: models.HTML(sd.ipVersion),
	}
}

func (sd *selfhostde) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(sd.username, sd.password),
		Host:   "carol.selfhost.de",
		Path:   "/nic/update",
	}
	values := url.Values{}
	switch sd.host {
	case "@":
		values.Set("hostname", sd.domain)
	default:
		values.Set("hostname", fmt.Sprintf("%sd.%s", sd.host, sd.domain))
	}
	if !sd.useProviderIP {
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
	// see their PDF file
	switch status {
	case http.StatusOK: // DynDNS v2 specification
	case http.StatusNoContent: // no change
		return ip, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("bad credentials (%s)", http.StatusText(status))
	case http.StatusConflict:
		return nil, fmt.Errorf("no zone found")
	case http.StatusGone:
		return nil, fmt.Errorf("account inactive")
	case http.StatusLengthRequired:
		return nil, fmt.Errorf("incorrect IP %s", ip)
	case http.StatusPreconditionFailed:
		return nil, fmt.Errorf("private IPs cannot be routed: %s", ip)
	case http.StatusServiceUnavailable:
		return nil, fmt.Errorf("server overloaded")
	default:
		return nil, fmt.Errorf(http.StatusText(status))
	}
	s := string(content)
	switch {
	case strings.HasPrefix(s, notfqdn):
		return nil, fmt.Errorf("fully qualified domain name is not valid")
	case strings.HasPrefix(s, "abuse"), strings.HasPrefix(s, "badrequest"):
		return nil, fmt.Errorf("bad request sent")
	case strings.HasPrefix(s, "good"), strings.HasPrefix(s, "nochg"):
		return ip, nil
	default:
		return nil, fmt.Errorf("unknown response: %s", s)
	}
}
