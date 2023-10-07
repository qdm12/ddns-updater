package mikrotik

import (
	"context"
	"encoding/json"
	builtinErrors "errors"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"

	"github.com/go-routeros/routeros" //nolint:misspell
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

var (
	ErrorAddressListNotFound = builtinErrors.New("address list not found")
)

type Provider struct {
	ipVersion   ipversion.IPVersion
	routerIP    string
	username    string
	password    string
	addressList string
	client      *routeros.Client
}

func New(data json.RawMessage, _, _ string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	if ipVersion == ipversion.IP6 {
		return nil, fmt.Errorf("%w", errors.ErrIPv6NotSupported)
	}
	extraSettings := struct {
		RouterIP    string `json:"router_ip"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		AddressList string `json:"address_list"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		ipVersion:   ipVersion,
		routerIP:    extraSettings.RouterIP,
		username:    extraSettings.Username,
		password:    extraSettings.Password,
		addressList: extraSettings.AddressList,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var hostRegex = regexp.MustCompile(`^[a-zA-Z]{2,}$`)

func (p *Provider) authenticate() (*routeros.Client, error) {
	return routeros.Dial(p.routerIP, p.username, p.password)
}

type AddressListItem struct {
	ID      string
	List    string
	Address string
}

func (i *AddressListItem) found() bool {
	return i.ID != "" && i.Address != ""
}

func (p *Provider) getListIDAndValue() (item *AddressListItem, err error) {
	resp, err := p.client.Run("/ip/firewall/address-list/print", queryParam("disabled", "false"), queryParam("list", p.addressList))
	if err != nil {
		return &AddressListItem{}, err
	}

	if len(resp.Re) > 1 {
		return &AddressListItem{}, fmt.Errorf("%w: more than one item in the address list, please remove extra entries", errors.ErrConflictingRecord)
	}

	if len(resp.Re) == 0 {
		return &AddressListItem{}, nil
	}

	re := resp.Re[0]

	item = &AddressListItem{
		ID:      re.Map[".id"],
		List:    re.Map["list"],
		Address: re.Map["address"],
	}

	return item, nil
}

func (p *Provider) setListValue(id string, value string) error {
	_, err := p.client.Run("/ip/firewall/address-list/set", identityParam("id", id), valueParam("address", value))
	return err
}

func (p *Provider) addListValue(listName string, value string) error {
	_, err := p.client.Run("/ip/firewall/address-list/add", valueParam("list", listName), valueParam("address", value))
	return err
}

func (p *Provider) isValid() error {
	if !hostRegex.MatchString(p.addressList) {
		return fmt.Errorf("%w: host %q does not match regex %q",
			errors.ErrKeyNotValid, p.addressList, hostRegex)
	}
	if p.routerIP == "" {
		return fmt.Errorf("%w: router_ip cannot be empty", errors.ErrKeyNotSet)
	}
	var err error
	p.client, err = p.authenticate()
	if err != nil {
		return fmt.Errorf("%w: error authenticating with router: %w", errors.ErrAuth, err)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.Domain(), p.addressList, constants.Namecheap, p.ipVersion)
}

func (p *Provider) Domain() string {
	return "N/A"
}

func (p *Provider) Host() string {
	return p.addressList
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return "N/A"
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    p.Domain(),
		Host:      p.addressList,
		Provider:  fmt.Sprintf("<a href=\"http://%s\">Web Config</a>", p.routerIP),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(_ context.Context, _ *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	if p.client == nil {
		p.client, err = p.authenticate()
		if err != nil {
			return netip.Addr{}, fmt.Errorf("%w: error authenticating with router: %w", errors.ErrAuth, err)
		}
	}

	listItem, err := p.getListIDAndValue()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: unable to retrieve address list items", errors.ErrUnsuccessful)
	}

	if listItem.found() {
		err = p.setListValue(listItem.ID, ip.String())
		if err != nil {
			return netip.Addr{}, err
		}
	} else {
		err = p.addListValue(p.addressList, ip.String())
		if err != nil {
			return netip.Addr{}, err
		}
	}

	updatedItem, _ := p.getListIDAndValue()

	newIP, err = netip.ParseAddr(updatedItem.Address)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}

func identityParam(key string, value string) string {
	return fmt.Sprintf("=.%s=%s", key, value)
}

func queryParam(key string, value string) string {
	return fmt.Sprintf("?%s=%s", key, value)
}

func valueParam(key string, value string) string {
	return fmt.Sprintf("=%s=%s", key, value)
}
