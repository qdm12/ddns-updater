package health

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// ttlProviderMock wraps MockProvider with a TTL method for the
// ttlProvider interface.
type ttlProviderMock struct {
	*MockProvider
	ttl *uint32
}

func (m *ttlProviderMock) TTL() (ttl *uint32) {
	return m.ttl
}

type testRecord struct {
	status    models.Status
	proxied   bool
	history   []models.HistoryEvent
	ttlMethod bool
	ttl       *uint32
	lookupIPs []net.IP
	lookupErr error
}

func Test_isHealthy(t *testing.T) {
	t.Parallel()

	const hostnameSuffix = "example.com"
	oldIPv4 := netip.MustParseAddr("1.2.3.4")
	newIPv4 := netip.MustParseAddr("5.6.7.8")
	oldIPv6 := netip.MustParseAddr("2001:db8::1")
	newIPv6 := netip.MustParseAddr("2001:db8::2")
	lastUpdate := time.Date(2025, time.May, 1, 12, 0, 0, 0, time.UTC)
	const ttlSeconds = 300
	ttlValue := uint32(ttlSeconds)
	lookupError := errors.New("lookup error")

	currentHistory := []models.HistoryEvent{
		{IP: oldIPv4, Time: lastUpdate.Add(-24 * time.Hour)},
		{IP: newIPv4, Time: lastUpdate},
	}

	testCases := map[string]struct {
		records     []testRecord
		now         time.Time
		wantErr     error
		wantErrPart string
	}{
		"no_records": {},
		"record_update_failed": {
			records: []testRecord{
				{status: constants.FAIL, history: []models.HistoryEvent{{IP: newIPv4, Time: lastUpdate}}},
			},
			wantErr:     ErrRecordUpdateFailed,
			wantErrPart: "record0.example.com",
		},
		"proxied_record_skipped": {
			records: []testRecord{
				{proxied: true, history: []models.HistoryEvent{{IP: newIPv4, Time: lastUpdate}}},
			},
		},
		"record_ip_not_set": {
			records: []testRecord{
				{status: constants.SUCCESS},
			},
			wantErr:     ErrRecordIPNotSet,
			wantErrPart: "record0.example.com",
		},
		"lookup_error": {
			records: []testRecord{
				{status: constants.SUCCESS, history: currentHistory, lookupErr: lookupError},
			},
			wantErr: lookupError,
		},
		"lookup_current_ipv4_found": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					lookupIPs: []net.IP{net.ParseIP(newIPv4.String())},
				},
			},
		},
		"lookup_current_ipv6_found": {
			records: []testRecord{
				{
					status: constants.SUCCESS,
					history: []models.HistoryEvent{
						{IP: oldIPv6, Time: lastUpdate.Add(-24 * time.Hour)},
						{IP: newIPv6, Time: lastUpdate},
					},
					lookupIPs: []net.IP{net.ParseIP(newIPv6.String())},
				},
			},
		},
		"lookup_no_match_no_ttl": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					lookupIPs: []net.IP{net.ParseIP(oldIPv4.String())},
				},
			},
			wantErr:     ErrLookupMismatch,
			wantErrPart: "1.2.3.4 instead of 5.6.7.8",
		},
		"lookup_previous_ip_within_ttl": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					ttlMethod: true,
					ttl:       &ttlValue,
					lookupIPs: []net.IP{net.ParseIP(oldIPv4.String())},
				},
			},
			now: lastUpdate.Add(2 * time.Minute),
		},
		"lookup_previous_ip_at_ttl_boundary": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					ttlMethod: true,
					ttl:       &ttlValue,
					lookupIPs: []net.IP{net.ParseIP(oldIPv4.String())},
				},
			},
			now:         lastUpdate.Add(ttlSeconds * time.Second),
			wantErr:     ErrLookupMismatch,
			wantErrPart: "1.2.3.4 instead of 5.6.7.8",
		},
		"lookup_previous_ip_outside_ttl": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					ttlMethod: true,
					ttl:       &ttlValue,
					lookupIPs: []net.IP{net.ParseIP(oldIPv4.String())},
				},
			},
			now:         lastUpdate.Add(2 * time.Hour),
			wantErr:     ErrLookupMismatch,
			wantErrPart: "1.2.3.4 instead of 5.6.7.8",
		},
		"lookup_previous_ip_ttl_unknown": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					ttlMethod: true,
					lookupIPs: []net.IP{net.ParseIP(oldIPv4.String())},
				},
			},
			now:         lastUpdate.Add(2 * time.Minute),
			wantErr:     ErrLookupMismatch,
			wantErrPart: "1.2.3.4 instead of 5.6.7.8",
		},
		"lookup_previous_and_current_ip_found": {
			records: []testRecord{
				{
					status:  constants.SUCCESS,
					history: currentHistory,
					lookupIPs: []net.IP{
						net.ParseIP(oldIPv4.String()),
						net.ParseIP(newIPv4.String()),
					},
				},
			},
		},
		"multiple_records_second_failed": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					lookupIPs: []net.IP{net.ParseIP(newIPv4.String())},
				},
				{
					status:  constants.FAIL,
					history: []models.HistoryEvent{{IP: newIPv4, Time: lastUpdate}},
				},
			},
			wantErr:     ErrRecordUpdateFailed,
			wantErrPart: "record1.example.com",
		},
		"multiple_records_healthy": {
			records: []testRecord{
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					lookupIPs: []net.IP{net.ParseIP(newIPv4.String())},
				},
				{
					status:    constants.SUCCESS,
					history:   currentHistory,
					lookupIPs: []net.IP{net.ParseIP(newIPv4.String())},
				},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			ctx := t.Context()

			dbMock := NewMockAllSelecter(ctrl)
			resolverMock := NewMockLookupIPer(ctrl)

			healthRecords := make([]records.Record, len(testCase.records))
			for i := range testCase.records {
				recordCase := &testCase.records[i]

				providerMock := NewMockProvider(ctrl)
				var providerValue provider.Provider = providerMock
				if recordCase.ttlMethod {
					providerValue = &ttlProviderMock{
						MockProvider: providerMock,
						ttl:          recordCase.ttl,
					}
				}

				healthRecords[i] = records.Record{
					Provider: providerValue,
					History:  recordCase.history,
					Status:   recordCase.status,
				}

				hostname := fmt.Sprintf("record%d.%s", i, hostnameSuffix)

				switch {
				case recordCase.status == constants.FAIL:
					providerMock.EXPECT().String().Return(hostname)
				case recordCase.proxied:
					providerMock.EXPECT().Proxied().Return(true)
				default:
					providerMock.EXPECT().Proxied().Return(false)
					providerMock.EXPECT().BuildDomainName().Return(hostname)
					if models.History(recordCase.history).GetCurrentIP().IsValid() {
						resolverMock.EXPECT().LookupIP(ctx, "ip", hostname).
							Return(recordCase.lookupIPs, recordCase.lookupErr)
					}
				}
			}

			dbMock.EXPECT().SelectAll().Return(healthRecords)

			err := isHealthy(ctx, dbMock, resolverMock, testCase.now)
			if testCase.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, testCase.wantErr)
				if testCase.wantErrPart != "" {
					assert.ErrorContains(t, err, testCase.wantErrPart)
				}
			}
		})
	}
}

func Test_MakeIsHealthy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	ctx := t.Context()

	const hostname = "record0.example.com"
	oldIPv4 := netip.MustParseAddr("1.2.3.4")
	newIPv4 := netip.MustParseAddr("5.6.7.8")
	lastUpdate := time.Date(2025, time.May, 1, 12, 0, 0, 0, time.UTC)
	ttlValue := uint32(300)
	now := lastUpdate.Add(2 * time.Minute)

	dbMock := NewMockAllSelecter(ctrl)
	resolverMock := NewMockLookupIPer(ctrl)

	providerMock := NewMockProvider(ctrl)
	providerMock.EXPECT().Proxied().Return(false)
	providerMock.EXPECT().BuildDomainName().Return(hostname)

	record := records.Record{
		Provider: &ttlProviderMock{MockProvider: providerMock, ttl: &ttlValue},
		History: []models.HistoryEvent{
			{IP: oldIPv4, Time: lastUpdate.Add(-24 * time.Hour)},
			{IP: newIPv4, Time: lastUpdate},
		},
		Status: constants.SUCCESS,
	}

	dbMock.EXPECT().SelectAll().Return([]records.Record{record})
	resolverMock.EXPECT().LookupIP(ctx, "ip", hostname).
		Return([]net.IP{net.ParseIP(oldIPv4.String())}, nil)

	healthcheck := MakeIsHealthy(dbMock, resolverMock, func() time.Time { return now })
	assert.NoError(t, healthcheck(ctx))
}
