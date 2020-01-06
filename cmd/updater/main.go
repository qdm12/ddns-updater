package main

import (
	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/qdm12/ddns-updater/internal/database"
	"github.com/qdm12/ddns-updater/internal/env"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/router"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/healthcheck"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/server"

	"github.com/kyokomi/emoji"
	"github.com/qdm12/golibs/signals"
)

func main() {
	if healthcheck.Mode(os.Args) {
		if err := healthcheck.Query(); err != nil {
			logging.Err(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	fmt.Println("#################################")
	fmt.Println("##### DDNS Universal Updater ####")
	fmt.Println("######## by Quentin McGaw #######")
	fmt.Println("######## Give some " + emoji.Sprint(":heart:") + "at #########")
	fmt.Println("# github.com/qdm12/ddns-updater #")
	fmt.Print("#################################\n\n")
	envParams := libparams.NewEnvParams()
	encoding, level, nodeID, err := envParams.GetLoggerConfig()
	if err != nil {
		logging.Error(err.Error())
	} else {
		logging.InitLogger(encoding, level, nodeID)
	}
	var e env.Env
	HTTPTimeout, err := envParams.GetHTTPTimeout(time.Millisecond, libparams.Default("3000"))
	e.CheckError(err)
	e.Client = network.NewClient(HTTPTimeout)
	e.Gotify, err = setupGotify(envParams)
	listeningPort, err := envParams.GetListeningPort()
	e.FatalOnError(err)
	rootURL, err := envParams.GetRootURL()
	e.FatalOnError(err)
	delay, err := envParams.GetDuration("DELAY", time.Second, libparams.Default("600"))
	e.FatalOnError(err)
	dir, err := envParams.GetExeDir()
	e.FatalOnError(err)
	dataDir, err := params.GetDataDir(envParams, dir)
	e.FatalOnError(err)
	e.SQL, err = database.NewDB(dataDir)
	e.FatalOnError(err)
	defer e.SQL.Close()
	go signals.WaitForExit(e.ShutdownFromSignal)
	settings, warnings, err := params.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		e.Warn(w)
	}
	if err != nil {
		e.Fatal(err)
	}
	logging.Infof("Found %d settings to update records", len(settings))
	for _, err := range network.NewConnectivity(5 * time.Second).Checks("google.com") {
		e.Warn(err)
	}
	var recordsConfigs []models.RecordConfigType
	for _, s := range settings {
		logging.Infof("Reading history from database: domain %s host %s", s.Domain, s.Host)
		ips, tSuccess, err := e.SQL.GetIps(s.Domain, s.Host)
		if err != nil {
			e.FatalOnError(err)
		}
		recordsConfigs = append(recordsConfigs, models.NewRecordConfig(s, ips, tSuccess))
	}
	chForce := make(chan struct{})
	chQuit := make(chan struct{})
	defer close(chForce)
	go update.TriggerServer(delay, chForce, chQuit, recordsConfigs, e.Client, e.SQL, e.Gotify)
	chForce <- struct{}{}
	productionRouter := router.CreateRouter(rootURL, dir, chForce, recordsConfigs, e.Gotify)
	healthcheckRouter := healthcheck.CreateRouter(func() error {
		return router.IsHealthy(recordsConfigs)
	})
	logging.Infof("Web UI listening at address 0.0.0.0:%s with root URL %s", listeningPort, rootURL)
	if e.Gotify != nil {
		e.Gotify.Notify("DDNS Updater", 1, "Just launched\nIt has %d records to watch", len(recordsConfigs))
	}
	serverErrs := server.RunServers(
		server.Settings{Name: "production", Addr: "0.0.0.0:" + listeningPort, Handler: productionRouter},
		server.Settings{Name: "healthcheck", Addr: "127.0.0.1:9999", Handler: healthcheckRouter},
	)
	for _, err := range serverErrs {
		e.CheckError(err)
	}
	if len(serverErrs) > 0 {
		e.Fatal(serverErrs)
	}
}

func setupGotify(envParams libparams.EnvParams) (admin.Gotify, error) {
	URL, err := envParams.GetGotifyURL()
	if err != nil {
		return nil, err
	} else if URL == nil {
		return nil, nil
	}
	token, err := envParams.GetGotifyToken()
	if err != nil {
		return nil, err
	}
	return admin.NewGotify(*URL, token, &http.Client{Timeout: time.Second}), nil
}
