package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/health"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/server"
	"github.com/qdm12/ddns-updater/internal/splash"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network/connectivity"
)

var (
	buildInfo models.BuildInformation
	version   = "unknown"
	commit    = "unknown"
	buildDate = "an unknown date"
)

func main() {
	buildInfo.Version = version
	buildInfo.Commit = commit
	buildInfo.BuildDate = buildDate
	os.Exit(_main(context.Background(), time.Now))
}

type allParams struct {
	period          time.Duration
	cooldown        time.Duration
	ipMethod        models.IPMethod
	ipv4Method      models.IPMethod
	ipv6Method      models.IPMethod
	dir             string
	dataDir         string
	listeningPort   uint16
	rootURL         string
	backupPeriod    time.Duration
	backupDirectory string
}

func _main(ctx context.Context, timeNow func() time.Time) int {
	if health.IsClientMode(os.Args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		client := health.NewClient()
		if err := client.Query(ctx); err != nil {
			fmt.Println(err)
			return 1
		}
		return 0
	}

	fmt.Println(splash.Splash(buildInfo))

	// Setup logger
	paramsReader := params.NewReader(logging.New(logging.StdLog)) // use a temporary logger
	logLevel, logCaller, err := paramsReader.LoggerConfig()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	logger := logging.New(logging.StdLog, logging.SetLevel(logLevel), logging.SetCaller(logCaller))
	paramsReader = params.NewReader(logger)

	notify, err := setupGotify(paramsReader, logger)
	if err != nil {
		logger.Error(err)
		return 1
	}

	p, err := getParams(paramsReader, logger)
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}

	persistentDB, err := persistence.NewJSON(p.dataDir)
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}
	settings, warnings, err := paramsReader.JSONSettings(filepath.Join(p.dataDir, "config.json"))
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
			notify(4, err)
			return 1
		}
		records[i] = recordslib.New(s, events)
	}
	HTTPTimeout, err := paramsReader.HTTPTimeout()
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}
	client := &http.Client{Timeout: HTTPTimeout}
	defer client.CloseIdleConnections()
	db := data.NewDatabase(records, persistentDB)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(err)
		}
	}()

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	updater := update.NewUpdater(db, client, notify, logger)
	ipGetter := update.NewIPGetter(client, p.ipMethod, p.ipv4Method, p.ipv6Method)
	runner := update.NewRunner(db, updater, ipGetter, p.cooldown, logger, timeNow)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go runner.Run(ctx, p.period)

	// note: errors are logged within the goroutine,
	// no need to collect the resulting errors.
	go runner.ForceUpdate(ctx)

	const healthServerAddr = "127.0.0.1:9999"
	isHealthy := health.MakeIsHealthy(db, net.LookupIP, logger)
	healthServer := health.NewServer(healthServerAddr,
		logger.NewChild(logging.SetPrefix("healthcheck server: ")),
		isHealthy)
	wg.Add(1)
	go healthServer.Run(ctx, wg)

	address := fmt.Sprintf("0.0.0.0:%d", p.listeningPort)
	uiDir := p.dir + "/ui"
	server := server.New(ctx, address, p.rootURL, uiDir, db, logger.NewChild(logging.SetPrefix("http server: ")), runner)
	wg.Add(1)
	go server.Run(ctx, wg)
	notify(1, fmt.Sprintf("Launched with %d records to watch", len(records)))

	go backupRunLoop(ctx, p.backupPeriod, p.dir, p.backupDirectory,
		logger.NewChild(logging.SetPrefix("backup: ")), timeNow)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	select {
	case signal := <-osSignals:
		message := fmt.Sprintf("Stopping program: caught OS signal %q", signal)
		logger.Warn(message)
		notify(2, message)
		return 1
	case <-ctx.Done():
		message := fmt.Sprintf("Stopping program: %s", ctx.Err())
		logger.Warn(message)
		return 1
	}
}

func setupGotify(paramsReader params.Reader, logger logging.Logger) (
	notify func(priority int, messageArgs ...interface{}), err error) {
	gotifyURL, err := paramsReader.GotifyURL()
	if err != nil {
		return nil, err
	} else if gotifyURL == nil {
		return func(priority int, messageArgs ...interface{}) {}, nil
	}
	gotifyToken, err := paramsReader.GotifyToken()
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
	p.period, warnings, err = paramsReader.Period()
	for _, warning := range warnings {
		logger.Warn(warning)
	}
	if err != nil {
		return p, err
	}
	p.cooldown, err = paramsReader.CooldownPeriod()
	if err != nil {
		return p, err
	}
	p.ipMethod, err = paramsReader.IPMethod()
	if err != nil {
		return p, err
	}
	p.ipv4Method, err = paramsReader.IPv4Method()
	if err != nil {
		return p, err
	}
	p.ipv6Method, err = paramsReader.IPv6Method()
	if err != nil {
		return p, err
	}
	p.dir, err = paramsReader.ExeDir()
	if err != nil {
		return p, err
	}
	p.dataDir, err = paramsReader.DataDir(p.dir)
	if err != nil {
		return p, err
	}
	p.listeningPort, _, err = paramsReader.ListeningPort()
	if err != nil {
		return p, err
	}
	p.rootURL, err = paramsReader.RootURL()
	if err != nil {
		return p, err
	}
	p.backupPeriod, err = paramsReader.BackupPeriod()
	if err != nil {
		return p, err
	}
	p.backupDirectory, err = paramsReader.BackupDirectory()
	if err != nil {
		return p, err
	}
	return p, nil
}

func backupRunLoop(ctx context.Context, backupPeriod time.Duration, exeDir, outputDir string,
	logger logging.Logger, timeNow func() time.Time) {
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
