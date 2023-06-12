package env

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ipv6DecimalPrefixToBits(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		prefixDecimal string
		maskBits      uint8
		err           error
	}{
		"empty": {
			err: fmt.Errorf(`parsing prefix decimal as uint8: ` +
				`strconv.ParseUint: parsing "": invalid syntax`),
		},
		"malformed": {
			prefixDecimal: "malformed",
			err: fmt.Errorf(`parsing prefix decimal as uint8: ` +
				`strconv.ParseUint: parsing "malformed": invalid syntax`),
		},
		"with leading slash": {
			prefixDecimal: "/78",
			maskBits:      78,
		},
		"without leading slash": {
			prefixDecimal: "78",
			maskBits:      78,
		},
		"full IPv6 mask": {
			prefixDecimal: "/128",
			maskBits:      128,
		},
		"zero IPv6 mask": {
			prefixDecimal: "/0",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ipMask, err := ipv6DecimalPrefixToBits(testCase.prefixDecimal)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.maskBits, ipMask)
		})
	}
}
