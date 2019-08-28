package healthcheck

import (
	"net/http"
	"os"
	"time"

	"ddns-updater/pkg/params"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Mode checks if the program is
// launched to run the Docker internal healthcheck.
func Mode() bool {
	args := os.Args
	if len(args) > 1 && args[1] == "healthcheck" {
		// either healthcheck mode or failure
		encoding := params.GetLoggerMode()
		nodeID := params.GetNodeID()
		config := zap.Config{
			Level:    zap.NewAtomicLevelAt(zap.InfoLevel),
			Encoding: encoding,
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				MessageKey:     "msg",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stdout"},
		}
		logger, err := config.Build()
		if err != nil {
			zap.S().Fatal(err)
		}
		logger = logger.With(zap.Int("node_id", nodeID))
		zap.ReplaceGlobals(logger)
		if len(args) > 2 {
			zap.S().Fatalf("Too many arguments provided for command healthcheck: %s", args[2:])
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
	targetURL := "http://127.0.0.1:" + listeningPort + rootURL + "/healthcheck"
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		zap.S().Fatalf("Cannot build HTTP request: %s", err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		zap.S().Fatalf("Cannot execute HTTP request: %s", err)
	}
	if response.StatusCode != 200 {
		zap.S().Fatalf("Status code is %s for %s", response.Status, targetURL)
	}
	os.Exit(0)
}
