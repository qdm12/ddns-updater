package config

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Resolver struct {
	Address *string
	Timeout time.Duration
	// DOHURL is the DNS over HTTPS server URL to use, for example
	// https://cloudflare-dns.com/dns-query. It is useful when the local
	// DNS servers do not resolve the names needed, or when plain DNS on
	// port 53 is blocked on the network. It is empty when not used.
	DOHURL *string
}

func (r *Resolver) setDefaults() {
	r.Address = gosettings.DefaultPointer(r.Address, "")
	const defaultTimeout = 5 * time.Second
	r.Timeout = gosettings.DefaultComparable(r.Timeout, defaultTimeout)
	r.DOHURL = gosettings.DefaultPointer(r.DOHURL, "")
}

func (r Resolver) Validate() (err error) {
	if *r.Address != "" {
		host, port, err := net.SplitHostPort(*r.Address)
		if err != nil {
			return fmt.Errorf("splitting host and port from address: %w", err)
		}

		switch {
		case host == "":
			return fmt.Errorf("address host is empty: in %s", *r.Address)
		case port == "":
			return fmt.Errorf("address port is empty: in %s", *r.Address)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if r.Timeout < minTimeout {
		return fmt.Errorf("timeout is too low: %s is below the minimum %s", r.Timeout, minTimeout)
	}

	if *r.DOHURL != "" && !strings.HasPrefix(*r.DOHURL, "https://") {
		return fmt.Errorf("DNS over HTTPS URL must start with https://: %s", *r.DOHURL)
	}

	return nil
}

func (r Resolver) String() string {
	return r.ToLinesNode().String()
}

func (r Resolver) ToLinesNode() *gotree.Node {
	if *r.Address == "" && *r.DOHURL == "" {
		return gotree.New("Resolver: use Go default resolver")
	}

	node := gotree.New("Resolver")
	if *r.Address != "" {
		node.Appendf("Address: %s", *r.Address)
	}
	if *r.DOHURL != "" {
		node.Appendf("DNS over HTTPS URL: %s", *r.DOHURL)
	}
	node.Appendf("Timeout: %s", r.Timeout)
	return node
}

func (r *Resolver) read(reader *reader.Reader) (err error) {
	r.Address = reader.Get("RESOLVER_ADDRESS")
	if r.Address != nil { // conveniently add port 53 if not specified
		_, port, err := net.SplitHostPort(*r.Address)
		if err == nil && port == "" {
			*r.Address += ":53"
		}
	}
	r.DOHURL = reader.Get("RESOLVER_DOH_URL")
	r.Timeout, err = reader.Duration("RESOLVER_TIMEOUT")
	return err
}
