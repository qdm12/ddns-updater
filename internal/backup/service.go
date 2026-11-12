package backup

import (
	"context"
	"path/filepath"
	"strconv"
	"time"
)

type Service struct {
	// Injected fields
	backupPeriod   time.Duration
	dataDir        string
	configFilepath string
	outputDir      string
	logger         Logger
	keep           uint

	// Internal fields
	stopCh chan<- struct{}
	done   <-chan struct{}
}

func New(backupPeriod time.Duration,
	dataDir, configFilepath, outputDir string, logger Logger, keep uint,
) *Service {
	return &Service{
		logger:         logger,
		backupPeriod:   backupPeriod,
		dataDir:        dataDir,
		configFilepath: configFilepath,
		outputDir:      outputDir,
		keep:           keep,
	}
}

func (s *Service) String() string {
	return "backup"
}

const (
	zipFileNamePrefix = "ddns-updater-backup-"
	zipFileNameSuffix = ".zip"
)

func makeZipFileName() string {
	return zipFileNamePrefix + strconv.Itoa(int(time.Now().UnixNano())) + zipFileNameSuffix
}

func (s *Service) Start(ctx context.Context) (runError <-chan error, startErr error) {
	ready := make(chan struct{})
	runErrorCh := make(chan error)
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	done := make(chan struct{})
	s.done = done
	go run(ready, runErrorCh, stopCh, done,
		s.outputDir, s.dataDir, s.configFilepath, s.backupPeriod, s.logger, s.keep)
	select {
	case <-ready:
	case <-ctx.Done():
		return nil, s.Stop()
	}
	return runErrorCh, nil
}

func run(ready chan<- struct{}, runError chan<- error, stopCh <-chan struct{},
	done chan<- struct{}, outputDir, dataDir, configFilepath string,
	backupPeriod time.Duration, logger Logger, keep uint,
) {
	defer close(done)

	if backupPeriod == 0 {
		close(ready)
		logger.Info("disabled")
		return
	}

	logger.Info("each " + backupPeriod.String() +
		"; writing zip files to directory " + outputDir)
	updatesFilepath := filepath.Join(dataDir, "updates.json")
	inputFilepaths := []string{configFilepath, updatesFilepath}
	if configFilepath == "" {
		logger.Info("the configuration file is not backed up")
		inputFilepaths = []string{updatesFilepath}
	}
	timer := time.NewTimer(backupPeriod)
	close(ready)

	for {
		select {
		case <-timer.C:
		case <-stopCh:
			_ = timer.Stop()
			return
		}
		err := zipFiles(filepath.Join(outputDir, makeZipFileName()), inputFilepaths...)
		if err != nil {
			runError <- err
			return
		}
		err = pruneBackups(outputDir, keep)
		if err != nil {
			logger.Warn("pruning backups: " + err.Error())
		}
		timer.Reset(backupPeriod)
	}
}

func (s *Service) Stop() (err error) {
	close(s.stopCh)
	<-s.done
	return nil
}
