package update

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/ddns-updater/pkg/publicip/privateip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errNoInterfaceAddress = errors.New("private discovery failed")

// fakeFetcher stands in for a privateip-only fetcher: it returns one
// configurable result for every method and counts the calls.
type fakeFetcher struct {
	ip    netip.Addr
	err   error
	calls int
}

func (f *fakeFetcher) get() (netip.Addr, error) {
	f.calls++
	return f.ip, f.err
}

func (f *fakeFetcher) IP(context.Context) (netip.Addr, error)  { return f.get() }
func (f *fakeFetcher) IP4(context.Context) (netip.Addr, error) { return f.get() }
func (f *fakeFetcher) IP6(context.Context) (netip.Addr, error) { return f.get() }

type memDB struct{ records []records.Record }

func (m *memDB) Select(id uint) (records.Record, error) { return m.records[id], nil }
func (m *memDB) SelectAll() []records.Record            { return m.records }
func (m *memDB) Update(id uint, r records.Record) error { m.records[id] = r; return nil }

type spyUpdater struct{ ips []netip.Addr }

func (s *spyUpdater) Update(_ context.Context, _ uint, ip netip.Addr) error {
	s.ips = append(s.ips, ip)
	return nil
}

type fakeResolver struct{ ipv4 netip.Addr }

func (f fakeResolver) LookupNetIP(_ context.Context, network, _ string) ([]netip.Addr, error) {
	if network == "ip4" {
		return []netip.Addr{f.ipv4}, nil
	}
	return nil, errors.New("no such host")
}

type noopLogger struct{}

func (noopLogger) Debug(string) {}
func (noopLogger) Info(string)  {}
func (noopLogger) Warn(string)  {}
func (noopLogger) Error(string) {}

type noopHealthchecks struct{}

func (noopHealthchecks) Ping(context.Context, healthchecksio.State) error { return nil }

// fakeProvider embeds the interface: only the methods used by the service are implemented.
type fakeProvider struct {
	provider.Provider
	version ipversion.IPVersion
}

func (fakeProvider) BuildDomainName() string          { return "home.example.com" }
func (p fakeProvider) IPVersion() ipversion.IPVersion { return p.version }
func (fakeProvider) Proxied() bool                    { return false }
func (fakeProvider) IPv6Suffix() netip.Prefix         { return netip.Prefix{} }
func (fakeProvider) String() string                   { return "fake" }

type serviceFixture struct {
	service *Service
	db      *memDB
	fetcher *fakeFetcher
	updater *spyUpdater
	history models.History
}

func newServiceFixture(version ipversion.IPVersion) serviceFixture {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	dns := netip.MustParseAddr("192.168.50.159")
	history := models.History{{IP: dns, Time: now.Add(-time.Hour)}}
	db := &memDB{records: []records.Record{
		records.New(fakeProvider{version: version}, append(models.History(nil), history...)),
	}}
	fetcher := &fakeFetcher{}
	updater := &spyUpdater{}
	service := NewService(db, updater, fetcher, time.Minute, 5*time.Minute, noopLogger{},
		fakeResolver{ipv4: dns}, func() time.Time { return now }, noopHealthchecks{})
	return serviceFixture{service: service, db: db, fetcher: fetcher, updater: updater, history: history}
}

func Test_Service_updateNecessary_privateIPFailureThenRecovery(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version         ipversion.IPVersion
		recovered       string
		wantUpdateCalls []string
	}{
		"ipv4 or ipv6 changed":   {ipversion.IP4or6, "192.168.50.160", []string{"192.168.50.160"}},
		"ipv4 changed":           {ipversion.IP4, "192.168.50.160", []string{"192.168.50.160"}},
		"ipv4 or ipv6 unchanged": {ipversion.IP4or6, "192.168.50.159", []string{}},
		"ipv4 unchanged":         {ipversion.IP4, "192.168.50.159", []string{}},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newServiceFixture(testCase.version)

			// Cycle 1: discovery fails on all three tries.
			f.fetcher.err = errNoInterfaceAddress
			errs := f.service.updateNecessary(t.Context())

			require.NotEmpty(t, errs)
			assert.Equal(t, 3, f.fetcher.calls)
			assert.Empty(t, f.updater.ips)
			assert.Equal(t, f.history, f.db.records[0].History)
			assert.Equal(t, constants.FAIL, f.db.records[0].Status)
			assert.Equal(t, "public IP address not found", f.db.records[0].Message)

			// Cycle 2: the interface is back.
			f.fetcher.calls = 0
			f.fetcher.err = nil
			f.fetcher.ip = netip.MustParseAddr(testCase.recovered)
			errs = f.service.updateNecessary(t.Context())

			assert.Empty(t, errs)
			assert.Equal(t, 1, f.fetcher.calls)
			gotUpdates := make([]string, 0, len(f.updater.ips))
			for _, ip := range f.updater.ips {
				gotUpdates = append(gotUpdates, ip.String())
			}
			assert.Equal(t, testCase.wantUpdateCalls, gotUpdates)
			assert.Equal(t, f.history, f.db.records[0].History)
			if len(testCase.wantUpdateCalls) == 0 {
				// Known, deferred limitation: when the recovered address already matches DNS
				// nothing resets the status set by the first failed cycle. The spec must change
				// together with this assertion if status repair is implemented.
				assert.Equal(t, constants.FAIL, f.db.records[0].Status)
			}
		})
	}
}

func Test_Service_updateNecessary_privateIPv6Unsupported(t *testing.T) {
	t.Parallel()

	f := newServiceFixture(ipversion.IP6)
	f.fetcher.err = privateip.ErrIPv6NotSupported

	errs := f.service.updateNecessary(t.Context())

	require.NotEmpty(t, errs)
	require.ErrorIs(t, errs[0], privateip.ErrIPv6NotSupported)
	assert.Equal(t, 3, f.fetcher.calls)
	assert.Empty(t, f.updater.ips)
	assert.Equal(t, f.history, f.db.records[0].History)
}
