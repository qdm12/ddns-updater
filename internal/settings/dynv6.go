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
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type dynV6 struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	token         string
	useProviderIP bool
}

func NewDynV6(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &dynV6{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
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
		return errors.ErrEmptyToken
	case d.host == "*":
		return errors.ErrHostWildcard
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

func (d *dynV6) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *dynV6) Proxied() bool {
	return false
}

func (d *dynV6) BuildDomainName() string {
	return utils.BuildDomainName(d.host, d.domain)
}

func (d *dynV6) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://dynv6.com/\">DynV6 DNS</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *dynV6) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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
	values.Set("zone", d.BuildDomainName())
	if !d.useProviderIP {
		if isIPv4 {
			values.Set("ipv4", ip.String())
		} else {
			values.Set("ipv6", ip.String())
		}
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}
	return nil, fmt.Errorf("%w: %d: %s",
		errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
}
