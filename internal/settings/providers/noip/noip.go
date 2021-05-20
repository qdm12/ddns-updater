package noip

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type noip struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (n *noip, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	n = &noip{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := n.isValid(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *noip) isValid() error {
	const maxUsernameLength = 50
	switch {
	case len(n.username) == 0:
		return errors.ErrEmptyUsername
	case len(n.username) > maxUsernameLength:
		return fmt.Errorf("%w: longer than 50 characters", errors.ErrMalformedUsername)
	case len(n.password) == 0:
		return errors.ErrEmptyPassword
	case n.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (n *noip) String() string {
	return utils.ToString(n.domain, n.host, constants.NoIP, n.ipVersion)
}

func (n *noip) Domain() string {
	return n.domain
}

func (n *noip) Host() string {
	return n.host
}

func (n *noip) IPVersion() ipversion.IPVersion {
	return n.ipVersion
}

func (n *noip) Proxied() bool {
	return false
}

func (n *noip) BuildDomainName() string {
	return utils.BuildDomainName(n.host, n.domain)
}

func (n *noip) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", n.BuildDomainName(), n.BuildDomainName())),
		Host:      models.HTML(n.Host()),
		Provider:  "<a href=\"https://www.noip.com/\">NoIP</a>",
		IPVersion: models.HTML(n.ipVersion.String()),
	}
}

func (n *noip) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dynupdate.no-ip.com",
		Path:   "/nic/update",
		User:   url.UserPassword(n.username, n.password),
	}
	values := url.Values{}
	values.Set("hostname", n.BuildDomainName())
	if !n.useProviderIP {
		if ip.To4() == nil {
			values.Set("myipv6", ip.String())
		} else {
			values.Set("myip", ip.String())
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

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
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
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		ips := verification.NewVerifier().SearchIPv4(s)
		if ips == nil {
			return nil, errors.ErrNoResultReceived
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ips[0])
		}
		if !n.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	}
	return nil, errors.ErrUnknownResponse
}
