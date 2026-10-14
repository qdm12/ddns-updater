package params

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLogger struct{}

func (testLogger) Info(string)  {}
func (testLogger) Debug(string) {}

func TestEnvironmentConfigPersistence(t *testing.T) {
	t.Setenv("CONFIG", `{"settings":[]}`)
	filePath := filepath.Join(t.TempDir(), "config.json")

	reader := NewReader(testLogger{}, false)
	providers, warnings, err := reader.getProvidersFromEnv(filePath)
	require.NoError(t, err)
	assert.Empty(t, providers)
	assert.Empty(t, warnings)
	_, err = os.Stat(filePath)
	assert.ErrorIs(t, err, os.ErrNotExist)

	reader = NewReader(testLogger{}, true)
	_, _, err = reader.getProvidersFromEnv(filePath)
	require.NoError(t, err)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.JSONEq(t, `{"settings":[]}`, string(content))
}
