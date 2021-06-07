package freedns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	token     string
	logger    log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
		logger:    logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if len(p.token) == 0 {
		return errors.ErrEmptyToken
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.FreeDNS, p.ipVersion)
}

func (p *provider) Domain() string {
	return p.domain
}

func (p *provider) Host() string {
	return p.host
}

func (p *provider) Proxied() bool {
	return false
}

func (p *provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://freedns.afraid.org/\">FreeDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	var hostPrefix string
	if ip.To4() == nil {
		hostPrefix = "v6."
	}

	u := url.URL{
		Scheme: "https",
		Host:   hostPrefix + "sync.afraid.org",
		Path:   "/u/" + p.token + "/",
	}

	p.logger.Debug("HTTP GET: " + u.String())

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

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	}

	if s == "" {
		return nil, errors.ErrNoResultReceived
	}

	// Example: Updated demo.freshdns.com from 50.23.197.94 to 2607:f0d0:1102:d5::2
	words := strings.Fields(s)
	const expectedWords = 6
	if len(words) != expectedWords {
		return nil, fmt.Errorf("%w: not enough fields in response: %s", errors.ErrUnmarshalResponse, s)
	}

	ipString := words[5]

	newIP = net.ParseIP(ipString)
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, newIP)
	}

	return newIP, nil
}
