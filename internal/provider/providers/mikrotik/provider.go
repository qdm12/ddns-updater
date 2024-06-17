package mikrotik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	ipVersion     ipversion.IPVersion
	ipv6Suffix    netip.Prefix
	routerAddress netip.AddrPort
	username      string
	password      string
	addressList   string
}

type settings struct {
	RouterIP    netip.Addr `json:"router_ip"`
	Username    string     `json:"username"`
	Password    string     `json:"password"`
	AddressList string     `json:"address_list"`
}

func New(data json.RawMessage, ipVersion ipversion.IPVersion,
	ipv6Suffix netip.Prefix) (p *Provider, err error) {
	var providerSpecificSettings settings
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}
	err = validateSettings(providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}

	const routerPort = 8728
	return &Provider{
		ipVersion:     ipVersion,
		ipv6Suffix:    ipv6Suffix,
		routerAddress: netip.AddrPortFrom(providerSpecificSettings.RouterIP, routerPort),
		username:      providerSpecificSettings.Username,
		password:      providerSpecificSettings.Password,
		addressList:   providerSpecificSettings.AddressList,
	}, nil
}

var addressListRegex = regexp.MustCompile(`^[a-zA-Z]{2,}$`)

func validateSettings(settings settings) error {
	switch {
	case !addressListRegex.MatchString(settings.AddressList):
		return fmt.Errorf("%w: host %q does not match regex %q",
			errors.ErrKeyNotValid, settings.AddressList, addressListRegex)
	case !settings.RouterIP.IsValid():
		return fmt.Errorf("%w: router_ip cannot be empty", errors.ErrKeyNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.Domain(), p.addressList, constants.Mikrotik, p.ipVersion)
}

func (p *Provider) Domain() string {
	return "N / A"
}

func (p *Provider) Host() string {
	return "N / A"
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
	return ""
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    p.Domain(),
		Host:      p.addressList,
		Provider:  fmt.Sprintf("<a href=\"http://%s\">Mikrotik</a>", p.routerAddress),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(_ context.Context, _ *http.Client, ip netip.Addr) (
	newIP netip.Addr, err error) {
	client, err := newClient(p.routerAddress)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating client: %w", err)
	}
	defer client.Close()
	err = client.login(p.username, p.password)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("logging in router: %w", err)
	}

	addressListItems, err := getAddressListItems(client, p.addressList)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting address list items: %w", err)
	}

	if len(addressListItems) == 0 {
		_, err = client.Run("/ip/firewall/address-list/add",
			"=list="+p.addressList, "=address="+ip.String())
		if err != nil {
			return netip.Addr{}, fmt.Errorf("adding address list %q: %w",
				p.addressList, err)
		}
		return ip, nil
	}

	for _, addressListItem := range addressListItems {
		if addressListItem.address == ip.String() {
			continue // already up to date
		}
		_, err = client.Run("/ip/firewall/address-list/set",
			"=.id="+addressListItem.id, "=address="+ip.String())
		if err != nil {
			return netip.Addr{}, fmt.Errorf("setting address in address list id %q: %w",
				addressListItem.id, err)
		}
	}

	return ip, nil
}
