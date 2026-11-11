package update

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/provider/providers/opendns"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:maintidx
func Test_Service_updateNecessary(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		// initialState is the record held by the database before the cycle.
		initialState recordState
		// publicIPv4 is the address returned by the public IP address fetcher.
		// An invalid address makes the fetch fail.
		publicIPv4 netip.Addr
		wantErrs   []string
		// configure sets the mock expectations of the cycle, written out for
		// each test case so the subtest body has no logic.
		configure func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record)
	}{
		// The network is not up yet, for example just after a container start,
		// so the record goes to the public IP unknown state.
		"public_ip_not_found_and_never_updated": {
			initialState: recordState{
				status: constants.UNSET,
			},
			wantErrs: []string{
				"obtaining ipv4 address: after 3 tries, errors were: network is unreachable, " +
					"network is unreachable, network is unreachable",
			},
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.fetcher.EXPECT().IP4(ctx).Times(3).
					Return(netip.Addr{}, errors.New("network is unreachable"))
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 1 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 2 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 3 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Error("obtaining ipv4 address: after 3 tries, errors were: " +
					"network is unreachable, network is unreachable, network is unreachable")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: invalid IP, v6: invalid IP")
				mocks.logger.EXPECT().Warn("Skipping update for ddns.example.com because ipv4 address was not found")
				mocks.db.EXPECT().Select(uint(0)).Return(initialRecord, nil)
				mocks.db.EXPECT().Update(uint(0), recordMatcher{want: recordState{
					status: constants.PUBLICIPUNKNOWN,
					time:   time.Date(2026, time.October, 30, 3, 25, 34, 0, time.UTC),
				}}).Return(nil)
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Fail).Return(nil)
			},
		},
		// The public IP unknown state is re-evaluated at every cycle. Here the
		// public IP address is still not found and nothing is written.
		"public_ip_not_found_and_already_unknown": {
			initialState: recordState{
				status: constants.PUBLICIPUNKNOWN,
				time:   time.Date(2026, time.October, 30, 3, 15, 34, 0, time.UTC),
			},
			wantErrs: []string{
				"obtaining ipv4 address: after 3 tries, errors were: network is unreachable, " +
					"network is unreachable, network is unreachable",
			},
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.fetcher.EXPECT().IP4(ctx).Times(3).
					Return(netip.Addr{}, errors.New("network is unreachable"))
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 1 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 2 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 3 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Error("obtaining ipv4 address: after 3 tries, errors were: " +
					"network is unreachable, network is unreachable, network is unreachable")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: invalid IP, v6: invalid IP")
				mocks.logger.EXPECT().Warn("Skipping update for ddns.example.com because ipv4 address was not found")
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Fail).Return(nil)
			},
		},
		// The network goes down after a record was up to date, so the record can
		// no longer be verified and goes the public IP unknown state.
		"public_ip_not_found_after_being_up_to_date": {
			initialState: recordState{
				status:  constants.UPTODATE,
				message: "no IP change for 1 hour",
				time:    time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC),
				history: []models.HistoryEvent{
					{IP: netip.MustParseAddr("203.0.113.45"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
				},
			},
			wantErrs: []string{
				"obtaining ipv4 address: after 3 tries, errors were: network is unreachable, " +
					"network is unreachable, network is unreachable",
			},
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.fetcher.EXPECT().IP4(ctx).Times(3).
					Return(netip.Addr{}, errors.New("network is unreachable"))
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 1 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 2 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 3 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Error("obtaining ipv4 address: after 3 tries, errors were: " +
					"network is unreachable, network is unreachable, network is unreachable")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: invalid IP, v6: invalid IP")
				mocks.logger.EXPECT().Warn("Skipping update for ddns.example.com because ipv4 address was not found")
				mocks.db.EXPECT().Select(uint(0)).Return(initialRecord, nil)
				mocks.db.EXPECT().Update(uint(0), recordMatcher{want: recordState{
					status: constants.PUBLICIPUNKNOWN,
					time:   time.Date(2026, time.October, 30, 3, 25, 34, 0, time.UTC),
					history: []models.HistoryEvent{
						{IP: netip.MustParseAddr("203.0.113.45"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
					},
				}}).Return(nil)
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Fail).Return(nil)
			},
		},
		// A failed update is kept, because its message tells why the record is
		// not up to date, whereas the public IP unknown state would not.
		"public_ip_not_found_after_update_failure": {
			initialState: recordState{
				status:  constants.FAIL,
				message: "HTTP error 500",
				time:    time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC),
			},
			wantErrs: []string{
				"obtaining ipv4 address: after 3 tries, errors were: network is unreachable, " +
					"network is unreachable, network is unreachable",
			},
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.fetcher.EXPECT().IP4(ctx).Times(3).
					Return(netip.Addr{}, errors.New("network is unreachable"))
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 1 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 2 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Debug("obtaining ipv4 address: try 3 of 3 failed: network is unreachable")
				mocks.logger.EXPECT().Error("obtaining ipv4 address: after 3 tries, errors were: " +
					"network is unreachable, network is unreachable, network is unreachable")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: invalid IP, v6: invalid IP")
				mocks.logger.EXPECT().Warn("Skipping update for ddns.example.com because ipv4 address was not found")
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Fail).Return(nil)
			},
		},
		"public_ip_found_and_up_to_date": {
			initialState: recordState{
				status: constants.UNSET,
				history: []models.HistoryEvent{
					{IP: netip.MustParseAddr("203.0.113.45"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
				},
			},
			publicIPv4: netip.MustParseAddr("203.0.113.45"),
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.provider.EXPECT().Proxied().Return(true)
				mocks.fetcher.EXPECT().IP4(ctx).Return(netip.MustParseAddr("203.0.113.45"), nil)
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: 203.0.113.45, v6: invalid IP")
				mocks.logger.EXPECT().Debug("Last ipv4 address stored for ddns.example.com is " +
					"203.0.113.45 and your ipv4 address is 203.0.113.45, skipping update")
				mocks.db.EXPECT().Select(uint(0)).Return(initialRecord, nil)
				mocks.db.EXPECT().Update(uint(0), recordMatcher{want: recordState{
					status: constants.UPTODATE,
					time:   time.Date(2026, time.October, 30, 3, 25, 34, 0, time.UTC),
					history: []models.HistoryEvent{
						{IP: netip.MustParseAddr("203.0.113.45"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
					},
				}}).Return(nil)
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Ok).Return(nil)
			},
		},
		"public_ip_found_and_update_required": {
			initialState: recordState{
				status: constants.UNSET,
				history: []models.HistoryEvent{
					{IP: netip.MustParseAddr("198.51.100.5"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
				},
			},
			publicIPv4: netip.MustParseAddr("203.0.113.45"),
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(4).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.provider.EXPECT().Proxied().Return(true)
				mocks.fetcher.EXPECT().IP4(ctx).Return(netip.MustParseAddr("203.0.113.45"), nil)
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: 203.0.113.45, v6: invalid IP")
				mocks.logger.EXPECT().Info("Last ipv4 address stored for ddns.example.com is " +
					"198.51.100.5 and your ipv4 address is 203.0.113.45")
				mocks.provider.EXPECT().String().Return("ddns.example.com")
				mocks.logger.EXPECT().Info("Updating record ddns.example.com to use 203.0.113.45")
				mocks.updater.EXPECT().Update(ctx, uint(0), netip.MustParseAddr("203.0.113.45")).Return(nil)
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Ok).Return(nil)
			},
		},
		// A failed update is not a transient state: the failure is kept until
		// the record is updated again.
		"update_failure_not_re_evaluated": {
			initialState: recordState{
				status:  constants.FAIL,
				message: "HTTP error 500",
				time:    time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC),
				history: []models.HistoryEvent{
					{IP: netip.MustParseAddr("203.0.113.45"), Time: time.Date(2026, time.October, 30, 2, 25, 34, 0, time.UTC)},
				},
			},
			publicIPv4: netip.MustParseAddr("203.0.113.45"),
			configure: func(ctx context.Context, mocks *serviceMocks, initialRecord records.Record) {
				mocks.db.EXPECT().SelectAll().Return([]records.Record{initialRecord})
				mocks.provider.EXPECT().IPVersion().Times(3).Return(ipversion.IP4)
				mocks.provider.EXPECT().BuildDomainName().Return("ddns.example.com")
				mocks.provider.EXPECT().Proxied().Return(true)
				mocks.fetcher.EXPECT().IP4(ctx).Return(netip.MustParseAddr("203.0.113.45"), nil)
				mocks.logger.EXPECT().Debug("configured to fetch IP: v4 or v6: false, v4: true, v6: false")
				mocks.logger.EXPECT().Debug("your public IP address are: v4 or v6: invalid IP, " +
					"v4: 203.0.113.45, v6: invalid IP")
				mocks.logger.EXPECT().Debug("Last ipv4 address stored for ddns.example.com is " +
					"203.0.113.45 and your ipv4 address is 203.0.113.45, skipping update")
				mocks.hio.EXPECT().Ping(ctx, healthchecksio.Ok).Return(nil)
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ctx := t.Context()

			mocks := newServiceMocks(ctrl)
			initialRecord := testCase.initialState.record(mocks.provider)
			testCase.configure(ctx, mocks, initialRecord)

			const period = 10 * time.Minute
			const cooldown = time.Duration(0)
			timeFunc := func() time.Time {
				return time.Date(2026, time.October, 30, 3, 25, 34, 0, time.UTC)
			}
			service := NewService(mocks.db, mocks.updater, mocks.fetcher, period, cooldown,
				mocks.logger, mocks.resolver, timeFunc, mocks.hio)

			errs := service.updateNecessary(ctx)

			assert.Len(t, errs, len(testCase.wantErrs))
			for i, wantErrMessage := range testCase.wantErrs {
				assert.ErrorContains(t, errs[i], wantErrMessage)
			}
		})
	}
}

// serviceMocks holds the generated mocks of a [Service].
type serviceMocks struct {
	db       *MockDatabase
	updater  *MockUpdaterInterface
	fetcher  *MockPublicIPFetcher
	logger   *MockLogger
	resolver *MockLookupIPer
	hio      *MockHealthchecksIOClient
	provider *MockProvider
}

func newServiceMocks(ctrl *gomock.Controller) *serviceMocks {
	return &serviceMocks{
		db:       NewMockDatabase(ctrl),
		updater:  NewMockUpdaterInterface(ctrl),
		fetcher:  NewMockPublicIPFetcher(ctrl),
		logger:   NewMockLogger(ctrl),
		resolver: NewMockLookupIPer(ctrl),
		hio:      NewMockHealthchecksIOClient(ctrl),
		provider: NewMockProvider(ctrl),
	}
}

type recordState struct {
	status  models.Status
	message string
	time    time.Time
	history []models.HistoryEvent
}

func (s recordState) record(provider provider.Provider) records.Record {
	return records.Record{
		Provider: provider,
		History:  s.history,
		Status:   s.status,
		Message:  s.message,
		Time:     s.time,
	}
}

// recordMatcher matches a [records.Record] on the fields set by an update
// cycle, since the record embeds a provider interface which cannot be
// compared.
type recordMatcher struct {
	want recordState
}

func (m recordMatcher) Matches(argument any) bool {
	record, ok := argument.(records.Record)
	if !ok {
		return false
	}
	return record.Status == m.want.status &&
		record.Message == m.want.message &&
		record.Time.Equal(m.want.time) &&
		slices.Equal(record.History, models.History(m.want.history))
}

func (m recordMatcher) String() string {
	return "record with status " + string(m.want.status) +
		", message " + m.want.message +
		", time " + m.want.time.String() +
		" and " + strconv.Itoa(len(m.want.history)) + " history event(s)"
}

// The IP address an OpenDNS record updates is the one of a network declared in
// the account, and not the one of a DNS record, the same as for proxied records.
// Therefore no DNS lookup is done and the last IP address sent to the provider
// decides if an update is needed.
func Test_Service_shouldUpdateRecord(t *testing.T) {
	t.Parallel()

	const (
		domain = "example.com"
		owner  = "sub"
	)

	testCases := map[string]struct {
		lastIP           netip.Addr
		publicIP         netip.Addr
		update           bool
		expectedInfoLog  string
		expectedDebugLog string
	}{
		"no_history_yet": {
			publicIP:        netip.MustParseAddr("203.0.113.45"),
			update:          true,
			expectedInfoLog: "Last ipv4 address stored for sub.example.com is invalid IP and your ipv4 address is 203.0.113.45",
		},
		"unchanged_ip": {
			lastIP:   netip.MustParseAddr("203.0.113.45"),
			publicIP: netip.MustParseAddr("203.0.113.45"),
			expectedDebugLog: "Last ipv4 address stored for sub.example.com is 203.0.113.45 and your ipv4 " +
				"address is 203.0.113.45, skipping update",
		},
		"changed_ip": {
			lastIP:   netip.MustParseAddr("203.0.113.44"),
			publicIP: netip.MustParseAddr("203.0.113.45"),
			update:   true,
			expectedInfoLog: "Last ipv4 address stored for sub.example.com is 203.0.113.44 and your ipv4 " +
				"address is 203.0.113.45",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			logger := NewMockLogger(ctrl)
			// No expectation on the resolver: the domain name of the record is
			// unrelated to the IP address updated, and any lookup makes this test
			// fail.
			resolver := NewMockLookupIPer(ctrl)

			data := json.RawMessage(`{"username": "username", "password": "password", "hostname": "Home"}`)
			provider, err := opendns.New(data, domain, owner, ipversion.IP4, netip.Prefix{})
			require.NoError(t, err)

			history := models.History(nil)
			if testCase.lastIP.IsValid() {
				const timeSinceSuccess = 24 * time.Hour
				history = models.History{models.HistoryEvent{
					IP:   testCase.lastIP,
					Time: time.Now().Add(-timeSinceSuccess),
				}}
			}
			record := records.Record{
				Provider: provider,
				History:  history,
			}

			s := &Service{
				cooldown: 0,
				resolver: resolver,
				logger:   logger,
				timeNow:  time.Now,
			}

			switch {
			case testCase.expectedInfoLog != "":
				logger.EXPECT().Info(testCase.expectedInfoLog)
			case testCase.expectedDebugLog != "":
				logger.EXPECT().Debug(testCase.expectedDebugLog)
			}

			publicIP := testCase.publicIP
			update := s.shouldUpdateRecord(t.Context(), record, publicIP, publicIP, netip.Addr{})

			assert.Equal(t, testCase.update, update)
		})
	}
}
