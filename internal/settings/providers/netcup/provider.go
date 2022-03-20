package netcup

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
	customerNumber int
	domain         string
	host           string
	ipVersion      ipversion.IPVersion
	apiKey         string
	password       string
	useProviderIP  bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		CustomerNumber int    `json:"customer_number"`
		ApiKey         string `json:"api_key"`
		Password       string `json:"password"`
		UseProviderIP  bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		customerNumber: extraSettings.CustomerNumber,
		apiKey:         extraSettings.ApiKey,
		password:       extraSettings.Password,
		useProviderIP:  extraSettings.UseProviderIP,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case p.customerNumber == 0:
		return errors.ErrEmptyCustomerNumber
	case p.apiKey == "":
		return errors.ErrEmptyAppKey
	case p.password == "":
		return errors.ErrEmptyPassword
	case p.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Netcup, p.ipVersion)
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
		Provider:  "<a href=\"https://www.netcup.eu/\">Netcup.eu</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "ccp.netcup.net",
		Path:     "/run/webservice/servers/endpoint.php",
		RawQuery: "JSON",
		//User:   url.UserPassword(p.username, p.password),
	}
	// https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON
	nc := NewClient(p.customerNumber, p.apiKey, p.password, u.String())

	err = nc.Login(ctx)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrBadRequest, err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, utils.ToSingleLine(s))
	}

	switch s {
	case "":
		return nil, errors.ErrNoResultReceived
	case constants.Nineoneone:
		return nil, errors.ErrDNSServerSide
	case constants.Abuse:
		return nil, errors.ErrAbuse
	case "!donator":
		return nil, errors.ErrFeatureUnavailable
	case constants.Badagent:
		return nil, errors.ErrBannedUserAgent
	case constants.Badauth:
		return nil, errors.ErrAuth
	case constants.Nohost:
		return nil, errors.ErrHostnameNotExists
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
	}
	if !p.useProviderIP && !ip.Equal(newIP) {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}
	return newIP, nil
}
