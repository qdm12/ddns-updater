package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateIPv6Prefix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		prefixDecimal string
		err           error
	}{
		"empty": {
			err: fmt.Errorf(`IPv6 prefix format is incorrect: ` +
				`cannot parse "" as uint8`),
		},
		"malformed": {
			prefixDecimal: "malformed",
			err: fmt.Errorf(`IPv6 prefix format is incorrect: ` +
				`cannot parse "malformed" as uint8`),
		},
		"with leading slash": {
			prefixDecimal: "/78",
		},
		"without leading slash": {
			prefixDecimal: "78",
		},
		"full IPv6 mask": {
			prefixDecimal: "/128",
		},
		"zero IPv6 mask": {
			prefixDecimal: "/0",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateIPv6Prefix(testCase.prefixDecimal)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
