package google

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
	"github.com/qdm12/golibs/verification"
)

type provider struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case len(p.username) == 0:
		return errors.ErrEmptyUsername
	case len(p.password) == 0:
		return errors.ErrEmptyPassword
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Google, p.ipVersion)
}

func (p *provider) Domain() string {
	return p.domain
}

func (p *provider) Host() string {
	return p.host
}

func (p *provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *provider) Proxied() bool {
	return false
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://domains.google.com/m/registrar\">Google</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "domains.google.com",
		Path:   "/nic/update",
		User:   url.UserPassword(p.username, p.password),
	}
	values := url.Values{}
	fqdn := utils.BuildURLQueryHostname(p.host, p.domain)
	values.Set("hostname", fqdn)
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
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	switch s {
	case "":
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	case constants.Nohost, constants.Notfqdn:
		return nil, errors.ErrHostnameNotExists
	case constants.Badauth:
		return nil, errors.ErrAuth
	case constants.Badagent:
		return nil, errors.ErrBannedUserAgent
	case constants.Abuse:
		return nil, errors.ErrAbuse
	case constants.Nineoneone:
		return nil, errors.ErrDNSServerSide
	case "conflict constants.A", "conflict constants.AAAA":
		return nil, errors.ErrConflictingRecord
	}

	if !strings.Contains(s, "nochg") && !strings.Contains(s, "good") {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}

	var ips []string
	verifier := verification.NewVerifier()
	if ip.To4() != nil {
		ips = verifier.SearchIPv4(s)
	} else {
		ips = verifier.SearchIPv6(s)
	}

	if len(ips) == 0 {
		return nil, errors.ErrNoIPInResponse
	}

	newIP = net.ParseIP(ips[0])
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ips[0])
	} else if !p.useProviderIP && !ip.Equal(newIP) {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}
	return newIP, nil
}
