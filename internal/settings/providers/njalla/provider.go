package njalla

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	key           string
	useProviderIP bool
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Key           string `json:"key"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		key:           extraSettings.Key,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if len(p.key) == 0 {
		return errors.ErrEmptyKey
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Njalla, p.ipVersion)
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
		Provider:  "<a href=\"https://njal.la/\">Njalla</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "njal.la",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("h", utils.BuildURLQueryHostname(p.host, p.domain))
	values.Set("k", p.key)
	updatingIP6 := ip.To4() == nil
	if p.useProviderIP {
		values.Set("auto", "")
	} else {
		if updatingIP6 {
			values.Set("aaaa", ip.String())
		} else {
			values.Set("a", ip.String())
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

	decoder := json.NewDecoder(response.Body)
	var respBody struct {
		Message string `json:"message"`
		Value   struct {
			A    string `json:"A"`
			AAAA string `json:"AAAA"`
		} `json:"value"`
	}
	if err := decoder.Decode(&respBody); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	switch response.StatusCode {
	case http.StatusOK:
		if respBody.Message != "record updated" {
			return nil, fmt.Errorf("%w: message received: %s", errors.ErrUnknownResponse, respBody.Message)
		}
		ipString := respBody.Value.A
		if updatingIP6 {
			ipString = respBody.Value.AAAA
		}
		newIP = net.ParseIP(ipString)
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ipString)
		} else if !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("%w: %s", errors.ErrAuth, respBody.Message)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", errors.ErrBadRequest, respBody.Message)
	}

	return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, respBody.Message)
}
