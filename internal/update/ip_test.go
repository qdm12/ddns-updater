package update

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/network/mock_network"
	"github.com/stretchr/testify/assert"
)

func Test_NewIPGetter(t *testing.T) {
	t.Parallel()
	client := network.NewClient(time.Second)
	ipMethod := models.IPMethod{Name: "ip"}
	ipv4Method := models.IPMethod{Name: "ipv4"}
	ipv6Method := models.IPMethod{Name: "ipv6"}
	ipGetter := NewIPGetter(client, ipMethod, ipv4Method, ipv6Method)
	assert.NotNil(t, ipGetter)
}

func Test_IP(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		ipMethod    models.IPMethod
		mockContent []byte
		ip          net.IP
	}{
		"url ipv4": {
			ipMethod:    models.IPMethod{URL: "https://opendns.com/ip"},
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
		"url ipv6": {
			ipMethod:    models.IPMethod{URL: "https://opendns.com/ip"},
			mockContent: []byte("blabla ad07:e846:51ac:6cd0:0000:0000:0000:0000 sds"),
			ip:          net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"cycle": {
			ipMethod:    models.IPMethod{Name: cycle},
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			client := mock_network.NewMockClient(mockCtrl)
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://diagnostic.opendns.com/myip"
			}
			client.EXPECT().GetContent(url).Return(tc.mockContent, http.StatusOK, nil).Times(1)
			ig := NewIPGetter(client, tc.ipMethod, models.IPMethod{}, models.IPMethod{})
			ip, err := ig.IP()
			assert.Nil(t, err)
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}

func Test_IPv4(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		ipMethod    models.IPMethod
		mockContent []byte
		ip          net.IP
	}{
		"url": {
			ipMethod:    models.IPMethod{URL: "https://opendns.com/ip"},
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
		"cycle": {
			ipMethod:    models.IPMethod{Name: cycle},
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			client := mock_network.NewMockClient(mockCtrl)
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://api.ipify.org"
			}
			client.EXPECT().GetContent(url).Return(tc.mockContent, http.StatusOK, nil).Times(1)
			ig := NewIPGetter(client, models.IPMethod{}, tc.ipMethod, models.IPMethod{})
			ip, err := ig.IPv4()
			assert.Nil(t, err)
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}

func Test_IPv6(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		ipMethod    models.IPMethod
		mockContent []byte
		ip          net.IP
	}{
		"url": {
			ipMethod:    models.IPMethod{URL: "https://ip6.ddnss.de/meineip.php"},
			mockContent: []byte("blabla ad07:e846:51ac:6cd0:0000:0000:0000:0000 sds"),
			ip:          net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"cycle": {
			ipMethod:    models.IPMethod{Name: cycle},
			mockContent: []byte("blabla ad07:e846:51ac:6cd0:0000:0000:0000:0000 sds"),
			ip:          net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			client := mock_network.NewMockClient(mockCtrl)
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://api6.ipify.org"
			}
			client.EXPECT().GetContent(url).Return(tc.mockContent, http.StatusOK, nil).Times(1)
			ig := NewIPGetter(client, models.IPMethod{}, models.IPMethod{}, tc.ipMethod)
			ip, err := ig.IPv6()
			assert.Nil(t, err)
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}
