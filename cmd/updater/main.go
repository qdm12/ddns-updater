package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/handlers"
	"github.com/qdm12/ddns-updater/internal/healthcheck"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/splash"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/golibs/admin"
	libhealthcheck "github.com/qdm12/golibs/healthcheck"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/network/connectivity"
	"github.com/qdm12/golibs/server"
)

func main() {
	os.Exit(_main(context.Background(), time.Now))
	// returns 1 on error
	// returns 2 on os signal
}

type allParams struct {
	period          time.Duration
	ipMethod        models.IPMethod
	ipv4Method      models.IPMethod
	ipv6Method      models.IPMethod
	dir             string
	dataDir         string
	listeningPort   string
	rootURL         string
	backupPeriod    time.Duration
	backupDirectory string
}

func _main(ctx context.Context, timeNow func() time.Time) int {
	if libhealthcheck.Mode(os.Args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		if err := libhealthcheck.Query(ctx); err != nil {
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

	p, err := getParams(paramsReader, logger)
	if err != nil {
		logger.Error(err)
		notify(4, err) //nolint:gomnd
		return 1
	}

	persistentDB, err := persistence.NewJSON(p.dataDir)
	if err != nil {
		logger.Error(err)
		notify(4, err) //nolint:gomnd
		return 1
	}
	settings, warnings, err := paramsReader.GetSettings(p.dataDir + "/config.json")
	for _, w := range warnings {
		logger.Warn(w)
		notify(2, w) //nolint:gomnd
	}
	if err != nil {
		logger.Error(err)
		notify(4, err) //nolint:gomnd
		return 1
	}
	if len(settings) > 1 {
		logger.Info("Found %d settings to update records", len(settings))
	} else if len(settings) == 1 {
		logger.Info("Found single setting to update record")
	}
	const connectivyCheckTimeout = 5 * time.Second
	for _, err := range connectivity.NewConnectivity(connectivyCheckTimeout).
		Checks(ctx, "google.com") {
		logger.Warn(err)
	}
	records := make([]recordslib.Record, len(settings))
	for i, s := range settings {
		logger.Info("Reading history from database: domain %s host %s", s.Domain(), s.Host())
		events, err := persistentDB.GetEvents(s.Domain(), s.Host())
		if err != nil {
			logger.Error(err)
			notify(4, err) //nolint:gomnd
			return 1
		}
		records[i] = recordslib.New(s, events)
	}
	HTTPTimeout, err := paramsReader.GetHTTPTimeout()
	if err != nil {
		logger.Error(err)
		notify(4, err) //nolint:gomnd
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
	updater := update.NewUpdater(db, client, notify)
	ipGetter := update.NewIPGetter(client, p.ipMethod, p.ipv4Method, p.ipv6Method)
	runner := update.NewRunner(db, updater, ipGetter, logger, timeNow)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	forceUpdate := runner.Run(ctx, p.period)
	forceUpdate()
	productionHandlerFunc := handlers.MakeHandler(p.rootURL, p.dir+"/ui", db, logger, forceUpdate, timeNow)
	healthcheckHandlerFunc := libhealthcheck.GetHandler(func() error {
		return healthcheck.IsHealthy(db, net.LookupIP, logger)
	})
	logger.Info("Web UI listening at address 0.0.0.0:%s with root URL %q", p.listeningPort, p.rootURL)
	notify(1, fmt.Sprintf("Launched with %d records to watch", len(records)))
	serverErrors := make(chan []error)
	go func() {
		serverErrors <- server.RunServers(ctx,
			server.Settings{Name: "production", Addr: "0.0.0.0:" + p.listeningPort, Handler: productionHandlerFunc},
			server.Settings{Name: "healthcheck", Addr: "127.0.0.1:9999", Handler: healthcheckHandlerFunc},
		)
	}()

	go backupRunLoop(ctx, p.backupPeriod, p.dir, p.backupDirectory, logger, timeNow)

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
		notify(2, message) //nolint:gomnd
		return 2           //nolint:gomnd
	case <-ctx.Done():
		message := fmt.Sprintf("Stopping program: %s", ctx.Err())
		logger.Warn(message)
		return 1
	}
}

func setupLogger() (logging.Logger, error) {
	paramsReader := params.NewReader(nil)
	encoding, level, err := paramsReader.GetLoggerConfig()
	if err != nil {
		return nil, err
	}
	return logging.NewLogger(encoding, level)
}

func setupGotify(paramsReader params.Reader, logger logging.Logger) (
	notify func(priority int, messageArgs ...interface{}), err error) {
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

func getParams(paramsReader params.Reader, logger logging.Logger) (p allParams, err error) {
	var warnings []string
	p.period, warnings, err = paramsReader.GetPeriod()
	for _, warning := range warnings {
		logger.Warn(warning)
	}
	if err != nil {
		return p, err
	}
	p.ipMethod, err = paramsReader.GetIPMethod()
	if err != nil {
		return p, err
	}
	p.ipv4Method, err = paramsReader.GetIPv4Method()
	if err != nil {
		return p, err
	}
	p.ipv6Method, err = paramsReader.GetIPv6Method()
	if err != nil {
		return p, err
	}
	p.dir, err = paramsReader.GetExeDir()
	if err != nil {
		return p, err
	}
	p.dataDir, err = paramsReader.GetDataDir(p.dir)
	if err != nil {
		return p, err
	}
	p.listeningPort, _, err = paramsReader.GetListeningPort()
	if err != nil {
		return p, err
	}
	p.rootURL, err = paramsReader.GetRootURL()
	if err != nil {
		return p, err
	}
	p.backupPeriod, err = paramsReader.GetBackupPeriod()
	if err != nil {
		return p, err
	}
	p.backupDirectory, err = paramsReader.GetBackupDirectory()
	if err != nil {
		return p, err
	}
	return p, nil
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
