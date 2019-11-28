package update

import (
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/database"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/admin"
)

// TriggerServer runs an infinite asynchronous periodic function that triggers updates
func TriggerServer(
	delay time.Duration,
	chForce, chQuit chan struct{}, // listener only
	recordsConfigs []models.RecordConfigType, // does not change size so no pointer needed
	httpClient *http.Client,
	db database.SQL,
	gotify *admin.Gotify,
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
			go periodicServer(&recordsConfigs[i], customDelay, httpClient, db, chForceArr[i], chQuitArr[i], gotify)
		} else {
			go periodicServer(&recordsConfigs[i], delay, httpClient, db, chForceArr[i], chQuitArr[i], gotify)
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
	db database.SQL,
	chForce, chQuit chan struct{},
	gotify *admin.Gotify,
) {
	ticker := time.NewTicker(delay)
	defer func() {
		ticker.Stop()
		close(chForce)
		close(chQuit)
	}()
	for {
		select {
		case <-ticker.C:
			go update(recordConfig, httpClient, db, gotify)
		case <-chForce:
			go update(recordConfig, httpClient, db, gotify)
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
