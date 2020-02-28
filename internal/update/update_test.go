package update

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
		"Google IP method": {
			IPMethod:    constants.OPENDNS,
			mockURL:     constants.IPMethodMapping()[constants.OPENDNS],
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
			client := &mocks.Client{}
			if len(tc.mockURL) != 0 {
				client.On("GetContent", tc.mockURL).Return(
					tc.mockContent, http.StatusOK, nil).Once()
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
			client.AssertExpectations(t)
		})
	}

}
