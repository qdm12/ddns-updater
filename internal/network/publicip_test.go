package network

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetPublicIP(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		IPVersion   models.IPVersion
		mockContent []byte
		mockStatus  int
		mockErr     error
		ip          net.IP
		err         error
	}{
		"network error": {
			IPVersion: constants.IPv4,
			mockErr:   fmt.Errorf("error"),
			err:       fmt.Errorf("cannot get public ipv4 address from https://getmyip.com: error"),
		},
		"bad status": {
			IPVersion:  constants.IPv4,
			mockStatus: http.StatusUnauthorized,
			err:        fmt.Errorf("cannot get public ipv4 address from https://getmyip.com: HTTP status code 401"),
		},
		"no IPs in content": {
			IPVersion:   constants.IPv4,
			mockContent: []byte(""),
			mockStatus:  http.StatusOK,
			err:         fmt.Errorf("no public ipv4 address found at https://getmyip.com"),
		},
		"multiple IPs in content": {
			IPVersion:   constants.IPv4,
			mockContent: []byte("10.10.10.10  50.50.50.50"),
			mockStatus:  http.StatusOK,
			err:         fmt.Errorf("multiple public ipv4 addresses found at https://getmyip.com: 10.10.10.10 50.50.50.50"),
		},
		"single IP in content": {
			IPVersion:   constants.IPv4,
			mockContent: []byte("10.10.10.10"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{10, 10, 10, 10},
		},
		"single IPv6 in content": {
			IPVersion:   constants.IPv6,
			mockContent: []byte("::fe"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xfe},
		},
	}
	const URL = "https://getmyip.com"
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := &mocks.Client{}
			client.On("GetContent", URL).Return(tc.mockContent, tc.mockStatus, tc.mockErr).Once()
			ip, err := GetPublicIP(client, URL, tc.IPVersion)
			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			fmt.Printf("%#v\n", ip)
			assert.True(t, tc.ip.Equal(ip))
			client.AssertExpectations(t)
		})
	}
}
