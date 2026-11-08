package reload

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/params"
	persistence "github.com/qdm12/ddns-updater/internal/persistence/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	duckdnsEntry = `{"domain":"example.duckdns.org","provider":"duckdns",` +
		`"token":"00000000-0000-0000-0000-000000000000"}`
	namecheapEntry = `{"domain":"example.com","password":` +
		`"e5322165c1d74692bfa6d807100c0310","provider":"namecheap"}`
)

type stubLogger struct{}

func (stubLogger) Info(string)  {}
func (stubLogger) Debug(string) {}
func (stubLogger) Warn(string)  {}
func (stubLogger) Error(string) {}

type stubSettingsUpdater struct{}

func (stubSettingsUpdater) SetPeriod(time.Duration)   {}
func (stubSettingsUpdater) SetCooldown(time.Duration) {}

func newTestService(t *testing.T, initialEntries ...string) (
	service *Service, configPath string, db *data.Database,
) {
	t.Helper()

	directory := t.TempDir()
	configPath = filepath.Join(directory, "config.json")

	settings := make([]json.RawMessage, len(initialEntries))
	for i, entry := range initialEntries {
		settings[i] = json.RawMessage(entry)
	}
	content := map[string]any{"settings": settings}
	contentBytes, err := json.Marshal(content)
	require.NoError(t, err)
	err = os.WriteFile(configPath, contentBytes, 0o600)
	require.NoError(t, err)

	persistentDB, err := persistence.NewDatabase(directory)
	require.NoError(t, err)
	db = data.NewDatabase(nil, persistentDB)

	logger := stubLogger{}
	service = New(Settings{
		ConfigFilePath:   configPath,
		SettingsFilePath: filepath.Join(directory, "settings.json"),
		JSONReader:       params.NewReader(logger),
		PersistentDB:     persistentDB,
		Database:         db,
		Updater:          stubSettingsUpdater{},
		Logger:           logger,
	})
	return service, configPath, db
}

func readEntries(t *testing.T, configPath string) (entries []Entry) {
	t.Helper()
	service := New(Settings{ConfigFilePath: configPath})
	entries, err := service.Entries()
	require.NoError(t, err)
	return entries
}

func Test_Entries(t *testing.T) {
	t.Parallel()

	service, _, _ := newTestService(t, duckdnsEntry, namecheapEntry)

	entries, err := service.Entries()
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, 0, entries[0].Index)
	assert.Equal(t, "duckdns", entries[0].Provider)
	assert.Equal(t, "example.duckdns.org", entries[0].Domain)
	assert.Empty(t, entries[0].IPVersion)
	assert.JSONEq(t, duckdnsEntry, string(entries[0].Raw))

	assert.Equal(t, 1, entries[1].Index)
	assert.Equal(t, "namecheap", entries[1].Provider)
	assert.Equal(t, "example.com", entries[1].Domain)
}

func Test_Entries_missingConfigFile(t *testing.T) {
	t.Parallel()

	service := New(Settings{ConfigFilePath: filepath.Join(t.TempDir(), "missing.json")})
	entries, err := service.Entries()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func Test_SaveEntry_append(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, configPath, db := newTestService(t, duckdnsEntry)

	result, err := service.SaveEntry(ctx, nil, json.RawMessage(namecheapEntry))
	require.NoError(t, err)
	assert.Equal(t, 2, result.RecordCount)
	assert.Len(t, db.SelectAll(), 2)

	entries := readEntries(t, configPath)
	require.Len(t, entries, 2)
	assert.Equal(t, "duckdns", entries[0].Provider)
	assert.Equal(t, "namecheap", entries[1].Provider)
}

func Test_SaveEntry_replace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, configPath, db := newTestService(t, duckdnsEntry)

	index := 0
	result, err := service.SaveEntry(ctx, &index, json.RawMessage(namecheapEntry))
	require.NoError(t, err)
	assert.Equal(t, 1, result.RecordCount)
	assert.Len(t, db.SelectAll(), 1)

	entries := readEntries(t, configPath)
	require.Len(t, entries, 1)
	assert.Equal(t, "namecheap", entries[0].Provider)
}

func Test_SaveEntry_invalidEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, configPath, _ := newTestService(t, duckdnsEntry)
	before, err := os.ReadFile(configPath)
	require.NoError(t, err)

	_, err = service.SaveEntry(ctx, nil, json.RawMessage(`{"domain":"example.com"}`))
	assert.ErrorIs(t, err, ErrInvalidEntry)
	assert.Contains(t, err.Error(), "provider is not set")

	_, err = service.SaveEntry(ctx, nil, json.RawMessage(`{"provider":"nope","domain":"example.com"}`))
	assert.ErrorIs(t, err, ErrInvalidEntry)

	_, err = service.SaveEntry(ctx, nil, json.RawMessage(`not json`))
	assert.ErrorIs(t, err, ErrInvalidEntry)

	after, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "an invalid entry must not be written")
}

func Test_SaveEntry_indexOutOfRange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _ := newTestService(t, duckdnsEntry)

	index := 1
	_, err := service.SaveEntry(ctx, &index, json.RawMessage(namecheapEntry))
	assert.ErrorIs(t, err, ErrEntryNotFound)

	index = -1
	_, err = service.SaveEntry(ctx, &index, json.RawMessage(namecheapEntry))
	assert.ErrorIs(t, err, ErrEntryNotFound)
}

func Test_DeleteEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, configPath, db := newTestService(t, duckdnsEntry, namecheapEntry)

	result, err := service.DeleteEntry(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, result.RecordCount)
	assert.Len(t, db.SelectAll(), 1)

	entries := readEntries(t, configPath)
	require.Len(t, entries, 1)
	assert.Equal(t, "namecheap", entries[0].Provider)

	_, err = service.DeleteEntry(ctx, 1)
	assert.ErrorIs(t, err, ErrEntryNotFound)

	// the configuration file is left untouched by the failed delete
	assert.Len(t, readEntries(t, configPath), 1)
}

func Test_EditingDisabledWithConfigEnv(t *testing.T) {
	t.Setenv("CONFIG", `{"settings":[]}`)

	ctx := context.Background()
	service, configPath, _ := newTestService(t, duckdnsEntry)
	before, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.NotEmpty(t, service.EditingDisabledReason())

	_, err = service.SaveEntry(ctx, nil, json.RawMessage(namecheapEntry))
	assert.ErrorIs(t, err, ErrEditingDisabled)

	_, err = service.DeleteEntry(ctx, 0)
	assert.ErrorIs(t, err, ErrEditingDisabled)

	after, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)

	t.Setenv("CONFIG", "")
	assert.Empty(t, New(Settings{}).EditingDisabledReason())
}
