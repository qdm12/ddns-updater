package dns

import (
	"context"
	"time"

	"github.com/miekg/dns"
)

//go:generate mockgen -destination=mock_$GOPACKAGE/$GOFILE . Client

// Client is an interface for the DNS client used in the implementation in this package.
// You SHOULD NOT use this interface anywhere as it is implementation specific.
type Client interface {
	ExchangeContext(ctx context.Context, m *dns.Msg, a string) (r *dns.Msg, rtt time.Duration, err error)
}
