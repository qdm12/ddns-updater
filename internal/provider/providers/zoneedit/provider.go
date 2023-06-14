package zoneedit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
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
	token         string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Token         string `json:"token"`
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
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.token == "":
		return fmt.Errorf("%w", errors.ErrEmptyToken)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Zoneedit, p.ipVersion)
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
		Provider:  "<a href=\"https://www.zoneedit.com/\">Zoneedit</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (
	newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.cp.zoneedit.com",
		Path:   "dyn/generic.php",
		User:   url.UserPassword(p.username, p.token),
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.host, p.domain))
	if !p.useProviderIP {
		values.Set("myip", ip.String())
	}
	if p.host == "*" {
		values.Set("wildcard", "ON")
	}
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
		return netip.Addr{}, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus,
			response.StatusCode, utils.ToSingleLine(s))
	}

	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, `success_code="200"`):
		ips := ipextract.IPv4(s)
		const expectedIPCount = 2
		if len(ips) != expectedIPCount {
			// can't really handle the ip check comparison, but the server
			// responded with a success code so do not return an error.
			return ip, nil
		}
		newIP = ips[1]
		if newIP.Compare(ip) != 0 {
			return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
				errors.ErrIPReceivedMismatch, ip, newIP)
		}
		return newIP, nil
	case s == "":
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrNoResultReceived)
	case strings.Contains(s, `error code="702"`),
		strings.Contains(s, "minimum 600 seconds between requests"):
		return netip.Addr{}, fmt.Errorf("%w: zoneedit requires 10 minutes between each request", errors.ErrAbuse)
	case strings.Contains(s, `error code="709"`),
		strings.Contains(s, "invalid hostname"):
		return netip.Addr{}, fmt.Errorf("%w: invalid request sent", errors.ErrAbuse)
	case strings.Contains(s, `error code="708"`),
		strings.Contains(s, "failed login"):
		return netip.Addr{}, fmt.Errorf("%w: for user %s", errors.ErrAuth, p.username)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, utils.ToSingleLine(s))
	}
}
