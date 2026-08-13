package rfc2136

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/miekg/dns"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	// server is the address of the name server accepting the
	// DNS update messages, in the form host:port.
	server string
	// zone is the fully qualified zone name to update.
	zone string
	// tsigKeyName is the fully qualified TSIG key name. If it is empty,
	// update messages are sent unsigned.
	tsigKeyName   string
	tsigSecret    string
	tsigAlgorithm string
	ttl           uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		Server        string `json:"server"`
		Zone          string `json:"zone"`
		TSIGKeyName   string `json:"tsig_key_name"`
		TSIGSecret    string `json:"tsig_secret"`
		TSIGAlgorithm string `json:"tsig_algorithm"`
		TTL           uint32 `json:"ttl"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	if providerSpecificSettings.Zone == "" {
		providerSpecificSettings.Zone = domain
	}
	if providerSpecificSettings.TSIGAlgorithm == "" {
		providerSpecificSettings.TSIGAlgorithm = dns.HmacSHA256
	}
	if providerSpecificSettings.TTL == 0 {
		const defaultTTL = 300
		providerSpecificSettings.TTL = defaultTTL
	}
	providerSpecificSettings.TSIGAlgorithm = dns.Fqdn(providerSpecificSettings.TSIGAlgorithm)
	if providerSpecificSettings.TSIGKeyName != "" {
		providerSpecificSettings.TSIGKeyName = dns.Fqdn(providerSpecificSettings.TSIGKeyName)
	}

	err = validateSettings(domain, providerSpecificSettings.Server,
		providerSpecificSettings.TSIGKeyName, providerSpecificSettings.TSIGSecret,
		providerSpecificSettings.TSIGAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:        domain,
		owner:         owner,
		ipVersion:     ipVersion,
		ipv6Suffix:    ipv6Suffix,
		server:        addressWithDefaultPort(providerSpecificSettings.Server),
		zone:          dns.Fqdn(providerSpecificSettings.Zone),
		tsigKeyName:   providerSpecificSettings.TSIGKeyName,
		tsigSecret:    providerSpecificSettings.TSIGSecret,
		tsigAlgorithm: providerSpecificSettings.TSIGAlgorithm,
		ttl:           providerSpecificSettings.TTL,
	}, nil
}

// addressWithDefaultPort adds the default DNS port to the server address
// given, if the address does not already specify a port.
func addressWithDefaultPort(server string) (address string) {
	const defaultPort = "53"
	_, _, err := net.SplitHostPort(server)
	if err != nil { // no port specified
		return net.JoinHostPort(server, defaultPort)
	}
	return server
}

func validateSettings(domain, server, tsigKeyName, tsigSecret, tsigAlgorithm string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch tsigAlgorithm {
	case dns.HmacSHA1, dns.HmacSHA224, dns.HmacSHA256, dns.HmacSHA384, dns.HmacSHA512:
	default:
		return fmt.Errorf("%w: %s", errors.ErrAlgorithmNotValid, tsigAlgorithm)
	}

	switch {
	case server == "":
		return fmt.Errorf("%w", errors.ErrServerNotSet)
	case tsigKeyName == "" && tsigSecret != "":
		return fmt.Errorf("%w", errors.ErrKeyNotSet)
	case tsigKeyName != "" && tsigSecret == "":
		return fmt.Errorf("%w", errors.ErrSecretNotSet)
	}

	if tsigSecret != "" {
		_, err = base64.StdEncoding.DecodeString(tsigSecret)
		if err != nil {
			return fmt.Errorf("%w: %w", errors.ErrSecretNotValid, err)
		}
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.RFC2136, p.ipVersion)
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
		Provider:  "<a href=\"https://datatracker.ietf.org/doc/html/rfc2136\">RFC 2136</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update replaces the A or AAAA record set of the domain name with a single
// record pointing at the IP address given, using a DNS update message as
// specified in https://datatracker.ietf.org/doc/html/rfc2136.
// The record is created if it does not exist yet.
// The HTTP client is unused since the update is sent over DNS.
func (p *Provider) Update(ctx context.Context, _ *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	message := p.newUpdateMessage(ip)

	// TCP is used instead of UDP to avoid truncated responses, since
	// updates are infrequent enough for the extra round trips not to matter.
	client := &dns.Client{Net: "tcp"}

	if p.tsigKeyName != "" {
		client.TsigSecret = map[string]string{p.tsigKeyName: p.tsigSecret}
		const fudgeSeconds = 300
		message.SetTsig(p.tsigKeyName, p.tsigAlgorithm, fudgeSeconds, time.Now().Unix())
	}

	response, _, err := client.ExchangeContext(ctx, message, p.server)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("exchanging update message with %s: %w", p.server, err)
	}

	if response.Rcode != dns.RcodeSuccess {
		return netip.Addr{}, fmt.Errorf("%w: %s", rcodeToError(response.Rcode), dns.RcodeToString[response.Rcode])
	}

	return ip, nil
}

// newUpdateMessage builds a DNS update message deleting the existing record
// set and adding a single record with the IP address given, which the server
// applies atomically, see https://datatracker.ietf.org/doc/html/rfc2136#section-2.5
func (p *Provider) newUpdateMessage(ip netip.Addr) (message *dns.Msg) {
	header := dns.RR_Header{
		Name:   dns.Fqdn(utils.BuildDomainName(p.owner, p.domain)),
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
		Ttl:    p.ttl,
	}

	var record dns.RR
	if ip.Is6() {
		header.Rrtype = dns.TypeAAAA
		record = &dns.AAAA{Hdr: header, AAAA: net.IP(ip.AsSlice())}
	} else {
		record = &dns.A{Hdr: header, A: net.IP(ip.AsSlice())}
	}

	message = new(dns.Msg)
	message.SetUpdate(p.zone)
	message.RemoveRRset([]dns.RR{record})
	message.Insert([]dns.RR{record})
	return message
}

func rcodeToError(rcode int) (err error) {
	switch rcode {
	case dns.RcodeNotAuth:
		return errors.ErrAuth
	case dns.RcodeRefused:
		return errors.ErrBadRequest
	case dns.RcodeNotZone, dns.RcodeNameError:
		return errors.ErrZoneNotFound
	case dns.RcodeFormatError:
		return errors.ErrBadRequest
	case dns.RcodeServerFailure:
		return errors.ErrDNSServerSide
	default:
		return errors.ErrUnknownResponse
	}
}
