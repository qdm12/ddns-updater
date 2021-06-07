package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/params"
)

type commonSettings struct {
	Provider  string `json:"provider"`
	Domain    string `json:"domain"`
	Host      string `json:"host"`
	IPVersion string `json:"ip_version"`
	// Retro values for warnings
	IPMethod *string `json:"ip_method,omitempty"`
	Delay    *uint64 `json:"delay,omitempty"`
}

// JSONSettings obtain the update settings from the JSON content, first trying from the environment variable CONFIG
// and then from the file config.json.
func (r *reader) JSONSettings(filePath string, logger log.Logger) (
	allSettings []settings.Settings, warnings []string, err error) {
	allSettings, warnings, err = r.getSettingsFromEnv(logger)
	if allSettings != nil || warnings != nil || err != nil {
		return allSettings, warnings, err
	}
	return r.getSettingsFromFile(filePath, logger)
}

// getSettingsFromFile obtain the update settings from config.json.
func (r *reader) getSettingsFromFile(filePath string, logger log.Logger) (
	allSettings []settings.Settings, warnings []string, err error) {
	bytes, err := r.readFile(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
		const mode = fs.FileMode(0600)
		return nil, nil, r.writeFile(filePath, []byte(`{}`), mode)
	}
	return extractAllSettings(bytes, logger)
}

// getSettingsFromEnv obtain the update settings from the environment variable CONFIG.
func (r *reader) getSettingsFromEnv(logger log.Logger) (allSettings []settings.Settings, warnings []string, err error) {
	s, err := r.env.Get("CONFIG", params.CaseSensitiveValue())
	if err != nil {
		return nil, nil, err
	} else if s == "" {
		return nil, nil, nil
	}
	return extractAllSettings([]byte(s), logger)
}

func extractAllSettings(jsonBytes []byte, logger log.Logger) (
	allSettings []settings.Settings, warnings []string, err error) {
	config := struct {
		CommonSettings []commonSettings `json:"settings"`
	}{}
	rawConfig := struct {
		Settings []json.RawMessage `json:"settings"`
	}{}
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(jsonBytes, &rawConfig); err != nil {
		return nil, nil, err
	}
	matcher := regex.NewMatcher()

	for i, common := range config.CommonSettings {
		newSettings, newWarnings, err := makeSettingsFromObject(common, rawConfig.Settings[i], matcher, logger)
		warnings = append(warnings, newWarnings...)
		if err != nil {
			return nil, warnings, err
		}
		allSettings = append(allSettings, newSettings...)
	}

	return allSettings, warnings, nil
}

func makeSettingsFromObject(common commonSettings, rawSettings json.RawMessage,
	matcher regex.Matcher, logger log.Logger) (
	settingsSlice []settings.Settings, warnings []string, err error) {
	provider := models.Provider(common.Provider)
	if provider == constants.DuckDNS { // only hosts, no domain
		if len(common.Domain) > 0 { // retro compatibility
			if len(common.Host) == 0 {
				common.Host = strings.TrimSuffix(common.Domain, ".duckdns.org")
				warnings = append(warnings,
					fmt.Sprintf("DuckDNS record should have %q specified as host instead of %q as domain",
						common.Host, common.Domain))
			} else {
				warnings = append(warnings,
					fmt.Sprintf("ignoring domain %q because host %q is specified for DuckDNS record",
						common.Domain, common.Host))
			}
		}
	}
	hosts := strings.Split(common.Host, ",")

	if len(common.IPVersion) == 0 {
		common.IPVersion = ipversion.IP4or6.String()
	}
	ipVersion, err := ipversion.Parse(common.IPVersion)
	if err != nil {
		return nil, nil, err
	}

	settingsSlice = make([]settings.Settings, len(hosts))
	for i, host := range hosts {
		settingsSlice[i], err = settings.New(provider, rawSettings, common.Domain,
			host, ipVersion, matcher, logger)
		if err != nil {
			return nil, warnings, err
		}
	}
	return settingsSlice, warnings, nil
}
