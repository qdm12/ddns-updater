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
	domain     string
	host       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	key        string
	secret     string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:     domain,
		host:       host,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		key:        extraSettings.Key,
		secret:     extraSettings.Secret,
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
		return fmt.Errorf("%w: key %q does not match regex %s",
			errors.ErrKeyNotValid, p.key, keyRegex)
	case p.secret == "":
		return fmt.Errorf("%w", errors.ErrSecretNotSet)
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

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.godaddy.com/en-ie\">GoDaddy</a>",
		IPVersion: p.ipVersion.String(),
	}
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
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetAuthSSOKey(request, p.key, p.secret)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}

	err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
	var parsedJSON struct {
		Message string `json:"message"`
	}
	jsonErr := json.Unmarshal(b, &parsedJSON)
	if jsonErr != nil || parsedJSON.Message == "" {
		return netip.Addr{}, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
	}
	return netip.Addr{}, fmt.Errorf("%w: %s", err, parsedJSON.Message)
}
