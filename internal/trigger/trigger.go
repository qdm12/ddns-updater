package trigger

import (
	"context"
	"time"

	"github.com/qdm12/ddns-updater/internal/update"
)

type Trigger interface {
	Run(ctx context.Context, onError func(err error))
	Force() (errors []error)
}

type trigger struct {
	defaultPeriod time.Duration
	idToPeriod    map[int]time.Duration
	updater       update.Updater
}

func NewTrigger(defaultPeriod time.Duration, idToPeriod map[int]time.Duration, updater update.Updater) Trigger {
	return &trigger{
		defaultPeriod: defaultPeriod,
		idToPeriod:    idToPeriod,
		updater:       updater,
	}
}

// Run runs an infinite asynchronous periodic function that triggers updates
func (t *trigger) Run(ctx context.Context, onError func(err error)) {
	errors := make(chan error)
	for id, customPeriod := range t.idToPeriod {
		period := t.defaultPeriod
		if customPeriod > 0 {
			period = customPeriod
		}
		go t.periodicRun(ctx, id, period, errors)
	}
	for {
		select {
		case err := <-errors:
			onError(err)
		case <-ctx.Done():
			break
		}
	}
}

func (t *trigger) periodicRun(ctx context.Context, id int, period time.Duration, errors chan<- error) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := t.updater.Update(id); err != nil {
				errors <- err
			}
		case <-ctx.Done(): // waits for update to finish
			return
		}
	}
}

func (t *trigger) Force() (errors []error) {
	for id := range t.idToPeriod {
		if err := t.updater.Update(id); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
