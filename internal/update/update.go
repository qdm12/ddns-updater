package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	settingserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
)

type Updater struct {
	db             Database
	client         *http.Client
	shoutrrrClient ShoutrrrClient
	logger         DebugLogger
}

func NewUpdater(db Database, client *http.Client, shoutrrrClient ShoutrrrClient, logger DebugLogger) *Updater {
	client = makeLogClient(client, logger)
	return &Updater{
		db:             db,
		client:         client,
		shoutrrrClient: shoutrrrClient,
		logger:         logger,
	}
}

func (u *Updater) Update(ctx context.Context, id uint, ip netip.Addr, now time.Time) (err error) {
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
	newIP, err := record.Provider.Update(ctx, u.client, ip)
	if err != nil {
		record.Message = err.Error()
		if errors.Is(err, settingserrors.ErrAbuse) {
			lastBan := time.Unix(now.Unix(), 0)
			record.LastBan = &lastBan
			domainName := record.Provider.BuildDomainName()
			message := domainName + ": " + record.Message +
				", no more updates will be attempted for an hour"
			u.shoutrrrClient.Notify(message)
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
	u.shoutrrrClient.Notify(record.Provider.BuildDomainName() + " " + record.Message)
	return u.db.Update(id, record) // persists some data if needed (i.e new IP)
}
