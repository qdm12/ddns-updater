package opendns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	username   string
	password   string
	hostname   string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Hostname string `json:"hostname"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	// The network name declared in the OpenDNS account, defaulting to the
	// domain name of the record, which is also a valid network label when the
	// network was named after a domain name.
	hostname := extraSettings.Hostname
	if hostname == "" {
		hostname = utils.BuildURLQueryHostname(owner, domain)
	}

	err = validateSettings(domain, owner, extraSettings.Username, extraSettings.Password)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		username:   extraSettings.Username,
		password:   extraSettings.Password,
		hostname:   hostname,
	}, nil
}

func validateSettings(domain, owner, username, password string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("domain is not valid: %w", err)
	}

	switch {
	case owner == "*":
		return errors.New("owner cannot be \"*\"")
	case username == "":
		return errors.New("username is not set")
	case password == "":
		return errors.New("password is not set")
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.OpenDNS, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

// Proxied is always true because the update API changes the IP address of the
// network declared in the account, named by the hostname setting, and not the IP
// address of a DNS record. Resolving the domain name of the record tells nothing
// about the state of the update, so the last IP address sent to OpenDNS decides
// if an update is needed, like for proxied Cloudflare records.
func (p *Provider) Proxied() bool {
	return true
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://opendns.com/\">Opendns DNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(p.username, p.password),
		Host:   "updates.opendns.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("hostname", p.hostname)
	values.Set("myip", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("HTTP status is not valid: %d: %s", response.StatusCode, s)
	}

	// Responses are <status> [<ip address>], and the IP address is present when
	// the status is good (record updated) or nochg (record already up to date).
	status, responseIPString, _ := strings.Cut(s, " ")
	switch status {
	case "good", "nochg":
		newIP, err = netip.ParseAddr(responseIPString)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("malformed IP address received: %s", s)
		}
		if newIP.Compare(ip) != 0 {
			return netip.Addr{}, fmt.Errorf("mismatching IP address received: sent ip %s to update but received %s", ip, newIP)
		}
		return newIP, nil
	case "noauth":
		return netip.Addr{}, errors.New("update is not authorized")
	case constants.Badauth:
		return netip.Addr{}, errors.New("bad authentication")
	case constants.Nohost, constants.Notfqdn:
		return netip.Addr{}, errors.New("hostname does not exist: check the network name")
	case constants.Badagent:
		return netip.Addr{}, errors.New("user agent is banned")
	case constants.Abuse:
		return netip.Addr{}, fmt.Errorf("%w", ddnserrors.ErrBannedAbuse)
	case "dnserr", constants.Nineoneone:
		return netip.Addr{}, errors.New("server side DNS error")
	default:
		return netip.Addr{}, fmt.Errorf("unknown response received: %s", s)
	}
}
