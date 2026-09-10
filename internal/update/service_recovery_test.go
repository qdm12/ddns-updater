package update

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/provider/providers/example"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceUpdateNecessaryRecoversFailedRecordAfterDNSMatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	publicIP := netip.MustParseAddr("192.0.2.10")
	previousIP := netip.MustParseAddr("192.0.2.9")
	oldTime := now.Add(-2 * time.Hour)

	testCases := map[string]struct {
		history             models.History
		irrelevantLookupErr error
		expectedHistory     models.History
	}{
		"empty history": {
			expectedHistory: models.History{{IP: publicIP, Time: now}},
		},
		"stale history is corrected": {
			history: models.History{{IP: previousIP, Time: oldTime}},
			expectedHistory: models.History{
				{IP: previousIP, Time: oldTime},
				{IP: publicIP, Time: now},
			},
		},
		"matching history is not duplicated": {
			history:         models.History{{IP: publicIP, Time: oldTime}},
			expectedHistory: models.History{{IP: publicIP, Time: oldTime}},
		},
		"matching IPv4 despite IPv6 lookup failure": {
			irrelevantLookupErr: errors.New("IPv6 DNS unavailable"),
			expectedHistory:     models.History{{IP: publicIP, Time: now}},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			record := newRecoveryTestRecord(t, testCase.history)
			db := &recoveryTestDatabase{records: []records.Record{record}}
			updater := &recoveryTestUpdater{}
			service := newRecoveryTestService(db, updater,
				recoveryTestIPFetcher{ip4: publicIP},
				recoveryTestResolver{
					ip4:     []netip.Addr{publicIP},
					ipv6Err: testCase.irrelevantLookupErr,
				}, now, 0)

			errs := service.updateNecessary(context.Background())

			require.Empty(t, errs)
			updatedRecord, err := db.Select(0)
			require.NoError(t, err)
			assert.Equal(t, constants.UPTODATE, updatedRecord.Status)
			assert.Empty(t, updatedRecord.Message)
			assert.Equal(t, now, updatedRecord.Time)
			assert.Equal(t, testCase.expectedHistory, updatedRecord.History)
			assert.Zero(t, updater.calls.Load())
		})
	}
}

func TestServiceUpdateNecessaryRecoversFailedIPv6RecordWithSuffix(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	publicIP := netip.MustParseAddr("2001:db8:1:2:aaaa:bbbb:cccc:dddd")
	ipv6Suffix := netip.MustParsePrefix("::1234/64")
	recordIP := netip.MustParseAddr("2001:db8:1:2::1234")
	record := newRecoveryTestRecordWithIPSettings(t, nil, ipversion.IP6, ipv6Suffix)
	db := &recoveryTestDatabase{records: []records.Record{record}}
	updater := &recoveryTestUpdater{}
	service := newRecoveryTestService(db, updater,
		recoveryTestIPFetcher{ip6: publicIP},
		recoveryTestResolver{ip6: []netip.Addr{recordIP}}, now, 0)

	errs := service.updateNecessary(context.Background())

	require.Empty(t, errs)
	updatedRecord, err := db.Select(0)
	require.NoError(t, err)
	assert.Equal(t, constants.UPTODATE, updatedRecord.Status)
	assert.Empty(t, updatedRecord.Message)
	assert.Equal(t, models.History{{IP: recordIP, Time: now}}, updatedRecord.History)
	assert.Zero(t, updater.calls.Load())
}

func TestServiceUpdateNecessaryDoesNotHideUnverifiedFailure(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	publicIP := netip.MustParseAddr("192.0.2.10")
	lookupErr := errors.New("DNS unavailable")

	testCases := map[string]struct {
		ipFetcher      recoveryTestIPFetcher
		resolver       recoveryTestResolver
		proxied        bool
		cooldown       time.Duration
		history        models.History
		lastBan        *time.Time
		updaterErr     error
		expectedCalls  int32
		expectedErrors int
	}{
		"public IP missing": {},
		"DNS lookup failed": {
			ipFetcher:      recoveryTestIPFetcher{ip4: publicIP},
			resolver:       recoveryTestResolver{err: lookupErr},
			updaterErr:     lookupErr,
			expectedCalls:  1,
			expectedErrors: 1,
		},
		"cooldown skipped DNS verification": {
			ipFetcher: recoveryTestIPFetcher{ip4: publicIP},
			cooldown:  time.Hour,
			history: models.History{{
				IP:   publicIP,
				Time: now.Add(-time.Minute),
			}},
		},
		"ban skipped DNS verification": {
			ipFetcher: recoveryTestIPFetcher{ip4: publicIP},
			lastBan:   new(now.Add(-time.Minute)),
		},
		"proxied record only matched local history": {
			ipFetcher: recoveryTestIPFetcher{ip4: publicIP},
			proxied:   true,
			history: models.History{{
				IP:   publicIP,
				Time: now.Add(-2 * time.Hour),
			}},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			record := newRecoveryTestRecord(t, testCase.history)
			if testCase.proxied {
				record.Provider = recoveryTestProxiedProvider{Provider: record.Provider}
			}
			record.LastBan = testCase.lastBan
			db := &recoveryTestDatabase{records: []records.Record{record}}
			updater := &recoveryTestUpdater{err: testCase.updaterErr}
			service := newRecoveryTestService(db, updater, testCase.ipFetcher,
				testCase.resolver, now, testCase.cooldown)

			errs := service.updateNecessary(context.Background())

			assert.Len(t, errs, testCase.expectedErrors)
			updatedRecord, err := db.Select(0)
			require.NoError(t, err)
			assert.Equal(t, constants.FAIL, updatedRecord.Status)
			assert.Equal(t, "stale provider failure", updatedRecord.Message)
			assert.Equal(t, testCase.history, updatedRecord.History)
			assert.Equal(t, testCase.expectedCalls, updater.calls.Load())
		})
	}
}

func newRecoveryTestRecord(t *testing.T, history models.History) records.Record {
	t.Helper()
	return newRecoveryTestRecordWithIPSettings(t, history, ipversion.IP4, netip.Prefix{})
}

func newRecoveryTestRecordWithIPSettings(t *testing.T, history models.History,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix,
) records.Record {
	t.Helper()
	provider, err := example.New(json.RawMessage(`{"username":"user","password":"password"}`),
		"example.com", "record", ipVersion, ipv6Suffix)
	require.NoError(t, err)
	return records.Record{
		Provider: provider,
		History:  history,
		Status:   constants.FAIL,
		Message:  "stale provider failure",
	}
}

func newRecoveryTestService(db Database, updater UpdaterInterface, ipFetcher PublicIPFetcher,
	resolver LookupIPer, now time.Time, cooldown time.Duration,
) *Service {
	return NewService(db, updater, ipFetcher, time.Hour, cooldown, recoveryTestLogger{}, resolver,
		func() time.Time { return now }, recoveryTestHealthchecksClient{})
}

type recoveryTestDatabase struct {
	mu      sync.Mutex
	records []records.Record
}

func (d *recoveryTestDatabase) Select(recordID uint) (records.Record, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.records[recordID], nil
}

func (d *recoveryTestDatabase) SelectAll() []records.Record {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]records.Record(nil), d.records...)
}

func (d *recoveryTestDatabase) Update(recordID uint, record records.Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records[recordID] = record
	return nil
}

type recoveryTestUpdater struct {
	calls atomic.Int32
	err   error
}

func (u *recoveryTestUpdater) Update(context.Context, uint, netip.Addr) error {
	u.calls.Add(1)
	return u.err
}

type recoveryTestIPFetcher struct {
	ip4 netip.Addr
	ip6 netip.Addr
}

func (f recoveryTestIPFetcher) IP(context.Context) (netip.Addr, error)  { return netip.Addr{}, nil }
func (f recoveryTestIPFetcher) IP4(context.Context) (netip.Addr, error) { return f.ip4, nil }
func (f recoveryTestIPFetcher) IP6(context.Context) (netip.Addr, error) { return f.ip6, nil }

type recoveryTestResolver struct {
	ip4     []netip.Addr
	ip6     []netip.Addr
	err     error
	ipv6Err error
}

func (r recoveryTestResolver) LookupNetIP(_ context.Context, network, _ string) ([]netip.Addr, error) {
	if r.err != nil {
		return nil, r.err
	}
	if network == "ip4" {
		return r.ip4, nil
	}
	if r.ipv6Err != nil {
		return nil, r.ipv6Err
	}
	if len(r.ip6) > 0 {
		return r.ip6, nil
	}
	return nil, errors.New("no such host")
}

type recoveryTestProxiedProvider struct {
	provider.Provider
}

func (recoveryTestProxiedProvider) Proxied() bool { return true }

type recoveryTestLogger struct{}

func (recoveryTestLogger) Debug(string) {}
func (recoveryTestLogger) Info(string)  {}
func (recoveryTestLogger) Warn(string)  {}
func (recoveryTestLogger) Error(string) {}

type recoveryTestHealthchecksClient struct{}

func (recoveryTestHealthchecksClient) Ping(context.Context, healthchecksio.State) error { return nil }
