package update

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

func Test_incCounter(t *testing.T) {
	t.Parallel()
	const initValue = 100
	u := &updater{
		counter: initValue,
	}
	counter := u.incCounter()
	assert.Equal(t, initValue, counter)
	counter = u.incCounter()
	assert.Equal(t, initValue+1, counter)
}

func Test_getPublicIP(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		IPMethod    models.IPMethod
		mockURL     string
		mockContent []byte
		ip          net.IP
		err         error
	}{
		"bad IP method": {
			IPMethod: "abc",
			err:      fmt.Errorf("IP method \"abc\" not supported"),
		},
		"provider IP method": {
			IPMethod: constants.PROVIDER,
		},
		"OpenDNS IP method": {
			IPMethod:    constants.OPENDNS,
			mockURL:     constants.IPMethodMapping()[constants.OPENDNS],
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
		"Custom URL IP method": {
			IPMethod:    models.IPMethod("https://ipinfo.io/ip"),
			mockURL:     "https://ipinfo.io/ip",
			mockContent: []byte("blabla 58.67.201.151.25 sds"),
			ip:          net.IP{58, 67, 201, 151},
		},
		"Cycle IP method": {
			IPMethod:    constants.CYCLE,
			mockURL:     constants.IPMethodMapping()[constants.OPENDNS],
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
			if len(tc.mockURL) != 0 {
				client.EXPECT().GetContent(tc.mockURL).Return(tc.mockContent, http.StatusOK, nil).Times(1)
			}
			u := &updater{
				client:    client,
				ipMethods: []models.IPMethod{constants.OPENDNS, constants.IPINFO},
			}
			ip, err := u.getPublicIP(tc.IPMethod, constants.IPv4)
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
