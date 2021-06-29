package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/config"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/health"
	"github.com/qdm12/ddns-updater/internal/models"
	jsonparams "github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/server"
	"github.com/qdm12/ddns-updater/internal/splash"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network/connectivity"
	"github.com/qdm12/golibs/params"
)

//nolint:gochecknoglobals
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
	env := params.NewEnv()
	os.Exit(_main(context.Background(), env, time.Now))
}

func _main(ctx context.Context, env params.Env, timeNow func() time.Time) int {
	if health.IsClientMode(os.Args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		client := health.NewClient()
		var healthConfig config.Health
		_, err := healthConfig.Get(env)
		if err != nil {
			fmt.Println(err)
			return 1 // TODO use healthConfig.Port
		}
		if err := client.Query(ctx, healthConfig.ServerAddress); err != nil {
			fmt.Println(err)
			return 1
		}
		return 0
	}

	fmt.Println(splash.Splash(buildInfo))

	var config config.Config
	warnings, err := config.Get(env)
	for _, warning := range warnings {
		fmt.Println(warning)
	}
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Setup logger
	loggerSettings := logging.Settings{
		Level:  config.Logger.Level,
		Caller: config.Logger.Caller}
	logger := logging.NewParent(loggerSettings)

	notify := setupGotify(config.Gotify, logger)

	persistentDB, err := persistence.NewJSON(config.Paths.DataDir)
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}

	jsonReader := jsonparams.NewReader(logger)
	settings, warnings, err := jsonReader.JSONSettings(config.Paths.JSON, logger)
	for _, w := range warnings {
		logger.Warn(w)
		notify(2, w)
	}
	if err != nil {
		logger.Error(err)
		notify(4, err)
		return 1
	}

	L := len(settings)
	switch L {
	case 0:
		logger.Warn("Found no setting to update record")
	case 1:
		logger.Info("Found single setting to update record")
	default:
		logger.Info("Found %d settings to update records", len(settings))
	}

	client := &http.Client{Timeout: config.Client.Timeout}

	connectivity := connectivity.NewConnectivity(net.DefaultResolver, client)
	for _, err := range connectivity.Checks(ctx, "github.com") {
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

	defer client.CloseIdleConnections()
	db := data.NewDatabase(records, persistentDB)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(err)
		}
	}()

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	config.PubIP.HTTPSettings.Client = client

	ipGetter, err := publicip.NewFetcher(config.PubIP.DNSSettings, config.PubIP.HTTPSettings)
	if err != nil {
		logger.Error(err)
		return 1
	}

	updater := update.NewUpdater(db, client, notify, logger)
	runner := update.NewRunner(db, updater, ipGetter, config.IPv6.Mask,
		config.Update.Cooldown, logger, timeNow)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go runner.Run(ctx, config.Update.Period)

	// note: errors are logged within the goroutine,
	// no need to collect the resulting errors.
	go runner.ForceUpdate(ctx)

	isHealthy := health.MakeIsHealthy(db, net.LookupIP, logger)
	healthServer := health.NewServer(config.Health.ServerAddress,
		logger.NewChild(logging.Settings{Prefix: "healthcheck server: "}),
		isHealthy)
	wg.Add(1)
	go healthServer.Run(ctx, wg)

	address := ":" + strconv.Itoa(int(config.Server.Port))
	serverLogger := logger.NewChild(logging.Settings{Prefix: "http server: "})
	server := server.New(ctx, address, config.Server.RootURL, db, serverLogger, runner)
	wg.Add(1)
	go server.Run(ctx, wg)
	notify(1, fmt.Sprintf("Launched with %d records to watch", len(records)))

	go backupRunLoop(ctx, config.Backup.Period, config.Paths.DataDir, config.Backup.Directory,
		logger.NewChild(logging.Settings{Prefix: "backup: "}), timeNow)

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

func setupGotify(config config.Gotify, logger logging.Logger) (
	notify func(priority int, messageArgs ...interface{})) {
	if config.URL == nil {
		return func(priority int, messageArgs ...interface{}) {}
	}
	gotify := admin.NewGotify(*config.URL, config.Token, &http.Client{Timeout: time.Second})
	return func(priority int, messageArgs ...interface{}) {
		if err := gotify.Notify("DDNS Updater", priority, messageArgs...); err != nil {
			logger.Error(err)
		}
	}
}

func backupRunLoop(ctx context.Context, backupPeriod time.Duration, dataDir, outputDir string,
	logger logging.Logger, timeNow func() time.Time) {
	if backupPeriod == 0 {
		logger.Info("disabled")
		return
	}
	logger.Info("each %s; writing zip files to directory %s", backupPeriod, outputDir)
	ziper := backup.NewZiper()
	timer := time.NewTimer(backupPeriod)
	for {
		fileName := "ddns-updater-backup-" + strconv.Itoa(int(timeNow().UnixNano())) + ".zip"
		zipFilepath := filepath.Join(outputDir, fileName)
		if err := ziper.ZipFiles(
			zipFilepath,
			filepath.Join(dataDir, "updates.json"),
			filepath.Join(dataDir, "config.json"),
		); err != nil {
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
