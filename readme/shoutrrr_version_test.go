package readme

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

var regexShoutrrrURL = regexp.MustCompile(`https://containrrr.dev/shoutrrr/v[0-9.]+/services/overview/`)

func Test_Readme_Shoutrrr_Version(t *testing.T) {
	t.Parallel()

	goModBytes, err := os.ReadFile("../go.mod")
	require.NoError(t, err)

	goMod, err := modfile.Parse("../go.mod", goModBytes, nil)
	require.NoError(t, err)

	shoutrrrVersion := ""
	for _, require := range goMod.Require {
		if require.Mod.Path != "github.com/containrrr/shoutrrr" {
			continue
		}
		shoutrrrVersion = require.Mod.Version
	}
	require.NotEmpty(t, shoutrrrVersion)

	// Remove bugfix suffix from version
	lastDot := strings.LastIndex(shoutrrrVersion, ".")
	require.GreaterOrEqual(t, lastDot, 0)
	urlShoutrrrVersion := shoutrrrVersion[:lastDot]

	expectedShoutrrrURL := "https://containrrr.dev/shoutrrr/" +
		urlShoutrrrVersion + "/services/overview/"

	readmeBytes, err := os.ReadFile("../README.md")
	require.NoError(t, err)
	readmeString := string(readmeBytes)

	readmeShoutrrrURLs := regexShoutrrrURL.FindAllString(readmeString, -1)
	require.NotEmpty(t, readmeShoutrrrURLs)

	for _, readmeShoutrrrURL := range readmeShoutrrrURLs {
		if readmeShoutrrrURL != expectedShoutrrrURL {
			t.Errorf("README.md contains outdated shoutrrr URL: %s should be %s",
				readmeShoutrrrURL, expectedShoutrrrURL)
		}
	}
}
