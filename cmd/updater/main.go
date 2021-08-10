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
	"syscall"
	"time"
	_ "time/tzdata"

	_ "github.com/breml/rootcerts"
	"github.com/containrrr/shoutrrr"
	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/config"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/health"
	"github.com/qdm12/ddns-updater/internal/models"
	jsonparams "github.com/qdm12/ddns-updater/internal/params"
	"github.com/qdm12/ddns-updater/internal/persistence"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/server"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network/connectivity"
	"github.com/qdm12/golibs/params"
	"github.com/qdm12/goshutdown"
	"github.com/qdm12/gosplash"
)

//nolint:gochecknoglobals
var (
	version   = "unknown"
	commit    = "unknown"
	buildDate = "an unknown date"
)

func main() {
	buildInfo := models.BuildInformation{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	}
	env := params.NewEnv()
	logger := logging.NewParent(logging.Settings{Writer: os.Stdout})

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	ctx, cancel := context.WithCancel(ctx)

	errorCh := make(chan error)
	go func() {
		errorCh <- _main(ctx, env, os.Args, logger, buildInfo, time.Now)
	}()

	select {
	case <-ctx.Done():
		stop()
		logger.Warn("Caught OS signal, shutting down")
	case err := <-errorCh:
		stop()
		close(errorCh)
		if err == nil { // expected exit such as healthcheck
			os.Exit(0)
		}
		logger.Error(err)
		cancel()
	}

	const shutdownGracePeriod = 5 * time.Second
	timer := time.NewTimer(shutdownGracePeriod)
	select {
	case err := <-errorCh:
		if !timer.Stop() {
			<-timer.C
		}
		if err != nil {
			logger.Error(err)
		}
		logger.Info("Shutdown successful")
	case <-timer.C:
		logger.Warn("Shutdown timed out")
	}

	os.Exit(1)
}

func _main(ctx context.Context, env params.Env, args []string, logger logging.ParentLogger,
	buildInfo models.BuildInformation, timeNow func() time.Time) (err error) {
	if health.IsClientMode(args) {
		// Running the program in a separate instance through the Docker
		// built-in healthcheck, in an ephemeral fashion to query the
		// long running instance of the program about its status
		client := health.NewClient()
		var healthConfig config.Health
		_, err := healthConfig.Get(env)
		if err != nil {
			return err
		}
		if err := client.Query(ctx, healthConfig.Port); err != nil {
			return err
		}
		return nil
	}

	announcementExp, err := time.Parse(time.RFC3339, "2021-07-22T00:00:00Z")
	if err != nil {
		return err
	}
	splashSettings := gosplash.Settings{
		User:         "qdm12",
		Repository:   "gluetun",
		Emails:       []string{"quentin.mcgaw@gmail.com"},
		Version:      buildInfo.Version,
		Commit:       buildInfo.Commit,
		BuildDate:    buildInfo.BuildDate,
		Announcement: "",
		AnnounceExp:  announcementExp,
		// Sponsor information
		PaypalUser:    "qmcgaw",
		GithubSponsor: "qdm12",
	}
	for _, line := range gosplash.MakeLines(splashSettings) {
		fmt.Println(line)
	}

	var config config.Config
	warnings, err := config.Get(env)
	for _, warning := range warnings {
		logger.Warn(warning)
	}
	if err != nil {
		return err
	}

	// Setup logger
	loggerSettings := logging.Settings{
		Level:  config.Logger.Level,
		Caller: config.Logger.Caller}
	logger = logging.NewParent(loggerSettings)

	sender, err := shoutrrr.CreateSender(config.Shoutrrr.Addresses...)
	if err != nil {
		return err
	}
	notify := func(message string) {
		errs := sender.Send(message, &config.Shoutrrr.Params)
		for _, err := range errs {
			if err != nil {
				logger.Error(err.Error())
			}
		}
	}

	persistentDB, err := persistence.NewJSON(config.Paths.DataDir)
	if err != nil {
		notify(err.Error())
		return err
	}

	jsonReader := jsonparams.NewReader(logger)
	settings, warnings, err := jsonReader.JSONSettings(config.Paths.JSON, logger)
	for _, w := range warnings {
		logger.Warn(w)
		notify(w)
	}
	if err != nil {
		notify(err.Error())
		return err
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
			notify(err.Error())
			return err
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

	config.PubIP.HTTPSettings.Client = client

	ipGetter, err := publicip.NewFetcher(config.PubIP.DNSSettings, config.PubIP.HTTPSettings)
	if err != nil {
		return err
	}

	updater := update.NewUpdater(db, client, notify, logger)
	runner := update.NewRunner(db, updater, ipGetter, config.Update.Period,
		config.IPv6.Mask, config.Update.Cooldown, logger, timeNow)

	runnerHandler, runnerCtx, runnerDone := goshutdown.NewGoRoutineHandler(
		"runner", goshutdown.GoRoutineSettings{})
	go runner.Run(runnerCtx, runnerDone)

	// note: errors are logged within the goroutine,
	// no need to collect the resulting errors.
	go runner.ForceUpdate(ctx)

	isHealthy := health.MakeIsHealthy(db, net.LookupIP, logger)
	healthServer := health.NewServer(config.Health.ServerAddress,
		logger.NewChild(logging.Settings{Prefix: "healthcheck server: "}),
		isHealthy)
	healthServerHandler, healthServerCtx, healthServerDone := goshutdown.NewGoRoutineHandler(
		"health server", goshutdown.GoRoutineSettings{})
	go healthServer.Run(healthServerCtx, healthServerDone)

	address := ":" + strconv.Itoa(int(config.Server.Port))
	serverLogger := logger.NewChild(logging.Settings{Prefix: "http server: "})
	server := server.New(ctx, address, config.Server.RootURL, db, serverLogger, runner)
	serverHandler, serverCtx, serverDone := goshutdown.NewGoRoutineHandler(
		"server", goshutdown.GoRoutineSettings{})
	go server.Run(serverCtx, serverDone)
	notify("Launched with " + strconv.Itoa(len(records)) + " records to watch")

	backupHandler, backupCtx, backupDone := goshutdown.NewGoRoutineHandler(
		"backup", goshutdown.GoRoutineSettings{})
	go backupRunLoop(backupCtx, backupDone, config.Backup.Period, config.Paths.DataDir, config.Backup.Directory,
		logger.NewChild(logging.Settings{Prefix: "backup: "}), timeNow)

	shutdownGroup := goshutdown.NewGroupHandler("", goshutdown.GroupSettings{})
	shutdownGroup.Add(runnerHandler, healthServerHandler, serverHandler, backupHandler)

	<-ctx.Done()

	if err := shutdownGroup.Shutdown(context.Background()); err != nil {
		notify(err.Error())
		return err
	}
	return nil
}

func backupRunLoop(ctx context.Context, done chan<- struct{}, backupPeriod time.Duration,
	dataDir, outputDir string, logger logging.Logger, timeNow func() time.Time) {
	defer close(done)
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
