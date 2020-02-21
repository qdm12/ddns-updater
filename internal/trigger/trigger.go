package trigger

import (
	"context"
	"time"

	"github.com/qdm12/ddns-updater/internal/update"
)

// StartUpdates starts periodic updates
func StartUpdates(ctx context.Context, updater update.Updater, idPeriodMapping map[int]time.Duration, onError func(err error)) (forceUpdate func()) {
	errors := make(chan error)
	triggers := make([]chan struct{}, len(idPeriodMapping))
	for id, period := range idPeriodMapping {
		triggers[id] = make(chan struct{})
		go func(id int, period time.Duration) {
			ticker := time.NewTicker(period)
			defer ticker.Stop()
			for {
				select {
				case <-triggers[id]:
					if err := updater.Update(id); err != nil {
						errors <- err
					}
				case <-ticker.C:
					if err := updater.Update(id); err != nil {
						errors <- err
					}
				case <-ctx.Done():
					return
				}
			}
		}(id, period)
	}
	// collects errors only
	go func() {
		for {
			select {
			case err := <-errors:
				onError(err)
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() {
		for i := range triggers {
			triggers[i] <- struct{}{}
		}
	}
}
