package godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	key       string
	secret    string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		secret:    extraSettings.Secret,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var keyRegex = regexp.MustCompile(`^[A-Za-z0-9]{8,14}\_[A-Za-z0-9]{21,22}$`)

func (p *Provider) isValid() error {
	switch {
	case !keyRegex.MatchString(p.key):
		return fmt.Errorf("%w", errors.ErrMalformedKey)
	case p.secret == "":
		return fmt.Errorf("%w", errors.ErrEmptySecret)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.GoDaddy, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://godaddy.com\">GoDaddy</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAuthSSOKey(request, p.key, p.secret)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	type goDaddyPutBody struct {
		Data string `json:"data"` // IP address to update to
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.godaddy.com",
		Path:   fmt.Sprintf("/v1/domains/%s/records/%s/%s", p.domain, recordType, p.host),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	requestData := []goDaddyPutBody{
		{Data: ip.String()},
	}
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
	var parsedJSON struct {
		Message string `json:"message"`
	}
	jsonErr := json.Unmarshal(b, &parsedJSON)
	if jsonErr != nil || parsedJSON.Message == "" {
		return netip.Addr{}, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
	}
	return netip.Addr{}, fmt.Errorf("%w: %s", err, parsedJSON.Message)
}
