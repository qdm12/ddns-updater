package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "github.com/breml/rootcerts"
	"github.com/qdm12/ddns-updater/internal/backup"
	"github.com/qdm12/ddns-updater/internal/config"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/health"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	jsonparams "github.com/qdm12/ddns-updater/internal/params"
	persistence "github.com/qdm12/ddns-updater/internal/persistence/json"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/resolver"
	"github.com/qdm12/ddns-updater/internal/server"
	"github.com/qdm12/ddns-updater/internal/shoutrrr"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/goshutdown"
	"github.com/qdm12/gosplash"
	"github.com/qdm12/log"
)

//nolint:gochecknoglobals
var (
	version = "unknown"
	commit  = "unknown"
	date    = "an unknown date"
)

func main() {
	buildInfo := models.BuildInformation{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
	logger := log.New()

	reader := reader.New(reader.Settings{
		HandleDeprecatedKey: func(source, oldKey, newKey string) {
			logger.Warnf("%q key %s is deprecated, please use %q instead",
				source, oldKey, newKey)
		},
	})

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	ctx, cancel := context.WithCancel(ctx)

	errorCh := make(chan error)
	go func() {
		errorCh <- _main(ctx, reader, os.Args, logger, buildInfo, time.Now)
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
		logger.Error(err.Error())
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
			logger.Error(err.Error())
		}
		logger.Info("Shutdown successful")
	case <-timer.C:
		logger.Warn("Shutdown timed out")
	}

	os.Exit(1)
}

func _main(ctx context.Context, reader *reader.Reader, args []string, logger log.LoggerInterface,
	buildInfo models.BuildInformation, timeNow func() time.Time) (err error) {
	if len(args) > 1 {
		switch args[1] {
		case "version", "-version", "--version":
			fmt.Println(buildInfo.VersionString())
			return nil
		case "healthcheck":
			// Running the program in a separate instance through the Docker
			// built-in healthcheck, in an ephemeral fashion to query the
			// long running instance of the program about its status

			var healthSettings config.Health
			healthSettings.Read(reader)
			healthSettings.SetDefaults()
			err = healthSettings.Validate()
			if err != nil {
				return fmt.Errorf("health settings: %w", err)
			}

			client := health.NewClient()
			return client.Query(ctx, *healthSettings.ServerAddress)
		}
	}

	announcementExp, err := time.Parse(time.RFC3339, "2023-07-15T00:00:00Z")
	if err != nil {
		return err
	}
	splashSettings := gosplash.Settings{
		User:         "qdm12",
		Repository:   "ddns-updater",
		Emails:       []string{"quentin.mcgaw@gmail.com"},
		Version:      buildInfo.Version,
		Commit:       buildInfo.Commit,
		BuildDate:    buildInfo.Date,
		Announcement: "Public IP dns provider GOOGLE, see https://github.com/qdm12/ddns-updater/issues/492",
		AnnounceExp:  announcementExp,
		// Sponsor information
		PaypalUser:    "qmcgaw",
		GithubSponsor: "qdm12",
	}
	for _, line := range gosplash.MakeLines(splashSettings) {
		fmt.Println(line)
	}

	var config config.Config
	err = config.Read(reader, logger)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}
	config.SetDefaults()
	err = config.Validate()
	if err != nil {
		return fmt.Errorf("settings validation: %w", err)
	}

	logger.Patch(config.Logger.ToOptions()...)

	logger.Info(config.String())

	shoutrrrSettings := shoutrrr.Settings{
		Addresses:    config.Shoutrrr.Addresses,
		DefaultTitle: config.Shoutrrr.DefaultTitle,
		Logger:       logger.New(log.SetComponent("shoutrrr")),
	}
	shoutrrrClient, err := shoutrrr.New(shoutrrrSettings)
	if err != nil {
		return fmt.Errorf("setting up Shoutrrr: %w", err)
	}

	persistentDB, err := persistence.NewDatabase(*config.Paths.DataDir)
	if err != nil {
		shoutrrrClient.Notify(err.Error())
		return err
	}

	jsonReader := jsonparams.NewReader(logger)
	jsonFilepath := filepath.Join(*config.Paths.DataDir, "config.json")
	providers, warnings, err := jsonReader.JSONProviders(jsonFilepath)
	for _, w := range warnings {
		logger.Warn(w)
		shoutrrrClient.Notify(w)
	}
	if err != nil {
		shoutrrrClient.Notify(err.Error())
		return err
	}

	L := len(providers)
	switch L {
	case 0:
		logger.Warn("Found no setting to update record")
	case 1:
		logger.Info("Found single setting to update record")
	default:
		logger.Info("Found " + fmt.Sprint(len(providers)) + " settings to update records")
	}

	client := &http.Client{Timeout: config.Client.Timeout}

	err = health.CheckHTTP(ctx, client)
	if err != nil {
		logger.Warn(err.Error())
	}

	records := make([]recordslib.Record, len(providers))
	for i, provider := range providers {
		logger.Info("Reading history from database: domain " +
			provider.Domain() + " host " + provider.Host() +
			" " + provider.IPVersion().String())
		events, err := persistentDB.GetEvents(provider.Domain(),
			provider.Host(), provider.IPVersion())
		if err != nil {
			shoutrrrClient.Notify(err.Error())
			return err
		}
		records[i] = recordslib.New(provider, events)
	}

	defer client.CloseIdleConnections()
	db := data.NewDatabase(records, persistentDB)
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	httpSettings := publicip.HTTPSettings{
		Enabled: *config.PubIP.HTTPEnabled,
		Client:  client,
		Options: config.PubIP.ToHTTPOptions(),
	}
	dnsSettings := publicip.DNSSettings{
		Enabled: *config.PubIP.DNSEnabled,
		Options: config.PubIP.ToDNSPOptions(),
	}

	ipGetter, err := publicip.NewFetcher(dnsSettings, httpSettings)
	if err != nil {
		return err
	}

	resolverSettings := resolver.Settings{
		Address: config.Resolver.Address,
		Timeout: config.Resolver.Timeout,
	}
	resolver, err := resolver.New(resolverSettings)
	if err != nil {
		return fmt.Errorf("creating resolver: %w", err)
	}

	hioClient := healthchecksio.New(client, *config.Health.HealthchecksioUUID)

	updater := update.NewUpdater(db, client, shoutrrrClient, logger, timeNow)
	runner := update.NewRunner(db, updater, ipGetter, config.Update.Period,
		config.Update.Cooldown, logger, resolver, timeNow, hioClient)

	runnerHandler, runnerCtx, runnerDone := goshutdown.NewGoRoutineHandler("runner")
	go runner.Run(runnerCtx, runnerDone)

	// note: errors are logged within the goroutine,
	// no need to collect the resulting errors.
	go runner.ForceUpdate(ctx)

	isHealthy := health.MakeIsHealthy(db, resolver)
	healthLogger := logger.New(log.SetComponent("healthcheck server"))
	healthServer := health.NewServer(*config.Health.ServerAddress,
		healthLogger, isHealthy)
	healthServerHandler, healthServerCtx, healthServerDone := goshutdown.NewGoRoutineHandler("health server")
	go healthServer.Run(healthServerCtx, healthServerDone)

	serverLogger := logger.New(log.SetComponent("http server"))
	server := server.New(ctx, config.Server.ListeningAddress, config.Server.RootURL,
		db, serverLogger, runner)
	serverHandler, serverCtx, serverDone := goshutdown.NewGoRoutineHandler("server")
	go server.Run(serverCtx, serverDone)
	shoutrrrClient.Notify("Launched with " + strconv.Itoa(len(records)) + " records to watch")

	backupHandler, backupCtx, backupDone := goshutdown.NewGoRoutineHandler("backup")
	backupLogger := logger.New(log.SetComponent("backup"))
	go backupRunLoop(backupCtx, backupDone, *config.Backup.Period, *config.Paths.DataDir,
		*config.Backup.Directory, backupLogger, timeNow)

	shutdownGroup := goshutdown.NewGroupHandler("")
	shutdownGroup.Add(runnerHandler, healthServerHandler, serverHandler, backupHandler)

	<-ctx.Done()

	err = shutdownGroup.Shutdown(context.Background())
	if err != nil {
		exitHealthchecksio(hioClient, logger, healthchecksio.Exit1)
		shoutrrrClient.Notify(err.Error())
		return err
	}

	exitHealthchecksio(hioClient, logger, healthchecksio.Exit0)
	return nil
}

type InfoErroer interface {
	Info(s string)
	Error(s string)
}

func backupRunLoop(ctx context.Context, done chan<- struct{}, backupPeriod time.Duration,
	dataDir, outputDir string, logger InfoErroer, timeNow func() time.Time) {
	defer close(done)
	if backupPeriod == 0 {
		logger.Info("disabled")
		return
	}
	logger.Info("each " + backupPeriod.String() +
		"; writing zip files to directory " + outputDir)
	ziper := backup.NewZiper()
	timer := time.NewTimer(backupPeriod)
	for {
		fileName := "ddns-updater-backup-" + strconv.Itoa(int(timeNow().UnixNano())) + ".zip"
		zipFilepath := filepath.Join(outputDir, fileName)
		err := ziper.ZipFiles(
			zipFilepath,
			filepath.Join(dataDir, "updates.json"),
			filepath.Join(dataDir, "config.json"),
		)
		if err != nil {
			logger.Error(err.Error())
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

func exitHealthchecksio(hioClient *healthchecksio.Client,
	logger log.LoggerInterface, state healthchecksio.State) {
	err := hioClient.Ping(context.Background(), state)
	if err != nil {
		logger.Error(err.Error())
	}
}
