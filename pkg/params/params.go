package params

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"

	"github.com/spf13/viper"
)

// GetListeningPort obtains and checks the listening port from Viper (env variable or config file, etc.)
func GetListeningPort() (listeningPort string) {
	listeningPort = viper.GetString("listeningPort")
	verifyListeningPort(listeningPort)
	return listeningPort
}

// GetRootURL obtains and checks the root URL from Viper (env variable or config file, etc.)
func GetRootURL() string {
	rootURL := viper.GetString("rooturl")
	verifyRootURL(rootURL)
	rootURL = strings.ReplaceAll(rootURL, "//", "/")
	return strings.TrimSuffix(rootURL, "/") // already have / from paths of router
}

// GetDelay obtains the global delay duration between each updates from Viper (env variable or config file, etc.)
func GetDelay() time.Duration {
	delayStr := viper.GetString("delay")
	delayInt, err := strconv.ParseInt(delayStr, 10, 64)
	if err != nil {
		logging.Fatal("delay %s is not a valid integer", delayStr)
	}
	if delayInt < 10 {
		logging.Fatal("delay %d must be bigger than 10 seconds", delayInt)
	}
	return time.Duration(delayInt)
}

// GetDataDir obtains the data directory from Viper (env variable or config file, etc.)
func GetDataDir(dir string) string {
	dataDir := viper.GetString("datadir")
	if len(dataDir) == 0 {
		dataDir = dir + "/data"
	}
	return dataDir
}

// GetDir obtains the executable directory
func GetDir() (dir string) {
	ex, err := os.Executable()
	if err != nil {
		logging.Fatal("%s", err)
	}
	return filepath.Dir(ex)
}

// GetLoggerMode obtains the logging mode from Viper (env variable or config file, etc.)
func GetLoggerMode() logging.Mode {
	s := viper.GetString("logging")
	return logging.ParseMode(s)
}

// GetLoggerLevel obtains the logging level from Viper (env variable or config file, etc.)
func GetLoggerLevel() logging.Level {
	s := viper.GetString("loglevel")
	return logging.ParseLevel(s)
}

// GetNodeID obtains the node instance ID from Viper (env variable or config file, etc.)
func GetNodeID() int {
	nodeID := viper.GetString("nodeid")
	value, err := strconv.Atoi(nodeID)
	if err != nil {
		logging.Fatal("Node ID %s is not a valid integer", nodeID)
	}
	return value
}

func stringInAny(s string, ss ...string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func getSettingsEnv() (settings []models.SettingsType, warnings []string, err error) {
	var i uint64 = 1
	for {
		s := os.Getenv(fmt.Sprintf("RECORD%d", i))
		if s == "" {
			break
		}
		x := strings.Split(s, ",")
		if len(x) != 5 {
			warnings = append(warnings, "configuration entry "+s+" should be in the format 'domain,host,provider,ipmethod,password'")
			continue
		}
		provider, err := models.ParseProvider(x[2])
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		IPMethod, err := models.ParseIPMethod(x[3])
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		var host, key, secret, token, password string
		switch provider {
		case models.PROVIDERGODADDY:
			host = x[1]
			arr := strings.Split(x[4], ":")
			if len(arr) != 2 {
				warnings = append(warnings, "GoDaddy password (key:secret) is not valid for entry "+s)
				continue
			}
			key = arr[0]
			secret = arr[1]
		case models.PROVIDERNAMECHEAP:
			host = x[1]
			password = x[4]
		case models.PROVIDERDUCKDNS:
			host = "@"
			token = x[4]
		case models.PROVIDERDREAMHOST:
			host = "@"
			key = x[4]
		}
		setting := models.SettingsType{
			Domain:   x[0],
			Host:     host,
			Provider: provider,
			IPmethod: IPMethod,
			Password: password,
			Key:      key,
			Secret:   secret,
			Token:    token,
		}
		err = setting.Verify()
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		settings = append(settings, setting)
		i++
	}
	if len(settings) == 0 {
		return nil, warnings, fmt.Errorf("no valid settings was found in the environment variables")
	}
	return settings, warnings, nil
}

// GetAllSettings reads all settings from environment variables and config.json
func GetAllSettings(dir string) []models.SettingsType {
	settingsEnv, warningsEnv, errEnv := getSettingsEnv()
	settingsJSON, warningsJSON, errJSON := getSettingsJSON(dir + "/config.json")
	if errEnv != nil && errJSON != nil {
		logging.Fatal("%s AND %s", errEnv, errJSON)
	} else if errEnv != nil {
		logging.Warn("%s", errEnv)
	} else if errJSON != nil {
		logging.Warn("%s", errJSON)
	}
	for _, w := range append(warningsEnv, warningsJSON...) {
		logging.Warn(w)
	}
	return append(settingsEnv, settingsJSON...)
}
