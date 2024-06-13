package params

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_makeIPv6Suffix(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		prefixBitsString string
		suffix           netip.Prefix
		errWrapped       error
		errMessage       string
	}{
		"empty": {
			errWrapped: errors.New(`IPv6 prefix format is incorrect: ` +
				`cannot parse "" as uint8`),
		},
		"malformed": {
			prefixBitsString: "malformed",
			errWrapped: errors.New(`IPv6 prefix format is incorrect: ` +
				`cannot parse "malformed" as uint8`),
		},
		"with_leading_slash": {
			prefixBitsString: "/78",
			suffix:           netip.MustParsePrefix("::/50"),
		},
		"without_leading_slash": {
			prefixBitsString: "78",
			suffix:           netip.MustParsePrefix("::/50"),
		},
		"full_IPv6_mask": {
			prefixBitsString: "/128",
			suffix:           netip.MustParsePrefix("::/0"),
		},
		"zero_IPv6_mask": {
			prefixBitsString: "/0",
			suffix:           netip.MustParsePrefix("::/128"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			suffix, err := makeIPv6Suffix(testCase.prefixBitsString)

			if testCase.errWrapped != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.errWrapped.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.suffix, suffix)
		})
	}
}
