package update

import (
	"fmt"
	"net"
	"time"

	"github.com/qdm12/golibs/logging"
	libnetwork "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/network"
)

type Updater interface {
	Update(id int) error
}

type updater struct {
	db       data.Database
	logger   logging.Logger
	client   libnetwork.Client
	notify   notifyFunc
	verifier verification.Verifier
}

type notifyFunc func(priority int, messageArgs ...interface{})

func NewUpdater(db data.Database, logger logging.Logger, client libnetwork.Client, notify notifyFunc) Updater {
	return &updater{
		db:       db,
		logger:   logger,
		client:   client,
		notify:   notify,
		verifier: verification.NewVerifier(),
	}
}

func (u *updater) Update(id int) error {
	record, err := u.db.Select(id)
	if err != nil {
		return err
	}
	record.Time = time.Now()
	record.Status = constants.UPDATING
	if err := u.db.Update(id, record); err != nil {
		return err
	}
	status, message, newIP, err := u.update(
		record.Settings,
		record.History.GetCurrentIP(),
		record.History.GetDurationSinceSuccess())
	record.Status = status
	record.Message = message
	if err != nil {
		if len(record.Message) == 0 {
			record.Message = err.Error()
		}
		if updateErr := u.db.Update(id, record); updateErr != nil {
			return fmt.Errorf("%s, %s", err, updateErr)
		}
		return err
	}
	if newIP != nil {
		record.History.SuccessTime = time.Now()
		record.History.IPs = append(record.History.IPs, newIP)
		u.notify(1, fmt.Sprintf("%s %s", record.Settings.BuildDomainName(), message))
	}
	return u.db.Update(id, record) // persists some data if needed (i.e new IP)
}

func (u *updater) update(settings models.Settings, currentIP net.IP, durationSinceSuccess string) (status models.Status, message string, newIP net.IP, err error) {
	// Get the public IP address
	ip, err := getPublicIP(u.client, settings.IPMethod) // Note: empty IP means DNS provider provided
	if err != nil {
		return constants.FAIL, "", nil, err
	}
	if ip != nil && ip.Equal(currentIP) {
		return constants.UPTODATE, fmt.Sprintf("No IP change for %s", durationSinceSuccess), nil, nil
	}

	// Update the record
	switch settings.Provider {
	case constants.NAMECHEAP:
		ip, err = updateNamecheap(
			u.client,
			settings.Host,
			settings.Domain,
			settings.Password,
			ip,
		)
	case constants.GODADDY:
		err = updateGoDaddy(
			u.client,
			settings.Host,
			settings.Domain,
			settings.Key,
			settings.Secret,
			ip,
		)
	case constants.DUCKDNS:
		ip, err = updateDuckDNS(
			u.client,
			settings.Domain,
			settings.Token,
			ip,
		)
	case constants.DREAMHOST:
		err = updateDreamhost(
			u.client,
			settings.Domain,
			settings.Key,
			settings.BuildDomainName(),
			ip,
		)
	case constants.CLOUDFLARE:
		err = updateCloudflare(
			u.client,
			settings.ZoneIdentifier,
			settings.Identifier,
			settings.Host,
			settings.Email,
			settings.Key,
			settings.UserServiceKey,
			settings.Token,
			settings.Proxied,
			settings.Ttl,
			ip,
		)
	case constants.NOIP:
		ip, err = updateNoIP(
			u.client,
			settings.BuildDomainName(),
			settings.Username,
			settings.Password,
			ip,
		)
	case constants.DNSPOD:
		err = updateDNSPod(
			u.client,
			settings.Domain,
			settings.Host,
			settings.Token,
			ip,
		)
	default:
		err = fmt.Errorf("provider %q is not supported", settings.Provider)
	}
	if err != nil {
		return constants.FAIL, "", nil, err
	}
	if ip != nil && ip.Equal(currentIP) {
		return constants.UPTODATE, fmt.Sprintf("No IP change for %s", durationSinceSuccess), nil, nil
	}
	return constants.SUCCESS, fmt.Sprintf("changed to %s", ip.String()), ip, nil
}

func getPublicIP(client libnetwork.Client, IPMethod models.IPMethod) (ip net.IP, err error) {
	switch IPMethod {
	case constants.PROVIDER:
		return nil, nil
	case constants.GOOGLE:
		return network.GetPublicIP(client, "https://google.com/search?q=ip")
	case constants.OPENDNS:
		return network.GetPublicIP(client, "https://diagnostic.opendns.com/myip")
	}
	return nil, fmt.Errorf("IP method %q not supported", IPMethod)
}
