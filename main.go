package main

import (
	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ddns-updater/pkg/database"
	"ddns-updater/pkg/healthcheck"
	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/network"
	"ddns-updater/pkg/params"
	"ddns-updater/pkg/server"
	"ddns-updater/pkg/update"

	"github.com/kyokomi/emoji"
)

func init() {
	loggerMode := params.GetLoggerMode()
	logging.SetGlobalLoggerMode(loggerMode)
	nodeID := params.GetNodeID()
	logging.SetGlobalLoggerNodeID(nodeID)
	loggerLevel := params.GetLoggerLevel()
	logging.SetGlobalLoggerLevel(loggerLevel)
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
	httpClient := &http.Client{Timeout: 10 * time.Second}
	dir := params.GetDir()
	listeningPort := params.GetListeningPort()
	rootURL := params.GetRootURL()
	delay := params.GetDelay()
	dataDir := params.GetDataDir(dir)
	settings, warnings, err := params.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		logging.Warn(w)
	}
	if err != nil {
		logging.Fatal("%s", err)
	}
	logging.Info("Found %d settings to update records", len(settings))
	errs := network.ConnectivityChecks(httpClient, []string{"google.com"})
	for _, err := range errs {
		logging.Warn("%s", err)
	}
	sqlDb, err := database.NewDb(dataDir)
	if err != nil {
		logging.Fatal("%s", err)
	}
	var recordsConfigs []models.RecordConfigType
	for _, s := range settings {
		logging.Info("Reading history from database for domain and host: %s %s", s.Domain, s.Host)
		ips, tSuccess, err := sqlDb.GetIps(s.Domain, s.Host)
		if err != nil {
			logging.Fatal("%s", err)
		}
		recordsConfigs = append(recordsConfigs, models.NewRecordConfig(s, ips, tSuccess))
	}
	chForce := make(chan struct{})
	chQuit := make(chan struct{})
	defer close(chForce)
	go waitForExit(httpClient, chQuit)
	go update.TriggerServer(delay, chForce, chQuit, recordsConfigs, httpClient, sqlDb)
	chForce <- struct{}{}
	router := server.CreateRouter(rootURL, dir, chForce, recordsConfigs)
	logging.Info("Web UI listening on 0.0.0.0:%s%s", listeningPort, rootURL)
	err = http.ListenAndServe("0.0.0.0:"+listeningPort, router)
	if err != nil {
		logging.Fatal("%s", err)
	}
}

func waitForExit(httpClient *http.Client, chQuit chan struct{}) {
	signals := make(chan os.Signal)
	signal.Notify(signals,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		os.Interrupt,
	)
	signal := <-signals
	logging.Warn("Caught OS signal: %s", signal)
	logging.Info("Closing HTTP client idle connections")
	httpClient.CloseIdleConnections()
	logging.Info("Sending quit signal to goroutines")
	chQuit <- struct{}{} // this closes chQuit implicitely
	os.Exit(0)
}
