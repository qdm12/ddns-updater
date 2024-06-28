package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/qdm12/ddns-updater/internal/noop"
	jsonparams "github.com/qdm12/ddns-updater/internal/params"
	persistence "github.com/qdm12/ddns-updater/internal/persistence/json"
	"github.com/qdm12/ddns-updater/internal/provider"
	recordslib "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/resolver"
	"github.com/qdm12/ddns-updater/internal/server"
	"github.com/qdm12/ddns-updater/internal/shoutrrr"
	"github.com/qdm12/ddns-updater/internal/update"
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/goservices"
	"github.com/qdm12/gosettings/reader"
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

	printSplash(buildInfo)

	config, err := readConfig(reader, logger)
	if err != nil {
		return err
	}

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
	providers, warnings, err := jsonReader.JSONProviders(*config.Paths.Config)
	for _, w := range warnings {
		logger.Warn(w)
		shoutrrrClient.Notify(w)
	}
	if err != nil {
		shoutrrrClient.Notify(err.Error())
		return err
	}

	logProvidersCount(len(providers), logger)

	client := &http.Client{Timeout: config.Client.Timeout}
	defer client.CloseIdleConnections()

	err = health.CheckHTTP(ctx, client)
	if err != nil {
		logger.Warn(err.Error())
	}

	records, err := readRecords(providers, persistentDB, logger, shoutrrrClient)
	if err != nil {
		return fmt.Errorf("reading records: %w", err)
	}

	db := data.NewDatabase(records, persistentDB)

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

	hioClient := healthchecksio.New(client, config.Health.HealthchecksioBaseURL,
		*config.Health.HealthchecksioUUID)

	updater := update.NewUpdater(db, client, shoutrrrClient, logger, timeNow)
	updaterService := update.NewService(db, updater, ipGetter, config.Update.Period,
		config.Update.Cooldown, logger, resolver, timeNow, hioClient)

	healthServer, err := createHealthServer(db, resolver, logger, *config.Health.ServerAddress)
	if err != nil {
		return fmt.Errorf("creating health server: %w", err)
	}

	server, err := createServer(ctx, config.Server, logger, db, updaterService)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	var backupService goservices.Service
	backupLogger := logger.New(log.SetComponent("backup"))
	backupService = backup.New(*config.Backup.Period, *config.Paths.DataDir,
		*config.Backup.Directory, backupLogger)
	backupService, err = goservices.NewRestarter(goservices.RestarterSettings{Service: backupService})
	if err != nil {
		return fmt.Errorf("creating backup restarter: %w", err)
	}

	servicesSequence, err := goservices.NewSequence(goservices.SequenceSettings{
		ServicesStart: []goservices.Service{db, updaterService, healthServer, server, backupService},
		ServicesStop:  []goservices.Service{server, healthServer, updaterService, backupService, db},
	})
	if err != nil {
		return fmt.Errorf("creating services sequence: %w", err)
	}

	runError, startErr := servicesSequence.Start(ctx)
	if startErr != nil {
		return fmt.Errorf("starting services: %w", startErr)
	}

	// note: errors are logged within the goroutine,
	// no need to collect the resulting errors.
	go updaterService.ForceUpdate(ctx)

	shoutrrrClient.Notify("Launched with " + strconv.Itoa(len(records)) + " records to watch")

	select {
	case <-ctx.Done():
	case err = <-runError:
		exitHealthchecksio(hioClient, logger, healthchecksio.Exit1)
		shoutrrrClient.Notify(err.Error())
		return fmt.Errorf("exiting due to critical error: %w", err)
	}

	err = servicesSequence.Stop()
	if err != nil {
		exitHealthchecksio(hioClient, logger, healthchecksio.Exit1)
		shoutrrrClient.Notify(err.Error())
		return fmt.Errorf("stopping failed: %w", err)
	}

	exitHealthchecksio(hioClient, logger, healthchecksio.Exit0)
	return nil
}

func printSplash(buildInfo models.BuildInformation) {
	announcementExp, err := time.Parse(time.RFC3339, "2023-07-15T00:00:00Z")
	if err != nil {
		panic(err)
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
}

func readConfig(reader *reader.Reader, logger log.LoggerInterface) (
	config config.Config, err error) {
	err = config.Read(reader, logger)
	if err != nil {
		return config, fmt.Errorf("reading settings: %w", err)
	}
	config.SetDefaults()
	err = config.Validate()
	if err != nil {
		return config, fmt.Errorf("settings validation: %w", err)
	}

	logger.Patch(config.Logger.ToOptions()...)
	logger.Info(config.String())

	return config, nil
}

func logProvidersCount(providersCount int, logger log.LeveledLogger) {
	switch providersCount {
	case 0:
		logger.Warn("Found no setting to update record")
	case 1:
		logger.Info("Found single setting to update record")
	default:
		logger.Info("Found " + strconv.Itoa(providersCount) + " settings to update records")
	}
}

func readRecords(providers []provider.Provider, persistentDB *persistence.Database,
	logger log.LoggerInterface, shoutrrrClient *shoutrrr.Client) (
	records []recordslib.Record, err error) {
	records = make([]recordslib.Record, len(providers))
	for i, provider := range providers {
		logger.Info("Reading history from database: domain " +
			provider.Domain() + " owner " + provider.Owner() +
			" " + provider.IPVersion().String())
		events, err := persistentDB.GetEvents(provider.Domain(),
			provider.Owner(), provider.IPVersion())
		if err != nil {
			shoutrrrClient.Notify(err.Error())
			return nil, err
		}
		records[i] = recordslib.New(provider, events)
	}
	return records, nil
}

func exitHealthchecksio(hioClient *healthchecksio.Client,
	logger log.LoggerInterface, state healthchecksio.State) {
	const timeout = 3 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := hioClient.Ping(ctx, state)
	if err != nil {
		logger.Error(err.Error())
	}
}

//nolint:ireturn
func createHealthServer(db health.AllSelecter, resolver health.LookupIPer,
	logger log.LoggerInterface, serverAddress string) (
	healthServer goservices.Service, err error) {
	if !health.IsDocker() {
		return noop.New("healthcheck server"), nil
	}
	isHealthy := health.MakeIsHealthy(db, resolver)
	healthLogger := logger.New(log.SetComponent("healthcheck server"))
	return health.NewServer(serverAddress, healthLogger, isHealthy)
}

//nolint:ireturn
func createServer(ctx context.Context, config config.Server,
	logger log.LoggerInterface, db server.Database,
	updaterService server.UpdateForcer) (
	service goservices.Service, err error) {
	if !*config.Enabled {
		return noop.New("server"), nil
	}
	serverLogger := logger.New(log.SetComponent("http server"))
	return server.New(ctx, config.ListeningAddress, config.RootURL,
		db, serverLogger, updaterService)
}
