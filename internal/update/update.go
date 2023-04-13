package update

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	settingserrors "github.com/qdm12/ddns-updater/internal/settings/errors"
)

type Updater struct {
	db     Database
	client *http.Client
	notify notifyFunc
	logger DebugLogger
}

type notifyFunc func(message string)

func NewUpdater(db Database, client *http.Client, notify notifyFunc, logger DebugLogger) *Updater {
	client = makeLogClient(client, logger)
	return &Updater{
		db:     db,
		client: client,
		notify: notify,
		logger: logger,
	}
}

func (u *Updater) Update(ctx context.Context, id uint, ip net.IP, now time.Time) (err error) {
	record, err := u.db.Select(id)
	if err != nil {
		return err
	}
	record.Time = now
	record.Status = constants.UPDATING
	err = u.db.Update(id, record)
	if err != nil {
		return err
	}
	record.Status = constants.FAIL
	newIP, err := record.Settings.Update(ctx, u.client, ip)
	if err != nil {
		record.Message = err.Error()
		if errors.Is(err, settingserrors.ErrAbuse) {
			lastBan := time.Unix(now.Unix(), 0)
			record.LastBan = &lastBan
			domainName := record.Settings.BuildDomainName()
			message := domainName + ": " + record.Message +
				", no more updates will be attempted for an hour"
			u.notify(message)
			err = fmt.Errorf("%w: for domain %s, no more update will be attempted for 1h", err, domainName)
		} else {
			record.LastBan = nil // clear a previous ban
		}
		if updateErr := u.db.Update(id, record); updateErr != nil {
			return fmt.Errorf("%w (with database update error: %w)", err, updateErr)
		}
		return err
	}
	record.Status = constants.SUCCESS
	record.Message = fmt.Sprintf("changed to %s", ip.String())
	record.History = append(record.History, models.HistoryEvent{
		IP:   newIP,
		Time: now,
	})
	u.notify(record.Settings.BuildDomainName() + " " + record.Message)
	return u.db.Update(id, record) // persists some data if needed (i.e new IP)
}
