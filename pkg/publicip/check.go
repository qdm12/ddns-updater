package publicip

import (
	"context"
	"net/netip"
	"sync"
	"time"
)

const (
	// MethodDNS is the name of the DNS over TLS public IP fetching method.
	MethodDNS = "DNS"
	// MethodHTTP is the name of the HTTPS public IP fetching method.
	MethodHTTP = "HTTP"
)

// FetchResult is the outcome of a single public IP lookup.
type FetchResult struct {
	// IP is the public IP address found. It is the zero value if Err is set.
	IP netip.Addr
	// Err is the error encountered, or nil on success.
	Err error
	// Duration is how long the lookup took.
	Duration time.Duration
}

// CheckResult holds the IPv4 and IPv6 lookup results of one fetching method.
type CheckResult struct {
	// Method is MethodDNS or MethodHTTP.
	Method string
	IPv4   FetchResult
	IPv6   FetchResult
}

// Check queries the public IPv4 and IPv6 addresses with every enabled
// fetching method separately (instead of cycling between them like the
// IP, IP4 and IP6 methods do). It always returns one result per enabled
// method, in a stable order, and never fails as a whole: errors are
// reported in each result. All lookups run concurrently.
func (f *Fetcher) Check(ctx context.Context) (results []CheckResult) {
	results = make([]CheckResult, len(f.fetchers))

	var wg sync.WaitGroup
	for i, fetcher := range f.fetchers {
		results[i].Method = f.names[i]

		wg.Add(2) //nolint:mnd
		go func(fetch func(context.Context) (netip.Addr, error), out *FetchResult) {
			defer wg.Done()
			*out = timedFetch(ctx, fetch)
		}(fetcher.IP4, &results[i].IPv4)
		go func(fetch func(context.Context) (netip.Addr, error), out *FetchResult) {
			defer wg.Done()
			*out = timedFetch(ctx, fetch)
		}(fetcher.IP6, &results[i].IPv6)
	}
	wg.Wait()

	return results
}

func timedFetch(ctx context.Context,
	fetch func(context.Context) (netip.Addr, error),
) (result FetchResult) {
	start := time.Now()
	result.IP, result.Err = fetch(ctx)
	result.Duration = time.Since(start)
	if result.Err != nil {
		result.IP = netip.Addr{}
	}
	return result
}
