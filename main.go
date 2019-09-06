package main

import (
	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ddns-updater/pkg/admin"
	"ddns-updater/pkg/database"
	"ddns-updater/pkg/healthcheck"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/network"
	"ddns-updater/pkg/params"
	"ddns-updater/pkg/server"
	"ddns-updater/pkg/update"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kyokomi/emoji"
)

func init() {
	encoding := params.GetLoggerMode()
	level := params.GetLoggerLevel()
	nodeID := params.GetNodeID()
	config := zap.Config{
		Level:    zap.NewAtomicLevelAt(level),
		Encoding: encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
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
}

func main() {
	if healthcheck.Mode() {
		healthcheck.Query()
	}
	fmt.Println("#################################")
	fmt.Println("##### DDNS Universal Updater ####")
	fmt.Println("######## by Quentin McGaw #######")
	fmt.Println("######## Give some " + emoji.Sprint(":heart:") + "at #########")
	fmt.Println("# github.com/qdm12/ddns-updater #")
	fmt.Print("#################################\n\n")
	httpClient := &http.Client{Timeout: time.Second}
	dir := params.GetDir()
	listeningPort := params.GetListeningPort()
	rootURL := params.GetRootURL()
	delay := params.GetDelay()
	dataDir := params.GetDataDir(dir)
	settings, warnings, err := params.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		zap.S().Warn(w)
	}
	if err != nil {
		zap.S().Fatal(err)
	}
	zap.S().Infof("Found %d settings to update records", len(settings))
	gotifyURL := params.GetGotifyURL()
	gotifyToken := params.GetGotifyToken()
	gotify, err := admin.NewGotify(gotifyURL, gotifyToken, httpClient)
	if err != nil {
		zap.S().Warn("Gotify not activated: %s", err)
	}
	httpClient.Timeout = 10 * time.Second
	errs := network.ConnectivityChecks(httpClient, []string{"google.com"})
	for _, err := range errs {
		zap.S().Warn(err)
	}
	sqlDb, err := database.NewDb(dataDir)
	if err != nil {
		zap.S().Fatal(err)
	}
	var recordsConfigs []models.RecordConfigType
	for _, s := range settings {
		zap.S().Infof("Reading history from database for domain and host: %s %s", s.Domain, s.Host)
		ips, tSuccess, err := sqlDb.GetIps(s.Domain, s.Host)
		if err != nil {
			zap.S().Fatal(err)
		}
		recordsConfigs = append(recordsConfigs, models.NewRecordConfig(s, ips, tSuccess))
	}
	chForce := make(chan struct{})
	chQuit := make(chan struct{})
	defer close(chForce)
	go waitForExit(httpClient, chQuit, gotify)
	go update.TriggerServer(delay, chForce, chQuit, recordsConfigs, httpClient, sqlDb, gotify)
	chForce <- struct{}{}
	router := server.CreateRouter(rootURL, dir, chForce, recordsConfigs, gotify)
	zap.S().Infof("Web UI listening on 0.0.0.0:%s%s", listeningPort, rootURL)
	gotify.Notify("DDNS Updater", 1, "Just launched\nIt has %d records to watch", len(recordsConfigs))
	err = http.ListenAndServe("0.0.0.0:"+listeningPort, router)
	if err != nil {
		zap.S().Fatal(err)
	}
}

func waitForExit(httpClient *http.Client, chQuit chan struct{}, gotify *admin.Gotify) {
	signals := make(chan os.Signal)
	signal.Notify(signals,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		os.Interrupt,
	)
	signal := <-signals
	zap.S().Warnf("Caught OS signal: %s", signal)
	gotify.Notify("DDNS Updater", 4, "Caught OS signal: %s", signal)
	zap.S().Info("Closing HTTP client idle connections")
	httpClient.CloseIdleConnections()
	zap.S().Info("Sending quit signal to goroutines")
	chQuit <- struct{}{} // this closes chQuit implicitely
	os.Exit(0)
}
