package update

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_NewIPGetter(t *testing.T) {
	t.Parallel()
	client := &http.Client{}
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

			ctx := context.Background()
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://diagnostic.opendns.com/myip"
			}

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, url, r.URL.String())
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(tc.mockContent)),
					}, nil
				}),
			}

			ig := NewIPGetter(client, tc.ipMethod, models.IPMethod{}, models.IPMethod{})
			ip, err := ig.IP(ctx)
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

			ctx := context.Background()
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://api.ipify.org"
			}

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, url, r.URL.String())
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(tc.mockContent)),
					}, nil
				}),
			}

			ig := NewIPGetter(client, models.IPMethod{}, tc.ipMethod, models.IPMethod{})
			ip, err := ig.IPv4(ctx)
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

			ctx := context.Background()
			url := tc.ipMethod.URL
			if tc.ipMethod.Name == cycle {
				url = "https://api6.ipify.org"
			}

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, url, r.URL.String())
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(tc.mockContent)),
					}, nil
				}),
			}

			ig := NewIPGetter(client, models.IPMethod{}, models.IPMethod{}, tc.ipMethod)
			ip, err := ig.IPv6(ctx)
			assert.Nil(t, err)
			assert.True(t, tc.ip.Equal(ip))
		})
	}
}
