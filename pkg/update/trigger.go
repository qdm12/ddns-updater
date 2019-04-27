package update

import (
	"time"
	"net/http"
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
			// Wait for all updates to stop updating or being read
			for i := range recordsConfigs {
				recordsConfigs[i].M.Lock()
			}
			for i := range recordsConfigs {
				recordsConfigs[i].M.Unlock()
			}
			ticker.Stop()
			return
		}
	}
}