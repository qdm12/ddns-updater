package config

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Resolver struct {
	Address *string
	Timeout time.Duration
}

func (r *Resolver) setDefaults() {
	r.Address = gosettings.DefaultPointer(r.Address, "")
	const defaultTimeout = 5 * time.Second
	r.Timeout = gosettings.DefaultComparable(r.Timeout, defaultTimeout)
}

var (
	ErrAddressHostEmpty = errors.New("address host is empty")
	ErrAddressPortEmpty = errors.New("address port is empty")
	ErrTimeoutTooLow    = errors.New("timeout is too low")
)

func (r Resolver) Validate() (err error) {
	if *r.Address != "" {
		host, port, err := net.SplitHostPort(*r.Address)
		if err != nil {
			return fmt.Errorf("splitting host and port from address: %w", err)
		}

		switch {
		case host == "":
			return fmt.Errorf("%w: in %s", ErrAddressHostEmpty, *r.Address)
		case port == "":
			return fmt.Errorf("%w: in %s", ErrAddressPortEmpty, *r.Address)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if r.Timeout < minTimeout {
		return fmt.Errorf("%w: %s is below the minimum %s",
			ErrTimeoutTooLow, r.Timeout, minTimeout)
	}

	return nil
}

func (r Resolver) String() string {
	return r.ToLinesNode().String()
}

func (r Resolver) ToLinesNode() *gotree.Node {
	if *r.Address == "" {
		return gotree.New("Resolver: use Go default resolver")
	}

	node := gotree.New("Resolver")
	node.Appendf("Address: %s", *r.Address)
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
	r.Timeout, err = reader.Duration("RESOLVER_TIMEOUT")
	return err
}
