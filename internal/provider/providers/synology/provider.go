package synology

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	dsID       string
	dsSerial   string
	macAddress string
	authKey    string
	apiKey     string
}

type extraSettings struct {
	MydsID     string `json:"myds_id"`
	Serial     string `json:"serial"`
	MACAddress string `json:"mac_address"`
	AuthKey    string `json:"auth_key"`
	ApiKey     string `json:"api_key"`
}

type apiResponse struct {
	Code string `json:"code"`
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {

	var eSettings extraSettings

	err = json.Unmarshal(data, &eSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, &eSettings)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		dsID:       eSettings.MydsID,
		dsSerial:   eSettings.Serial,
		macAddress: eSettings.MACAddress,
		authKey:    eSettings.AuthKey,
		apiKey:     eSettings.ApiKey,
	}, nil
}

var supportedDomains []string = []string{
	"synology.me",
	"DiskStation.me",
	"i234.me",
	"DCloud.biz",
	"DCloud.me",
	"DCloud.mobi",
	"DSmyNAS.com",
	"DSmyNAS.net",
	"DSmyNAS.org",
	"FamilyDS.com",
	"FamilyDS.net",
	"FamilyDS.org",
}

func validateSettings(domain string, es *extraSettings) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	var matchDomain int = -1
	for idx, suffix := range supportedDomains {
		if strings.HasSuffix(domain, strings.ToLower(suffix)) {
			matchDomain = idx
		}
	}
	if matchDomain < 0 {
		return fmt.Errorf("Invalid domain %s: must be one of %s", domain, supportedDomains)
	}

	var entries []string

	if es.MydsID == "" {
		entries = append(entries, "myds_id")
	}
	if es.Serial == "" {
		entries = append(entries, "serial")
	}
	if es.AuthKey == "" {
		entries = append(entries, "auth_key")
	}
	if es.ApiKey == "" {
		entries = append(entries, "api_key")
	}

	if len(entries) != 0 {
		return fmt.Errorf("Missing values for given keys: %s", entries)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Synology, p.ipVersion)
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

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  fmt.Sprintf("<a href=\"https://account.synology.com/en-global/device/%s\">Synology DNS</a>", p.dsSerial),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) getMacAddress() (string, error) {
	macAddress := p.macAddress
	if macAddress != "" {
		return macAddress, nil
	}

	ifas, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("obtaining net interfaces: %w", err)
	}

	var netFlags = net.FlagUp | net.FlagRunning
	for _, ifa := range ifas {
		// Skip not operational network interface
		if (ifa.Flags & netFlags) != netFlags {
			continue
		}

		addr := ifa.HardwareAddr.String()
		// Found Synology network card.
		if strings.HasPrefix(addr, "00:11:32") {
			return addr, nil
		}

		// Use first network interface on Linux or Mac
		if strings.HasPrefix(ifa.Name, "eth0") || strings.HasPrefix("en0", ifa.Name) {
			macAddress = addr
		}
		if macAddress == "" {
			macAddress = addr
		}
	}
	if macAddress == "" {
		// Use some made up mac address as it's needed.
		return "00:11:32:00:00:00", nil
	}

	return macAddress, nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "ddns.synology.com",
		Path:   "/main.php",
	}
	hostname := utils.BuildURLQueryHostname(p.owner, p.domain)
	values := url.Values{}
	values.Set("hostname", hostname)
	values.Set("auth_key", p.authKey)
	values.Set("api_key", p.apiKey)
	if ip.Is4() {
		values.Set("ipv4", ip.String())
	}
	if ip.Is6() {
		values.Set("ipv6", ip.String())
	}

	macAddress, err := p.getMacAddress()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("obtaining mac address: %w", err)
	}
	values.Set("mac", macAddress)
	values.Set("myds_id", p.dsID)
	values.Set("serial", p.dsSerial)
	values.Set("_", "hostname/create")

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	bodyString, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(bodyString))
	}

	var resp apiResponse
	json.Unmarshal([]byte(bodyString), &resp)

	switch resp.Code {
	case "good":
		return ip, nil
	case "apikey_not_found":
		return netip.Addr{}, errors.ErrAPIKeyNotSet
	case "badauth":
		return netip.Addr{}, errors.ErrBadRequest
	default:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, bodyString)
	}
}
