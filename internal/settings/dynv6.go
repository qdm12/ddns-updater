package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/golibs/network"
)

//nolint:maligned
type dynV6 struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	dnsLookup     bool
	token         string
	useProviderIP bool
}

func NewDynV6(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token         string `json:"string"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &dynV6{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		dnsLookup:     !noDNSLookup,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dynV6) isValid() error {
	switch {
	case len(d.token) == 0:
		return fmt.Errorf("token cannot be empty")
	case d.host == "*":
		return fmt.Errorf(`host cannot be "*"`)
	}
	return nil
}

func (d *dynV6) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: DynV6]", d.domain, d.host)
}

func (d *dynV6) Domain() string {
	return d.domain
}

func (d *dynV6) Host() string {
	return d.host
}

func (d *dynV6) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *dynV6) DNSLookup() bool {
	return d.dnsLookup
}

func (d *dynV6) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dynV6) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://dynv6.com/\">DynV6 DNS</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *dynV6) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	isIPv4 := ip.To4() != nil
	host := "dynv6.com"
	if isIPv4 {
		host = "ipv4." + host
	} else {
		host = "ipv6." + host
	}
	u := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/api/update",
	}
	values := url.Values{}
	values.Set("token", d.token)
	switch d.host {
	case "@":
		values.Set("zone", d.domain)
	default:
		values.Set("zone", fmt.Sprintf("%s.%s", d.host, d.domain))
	}
	if !d.useProviderIP {
		if isIPv4 {
			values.Set("ipv4", ip.String())
		} else {
			values.Set("ipv6", ip.String())
		}
	}
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	_, status, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(status))
	}
	return ip, nil
}
