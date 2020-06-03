package update

import (
	"fmt"
	"net"
	"time"

	netlib "github.com/qdm12/golibs/network"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
)

type Updater interface {
	Update(id int, ip net.IP, now time.Time) (err error)
}

type updater struct {
	db     data.Database
	client netlib.Client
	notify notifyFunc
}

type notifyFunc func(priority int, messageArgs ...interface{})

func NewUpdater(db data.Database, client netlib.Client, notify notifyFunc) Updater {
	return &updater{
		db:     db,
		client: client,
		notify: notify,
	}
}

func (u *updater) Update(id int, ip net.IP, now time.Time) (err error) {
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
	newIP, err := record.Settings.Update(u.client, ip)
	if err != nil {
		record.Message = err.Error()
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
