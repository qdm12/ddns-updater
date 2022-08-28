package http

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: time.Second}

	testCases := map[string]struct {
		options []Option
		fetcher *Fetcher
		err     error
	}{
		"no options": {
			fetcher: &Fetcher{
				client:  client,
				timeout: 5 * time.Second,
				ip4or6: urlsRing{
					counter: new(uint32),
					urls:    []string{"https://domains.google.com/checkip"},
				},
				ip4: urlsRing{
					counter: new(uint32),
					urls:    []string{"http://ip1.dynupdate.no-ip.com"},
				},
				ip6: urlsRing{
					counter: new(uint32),
					urls:    []string{"http://ip1.dynupdate6.no-ip.com"},
				},
			},
		},
		"with options": {
			options: []Option{
				SetProvidersIP(Opendns),
				SetProvidersIP4(Ipify),
				SetProvidersIP6(Ipify),
				SetTimeout(time.Second),
			},
			fetcher: &Fetcher{
				client:  client,
				timeout: time.Second,
				ip4or6: urlsRing{
					counter: new(uint32),
					urls:    []string{"https://diagnostic.opendns.com/myip"},
				},
				ip4: urlsRing{
					counter: new(uint32),
					urls:    []string{"https://api.ipify.org"},
				},
				ip6: urlsRing{
					counter: new(uint32),
					urls:    []string{"https://api6.ipify.org"},
				},
			},
		},
		"bad option": {
			options: []Option{
				SetProvidersIP(Provider("invalid")),
			},
			err: errors.New("unknown provider: invalid"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fetcher, err := New(client, testCase.options...)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
				assert.Nil(t, fetcher)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.fetcher, fetcher)
		})
	}
}
