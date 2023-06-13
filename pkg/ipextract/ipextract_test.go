package ipextract

import (
	"math/rand"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_IPv4(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		text      string
		extracted []netip.Addr
	}{
		"empty": {},
		"one_ipv4": {
			text:      "1.2.3.4",
			extracted: []netip.Addr{netip.MustParseAddr("1.2.3.4")},
		},
		"two_ipv4": {
			text: " 1.2.3.4 x.x.2.2 5.6.7.8.9 10.11.12.13",
			extracted: []netip.Addr{
				netip.MustParseAddr("1.2.3.4"),
				netip.MustParseAddr("10.11.12.13"),
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			extracted := IPv4(testCase.text)

			assert.Equal(t, testCase.extracted, extracted)
		})
	}
}

func Test_IPv6(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		text      string
		extracted []netip.Addr
	}{
		"empty": {},
		"ipv6_compact": {
			text:      "::1",
			extracted: []netip.Addr{netip.MustParseAddr("::1")},
		},
		"two_ipv6_compact": {
			text: ":1 ::1 ::0 ",
			extracted: []netip.Addr{
				netip.MustParseAddr("::1"),
				netip.MustParseAddr("::0"),
			},
		},
		"ipv6_A": {
			text:      "2408:8256:480:3162::cef",
			extracted: []netip.Addr{netip.MustParseAddr("2408:8256:480:3162::cef")},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			extracted := IPv6(testCase.text)

			assert.Equal(t, testCase.extracted, extracted)
		})
	}
}

func Fuzz_IPv6(f *testing.F) {
	f.Fuzz(func(t *testing.T, ipv6A, ipv6B, ipv6C []byte,
		garbageA, garbageB, garbageC string) {
		var arrayA [16]byte
		if len(ipv6A) > 0 {
			copy(arrayA[:], ipv6A)
		}

		var arrayB [16]byte
		if len(ipv6B) > 0 {
			copy(arrayB[:], ipv6B)
		}

		var arrayC [16]byte
		if len(ipv6C) > 0 {
			copy(arrayC[:], ipv6C)
		}

		text := garbageA +
			netip.AddrFrom16(arrayA).String() +
			garbageB +
			netip.AddrFrom16(arrayB).String() +
			garbageC +
			netip.AddrFrom16(arrayC).String() +
			garbageA

		_ = IPv6(text)
	})
}

func Benchmark_IPv6(b *testing.B) {
	source := rand.NewSource(time.Now().UnixNano())
	generator := rand.New(source) //nolint:gosec

	text := "garbage " +
		generateIPv6(generator) +
		"::99999" +
		generateIPv6(generator) +
		"1.2.3.4" +
		generateIPv6(generator) +
		" fac00"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IPv6(text)
	}
}

func generateIPv6(generator *rand.Rand) string {
	ipv6Bytes := make([]byte, 16)
	_, err := generator.Read(ipv6Bytes)
	if err != nil {
		panic(err)
	}
	return netip.AddrFrom16([16]byte(ipv6Bytes)).String()
}
