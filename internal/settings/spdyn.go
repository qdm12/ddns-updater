package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type spdyn struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	user          string
	password      string
	token         string
	useProviderIP bool
}

func NewSpdyn(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		User          string `json:"user"`
		Password      string `json:"password"`
		Token         string `json:"token"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	spdyn := &spdyn{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		user:          extraSettings.User,
		password:      extraSettings.Password,
		token:         extraSettings.Token,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := spdyn.isValid(); err != nil {
		return nil, err
	}
	return spdyn, nil
}

func (s *spdyn) isValid() error {
	if len(s.token) > 0 {
		return nil
	}
	switch {
	case len(s.user) == 0:
		return ErrEmptyUsername
	case len(s.password) == 0:
		return ErrEmptyPassword
	case s.host == "*":
		return ErrHostWildcard
	}
	return nil
}

func (s *spdyn) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Spdyn]", s.domain, s.host)
}

func (s *spdyn) Domain() string {
	return s.domain
}

func (s *spdyn) Host() string {
	return s.host
}

func (s *spdyn) IPVersion() ipversion.IPVersion {
	return s.ipVersion
}

func (s *spdyn) Proxied() bool {
	return false
}

func (s *spdyn) BuildDomainName() string {
	return buildDomainName(s.host, s.domain)
}

func (s *spdyn) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", s.BuildDomainName(), s.BuildDomainName())),
		Host:      models.HTML(s.Host()),
		Provider:  "<a href=\"https://spdyn.com/\">Spdyn DNS</a>",
		IPVersion: models.HTML(s.ipVersion),
	}
}

func (s *spdyn) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	// see https://wiki.securepoint.de/SPDyn/Variablen
	u := url.URL{
		Scheme: "https",
		Host:   "update.spdyn.de",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", s.BuildDomainName())
	if s.useProviderIP {
		values.Set("myip", "10.0.0.1")
	} else {
		values.Set("myip", ip.String())
	}
	if len(s.token) > 0 {
		values.Set("user", s.BuildDomainName())
		values.Set("pass", s.token)
	} else {
		values.Set("user", s.user)
		values.Set("pass", s.password)
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

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	bodyString := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyDataToSingleLine(bodyString))
	}

	switch bodyString {
	case abuse, "numhost":
		return nil, ErrAbuse
	case badauth, "!yours":
		return nil, ErrAuth
	case "good":
		return ip, nil
	case notfqdn:
		return nil, fmt.Errorf("%w: not fqdn", ErrBadRequest)
	case "nochg":
		return ip, nil
	case "nohost", "fatal":
		return nil, ErrHostnameNotExists
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
}
