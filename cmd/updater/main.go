package main

import (
	"context"
	"net"

	"fmt"
	"net/http"
	"os"
	"time"

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
	"github.com/qdm12/ddns-updater/internal/splash"
	"github.com/qdm12/ddns-updater/internal/trigger"
	"github.com/qdm12/ddns-updater/internal/update"
)

func main() {
	logger, err := logging.NewLogger(logging.ConsoleEncoding, logging.InfoLevel, -1)
	if err != nil {
		panic(err)
	}
	paramsReader := params.NewReader(logger)
	encoding, level, nodeID, err := paramsReader.GetLoggerConfig()
	if err != nil {
		logger.Error(err)
	} else {
		logger, err = logging.NewLogger(encoding, level, nodeID)
		if err != nil {
			panic(err)
		}
	}
	if libhealthcheck.Mode(os.Args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		if err := libhealthcheck.Query(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	fmt.Println(splash.Splash(paramsReader))
	e := env.NewEnv(logger)
	gotifyURL, err := paramsReader.GetGotifyURL()
	e.FatalOnError(err)
	if gotifyURL != nil {
		gotifyToken, err := paramsReader.GetGotifyToken()
		e.FatalOnError(err)
		e.SetGotify(admin.NewGotify(*gotifyURL, gotifyToken, &http.Client{Timeout: time.Second}))
	}
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
	var persistentDB persistence.Database
	persistentDB, err = persistence.NewJSON(dataDir)
	e.FatalOnError(err)
	go signals.WaitForExit(e.ShutdownFromSignal)
	settings, warnings, err := paramsReader.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		e.Warn(w)
	}
	if err != nil {
		e.Fatal(err)
	}
	if len(settings) > 1 {
		logger.Info("Found %d settings to update records", len(settings))
	} else if len(settings) == 1 {
		logger.Info("Found single setting to update records")
	}
	for _, err := range network.NewConnectivity(5 * time.Second).Checks("google.com") {
		e.Warn(err)
	}
	records := make([]models.Record, len(settings))
	idToPeriod := make(map[int]time.Duration)
	i := 0
	for id, setting := range settings {
		logger.Info("Reading history from database: domain %s host %s", setting.Domain, setting.Host)
		events, err := persistentDB.GetEvents(setting.Domain, setting.Host)
		if err != nil {
			e.FatalOnError(err)
		}
		records[i] = models.NewRecord(setting, events)
		idToPeriod[id] = defaultPeriod
		if setting.Delay > 0 {
			idToPeriod[id] = setting.Delay
		}
		i++
	}
	HTTPTimeout, err := paramsReader.GetHTTPTimeout()
	e.FatalOnError(err)
	client := network.NewClient(HTTPTimeout)
	db := data.NewDatabase(records, persistentDB)
	e.SetDB(db)
	updater := update.NewUpdater(db, logger, client, e.Notify)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forceUpdate := trigger.StartUpdates(ctx, updater, idToPeriod, e.CheckError)
	forceUpdate()
	productionHandlerFunc := handlers.NewHandler(rootURL, dir, db, logger, forceUpdate, e.CheckError).GetHandlerFunc()
	healthcheckHandlerFunc := libhealthcheck.GetHandler(func() error {
		return healthcheck.IsHealthy(db, net.LookupIP, logger)
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
