package configfile

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Parse(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		data       string
		expected   []string
		errWrapped error
	}{
		"empty": {
			data:     "",
			expected: []string{},
		},
		"empty_object": {
			data:     "{}",
			expected: []string{},
		},
		"one_entry": {
			data:     `{"settings":[{"provider":"duckdns","domain":"example.duckdns.org"}]}`,
			expected: []string{`{"provider":"duckdns","domain":"example.duckdns.org"}`},
		},
		"two_entries": {
			data: `{"settings":[{"provider":"a","domain":"a.com"},
				{"provider":"b","domain":"b.com"}]}`,
			expected: []string{`{"provider":"a","domain":"a.com"}`, `{"provider":"b","domain":"b.com"}`},
		},
		"null_settings": {
			data:     `{"settings":null}`,
			expected: []string{},
		},
		"invalid_json": {
			data:       `{`,
			errWrapped: ErrInvalidConfig,
		},
		"not_an_object": {
			data:       `[]`,
			errWrapped: ErrInvalidConfig,
		},
		"settings_not_an_array": {
			data:       `{"settings":{}}`,
			errWrapped: ErrInvalidConfig,
		},
		"entry_not_an_object": {
			data:       `{"settings":["x"]}`,
			errWrapped: ErrInvalidConfig,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			document, err := Parse([]byte(testCase.data))
			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				return
			}
			assert.Equal(t, testCase.expected, rawMessagesToStrings(document.Entries()))
		})
	}
}

func Test_Document_mutations(t *testing.T) {
	t.Parallel()

	document := NewDocument()
	assert.Equal(t, 0, document.Len())
	assert.Empty(t, document.Entries())

	document.Add(json.RawMessage(`{"provider":"a"}`))
	document.Add(json.RawMessage(`{"provider":"b"}`))
	document.Add(json.RawMessage(`{"provider":"c"}`))
	assert.Equal(t, 3, document.Len())

	err := document.Set(1, json.RawMessage(`{"provider":"B"}`))
	require.NoError(t, err)
	assert.Equal(t, []string{`{"provider":"a"}`, `{"provider":"B"}`, `{"provider":"c"}`},
		rawMessagesToStrings(document.Entries()))

	err = document.Delete(0)
	require.NoError(t, err)
	assert.Equal(t, []string{`{"provider":"B"}`, `{"provider":"c"}`},
		rawMessagesToStrings(document.Entries()))

	err = document.Set(2, json.RawMessage(`{}`))
	assert.ErrorIs(t, err, ErrIndexOutOfRange)
	err = document.Set(-1, json.RawMessage(`{}`))
	assert.ErrorIs(t, err, ErrIndexOutOfRange)
	err = document.Delete(2)
	assert.ErrorIs(t, err, ErrIndexOutOfRange)
	err = document.Delete(-1)
	assert.ErrorIs(t, err, ErrIndexOutOfRange)
}

func Test_Document_EntriesReturnsCopy(t *testing.T) {
	t.Parallel()

	document := NewDocument()
	document.Add(json.RawMessage(`{"provider":"a"}`))

	entries := document.Entries()
	entries[0] = json.RawMessage(`{"provider":"mutated"}`)

	assert.Equal(t, []string{`{"provider":"a"}`}, rawMessagesToStrings(document.Entries()))
}

func Test_Document_Marshal(t *testing.T) {
	t.Parallel()

	document, err := Parse([]byte(`{"version":1,"settings":[{"provider":"a","domain":"a.com"}]}`))
	require.NoError(t, err)

	data, err := document.Marshal()
	require.NoError(t, err)

	expected := `{
    "settings": [
        {
            "provider": "a",
            "domain": "a.com"
        }
    ],
    "version": 1
}
`
	assert.Equal(t, expected, string(data))

	reparsed, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, rawMessagesToStrings(document.Entries()),
		rawMessagesToStrings(reparsed.Entries()))
}

func Test_File_ReadMissing(t *testing.T) {
	t.Parallel()

	file := NewFile(filepath.Join(t.TempDir(), "missing.json"))
	document, err := file.Read()
	require.NoError(t, err)
	assert.Equal(t, 0, document.Len())

	_, statErr := os.Stat(file.Path())
	assert.True(t, os.IsNotExist(statErr), "reading must not create the file")
}

func Test_File_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "config.json")
	file := NewFile(path)

	original := `{
    "comment": {"keep": "me"},
    "settings": [
        {"provider": "duckdns", "domain": "example.duckdns.org", "token": "abc"}
    ]
}
`
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(path, []byte(original), 0o600)
	require.NoError(t, err)

	document, err := file.Read()
	require.NoError(t, err)
	require.Equal(t, 1, document.Len())

	document.Add(json.RawMessage(`{"provider":"namecheap","domain":"example.com","password":"x"}`))
	err = document.Delete(0)
	require.NoError(t, err)
	document.Add(json.RawMessage(`{"provider":"dnsexit","domain":"example.com","api_key":"k"}`))
	err = file.Write(document)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
			"existing file permissions must be kept")
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	reparsed, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, []string{
		`{"provider":"namecheap","domain":"example.com","password":"x"}`,
		`{"provider":"dnsexit","domain":"example.com","api_key":"k"}`,
	}, rawMessagesToStrings(reparsed.Entries()))
	assert.JSONEq(t, `{"keep":"me"}`, string(reparsed.root["comment"]))

	// no temporary file should be left behind
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "config.json", entries[0].Name())
}

func Test_File_WriteCreatesFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	file := NewFile(path)

	document := NewDocument()
	document.Add(json.RawMessage(`{"provider":"duckdns"}`))
	err := file.Write(document)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"settings"`)

	reparsed, err := file.Read()
	require.NoError(t, err)
	assert.Equal(t, rawMessagesToStrings(document.Entries()),
		rawMessagesToStrings(reparsed.Entries()))
}

func rawMessagesToStrings(messages []json.RawMessage) []string {
	strings := make([]string, len(messages))
	for i, message := range messages {
		var buffer bytes.Buffer
		if err := json.Compact(&buffer, message); err != nil {
			panic(err)
		}
		strings[i] = buffer.String()
	}
	return strings
}
