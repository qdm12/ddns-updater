package update

import (
	"ddns-updater/pkg/database"
	"ddns-updater/pkg/models"
	"net/http"
	"time"
)

// TriggerServer runs an infinite asynchronous periodic function that triggers updates
func TriggerServer(
	delay time.Duration,
	chForce, chQuit chan struct{}, // listener only
	recordsConfigs []models.RecordConfigType, // does not change size so no pointer needed
	httpClient *http.Client,
	sqlDb *database.DB,
) {
	var chQuitArr, chForceArr []chan struct{}
	defer func() {
		for i := range chForceArr {
			close(chForceArr[i])
		}
		for i := range chQuitArr {
			close(chQuitArr[i])
		}
		close(chForce)
		close(chQuit)
	}()
	for i := range recordsConfigs {
		chForceArr = append(chForceArr, make(chan struct{}))
		chQuitArr = append(chQuitArr, make(chan struct{}))
		customDelay := recordsConfigs[i].Settings.Delay
		if customDelay > 0 {
			go periodicServer(&recordsConfigs[i], customDelay, httpClient, sqlDb, chForceArr[i], chQuitArr[i])
		} else {
			go periodicServer(&recordsConfigs[i], delay, httpClient, sqlDb, chForceArr[i], chQuitArr[i])
		}
	}
	// fan out channel signals
	for {
		select {
		case <-chForce:
			for i := range chForceArr {
				chForceArr[i] <- struct{}{}
			}
		case <-chQuit:
			for i := range chQuitArr {
				chQuitArr[i] <- struct{}{}
			}
			return
		}
	}
}

func periodicServer(
	recordConfig *models.RecordConfigType,
	delay time.Duration,
	httpClient *http.Client,
	sqlDb *database.DB,
	chForce, chQuit chan struct{},
) {
	ticker := time.NewTicker(delay * time.Second)
	defer func() {
		ticker.Stop()
		close(chForce)
		close(chQuit)
	}()
	for {
		select {
		case <-ticker.C:
			go update(recordConfig, httpClient, sqlDb)
		case <-chForce:
			go update(recordConfig, httpClient, sqlDb)
		case <-chQuit:
			recordConfig.IsUpdating.Lock() // wait for an eventual update to finish
			ticker.Stop()
			close(chForce)
			close(chQuit)
			recordConfig.IsUpdating.Unlock()
			return
		}
	}
}
