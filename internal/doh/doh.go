// Package doh provides a DNS over HTTPS resolver, which is useful when
// the local DNS servers do not resolve the names needed, or when plain
// DNS on port 53 is blocked on the network.
package doh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Client is a DNS over HTTPS resolver client, as defined by RFC 8484.
type Client struct {
	url     string
	timeout time.Duration
	client  *http.Client

	mutex sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	addresses []netip.Addr
	expiresAt time.Time
}

const (
	contentTypeDNSMessage = "application/dns-message"
	cacheDefaultTTL       = 60 * time.Second
	cacheMaxTTL           = 10 * time.Minute
)

// Option is an option to create a DNS over HTTPS client.
type Option func(client *Client)

// WithHTTPClient sets the HTTP client used to send the DNS over HTTPS
// requests. It is mostly useful for testing.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		client.client = httpClient
	}
}

// New creates a new DNS over HTTPS client.
func New(url string, timeout time.Duration, options ...Option) (client *Client, err error) {
	if url == "" {
		return nil, errors.New("DNS over HTTPS URL is empty")
	}

	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("DNS over HTTPS URL %q must start with https://", url)
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	client = &Client{
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				// The DNS over HTTPS server name must be resolved
				// through the regular resolver.
				ForceAttemptHTTP2: true,
			},
		},
		cache: make(map[string]cacheEntry),
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

// LookupNetIP resolves the host name given, and returns the IP addresses
// matching the network given, which can be "ip", "ip4" or "ip6".
func (c *Client) LookupNetIP(ctx context.Context, network, host string) (
	addresses []netip.Addr, err error,
) {
	if host == "" {
		return nil, errors.New("host is empty")
	}

	if net.ParseIP(host) != nil { // already an IP address
		address, err := netip.ParseAddr(host)
		if err != nil {
			return nil, fmt.Errorf("parsing IP address %q: %w", host, err)
		}
		return []netip.Addr{address}, nil
	}

	switch network {
	case "ip", "ip4", "ip6":
	default:
		return nil, fmt.Errorf("network %q is not supported", network)
	}

	c.mutex.Lock()
	entry, ok := c.cache[network+"/"+host]
	c.mutex.Unlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.addresses, nil
	}

	addresses, err = c.query(ctx, network, host)
	if err != nil {
		return nil, err
	}

	c.mutex.Lock()
	c.cache[network+"/"+host] = cacheEntry{
		addresses: addresses,
		expiresAt: time.Now().Add(cacheDefaultTTL),
	}
	c.mutex.Unlock()

	return addresses, nil
}

func (c *Client) query(ctx context.Context, network, host string) (
	addresses []netip.Addr, err error,
) {
	var queryTypes []uint16
	switch network {
	case "ip4":
		queryTypes = []uint16{dns.TypeA}
	case "ip6":
		queryTypes = []uint16{dns.TypeAAAA}
	default:
		queryTypes = []uint16{dns.TypeA, dns.TypeAAAA}
	}

	for _, queryType := range queryTypes {
		found, err := c.queryType(ctx, host, queryType)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, found...)
	}

	if len(addresses) == 0 {
		// Use the same error message as the Go resolver, so that
		// callers can detect a name which does not exist.
		return nil, &net.DNSError{
			Err:        "no such host",
			Name:       host,
			IsNotFound: true,
		}
	}

	return addresses, nil
}

func (c *Client) queryType(ctx context.Context, host string, queryType uint16) (
	addresses []netip.Addr, err error,
) {
	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(host), queryType)
	message.RecursionDesired = true

	packed, err := message.Pack()
	if err != nil {
		return nil, fmt.Errorf("packing DNS query: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url,
		bytes.NewReader(packed))
	if err != nil {
		return nil, fmt.Errorf("creating DNS over HTTPS request: %w", err)
	}
	request.Header.Set("Content-Type", contentTypeDNSMessage)
	request.Header.Set("Accept", contentTypeDNSMessage)

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending DNS over HTTPS request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DNS over HTTPS server returned status %d", response.StatusCode)
	}

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading DNS over HTTPS response body: %w", err)
	}

	responseMessage := new(dns.Msg)
	err = responseMessage.Unpack(body.Bytes())
	if err != nil {
		return nil, fmt.Errorf("unpacking DNS over HTTPS response: %w", err)
	}

	if responseMessage.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("DNS over HTTPS server returned code %s",
			dns.RcodeToString[responseMessage.Rcode])
	}

	for _, answer := range responseMessage.Answer {
		switch record := answer.(type) {
		case *dns.A:
			address, ok := netip.AddrFromSlice(record.A.To4())
			if ok {
				addresses = append(addresses, address)
			}
		case *dns.AAAA:
			address, ok := netip.AddrFromSlice(record.AAAA.To16())
			if ok {
				addresses = append(addresses, address)
			}
		}
	}

	return addresses, nil
}
