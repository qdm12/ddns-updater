package network

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/qdm12/golibs/network/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetPublicIP(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		mockContent []byte
		mockStatus  int
		mockErr     error
		ip          net.IP
		err         error
	}{
		"network error": {
			mockErr: fmt.Errorf("error"),
			err:     fmt.Errorf("cannot get public IP address from https://getmyip.com: error"),
		},
		"bad status": {
			mockStatus: http.StatusUnauthorized,
			err:        fmt.Errorf("cannot get public IP address from https://getmyip.com: HTTP status code 401"),
		},
		"no IPs in content": {
			mockContent: []byte(""),
			mockStatus:  http.StatusOK,
			err:         fmt.Errorf("no public IPv4 address found at https://getmyip.com"),
		},
		"multiple IPs in content": {
			mockContent: []byte("10.10.10.10  50.50.50.50"),
			mockStatus:  http.StatusOK,
			err:         fmt.Errorf("2 public IPv4 addresses found at https://getmyip.com instead of 1"),
		},
		"single IP in content": {
			mockContent: []byte("10.10.10.10"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{10, 10, 10, 10},
		},
	}
	const URL = "https://getmyip.com"
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := &mocks.Client{}
			client.On("GetContent", URL).Return(tc.mockContent, tc.mockStatus, tc.mockErr).Once()
			ip, err := GetPublicIP(client, URL)
			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, tc.ip.Equal(ip))
			client.AssertExpectations(t)
		})
	}
}
