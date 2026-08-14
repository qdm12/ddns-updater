package nextdns

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	validData := json.RawMessage(`{"endpoint_id":"abcdef","api_guid":"0123456789abcdef"}`)

	provider, err := New(validData, "example.com", "owner",
		ipversion.IP4, netip.Prefix{})

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func Test_validateSettings(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		endpointID string
		apiGUID    string
		errWrapped error
		errMessage string
	}{
		"success": {
			endpointID: "abcdef",
			apiGUID:    "0123456789abcdef",
		},
		"endpoint_id_empty": {
			endpointID: "",
			apiGUID:    "0123456789abcdef",
			errWrapped: errors.ErrEndpointNotValid,
			errMessage: "endpoint is not valid: endpoint_id is not set",
		},
		"api_guid_empty": {
			endpointID: "abcdef",
			apiGUID:    "",
			errWrapped: errors.ErrEndpointNotValid,
			errMessage: "endpoint is not valid: api_guid is not set",
		},
		"both_empty": {
			endpointID: "",
			apiGUID:    "",
			errWrapped: errors.ErrEndpointNotValid,
			errMessage: "endpoint is not valid: endpoint_id is not set",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateSettings(testCase.endpointID, testCase.apiGUID)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
