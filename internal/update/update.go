package update

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	settingserrors "github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/golibs/logging"
)

type Updater interface {
	Update(ctx context.Context, id int, ip net.IP, now time.Time) (err error)
}

type updater struct {
	db     data.Database
	client *http.Client
	notify notifyFunc
	logger logging.Logger
}

type notifyFunc func(message string)

func NewUpdater(db data.Database, client *http.Client, notify notifyFunc, logger logging.Logger) Updater {
	client = makeLogClient(client, logger)
	return &updater{
		db:     db,
		client: client,
		notify: notify,
		logger: logger,
	}
}

func (u *updater) Update(ctx context.Context, id int, ip net.IP, now time.Time) (err error) {
	record, err := u.db.Select(id)
	if err != nil {
		return err
	}
	record.Time = now
	record.Status = constants.UPDATING
	if err := u.db.Update(id, record); err != nil {
		return err
	}
	record.Status = constants.FAIL
	newIP, err := record.Settings.Update(ctx, u.client, ip)
	if err != nil {
		record.Message = err.Error()
		if errors.Is(err, settingserrors.ErrAbuse) {
			lastBan := time.Unix(now.Unix(), 0)
			record.LastBan = &lastBan
			message := record.Settings.BuildDomainName() + ": " + record.Message +
				", no more updates will be attempted for an hour"
			u.notify(message)
			err = fmt.Errorf(message)
		} else {
			record.LastBan = nil // clear a previous ban
		}
		if updateErr := u.db.Update(id, record); updateErr != nil {
			return fmt.Errorf("%s, %s", err, updateErr)
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
