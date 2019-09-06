package params

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
		zap.S().Fatalf("delay %s is not a valid integer", delayStr)
	}
	if delayInt < 10 {
		zap.S().Fatalf("delay %d must be bigger than 10 seconds", delayInt)
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
		zap.S().Fatal(err)
	}
	return filepath.Dir(ex)
}

// GetLoggerMode obtains the logging mode from Viper (env variable or config file, etc.)
func GetLoggerMode() string {
	s := viper.GetString("logging")
	s = strings.ToLower(s)
	switch s {
	case "json":
		return "json" // zap style encoding
	case "human":
		return "console"
	case "console":
		return "console"
	case "":
		return "json"
	default:
		// uses the global logger
		zap.S().Warnf("Unrecognized logging mode %s", s)
		return "json"
	}
}

// GetLoggerLevel obtains the logging level from Viper (env variable or config file, etc.)
func GetLoggerLevel() zapcore.Level {
	s := viper.GetString("loglevel")
	s = strings.ToLower(s)
	switch s {
	case "info":
		return zap.InfoLevel
	case "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "":
		return zap.InfoLevel
	default:
		zap.S().Warn("Unrecognized logging level %s", s)
		return zap.InfoLevel
	}
}

// GetNodeID obtains the node instance ID from Viper (env variable or config file, etc.)
func GetNodeID() int {
	nodeID := viper.GetString("nodeid")
	value, err := strconv.Atoi(nodeID)
	if err != nil {
		zap.S().Fatalf("Node ID %s is not a valid integer", nodeID)
	}
	return value
}

// GetGotifyURL obtains the URL to the Gotify server
func GetGotifyURL() (URL *url.URL) {
	s := viper.GetString("gotifyurl")
	if s == "" {
		return nil
	}
	URL, err := url.Parse(s)
	if err != nil {
		zap.S().Fatalf("URL %s is not valid", s)
	}
	return URL
}

// GetGotifyToken obtains the token for the app on the Gotify server
func GetGotifyToken() (token string) {
	token = viper.GetString("gotifytoken")
	return token
}

func stringInAny(s string, ss ...string) bool {
	for _, x := range ss {
		if s == x {
			return true
		}
	}
	return false
}
