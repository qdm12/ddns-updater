package settings

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
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type he struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	password      string
	useProviderIP bool
}

func NewHe(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	h := &he{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := h.isValid(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *he) isValid() error {
	if len(h.password) == 0 {
		return errors.ErrEmptyPassword
	}
	return nil
}

func (h *he) String() string {
	return utils.ToString(h.domain, h.host, constants.HE, h.ipVersion)
}

func (h *he) Domain() string {
	return h.domain
}

func (h *he) Host() string {
	return h.host
}

func (h *he) IPVersion() ipversion.IPVersion {
	return h.ipVersion
}

func (h *he) Proxied() bool {
	return false
}

func (h *he) BuildDomainName() string {
	return utils.BuildDomainName(h.host, h.domain)
}

func (h *he) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", h.BuildDomainName(), h.BuildDomainName())),
		Host:      models.HTML(h.Host()),
		Provider:  "<a href=\"https://dns.he.net/\">he.net</a>",
		IPVersion: models.HTML(h.ipVersion.String()),
	}
}

func (h *he) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	fqdn := h.BuildDomainName()
	u := url.URL{
		Scheme: "https",
		Host:   "dyn.dns.he.net",
		Path:   "/nic/update",
		User:   url.UserPassword(fqdn, h.password),
	}
	values := url.Values{}
	values.Set("hostname", fqdn)
	if !h.useProviderIP {
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

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	switch s {
	case "":
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	case constants.Badauth:
		return nil, errors.ErrAuth
	}
	if strings.Contains(s, "nochg") || strings.Contains(s, "good") {
		verifier := verification.NewVerifier()
		ipsV4 := verifier.SearchIPv4(s)
		ipsV6 := verifier.SearchIPv6(s)
		ips := append(ipsV4, ipsV6...) //nolint:gocritic
		if ips == nil {
			return nil, errors.ErrNoResultReceived
		}
		newIP = net.ParseIP(ips[0])
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ips[0])
		} else if !h.useProviderIP && !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	}
	return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
}
