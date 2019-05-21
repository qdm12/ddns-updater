package params

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ddns-updater/pkg/logging"

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
