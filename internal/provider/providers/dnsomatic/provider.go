package dnsomatic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/ipextract"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var regexUsername = regexp.MustCompile(`^[a-zA-Z0-9@._-]{3,25}$`)

func (p *Provider) isValid() error {
	switch {
	case !regexUsername.MatchString(p.username):
		return fmt.Errorf("%w: %s", errors.ErrMalformedUsername, p.username)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.DNSOMatic, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.dnsomatic.com/\">dnsomatic</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Multiple hosts can be updated in one query, see https://www.dnsomatic.com/docs/api
	u := url.URL{
		Scheme: "https",
		Host:   "updates.dnsomatic.com",
		Path:   "/nic/update",
		User:   url.UserPassword(p.username, p.password),
	}
	values := url.Values{}
	if !p.useProviderIP {
		values.Set("myip", ip.String())
	}
	values.Set("wildcard", "NOCHG")
	if p.host == "*" {
		values.Set("hostname", p.domain)
		values.Set("wildcard", "ON")
	} else {
		values.Set("hostname", utils.BuildURLQueryHostname(p.host, p.domain))
	}
	values.Set("mx", "NOCHG")
	values.Set("backmx", "NOCHG")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, s)
	}

	switch s {
	case constants.Nohost, constants.Notfqdn:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrHostnameNotExists)
	case constants.Badauth:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAuth)
	case constants.Badagent:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrBannedUserAgent)
	case constants.Abuse:
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrAbuse)
	case "dnserr", constants.Nineoneone:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrDNSServerSide, s)
	}

	if !strings.Contains(s, "nochg") && !strings.Contains(s, "good") {
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}

	var ips []netip.Addr
	if ip.Is4() {
		ips = ipextract.IPv4(s)
	} else {
		ips = ipextract.IPv6(s)
	}

	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrNoIPInResponse)
	}

	newIP = ips[0]
	if !p.useProviderIP && ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
