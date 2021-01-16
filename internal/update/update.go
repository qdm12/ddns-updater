package update

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/golibs/logging"
	netlib "github.com/qdm12/golibs/network"
)

type Updater interface {
	Update(ctx context.Context, id int, ip net.IP, now time.Time) (err error)
}

type updater struct {
	db     data.Database
	client netlib.Client
	notify notifyFunc
	logger logging.Logger
}

type notifyFunc func(priority int, messageArgs ...interface{})

func NewUpdater(db data.Database, client netlib.Client, notify notifyFunc, logger logging.Logger) Updater {
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
		if errors.Is(err, settings.ErrAbuse) {
			lastBan := time.Unix(now.Unix(), 0)
			record.LastBan = &lastBan
			message := record.Settings.BuildDomainName() + ": " + record.Message + ", no more updates will be attempted"
			u.notify(3, message) //nolint:gomnd
			err = errors.New(message)
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
	u.notify(1, fmt.Sprintf("%s %s", record.Settings.BuildDomainName(), record.Message))
	return u.db.Update(id, record) // persists some data if needed (i.e new IP)
}
