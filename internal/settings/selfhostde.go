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
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type selfhostde struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func NewSelfhostde(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
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
		return ErrEmptyUsername
	case len(sd.password) == 0:
		return ErrEmptyPassword
	case sd.host == "*":
		return ErrHostWildcard
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

func (sd *selfhostde) IPVersion() ipversion.IPVersion {
	return sd.ipVersion
}

func (sd *selfhostde) Proxied() bool {
	return false
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

func (sd *selfhostde) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(sd.username, sd.password),
		Host:   "carol.selfhost.de",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", sd.BuildDomainName())
	if !sd.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// see their PDF file
	switch response.StatusCode {
	case http.StatusOK: // DynDNS v2 specification
	case http.StatusNoContent: // no change
		return ip, nil
	case http.StatusUnauthorized:
		return nil, ErrAuth
	case http.StatusConflict:
		return nil, ErrZoneNotFound
	case http.StatusGone:
		return nil, ErrAccountInactive
	case http.StatusLengthRequired:
		return nil, fmt.Errorf("%w: %s", ErrMalformedIPSent, ip)
	case http.StatusPreconditionFailed:
		return nil, fmt.Errorf("%w: %s", ErrPrivateIPSent, ip)
	case http.StatusServiceUnavailable:
		return nil, ErrDNSServerSide
	default:
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	s := string(b)

	switch {
	case strings.HasPrefix(s, notfqdn):
		return nil, ErrHostnameNotExists
	case strings.HasPrefix(s, "abuse"):
		return nil, ErrAbuse
	case strings.HasPrefix(s, "badrequest"):
		return nil, ErrBadRequest
	case strings.HasPrefix(s, "good"), strings.HasPrefix(s, "nochg"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
}
