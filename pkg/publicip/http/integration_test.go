//go:build integration
// +build integration

package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration(t *testing.T) {
	t.Parallel()

	client := &http.Client{}

	fetcher, err := New(client, SetProvidersIP(Ipify))
	require.NoError(t, err)

	ctx := context.Background()

	publicIP1, err := fetcher.IP4(ctx)
	require.NoError(t, err)
	assert.NotNil(t, publicIP1)

	publicIP2, err := fetcher.IP4(ctx)
	require.NoError(t, err)
	assert.NotNil(t, publicIP2)

	assert.Equal(t, publicIP1, publicIP2)

	t.Logf("Public IP is %s", publicIP1)
}
