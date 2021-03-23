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

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type freedns struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	token     string
}

func NewFreedns(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	f := &freedns{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
	}
	if err := f.isValid(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *freedns) isValid() error {
	if len(f.token) == 0 {
		return ErrEmptyToken
	}
	return nil
}

func (f *freedns) String() string {
	return toString(f.domain, f.host, constants.FREEDNS, f.ipVersion)
}

func (f *freedns) Domain() string {
	return f.domain
}

func (f *freedns) Host() string {
	return f.host
}

func (f *freedns) Proxied() bool {
	return false
}

func (f *freedns) IPVersion() ipversion.IPVersion {
	return f.ipVersion
}

func (f *freedns) BuildDomainName() string {
	return buildDomainName(f.host, f.domain)
}

func (f *freedns) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", f.BuildDomainName(), f.BuildDomainName())),
		Host:      models.HTML(f.Host()),
		Provider:  "<a href=\"https://freedns.afraid.org/\">FreeDNS</a>",
		IPVersion: models.HTML(f.ipVersion.String()),
	}
}

func (f *freedns) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	var hostPrefix string
	if ip.To4() == nil {
		hostPrefix = "v6."
	}

	u := url.URL{
		Scheme: "https",
		Host:   hostPrefix + "sync.afraid.org",
		Path:   "/u/" + f.token + "/",
	}

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
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, s)
	}

	if s == "" {
		return nil, ErrNoResultReceived
	}

	// Example: Updated demo.freshdns.com from 50.23.197.94 to 2607:f0d0:1102:d5::2
	words := strings.Fields(s)
	const expectedWords = 6
	if len(words) != expectedWords {
		return nil, fmt.Errorf("%w: not enough fields in response: %s", ErrUnmarshalResponse, s)
	}

	ipString := words[5]

	newIP = net.ParseIP(ipString)
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, newIP)
	}

	return newIP, nil
}
