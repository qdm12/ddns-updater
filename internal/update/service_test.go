package update

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Test_Service_updateNecessary checks the statuses set by an update cycle,
// and in particular that a record in the public IP unknown state is
// re-evaluated at each cycle, and does not generate a database update while
// its public IP address still cannot be found.
func Test_Service_updateNecessary(t *testing.T) {
	t.Parallel()

	const (
		hostname = "ddns.example.com"
		period   = 10 * time.Minute
		cooldown = time.Duration(0)
		// fetchTries mirrors the constant of tryAndRepeatGettingIP.
		fetchTries = 3
		ipv4Kind   = "ipv4"
	)
	cycleTime := time.Date(2026, time.October, 30, 3, 25, 34, 0, time.UTC)
	publicIPv4 := netip.MustParseAddr("203.0.113.45")
	staleIPv4 := netip.MustParseAddr("198.51.100.5")
	historyTime := cycleTime.Add(-time.Hour)
	fetchError := errors.New("network is unreachable")
	fetchErrorMessage := fmt.Sprintf(
		"obtaining %s address: after %d tries, errors were: %s",
		ipv4Kind, fetchTries, fetchError.Error()+", "+fetchError.Error()+", "+fetchError.Error())

	testCases := map[string]struct {
		// initialState is the record held by the database before the cycle.
		initialState recordState
		// publicIPv4 is the fetched public IP address. An invalid address
		// makes the public IP fetch fail.
		publicIPv4 netip.Addr
		// wantState is the record written back to the database, nil when the
		// cycle must not write anything.
		wantState *recordState
		// wantUpdateIP is the IP address the record must be updated to, an
		// invalid address meaning no update must be made.
		wantUpdateIP netip.Addr
		wantErrs     []string
	}{
		"public_ip_not_found_and_never_updated": {
			initialState: recordState{
				status: constants.UNSET,
			},
			wantState: &recordState{
				status: constants.PUBLICIPUNKNOWN,
				time:   cycleTime,
			},
			wantErrs: []string{fetchErrorMessage},
		},
		"public_ip_not_found_and_already_unknown": {
			initialState: recordState{
				status: constants.PUBLICIPUNKNOWN,
				time:   cycleTime.Add(-period),
			},
			wantErrs: []string{fetchErrorMessage},
		},
		"public_ip_found_and_up_to_date": {
			initialState: recordState{
				status: constants.UNSET,
				history: []models.HistoryEvent{
					{IP: publicIPv4, Time: historyTime},
				},
			},
			publicIPv4: publicIPv4,
			wantState: &recordState{
				status: constants.UPTODATE,
				time:   cycleTime,
				history: []models.HistoryEvent{
					{IP: publicIPv4, Time: historyTime},
				},
			},
		},
		"public_ip_found_and_update_required": {
			initialState: recordState{
				status: constants.UNSET,
				history: []models.HistoryEvent{
					{IP: staleIPv4, Time: historyTime},
				},
			},
			publicIPv4:   publicIPv4,
			wantUpdateIP: publicIPv4,
		},
		"update_failure_not_re_evaluated": {
			initialState: recordState{
				status:  constants.FAIL,
				message: "HTTP error 500",
				time:    historyTime,
				history: []models.HistoryEvent{
					{IP: publicIPv4, Time: historyTime},
				},
			},
			publicIPv4: publicIPv4,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ctx := t.Context()

			fetchFailed := !testCase.publicIPv4.IsValid()
			updateRequired := testCase.wantUpdateIP.IsValid()
			initialState := testCase.initialState
			initialStatus := initialState.status == constants.UNSET ||
				initialState.status == constants.PUBLICIPUNKNOWN

			dbMock := NewMockDatabase(ctrl)
			loggerMock := NewMockLogger(ctrl)
			updaterMock := NewMockUpdaterInterface(ctrl)
			fetcherMock := NewMockPublicIPFetcher(ctrl)
			resolverMock := NewMockLookupIPer(ctrl)
			hioMock := NewMockHealthchecksIOClient(ctrl)
			providerMock := NewMockProvider(ctrl)

			initialRecord := initialState.record(providerMock)

			dbMock.EXPECT().SelectAll().Return([]records.Record{initialRecord})
			providerMock.EXPECT().IPVersion().
				Times(ipVersionCalls(initialStatus, updateRequired)).
				Return(ipversion.IP4)
			providerMock.EXPECT().BuildDomainName().Return(hostname)
			loggerMock.EXPECT().Debug(
				"configured to fetch IP: v4 or v6: false, v4: true, v6: false")

			publicIPv4 := testCase.publicIPv4
			if fetchFailed {
				fetcherMock.EXPECT().IP4(ctx).Times(fetchTries).
					Return(netip.Addr{}, fetchError)
				for try := 1; try <= fetchTries; try++ {
					loggerMock.EXPECT().Debug(fmt.Sprintf(
						"obtaining %s address: try %d of %d failed: %s",
						ipv4Kind, try, fetchTries, fetchError.Error()))
				}
			} else {
				fetcherMock.EXPECT().IP4(ctx).Return(publicIPv4, nil)
			}
			loggerMock.EXPECT().Debug(fmt.Sprintf(
				"your public IP address are: v4 or v6: %s, v4: %s, v6: %s",
				netip.Addr{}, publicIPv4, netip.Addr{}))
			if fetchFailed {
				loggerMock.EXPECT().Error(fetchErrorMessage)
				loggerMock.EXPECT().Warn(fmt.Sprintf(
					"Skipping update for %s because %s address was not found",
					hostname, ipv4Kind))
			} else {
				providerMock.EXPECT().Proxied().Return(true)
				lastIP := models.History(initialState.history).GetCurrentIP()
				if updateRequired {
					loggerMock.EXPECT().Info(fmt.Sprintf(
						"Last %s address stored for %s is %s and your %s address is %s",
						ipv4Kind, hostname, lastIP, ipv4Kind, publicIPv4))
				} else {
					loggerMock.EXPECT().Debug(fmt.Sprintf(
						"Last %s address stored for %s is %s and your %s address"+
							" is %s, skipping update",
						ipv4Kind, hostname, lastIP, ipv4Kind, publicIPv4))
				}
			}

			if testCase.wantState != nil {
				dbMock.EXPECT().Select(uint(0)).Return(initialRecord, nil)
				dbMock.EXPECT().Update(uint(0), recordMatcher{want: *testCase.wantState}).Return(nil)
			}
			if updateRequired {
				providerMock.EXPECT().String().Return(hostname)
				loggerMock.EXPECT().Info(fmt.Sprintf("Updating record %s to use %s",
					hostname, publicIPv4))
				updaterMock.EXPECT().Update(ctx, uint(0), publicIPv4).Return(nil)
			}

			healthchecksIOState := healthchecksio.Ok
			if len(testCase.wantErrs) > 0 {
				healthchecksIOState = healthchecksio.Fail
			}
			hioMock.EXPECT().Ping(ctx, healthchecksIOState).Return(nil)

			service := NewService(dbMock, updaterMock, fetcherMock, period, cooldown,
				loggerMock, resolverMock, func() time.Time { return cycleTime }, hioMock)

			errs := service.updateNecessary(ctx)

			assert.Len(t, errs, len(testCase.wantErrs))
			for i, wantErrMessage := range testCase.wantErrs {
				assert.ErrorContains(t, errs[i], wantErrMessage)
			}
		})
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
	return fmt.Sprintf("record with status %q, message %q, time %s and %d history event(s)",
		string(m.want.status), m.want.message, m.want.time, len(m.want.history))
}

// ipVersionCalls returns the number of times the provider IPVersion method is
// called during an update cycle: once to know which public IP addresses to
// fetch, once to decide whether the record must be updated, plus once per
// record processing either by the initial status setting or by the update.
func ipVersionCalls(initialStatus, updateRequired bool) (calls int) {
	const baseCalls = 2
	calls = baseCalls
	if initialStatus && !updateRequired {
		calls++
	}
	if updateRequired {
		calls++
	}
	return calls
}
