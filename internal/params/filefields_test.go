package params

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_resolveFileFields(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")
	files := map[string]string{ //nolint:gosec // fake file contents, not credentials
		"/run/secrets/token":      "abc123\n",
		"/run/secrets/token_crlf": "abc123\r\n",
		"/run/secrets/password":   "p@ss\"word",
		"/run/secrets/two_lines":  "line1\nline2\n\n",
	}
	readFile := func(filename string) ([]byte, error) {
		content, ok := files[filename]
		if !ok {
			return nil, errTest
		}
		return []byte(content), nil
	}

	testCases := map[string]struct {
		settings   string
		expected   map[string]string
		unchanged  bool
		errWrapped error
		errMessage string
	}{
		"no_file_field": {
			settings:  `{"provider":"desec","token":"abc"}`,
			unchanged: true,
		},
		"bare_suffix_is_not_a_file_field": {
			settings:  `{"provider":"desec","_file":"/run/secrets/token"}`,
			unchanged: true,
		},
		"trailing_newline_removed": {
			settings: `{"provider":"desec","token_file":"/run/secrets/token"}`,
			expected: map[string]string{"provider": "desec", "token": "abc123"},
		},
		"trailing_crlf_removed": {
			settings: `{"provider":"desec","token_file":"/run/secrets/token_crlf"}`,
			expected: map[string]string{"provider": "desec", "token": "abc123"},
		},
		"only_one_trailing_newline_removed": {
			settings: `{"provider":"desec","token_file":"/run/secrets/two_lines"}`,
			expected: map[string]string{"provider": "desec", "token": "line1\nline2\n"},
		},
		"value_is_json_escaped": {
			settings: `{"provider":"x","username":"u","password_file":"/run/secrets/password"}`,
			expected: map[string]string{"provider": "x", "username": "u", "password": "p@ss\"word"},
		},
		"several_file_fields": {
			settings: `{"provider":"x","token_file":"/run/secrets/token","password_file":"/run/secrets/password"}`,
			expected: map[string]string{"provider": "x", "token": "abc123", "password": "p@ss\"word"},
		},
		"not_a_string": {
			settings:   `{"provider":"desec","token_file":1}`,
			errWrapped: ErrFileFieldNotString,
			errMessage: "file field value is not a string: token_file",
		},
		"empty_path": {
			settings:   `{"provider":"desec","token_file":""}`,
			errWrapped: ErrFileFieldEmptyPath,
			errMessage: "file field path is empty: token_file",
		},
		"conflict": {
			settings:   `{"provider":"desec","token":"abc","token_file":"/run/secrets/token"}`,
			errWrapped: ErrFileFieldConflict,
			errMessage: "field is set both directly and with its file variant: token and token_file",
		},
		"read_error": {
			settings:   `{"provider":"desec","token_file":"/run/secrets/missing"}`,
			errWrapped: ErrFileFieldRead,
			errMessage: "cannot read file field token_file: test error",
		},
		"not_an_object": {
			settings:   `["token_file"]`,
			errWrapped: errDecodeSettings,
			errMessage: "cannot decode settings object: " +
				"json: cannot unmarshal array into Go value of type map[string]json.RawMessage",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resolved, err := resolveFileFields(json.RawMessage(testCase.settings), readFile)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
				return
			}
			if testCase.unchanged {
				assert.Equal(t, testCase.settings, string(resolved))
				return
			}
			var fields map[string]string
			require.NoError(t, json.Unmarshal(resolved, &fields))
			assert.Equal(t, testCase.expected, fields)
		})
	}
}

func Test_extractAllSettings_fileField(t *testing.T) {
	t.Parallel()

	readFile := func(filename string) ([]byte, error) {
		if filename == "/run/secrets/desec_token" {
			return []byte("token-value\n"), nil
		}
		return nil, errors.New("not found")
	}

	config := []byte(`{"settings":[{"provider":"desec","domain":"sub.example.dedyn.io",` +
		`"token_file":"/run/secrets/desec_token"}]}`)
	providers, _, err := extractAllSettings(config, readFile)
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "sub", providers[0].Owner())

	// Without the token, the desec provider refuses the settings: the file field is what supplies it.
	config = []byte(`{"settings":[{"provider":"desec","domain":"sub.example.dedyn.io"}]}`)
	_, _, err = extractAllSettings(config, readFile)
	assert.Error(t, err)

	config = []byte(`{"settings":[{"provider":"desec","domain":"sub.example.dedyn.io",` +
		`"token_file":"/run/secrets/other"}]}`)
	_, _, err = extractAllSettings(config, readFile)
	assert.ErrorIs(t, err, ErrFileFieldRead)
	assert.ErrorContains(t, err, "settings 1: ")
}
