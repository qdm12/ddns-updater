package update

import (
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/providers/opendns"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
			record := librecords.Record{
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
