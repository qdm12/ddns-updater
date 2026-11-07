package update

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateProvider struct {
	ipv4Provider
	err error
}

func (p updateProvider) Update(_ context.Context, _ *http.Client, ip netip.Addr) (netip.Addr, error) {
	return ip, p.err
}

type noopShoutrrr struct{}

func (noopShoutrrr) Notify(string) {}

func Test_Updater_Update(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 30, 3, 25, 34, 0, time.UTC)
	ip := netip.MustParseAddr("1.2.3.4")

	testCases := map[string]struct {
		providerErr error
		status      models.Status
		message     string
	}{
		"success": {
			status:  constants.SUCCESS,
			message: "changed to 1.2.3.4",
		},
		"failure": {
			providerErr: errors.New("http 500"),
			status:      constants.FAIL,
			message:     "http 500",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := &memoryDatabase{records: []librecords.Record{{
				Provider: updateProvider{err: testCase.providerErr},
				Status:   constants.SUCCESS,
				Message:  "changed to 5.6.7.8",
			}}}
			now := start
			timeNow := func() time.Time {
				now = now.Add(time.Second)
				return now
			}
			updater := NewUpdater(db, nil, noopShoutrrr{}, nil, timeNow, false)

			err := updater.Update(t.Context(), 0, ip)

			if testCase.providerErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, testCase.providerErr)
			}
			require.Len(t, db.updates, 2)
			updating, finished := db.updates[0], db.updates[1]
			assert.Equal(t, constants.UPDATING, updating.Status)
			assert.Empty(t, updating.Message, "previous message must be cleared")
			assert.Equal(t, start.Add(time.Second), updating.Time)
			assert.Equal(t, testCase.status, finished.Status)
			assert.Equal(t, testCase.message, finished.Message)
			assert.Equal(t, start.Add(2*time.Second), finished.Time)
		})
	}
}
