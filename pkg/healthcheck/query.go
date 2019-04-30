package healthcheck

import (
	"net/http"
	"os"
	"time"

	"ddns-updater/pkg/params"
	"ddns-updater/pkg/logging"
)

// Mode checks if the program is
// launched to run the Docker internal healthcheck.
func Mode() bool {
	args := os.Args
	if len(args) > 1 && args[1] == "healthcheck" {
		// either healthcheck mode or failure
		nodeID := params.GetNodeID()
		logging.SetGlobalLoggerNodeID(nodeID)
		loggerMode := params.GetLoggerMode()
		logging.SetGlobalLoggerMode(loggerMode)
		// we don't care about the logger level as it will only be Fatal
		if len(args) > 2 {
			logging.Fatal("Too many arguments provided for command healthcheck: %s", args[2:])
		}
		return true
	}
	return false
}

// Query sends an HTTP request to
// the other instance of the program's healthcheck
// server route.
func Query() {
	rootURL := params.GetRootURL()
	listeningPort := params.GetListeningPort()
	targetURL := "http://127.0.0.1:"+listeningPort+rootURL+"/healthcheck"
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		logging.Fatal("Cannot build HTTP request: %s", err)
	}
	client := &http.Client{Timeout: time.Duration(1000) * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		logging.Fatal("Cannot execute HTTP request: %s", err)
	}
	if response.StatusCode != 200 {
		logging.Fatal("Status code is %s for %s", response.Status, targetURL)
	}
	os.Exit(0)
}
