package main

import (
	"context"
	"net"
	"os/signal"
	"syscall"

	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/qdm12/golibs/admin"
	libhealthcheck "github.com/qdm12/golibs/healthcheck"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/network/connectivity"
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/server"

	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/data"
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
	os.Exit(_main(context.Background(), time.Now))
	// returns 1 on error
	// returns 2 on os signal
}

func _main(ctx context.Context, timeNow func() time.Time) int {
	if libhealthcheck.Mode(os.Args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		if err := libhealthcheck.Query(); err != nil {
			fmt.Println(err)
			return 1
		}
		return 0
	}
	logger, err := setupLogger()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	paramsReader := params.NewReader(logger)

	fmt.Println(splash.Splash(
		paramsReader.GetVersion(),
		paramsReader.GetVcsRef(),
		paramsReader.GetBuildDate()))

	notify, err := setupGotify(paramsReader, logger)
	if err != nil {
		logger.Error(err)
		return 1
	}

	dir, dataDir, listeningPort, rootURL, defaultPeriod, backupPeriod, backupDirectory, err := getParams(paramsReader)
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}

	persistentDB, err := persistence.NewJSON(dataDir)
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}
	settings, warnings, err := paramsReader.GetSettings(dataDir + "/config.json")
	for _, w := range warnings {
		logger.Warn(w)
		notify(2, w)
	}
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}
	if len(settings) > 1 {
		logger.Info("Found %d settings to update records", len(settings))
	} else if len(settings) == 1 {
		logger.Info("Found single setting to update records")
	}
	for _, err := range connectivity.NewConnectivity(5 * time.Second).Checks("google.com") {
		logger.Warn(err)
	}
	records := make([]models.Record, len(settings))
	idToPeriod := make(map[int]time.Duration)
	i := 0
	for id, setting := range settings {
		logger.Info("Reading history from database: domain %s host %s", setting.Domain, setting.Host)
		events, err := persistentDB.GetEvents(setting.Domain, setting.Host)
		if err != nil {
			logger.Error(err)
			notify(4, err)
			return 1
		}
		records[i] = models.NewRecord(setting, events)
		idToPeriod[id] = defaultPeriod
		if setting.Delay > 0 {
			idToPeriod[id] = setting.Delay
		}
		i++
	}
	HTTPTimeout, err := paramsReader.GetHTTPTimeout()
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}
	client := network.NewClient(HTTPTimeout)
	defer client.Close()
	db := data.NewDatabase(records, persistentDB)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(err)
		}
	}()
	updater := update.NewUpdater(db, logger, client, notify)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	checkError := func(err error) {
		if err != nil {
			logger.Error(err)
		}
	}
	forceUpdate := trigger.StartUpdates(ctx, updater, idToPeriod, checkError)
	forceUpdate()
	productionHandlerFunc := handlers.NewHandler(rootURL, dir, db, logger, forceUpdate, checkError).GetHandlerFunc()
	healthcheckHandlerFunc := libhealthcheck.GetHandler(func() error {
		return healthcheck.IsHealthy(db, net.LookupIP, logger)
	})
	logger.Info("Web UI listening at address 0.0.0.0:%s with root URL %s", listeningPort, rootURL)
	notify(1, fmt.Sprintf("Launched with %d records to watch", len(records)))
	serverErrors := make(chan []error)
	go func() {
		serverErrors <- server.RunServers(ctx,
			server.Settings{Name: "production", Addr: "0.0.0.0:" + listeningPort, Handler: productionHandlerFunc},
			server.Settings{Name: "healthcheck", Addr: "127.0.0.1:9999", Handler: healthcheckHandlerFunc},
		)
	}()

	go backupRunLoop(ctx, backupPeriod, dir, backupDirectory, logger, timeNow)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	select {
	case errors := <-serverErrors:
		for _, err := range errors {
			logger.Error(err)
		}
		return 1
	case signal := <-osSignals:
		message := fmt.Sprintf("Stopping program: caught OS signal %q", signal)
		logger.Warn(message)
		notify(2, message)
		return 2
	case <-ctx.Done():
		message := fmt.Sprintf("Stopping program: %s", ctx.Err())
		logger.Warn(message)
		return 1
	}
}

func setupLogger() (logging.Logger, error) {
	paramsReader := params.NewReader(nil)
	encoding, level, nodeID, err := paramsReader.GetLoggerConfig()
	if err != nil {
		return nil, err
	}
	return logging.NewLogger(encoding, level, nodeID)
}

func setupGotify(paramsReader params.Reader, logger logging.Logger) (notify func(priority int, messageArgs ...interface{}), err error) {
	gotifyURL, err := paramsReader.GetGotifyURL()
	if err != nil {
		return nil, err
	} else if gotifyURL == nil {
		return func(priority int, messageArgs ...interface{}) {}, nil
	}
	gotifyToken, err := paramsReader.GetGotifyToken()
	if err != nil {
		return nil, err
	}
	gotify := admin.NewGotify(*gotifyURL, gotifyToken, &http.Client{Timeout: time.Second})
	return func(priority int, messageArgs ...interface{}) {
		if err := gotify.Notify("DDNS Updater", priority, messageArgs...); err != nil {
			logger.Error(err)
		}
	}, nil
}

func getParams(paramsReader params.Reader) (
	dir, dataDir,
	listeningPort, rootURL string,
	defaultPeriod time.Duration,
	backupPeriod time.Duration, backupDirectory string,
	err error) {
	dir, err = paramsReader.GetExeDir()
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	dataDir, err = paramsReader.GetDataDir(dir)
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	listeningPort, _, err = paramsReader.GetListeningPort()
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	rootURL, err = paramsReader.GetRootURL()
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	defaultPeriod, err = paramsReader.GetDelay(libparams.Default("10m"))
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}

	backupPeriod, err = paramsReader.GetBackupPeriod()
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	backupDirectory, err = paramsReader.GetBackupDirectory()
	if err != nil {
		return "", "", "", "", 0, 0, "", err
	}
	return dir, dataDir, listeningPort, rootURL, defaultPeriod, backupPeriod, backupDirectory, nil
}

func backupRunLoop(ctx context.Context, backupPeriod time.Duration, exeDir, outputDir string,
	logger logging.Logger, timeNow func() time.Time) {
	logger = logger.WithPrefix("backup: ")
	if backupPeriod == 0 {
		logger.Info("disabled")
		return
	}
	logger.Info("each %s; writing zip files to directory %s", backupPeriod, outputDir)
	ziper := backup.NewZiper()
	timer := time.NewTimer(backupPeriod)
	for {
		filepath := fmt.Sprintf("%s/ddns-updater-backup-%d.zip", outputDir, timeNow().UnixNano())
		if err := ziper.ZipFiles(
			filepath,
			fmt.Sprintf("%s/data/updates.json", exeDir),
			fmt.Sprintf("%s/data/config.json", exeDir)); err != nil {
			logger.Error(err)
		}
		select {
		case <-timer.C:
			timer.Reset(backupPeriod)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}
