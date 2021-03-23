package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
)

type njalla struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	key           string
	useProviderIP bool
}

func NewNjalla(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Key           string `json:"key"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	n := &njalla{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		key:           extraSettings.Key,
		useProviderIP: extraSettings.UseProviderIP,
	}
	if err := n.isValid(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *njalla) isValid() error {
	if len(n.key) == 0 {
		return ErrEmptyKey
	}
	return nil
}

func (n *njalla) String() string {
	return toString(n.domain, n.host, constants.NJALLA, n.ipVersion)
}

func (n *njalla) Domain() string {
	return n.domain
}

func (n *njalla) Host() string {
	return n.host
}

func (n *njalla) IPVersion() models.IPVersion {
	return n.ipVersion
}

func (n *njalla) Proxied() bool {
	return false
}

func (n *njalla) BuildDomainName() string {
	return buildDomainName(n.host, n.domain)
}

func (n *njalla) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", n.BuildDomainName(), n.BuildDomainName())),
		Host:      models.HTML(n.Host()),
		Provider:  "<a href=\"https://njal.la/\">Njalla</a>",
		IPVersion: models.HTML(n.ipVersion),
	}
}

func (n *njalla) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "njal.la",
		Path:   "/update",
	}
	values := url.Values{}
	values.Set("h", n.BuildDomainName())
	if n.host == "*" {
		values.Set("h", "*."+n.domain)
	}
	values.Set("k", n.key)
	updatingIP6 := ip.To4() == nil
	if n.useProviderIP {
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
	setUserAgent(request)

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
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	switch response.StatusCode {
	case http.StatusOK:
		if respBody.Message != "record updated" {
			return nil, fmt.Errorf("%w: message received: %s", ErrUnknownResponse, respBody.Message)
		}
		ipString := respBody.Value.A
		if updatingIP6 {
			ipString = respBody.Value.AAAA
		}
		newIP = net.ParseIP(ipString)
		if newIP == nil {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, ipString)
		} else if !ip.Equal(newIP) {
			return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, newIP.String())
		}
		return newIP, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("%w: %s", ErrAuth, respBody.Message)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, respBody.Message)
	}

	return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, respBody.Message)
}
