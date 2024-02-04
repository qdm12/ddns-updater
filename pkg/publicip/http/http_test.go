package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: time.Second}

	testCases := map[string]struct {
		options    []Option
		fetcher    *Fetcher
		err        error
		errMessage string
	}{
		"no options": {
			fetcher: &Fetcher{
				client:  client,
				timeout: 5 * time.Second,
				ip4or6: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://domains.google.com/checkip"},
				},
				ip4: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://api.ipify.org"},
				},
				ip6: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://api6.ipify.org"},
				},
			},
		},
		"with options": {
			options: []Option{
				SetProvidersIP(Google),
				SetProvidersIP4(Ipify),
				SetProvidersIP6(Ipify),
				SetTimeout(time.Second),
			},
			fetcher: &Fetcher{
				client:  client,
				timeout: time.Second,
				ip4or6: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://domains.google.com/checkip"},
				},
				ip4: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://api.ipify.org"},
				},
				ip6: &urlsRing{
					banned: map[int]string{},
					urls:   []string{"https://api6.ipify.org"},
				},
			},
		},
		"bad option": {
			options: []Option{
				SetProvidersIP(Provider("invalid")),
			},
			err:        ErrUnknownProvider,
			errMessage: "unknown public IP echo HTTP provider: invalid",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fetcher, err := New(client, testCase.options...)

			assert.ErrorIs(t, err, testCase.err)
			if testCase.err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.fetcher, fetcher)
		})
	}
}
