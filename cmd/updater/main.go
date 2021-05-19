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
	"github.com/qdm12/ddns-updater/pkg/publicip"
	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	pubiphttp "github.com/qdm12/ddns-updater/pkg/publicip/http"
	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/network/connectivity"
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
	os.Exit(_main(context.Background(), time.Now))
}

type allParams struct {
	period          time.Duration
	cooldown        time.Duration
	httpTimeout     time.Duration
	httpSettings    publicip.HTTPSettings
	dnsSettings     publicip.DNSSettings
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
	paramsReader := params.NewReader(logging.New(logging.Settings{})) // use a temporary logger
	logLevel, logCaller, err := paramsReader.LoggerConfig()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	logger := logging.NewParent(logging.Settings{Level: logLevel, Caller: logCaller})
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

	client := &http.Client{Timeout: p.httpTimeout}

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

	p.httpSettings.Client = client

	ipGetter, err := publicip.NewFetcher(p.dnsSettings, p.httpSettings)
	if err != nil {
		logger.Error(err)
		return 1
	}

	updater := update.NewUpdater(db, client, notify, logger)
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
		logger.NewChild(logging.Settings{Prefix: "healthcheck server: "}),
		isHealthy)
	wg.Add(1)
	go healthServer.Run(ctx, wg)

	address := fmt.Sprintf("0.0.0.0:%d", p.listeningPort)
	uiDir := p.dir + "/ui"
	serverLogger := logger.NewChild(logging.Settings{Prefix: "http server: "})
	server := server.New(ctx, address, p.rootURL, uiDir, db, serverLogger, runner)
	wg.Add(1)
	go server.Run(ctx, wg)
	notify(1, fmt.Sprintf("Launched with %d records to watch", len(records)))

	go backupRunLoop(ctx, p.backupPeriod, p.dir, p.backupDirectory,
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

	p.httpSettings.Enabled, p.dnsSettings.Enabled, err = paramsReader.PublicIPFetchers()
	if err != nil {
		return p, err
	}

	p.httpTimeout, err = paramsReader.HTTPTimeout()
	if err != nil {
		return p, err
	}

	httpIPProviders, err := paramsReader.PublicIPHTTPProviders()
	if err != nil {
		return p, err
	}
	httpIP4Providers, err := paramsReader.PublicIPv4HTTPProviders()
	if err != nil {
		return p, err
	}
	httpIP6Providers, err := paramsReader.PublicIPv6HTTPProviders()
	if err != nil {
		return p, err
	}
	p.httpSettings.Options = []pubiphttp.Option{
		pubiphttp.SetProvidersIP(httpIPProviders[0], httpIPProviders[1:]...),
		pubiphttp.SetProvidersIP4(httpIP4Providers[0], httpIP4Providers[1:]...),
		pubiphttp.SetProvidersIP6(httpIP6Providers[0], httpIP6Providers[1:]...),
	}

	dnsIPProviders, err := paramsReader.PublicIPDNSProviders()
	if err != nil {
		return p, err
	}
	p.dnsSettings.Options = []dns.Option{
		dns.SetProviders(dnsIPProviders[0], dnsIPProviders[1:]...),
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
