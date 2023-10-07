package mikrotik

import (
	"context"
	"encoding/json"
	builtinErrors "errors"
	"fmt"
	"net/http"
	"net/netip"
	"regexp"

	"github.com/go-routeros/routeros"
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
	routerIp    string
	username    string
	password    string
	addressList string
	client      *routeros.Client
}

func New(data json.RawMessage, domain, host string,
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
		routerIp:    extraSettings.RouterIP,
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
	return routeros.Dial(p.routerIp, p.username, p.password)
}

func (p *Provider) getListIdAndValue() (id string, value string, err error) {
	resp, err := p.client.Run("/ip/firewall/address-list/print", queryParam("disabled", "false"), queryParam("list", p.addressList))
	if err != nil {
		return "", "", err
	}

	if len(resp.Re) > 1 {
		return "", "", builtinErrors.New("more than one item in the address list, please remove extra entries")
	}

	if len(resp.Re) == 0 {
		return "", "", ErrorAddressListNotFound
	}

	re := resp.Re[0]

	value, ok := re.Map["address"]
	if !ok {
		return "", "", fmt.Errorf("unable to parse response: %v", re.Map)
	}

	id, ok = re.Map[".id"]
	if !ok {
		return "", "", fmt.Errorf("unable to parse response: %v", re.Map)
	}

	return id, value, nil
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
	if p.routerIp == "" {
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
		Provider:  fmt.Sprintf("<a href=\"http://%s\">Web Config</a>", p.routerIp),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	if p.client == nil {
		p.client, err = p.authenticate()
		if err != nil {
			return netip.Addr{}, err
		}
	}

	id, _, err := p.getListIdAndValue()
	if builtinErrors.Is(err, ErrorAddressListNotFound) { // Address list not found, create it
		err = p.addListValue(p.addressList, ip.String())
		if err != nil {
			return netip.Addr{}, err
		}
	} else if err != nil { // Some other error, abort
		return netip.Addr{}, err
	} else { // Address list found, update value
		err = p.setListValue(id, ip.String())
		if err != nil {
			return netip.Addr{}, err
		}
	}

	_, updated, _ := p.getListIdAndValue()

	newIP, err = netip.ParseAddr(updated)
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
