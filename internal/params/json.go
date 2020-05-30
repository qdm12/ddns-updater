package params

import (
	"encoding/json"
	"fmt"

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
	verifier := verification.NewVerifier()
	allSettings = make([]settings.Settings, len(config.CommonSettings))
	for i, s := range config.CommonSettings {
		if s.Delay != nil {
			warnings = append(warnings, "per record delay is not supported anymore and will be ignored")
		}
		if s.IPMethod != nil {
			warnings = append(warnings, "per record ip method is not supported anymore and will be ignored")
		}
		provider := models.Provider(s.Provider)
		switch provider {
		case constants.DREAMHOST, constants.DUCKDNS:
			if s.Host != "" && s.Host != "@" {
				warnings = append(warnings, fmt.Sprintf("Provider %s only supports @ host configurations, forcing host to @", provider))
			}
			s.Host = "@"
		}
		if len(s.Host) == 0 {
			return nil, nil, fmt.Errorf("host cannot be empty")
		}
		if !verifier.MatchDomain(s.Domain) {
			return nil, nil, fmt.Errorf("invalid domain name format %q", s.Domain)
		}

		ipVersion := models.IPVersion(s.IPVersion)
		if len(ipVersion) == 0 {
			ipVersion = constants.IPv4OrIPv6 // default
		}
		if err := settingsIPVersionChecks(ipVersion, provider); err != nil {
			return nil, nil, err
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
		case constants.DYN:
			settingsConstructor = settings.NewDyn
		default:
			return nil, nil, fmt.Errorf("provider %q is not supported", provider)
		}
		allSettings[i], err = settingsConstructor(rawConfig.Settings[i], s.Domain, s.Host, ipVersion, s.NoDNSLookup)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(allSettings) == 0 {
		warnings = append(warnings, "no settings found in config.json")
	}
	return allSettings, warnings, nil
}
