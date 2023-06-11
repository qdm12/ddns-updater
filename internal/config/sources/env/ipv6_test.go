package env

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ipv6DecimalPrefixToMask(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		prefixDecimal string
		ipMask        net.IPMask
		err           error
	}{
		"empty": {
			err: fmt.Errorf("cannot parse IP prefix: empty prefix"),
		},
		"malformed": {
			prefixDecimal: "malformed",
			err:           fmt.Errorf("cannot parse IP prefix: malformed"),
		},
		"with leading slash": {
			prefixDecimal: "/78",
			ipMask:        net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 252, 0, 0, 0, 0, 0, 0},
		},
		"without leading slash": {
			prefixDecimal: "78",
			ipMask:        net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 252, 0, 0, 0, 0, 0, 0},
		},
		"full IPv6 mask": {
			prefixDecimal: "/128",
			ipMask:        net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		"zero IPv6 mask": {
			prefixDecimal: "/0",
			ipMask:        net.IPMask{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ipMask, err := ipv6DecimalPrefixToMask(testCase.prefixDecimal)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.ipMask, ipMask)
		})
	}
}
