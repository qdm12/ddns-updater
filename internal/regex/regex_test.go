package regex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Matcher_DNSOMaticUsername(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		s     string
		match bool
	}{
		"empty": {},
		"email": {
			s:     "email@mail12.com",
			match: true,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			matcher := NewMatcher()
			match := matcher.DNSOMaticUsername(testCase.s)
			assert.Equal(t, testCase.match, match)
		})
	}
}
