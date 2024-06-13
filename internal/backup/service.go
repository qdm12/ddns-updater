package backup

import (
	"context"
	"path/filepath"
	"strconv"
	"time"
)

type Service struct {
	// Injected fields
	backupPeriod time.Duration
	dataDir      string
	outputDir    string
	logger       Logger

	// Internal fields
	stopCh chan<- struct{}
	done   <-chan struct{}
}

func New(backupPeriod time.Duration,
	dataDir, outputDir string, logger Logger) *Service {
	return &Service{
		logger:       logger,
		backupPeriod: backupPeriod,
		dataDir:      dataDir,
		outputDir:    outputDir,
	}
}

func (s *Service) String() string {
	return "backup"
}

func makeZipFileName() string {
	return "ddns-updater-backup-" + strconv.Itoa(int(time.Now().UnixNano())) + ".zip"
}

func (s *Service) Start(ctx context.Context) (runError <-chan error, startErr error) {
	ready := make(chan struct{})
	runErrorCh := make(chan error)
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	done := make(chan struct{})
	s.done = done
	go run(ready, runErrorCh, stopCh, done,
		s.outputDir, s.dataDir, s.backupPeriod, s.logger)
	select {
	case <-ready:
	case <-ctx.Done():
		return nil, s.Stop()
	}
	return runErrorCh, nil
}

func run(ready chan<- struct{}, runError chan<- error, stopCh <-chan struct{},
	done chan<- struct{}, outputDir, dataDir string, backupPeriod time.Duration,
	logger Logger) {
	defer close(done)

	if backupPeriod == 0 {
		close(ready)
		logger.Info("disabled")
		return
	}

	logger.Info("each " + backupPeriod.String() +
		"; writing zip files to directory " + outputDir)
	timer := time.NewTimer(backupPeriod)
	close(ready)

	for {
		select {
		case <-timer.C:
		case <-stopCh:
			_ = timer.Stop()
			return
		}
		err := zipFiles(
			filepath.Join(outputDir, makeZipFileName()),
			filepath.Join(dataDir, "config.json"),
			filepath.Join(dataDir, "updates.json"),
		)
		if err != nil {
			runError <- err
			return
		}
		timer.Reset(backupPeriod)
	}
}

func (s *Service) Stop() (err error) {
	close(s.stopCh)
	<-s.done
	return nil
}
