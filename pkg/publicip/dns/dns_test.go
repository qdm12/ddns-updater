package dns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	impl, err := New(SetTimeout(time.Hour))
	require.NoError(t, err)

	assert.NotNil(t, impl.ring.counter)
	assert.NotEmpty(t, impl.ring.providers)
}
