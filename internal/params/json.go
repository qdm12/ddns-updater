package params

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

type settingsType struct {
	Provider       string `json:"provider"`
	Domain         string `json:"domain"`
	IPMethod       string `json:"ip_method"`
	Delay          uint64 `json:"delay"`
	NoDNSLookup    bool   `json:"no_dns_lookup"`
	Host           string `json:"host"`
	Password       string `json:"password"`         // Namecheap, NoIP only
	Key            string `json:"key"`              // GoDaddy, Dreamhost and Cloudflare only
	Secret         string `json:"secret"`           // GoDaddy only
	Token          string `json:"token"`            // DuckDNS and Cloudflare only
	Email          string `json:"email"`            // Cloudflare only
	Username       string `json:"username"`         // NoIP only
	UserServiceKey string `json:"user_service_key"` // Cloudflare only
	ZoneIdentifier string `json:"zone_identifier"`  // Cloudflare only
	Identifier     string `json:"identifier"`       // Cloudflare only
	Proxied        bool   `json:"proxied"`          // Cloudflare only
	Ttl            uint   `json:"ttl"`              // Cloudflare only
}

// GetSettings obtain the update settings from config.json
func (p *params) GetSettings(filePath string) (settings []models.Settings, warnings []string, err error) {
	bytes, err := p.readFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	var config struct {
		Settings []settingsType `json:"settings"`
	}
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, nil, err
	}
	for _, s := range config.Settings {
		switch models.Provider(s.Provider) {
		case constants.DREAMHOST, constants.DUCKDNS:
			s.Host = "@" // only choice available
		}
		ipMethod := models.IPMethod(s.IPMethod)
		// Retro compatibility
		if ipMethod == constants.GOOGLE {
			p.logger.Warn("IP Method %q is no longer valid, please change it. Defaulting it to %s", constants.GOOGLE, constants.CYCLE)
			ipMethod = constants.CYCLE
		}
		setting := models.Settings{
			Provider:       models.Provider(s.Provider),
			Domain:         s.Domain,
			Host:           s.Host,
			IPMethod:       ipMethod,
			Delay:          time.Second * time.Duration(s.Delay),
			NoDNSLookup:    s.NoDNSLookup,
			Password:       s.Password,
			Key:            s.Key,
			Secret:         s.Secret,
			Token:          s.Token,
			Email:          s.Email,
			Username:       s.Username,
			UserServiceKey: s.UserServiceKey,
			ZoneIdentifier: s.ZoneIdentifier,
			Identifier:     s.Identifier,
			Proxied:        s.Proxied,
			Ttl:            s.Ttl,
		}
		if err := p.isConsistent(setting); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s for settings %s", err, setting.String()))
			continue
		}
		settings = append(settings, setting)
	}
	if len(settings) == 0 {
		return nil, warnings, fmt.Errorf("no settings found in config.json")
	}
	return settings, warnings, nil
}
