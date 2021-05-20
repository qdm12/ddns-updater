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
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type infomaniak struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
}

func NewInfomaniak(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (s Settings, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	i := &infomaniak{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := i.isValid(); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *infomaniak) isValid() error {
	switch {
	case len(i.username) == 0:
		return errors.ErrEmptyUsername
	case len(i.password) == 0:
		return errors.ErrEmptyPassword
	case i.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (i *infomaniak) String() string {
	return utils.ToString(i.domain, i.host, constants.Infomaniak, i.ipVersion)
}

func (i *infomaniak) Domain() string {
	return i.domain
}

func (i *infomaniak) Host() string {
	return i.host
}

func (i *infomaniak) IPVersion() ipversion.IPVersion {
	return i.ipVersion
}

func (i *infomaniak) Proxied() bool {
	return false
}

func (i *infomaniak) BuildDomainName() string {
	return utils.BuildDomainName(i.host, i.domain)
}

func (i *infomaniak) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", i.BuildDomainName(), i.BuildDomainName())),
		Host:      models.HTML(i.Host()),
		Provider:  "<a href=\"https://www.infomaniak.com/\">Infomaniak</a>",
		IPVersion: models.HTML(i.ipVersion.String()),
	}
}

func (i *infomaniak) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "infomaniak.com",
		Path:   "/nic/update",
		User:   url.UserPassword(i.username, i.password),
	}
	values := url.Values{}
	values.Set("hostname", i.domain)
	if i.host != "@" {
		values.Set("hostname", i.host+"."+i.domain)
	}
	if !i.useProviderIP {
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
			return nil, errors.ErrHostnameNotExists
		case constants.Badauth:
			return nil, errors.ErrAuth
		default:
			return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
		}
	default:
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	}
}
