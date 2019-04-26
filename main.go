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
	"ddns-updater/pkg/params"
	"ddns-updater/pkg/update"
	"ddns-updater/pkg/network"
	"ddns-updater/pkg/server"

	"github.com/kyokomi/emoji"
)

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
	go waitForExit()
	logging.SetGlobalLoggerLevel(logging.InfoLevel)
	loggerMode := params.GetLoggerMode()
	logging.SetGlobalLoggerMode(loggerMode)	
	nodeID := params.GetNodeID()
	logging.SetGlobalLoggerNodeID(nodeID)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	dir := params.GetDir()
	listeningPort := params.GetListeningPort()
	rootURL := params.GetRootURL()
	delay := params.GetDelay()
	recordsConfigs := params.GetRecordConfigs()
	logging.Info("Found %d records to update", len(recordsConfigs))
	go healthcheck.Serve(recordsConfigs)
	dataDir := params.GetDataDir(dir)
	errs := network.ConnectivityChecks(httpClient, []string{"google.com"})
	for _, err := range errs {
		logging.Warn("%s", err)
	}
	sqlDb, err := database.NewDb(dataDir)
	if err != nil {
		logging.Fatal("%s", err)
	}
	for i := range recordsConfigs {
		domain := recordsConfigs[i].Settings.Domain
		host := recordsConfigs[i].Settings.Host
		logging.Info("Reading history for domain %s and host %s", domain, host)
		ips, tSuccess, err := sqlDb.GetIps(domain, host)
		if err != nil {
			logging.Fatal("%s", err)
		}
		recordsConfigs[i].Lock()
		recordsConfigs[i].History.IPs = ips
		recordsConfigs[i].History.TSuccess = tSuccess
		recordsConfigs[i].Unlock()
	}
	forceCh := make(chan struct{})
	quitCh := make(chan struct{})
	go update.TriggerServer(delay, forceCh, quitCh, recordsConfigs, httpClient, sqlDb)
	forceCh <- struct{}{}
	router := server.CreateRouter(rootURL, dir, forceCh, recordsConfigs)
	logging.Info("Web UI listening on 0.0.0.0:%s", listeningPort)
	err = http.ListenAndServe("0.0.0.0:"+listeningPort, router)
	if err != nil {
		logging.Fatal("%s", err)
	}
}

func waitForExit() {
	signals := make(chan os.Signal)
	signal.Notify(signals,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		os.Interrupt,
	)
	signal := <-signals
	logging.Warn("Caught OS signal: %s", signal)
	os.Exit(0)
}
