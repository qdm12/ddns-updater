package params

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/golibs/verification"
)

// nolint: maligned
type commonSettings struct {
	Provider    string `json:"provider"`
	Domain      string `json:"domain"`
	Host        string `json:"host"`
	IPVersion   string `json:"ip_version"`
	NoDNSLookup bool   `json:"no_dns_lookup"`
	// Retro values for warnings
	IPMethod *string `json:"ip_method,omitempty"`
	Delay    *uint64 `json:"delay,omitempty"`
}

// GetSettings obtain the update settings from config.json
func (r *reader) GetSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error) {
	bytes, err := r.readFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	config := struct {
		CommonSettings []commonSettings `json:"settings"`
	}{}
	rawConfig := struct {
		Settings []json.RawMessage `json:"settings"`
	}{}
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(bytes, &rawConfig); err != nil {
		return nil, nil, err
	}
	for i, common := range config.CommonSettings {
		newSettings, newWarnings, err := makeSettingsFromObject(common, rawConfig.Settings[i])
		warnings = append(warnings, newWarnings...)
		if err != nil {
			return nil, warnings, err
		}
		allSettings = append(allSettings, newSettings...)
	}
	if len(allSettings) == 0 {
		warnings = append(warnings, "no settings found in config.json")
	}
	return allSettings, warnings, nil
}

func makeSettingsFromObject(common commonSettings, rawSettings json.RawMessage) (settingsSlice []settings.Settings, warnings []string, err error) {
	hosts := strings.Split(common.Host, ",")
	provider := models.Provider(common.Provider)
	switch provider {
	case constants.DREAMHOST, constants.DUCKDNS:
		for i, host := range hosts {
			if host != "" && host != "@" {
				warnings = append(warnings, fmt.Sprintf("Provider %s only supports @ host configurations, forcing host to @", provider))
			}
			hosts[i] = "@"
		}
	}
	for _, host := range hosts {
		if len(host) == 0 {
			return nil, warnings, fmt.Errorf("host cannot be empty")
		}
	}
	if !verification.NewVerifier().MatchDomain(common.Domain) {
		return nil, warnings, fmt.Errorf("invalid domain name format %q", common.Domain)
	}

	ipVersion := models.IPVersion(common.IPVersion)
	if len(ipVersion) == 0 {
		ipVersion = constants.IPv4OrIPv6 // default
	}
	if err := settingsIPVersionChecks(ipVersion, provider); err != nil {
		return nil, warnings, err
	}
	var settingsConstructor settings.Constructor
	switch provider {
	case constants.CLOUDFLARE:
		settingsConstructor = settings.NewCloudflare
	case constants.DDNSSDE:
		settingsConstructor = settings.NewDdnss
	case constants.DNSPOD:
		settingsConstructor = settings.NewDNSPod
	case constants.DREAMHOST:
		settingsConstructor = settings.NewDreamhost
	case constants.DUCKDNS:
		settingsConstructor = settings.NewDuckdns
	case constants.GODADDY:
		settingsConstructor = settings.NewGodaddy
	case constants.INFOMANIAK:
		settingsConstructor = settings.NewInfomaniak
	case constants.NAMECHEAP:
		settingsConstructor = settings.NewNamecheap
	case constants.NOIP:
		settingsConstructor = settings.NewNoip
	default:
		return nil, warnings, fmt.Errorf("provider %q is not supported", provider)
	}
	settingsSlice = make([]settings.Settings, len(hosts))
	for i, host := range hosts {
		settingsSlice[i], err = settingsConstructor(rawSettings, common.Domain, host, ipVersion, common.NoDNSLookup)
		if err != nil {
			return nil, warnings, err
		}
	}
	return settingsSlice, warnings, nil
}
