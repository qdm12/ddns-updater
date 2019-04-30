package params

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"fmt"

	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/regex"
	"github.com/spf13/viper"
)

// GetListeningPort obtains and checks the listening port from Viper (env variable or config file, etc.)
func GetListeningPort() (listeningPort string) {
	listeningPort = viper.GetString("listeningPort")
	value, err := strconv.Atoi(listeningPort)
	if err != nil {
		logging.Fatal("listening port %s is not a valid integer", listeningPort)
	} else if value < 1 {
		logging.Fatal("listening port %s cannot be lower than 1", listeningPort)
	} else if value < 1024 {
		if os.Geteuid() == 0 {
			logging.Warn("listening port %s allowed to be in the reserved system ports range as you are running as root", listeningPort)
		} else if os.Geteuid() == -1 {
			logging.Warn("listening port %s allowed to be in the reserved system ports range as you are running in Windows", listeningPort)
		} else {
			logging.Fatal("listening port %s cannot be in the reserved system ports range (1 to 1023) when running without root", listeningPort)
		}
	} else if value > 65535 {
		logging.Fatal("listening port %s cannot be higher than 65535", listeningPort)
	} else if value > 49151 {
		// dynamic and/or private ports.
		logging.Warn("listening port %s is in the dynamic/private ports range (above 49151)", listeningPort)
	} else if value == 9999 {
		logging.Fatal("listening port %s cannot be set to the local healthcheck port 9999", listeningPort)
	}
	return listeningPort
}

// GetRootURL obtains and checks the root URL from Viper (env variable or config file, etc.)
func GetRootURL() string {
	rootURL := viper.GetString("rooturl")
	if strings.ContainsAny(rootURL, " .?~#") {
		logging.Fatal("root URL %s contains invalid characters", rootURL)
	}
	rootURL = strings.ReplaceAll(rootURL, "//", "/")
	return strings.TrimSuffix(rootURL, "/") // already have / from paths of router
}

// GetDelay obtains and delay duration between each updates from Viper (env variable or config file, etc.)
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

// GetDataDir obtains and data directory from Viper (env variable or config file, etc.)
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

// GetRecordConfigs get the DNS update configurations from the environment variables RECORD1, RECORD2, ...
func GetRecordConfigs() (recordsConfigs []models.RecordConfigType) {
	var i uint64 = 1
	for {
		config := os.Getenv(fmt.Sprintf("RECORD%d", i))
		if config == "" {
			break
		}
		x := strings.Split(config, ",")
		if len(x) != 5 {
			logging.Fatal("The configuration entry %s should be in the format 'domain,host,provider,ipmethod,password'", config)
		}
		if !regex.Domain(x[0]) {
			logging.Fatal("The domain name %s is not valid for entry %s", x[0], config)
		}
		if len(x[1]) == 0 {
			logging.Fatal("The host for entry %s must have one character at least", config)
		} // TODO test when it does not exist
		if stringInAny(x[2], "duckdns", "dreamhost") && x[1] != "@" {
			logging.Fatal("The host %s can only be '@' for entry %s", x[1], config)
		}
		if !stringInAny(x[2], "namecheap", "godaddy", "duckdns", "dreamhost") {
			logging.Fatal("The DNS provider %s is not supported for entry %s", x[2], config)
		}
		if stringInAny(x[2], "namecheap", "duckdns") {
			if !stringInAny(x[3], "duckduckgo", "opendns", "provider") && regex.FindIP(x[3]) == "" {
				logging.Fatal("The IP query method %s is not valid for entry %s", x[3], config)
			}
		} else if !stringInAny(x[3], "duckduckgo", "opendns") && regex.FindIP(x[3]) == "" {
			logging.Fatal("The IP query method %s is not valid for entry %s", x[3], config)
		}
		if x[2] == "namecheap" && !regex.NamecheapPassword(x[4]) {
			logging.Fatal("The Namecheap password query parameter is not valid for entry %s", config)
		}
		if x[2] == "godaddy" && !regex.GodaddyKeySecret(x[4]) {
			logging.Fatal("The GoDaddy password (key:secret) query parameter is not valid for entry %s", config)
		}
		if x[2] == "duckdns" && !regex.DuckDNSToken(x[4]) {
			logging.Fatal("The DuckDNS password (token) query parameter is not valid for entry %s", config)
		}
		if x[2] == "dreamhost" && !regex.DreamhostKey(x[4]) {
			logging.Fatal("The Dreamhost password (key) query parameter is not valid for entry %s", config)
		}
		recordsConfigs = append(recordsConfigs, models.RecordConfigType{
			Settings: models.SettingsType{
				Domain:   x[0],
				Host:     x[1],
				Provider: x[2],
				IPmethod: x[3],
				Password: x[4],
			},
		})
		i++
	}
	if len(recordsConfigs) == 0 {
		logging.Fatal("No record to update was found in the environment variable RECORD1")
	}
	return recordsConfigs
}
