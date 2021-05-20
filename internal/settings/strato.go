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
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type strato struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	password      string
	useProviderIP bool
}

func NewStrato(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	ss := &strato{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := ss.isValid(); err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *strato) isValid() error {
	switch {
	case len(s.password) == 0:
		return errors.ErrEmptyPassword
	case s.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (s *strato) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Strato]", s.domain, s.host)
}

func (s *strato) Domain() string {
	return s.domain
}

func (s *strato) Host() string {
	return s.host
}

func (s *strato) IPVersion() ipversion.IPVersion {
	return s.ipVersion
}

func (s *strato) Proxied() bool {
	return false
}

func (s *strato) BuildDomainName() string {
	return utils.BuildDomainName(s.host, s.domain)
}

func (s *strato) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", s.BuildDomainName(), s.BuildDomainName())),
		Host:      models.HTML(s.Host()),
		Provider:  "<a href=\"https://strato.com/\">Strato DNS</a>",
		IPVersion: models.HTML(s.ipVersion.String()),
	}
}

func (s *strato) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(s.domain, s.password),
		Host:   "dyndns.strato.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", s.BuildDomainName())
	if !s.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	// headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	str := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, str)
	}

	switch {
	case strings.HasPrefix(str, constants.Notfqdn):
		return nil, errors.ErrHostnameNotExists
	case strings.HasPrefix(str, constants.Abuse):
		return nil, errors.ErrAbuse
	case strings.HasPrefix(str, "badrequest"):
		return nil, errors.ErrBadRequest
	case strings.HasPrefix(str, "constants.Badauth"):
		return nil, errors.ErrAuth
	case strings.HasPrefix(str, "good"), strings.HasPrefix(str, "nochg"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, str)
	}
}
