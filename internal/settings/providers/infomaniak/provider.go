package infomaniak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
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

func (p *Provider) isValid() error {
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	case p.host == "*":
		return fmt.Errorf("%w", errors.ErrHostWildcard)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Infomaniak, p.ipVersion)
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
		Provider:  "<a href=\"https://www.infomaniak.com/\">Infomaniak</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "infomaniak.com",
		Path:   "/nic/update",
		User:   url.UserPassword(p.username, p.password),
	}
	values := url.Values{}
	values.Set("hostname", utils.BuildURLQueryHostname(p.host, p.domain))
	if !p.useProviderIP {
		values.Set("myip", ip.String())
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

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	switch response.StatusCode {
	case http.StatusOK:
		switch {
		case strings.HasPrefix(s, "good "):
			newIP = net.ParseIP(s[5:])
			if newIP == nil {
				return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, s)
			} else if ip != nil && !ip.Equal(newIP) {
				return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP)
			}
			return newIP, nil
		case strings.HasPrefix(s, "nochg "):
			newIP = net.ParseIP(s[6:])
			if newIP == nil {
				return nil, fmt.Errorf("%w: in response %q", errors.ErrNoResultReceived, s)
			} else if ip != nil && !ip.Equal(newIP) {
				return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP)
			}
			return newIP, nil
		default:
			return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
		}
	case http.StatusBadRequest:
		switch s {
		case constants.Nohost:
			return nil, fmt.Errorf("%w", errors.ErrHostnameNotExists)
		case constants.Badauth:
			return nil, fmt.Errorf("%w", errors.ErrAuth)
		default:
			return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
		}
	default:
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	}
}
