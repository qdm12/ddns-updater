package network

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network/mock_network"
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
			err:       fmt.Errorf("cannot get public ipv4 address: error"),
		},
		"bad status": {
			IPVersion:  constants.IPv4,
			mockStatus: http.StatusUnauthorized,
			err:        fmt.Errorf("cannot get public ipv4 address from https://getmyip.com: HTTP status code 401"),
		},
		"ipv4 address": {
			IPVersion:   constants.IPv4,
			mockContent: []byte("55.55.55.55"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{55, 55, 55, 55},
		},
		"ipv6 address": {
			IPVersion:   constants.IPv6,
			mockContent: []byte("ad07:e846:51ac:6cd0:0000:0000:0000:0000"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"ipv4 or ipv6 found ipv4": {
			IPVersion:   constants.IPv4OrIPv6,
			mockContent: []byte("55.55.55.55"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{55, 55, 55, 55},
		},
		"ipv4 or ipv6 found ipv6": {
			IPVersion:   constants.IPv4OrIPv6,
			mockContent: []byte("ad07:e846:51ac:6cd0:0000:0000:0000:0000"),
			mockStatus:  http.StatusOK,
			ip:          net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"ipv4 or ipv6 not found": {
			IPVersion:   constants.IPv4OrIPv6,
			mockContent: []byte("abc"),
			mockStatus:  http.StatusOK,
			err:         fmt.Errorf("no public ipv4 address found, no public ipv6 address found"),
		},
		"unsupported ip version": {
			IPVersion:  models.IPVersion("x"),
			mockStatus: http.StatusOK,
			err:        fmt.Errorf("ip version \"x\" not supported"),
		},
	}
	const URL = "https://getmyip.com"
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			client := mock_network.NewMockClient(mockCtrl)
			client.EXPECT().GetContent(URL).Return(tc.mockContent, tc.mockStatus, tc.mockErr).Times(1)
			ip, err := GetPublicIP(client, URL, tc.IPVersion)
			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}

func Test_searchIP(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		IPVersion models.IPVersion
		s         string
		ip        net.IP
		err       error
	}{
		"unsupported ip version": {
			IPVersion: constants.IPv4OrIPv6,
			err:       fmt.Errorf("ip version \"ipv4 or ipv6\" is not supported for regex search"),
		},
		"no content": {
			IPVersion: constants.IPv4,
			err:       fmt.Errorf("no public ipv4 address found"),
		},
		"single ipv4 address": {
			IPVersion: constants.IPv4,
			s:         "abcd 55.55.55.55 abcd",
			ip:        net.IP{55, 55, 55, 55},
		},
		"single ipv6 address": {
			IPVersion: constants.IPv6,
			s:         "abcd bd07:e846:51ac:6cd0:0000:0000:0000:0000 abcd",
			ip:        net.IP{0xbd, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"single private ipv4 address": {
			IPVersion: constants.IPv4,
			s:         "abcd 10.0.0.3 abcd",
			err:       fmt.Errorf("no public ipv4 address found"),
		},
		"single private ipv6 address": {
			IPVersion: constants.IPv6,
			s:         "abcd ::1 abcd",
			err:       fmt.Errorf("no public ipv6 address found"),
		},
		"2 ipv4 addresses": {
			IPVersion: constants.IPv4,
			s:         "55.55.55.55 56.56.56.56",
			err:       fmt.Errorf("multiple public ipv4 addresses found: 55.55.55.55 56.56.56.56"),
		},
		"2 ipv6 addresses": {
			IPVersion: constants.IPv6,
			s:         "bd07:e846:51ac:6cd0:0000:0000:0000:0000  ad07:e846:51ac:6cd0:0000:0000:0000:0000",
			err:       fmt.Errorf("multiple public ipv6 addresses found: ad07:e846:51ac:6cd0:: bd07:e846:51ac:6cd0::"), //nolint:go-lint
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ip, err := searchIP(tc.IPVersion, tc.s)
			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}
