package main

import (
	"context"
	"net"

	_ "github.com/mattn/go-sqlite3"

	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kyokomi/emoji"
	"github.com/qdm12/golibs/admin"
	libhealthcheck "github.com/qdm12/golibs/healthcheck"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/server"
	"github.com/qdm12/golibs/signals"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/env"
	"github.com/qdm12/ddns-updater/internal/handlers"
	"github.com/qdm12/ddns-updater/internal/healthcheck"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence"
	libtrigger "github.com/qdm12/ddns-updater/internal/trigger"
	"github.com/qdm12/ddns-updater/internal/update"
)

func main() {
	logger, err := logging.NewLogger(logging.ConsoleEncoding, logging.InfoLevel, -1)
	if err != nil {
		panic(err)
	}
	paramsReader := params.NewParamsReader(logger)
	encoding, level, nodeID, err := paramsReader.GetLoggerConfig()
	if err != nil {
		logger.Error(err)
	} else {
		logger, err = logging.NewLogger(encoding, level, nodeID)
	}
	if libhealthcheck.Mode(os.Args) {
		if err := libhealthcheck.Query(); err != nil {
			logger.Error(err)
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
	e := env.NewEnv(logger)
	gotify, err := setupGotify(paramsReader)
	e.FatalOnError(err)
	e.SetGotify(gotify)
	listeningPort, warning, err := paramsReader.GetListeningPort()
	e.FatalOnError(err)
	if len(warning) > 0 {
		logger.Warn(warning)
	}
	rootURL, err := paramsReader.GetRootURL()
	e.FatalOnError(err)
	defaultPeriod, err := paramsReader.GetDelay(libparams.Default("10m"))
	e.FatalOnError(err)
	dir, err := paramsReader.GetExeDir()
	e.FatalOnError(err)
	dataDir, err := paramsReader.GetDataDir(dir)
	e.FatalOnError(err)
	persistentDB, err := persistence.NewSQLite(dataDir)
	e.FatalOnError(err)
	go signals.WaitForExit(e.ShutdownFromSignal)
	settings, warnings, err := paramsReader.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		e.Warn(w)
	}
	if err != nil {
		e.Fatal(err)
	}
	logger.Info("Found %d settings to update records", len(settings))
	for _, err := range network.NewConnectivity(5 * time.Second).Checks("google.com") {
		e.Warn(err)
	}
	var records []models.Record
	idToPeriod := make(map[int]time.Duration)
	for id, setting := range settings {
		logger.Info("Reading history from database: domain %s host %s", setting.Domain, setting.Host)
		ips, tSuccess, err := persistentDB.GetIPs(setting.Domain, setting.Host)
		if err != nil {
			e.FatalOnError(err)
		}
		records = append(records, models.NewRecord(setting, ips, tSuccess))
		idToPeriod[id] = setting.Delay
	}
	HTTPTimeout, err := paramsReader.GetHTTPTimeout()
	e.FatalOnError(err)
	client := network.NewClient(HTTPTimeout)
	db := data.NewDatabase(records, persistentDB)
	e.SetDb(db)
	updater := update.NewUpdater(db, logger, client, gotify)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	trigger := libtrigger.NewTrigger(defaultPeriod, idToPeriod, updater)
	go trigger.Run(ctx, func(err error) {
		e.CheckError(err)
	})
	for _, err := range trigger.Force() {
		e.CheckError(err)
	}
	productionHandlerFunc := handlers.NewHandler(rootURL, dir, db, logger, trigger.Force, e.CheckError).GetHandlerFunc()
	healthcheckHandlerFunc := libhealthcheck.GetHandler(func() error {
		return healthcheck.IsHealthy(db, net.LookupIP)
	})
	logger.Info("Web UI listening at address 0.0.0.0:%s with root URL %s", listeningPort, rootURL)
	e.Notify(1, fmt.Sprintf("Just launched\nIt has %d records to watch", len(records)))
	serverErrs := server.RunServers(
		server.Settings{Name: "production", Addr: "0.0.0.0:" + listeningPort, Handler: productionHandlerFunc},
		server.Settings{Name: "healthcheck", Addr: "127.0.0.1:9999", Handler: healthcheckHandlerFunc},
	)
	if len(serverErrs) > 0 {
		e.Fatal(serverErrs)
	}
}

func setupGotify(paramsReader params.ParamsReader) (admin.Gotify, error) {
	URL, err := paramsReader.GetGotifyURL()
	if err != nil {
		return nil, err
	} else if URL == nil {
		return nil, nil
	}
	token, err := paramsReader.GetGotifyToken()
	if err != nil {
		return nil, err
	}
	return admin.NewGotify(*URL, token, &http.Client{Timeout: time.Second}), nil
}
