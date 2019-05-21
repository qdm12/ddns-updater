package params

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"ddns-updater/pkg/models"
)

type configType struct {
	Settings []settingsType `json:"settings"`
}

type settingsType struct {
	Provider       string `json:"provider"`
	Domain         string `json:"domain"`
	IPMethod       string `json:"ip_method"`
	Delay          int    `json:"delay"`
	NoDNSLookup    bool   `json:"no_dns_lookup"`
	Host           string `json:"host"`
	Password       string `json:"password"`         // Namecheap only
	Key            string `json:"key"`              // GoDaddy, Dreamhost and Cloudflare only
	Secret         string `json:"secret"`           // GoDaddy only
	Token          string `json:"token"`            // DuckDNS only
	Email          string `json:"email"`            // Cloudflare only
	UserServiceKey string `json:"user_service_key"` // Cloudflare only
	ZoneIdentifier string `json:"zone_identifier"`  // Cloudflare only
	Identifier     string `json:"identifier"`       // Cloudflare only
	Proxied        bool   `json:"proxied"`          // Cloudflare only
}

// GetSettings obtain the update settings from config.json
func GetSettings(filePath string) (settings []models.SettingsType, warnings []string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	bytes, err := ioutil.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, nil, err
	}
	var config configType
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range config.Settings {
		provider, err := models.ParseProvider(s.Provider)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		IPMethod, err := models.ParseIPMethod(s.IPMethod)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		delay := time.Duration(s.Delay)
		host := s.Host
		if provider == models.PROVIDERDREAMHOST || provider == models.PROVIDERDUCKDNS {
			host = "@" // only one choice
		}
		setting := models.SettingsType{
			Provider:       provider,
			Domain:         s.Domain,
			Host:           host,
			IPmethod:       IPMethod,
			Delay:          delay,
			NoDNSLookup:    s.NoDNSLookup,
			Password:       s.Password,
			Key:            s.Key,
			Secret:         s.Secret,
			Token:          s.Token,
			Email:          s.Email,
			UserServiceKey: s.UserServiceKey,
			ZoneIdentifier: s.ZoneIdentifier,
			Identifier:     s.Identifier,
			Proxied:        s.Proxied,
		}
		err = setting.Verify()
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		settings = append(settings, setting)
	}
	if len(settings) == 0 {
		return nil, warnings, fmt.Errorf("no settings found in config.json")
	}
	return settings, warnings, nil
}
