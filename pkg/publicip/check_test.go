package publicip

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"
)

type fetcherFake struct {
	ip4, ip6       netip.Addr
	err4, err6     error
	delay          time.Duration
	receivedCtxErr error
}

func (f *fetcherFake) IP(context.Context) (netip.Addr, error) { return f.ip4, f.err4 }

func (f *fetcherFake) IP4(ctx context.Context) (netip.Addr, error) {
	time.Sleep(f.delay)
	return f.ip4, f.err4
}

func (f *fetcherFake) IP6(ctx context.Context) (netip.Addr, error) {
	time.Sleep(f.delay)
	// Returned together with an error on purpose, to check the IP is dropped.
	return f.ip6, f.err6
}

func Test_Fetcher_Check(t *testing.T) {
	t.Parallel()

	errIPv6 := errors.New("ipv6 not supported")
	const delay = 100 * time.Millisecond
	fetcher := &Fetcher{
		fetchers: []ipFetcher{
			&fetcherFake{
				ip4:   netip.MustParseAddr("203.0.113.7"),
				ip6:   netip.MustParseAddr("2001:db8::1"),
				err6:  errIPv6,
				delay: delay,
			},
			&fetcherFake{
				ip4:   netip.MustParseAddr("203.0.113.7"),
				ip6:   netip.MustParseAddr("2001:db8::2"),
				delay: delay,
			},
		},
		names:   []string{MethodDNS, MethodHTTP},
		counter: new(uint32),
	}

	start := time.Now()
	results := fetcher.Check(context.Background())
	elapsed := time.Since(start)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Method != MethodDNS || results[1].Method != MethodHTTP {
		t.Errorf("unexpected methods: %q, %q", results[0].Method, results[1].Method)
	}
	if results[0].IPv4.IP != netip.MustParseAddr("203.0.113.7") || results[0].IPv4.Err != nil {
		t.Errorf("unexpected DNS ipv4 result: %+v", results[0].IPv4)
	}
	if !errors.Is(results[0].IPv6.Err, errIPv6) || results[0].IPv6.IP.IsValid() {
		t.Errorf("expected the error and no IP for DNS ipv6, got %+v", results[0].IPv6)
	}
	if results[1].IPv6.IP != netip.MustParseAddr("2001:db8::2") || results[1].IPv6.Err != nil {
		t.Errorf("unexpected HTTP ipv6 result: %+v", results[1].IPv6)
	}
	for _, result := range results {
		for _, lookup := range []FetchResult{result.IPv4, result.IPv6} {
			if lookup.Duration < delay {
				t.Errorf("%s: duration %s is shorter than the %s delay", result.Method, lookup.Duration, delay)
			}
		}
	}
	// All four lookups run concurrently: well below the 4 * delay sequential time.
	if elapsed >= 3*delay {
		t.Errorf("lookups do not seem to run concurrently, took %s", elapsed)
	}
}

func Test_Fetcher_Check_contextCancelled(t *testing.T) {
	t.Parallel()
	// Check must always return one result per method, even with errors.
	errBoom := errors.New("boom")
	fetcher := &Fetcher{
		fetchers: []ipFetcher{&fetcherFake{err4: errBoom, err6: errBoom}},
		names:    []string{MethodHTTP},
		counter:  new(uint32),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := fetcher.Check(ctx)
	if len(results) != 1 || results[0].IPv4.Err == nil || results[0].IPv6.Err == nil {
		t.Errorf("unexpected results: %+v", results)
	}
}
