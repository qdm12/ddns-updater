package update

import (
	"time"
	"net/http"
	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/database"
)

// TriggerServer runs an infinite asynchronous periodic function that triggers updates
func TriggerServer(
	delay time.Duration,
	forceCh, quitCh chan struct{},
	recordsConfigs []models.RecordConfigType,
	httpClient *http.Client,
	sqlDb *database.DB,
) {
	ticker := time.NewTicker(delay * time.Second)
	defer func() {
		ticker.Stop()
		close(quitCh)
	}()
	for {
		select {
		case <-ticker.C:
			for i := range recordsConfigs {
				go update(&recordsConfigs[i], httpClient, sqlDb)
			}
		case <-forceCh:
			for i := range recordsConfigs {
				go update(&recordsConfigs[i], httpClient, sqlDb)
			}
		case <-quitCh:
			for {
				allUpdatesFinished := true
				for i := range recordsConfigs {
					if recordsConfigs[i].Status.Code == models.UPDATING {
						allUpdatesFinished = false
					}
				}
				if allUpdatesFinished {
					break
				}
				logging.Info("Waiting for updates to complete...")
				time.Sleep(400 * time.Millisecond)
			}
			ticker.Stop()
			return
		}
	}
}