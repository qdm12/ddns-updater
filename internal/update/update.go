package update

import (
	"fmt"
	"net"
	"time"

	"github.com/qdm12/golibs/admin"
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
	gotify   admin.Gotify
	verifier verification.Verifier
}

func NewUpdater(db data.Database, logger logging.Logger, client libnetwork.Client, gotify admin.Gotify) Updater {
	return &updater{
		db:       db,
		logger:   logger,
		client:   client,
		gotify:   gotify,
		verifier: verification.NewVerifier(),
	}
}

func (u *updater) Update(id int) error {
	recordConfig, err := u.db.Select(id)
	if err != nil {
		return err
	}
	recordConfig.Time = time.Now()
	recordConfig.Status = constants.UPDATING
	if err := u.db.Update(id, recordConfig); err != nil {
		return err
	}
	status, message, newIP, err := u.update(
		recordConfig.Settings,
		recordConfig.History.GetCurrentIP(),
		recordConfig.History.GetDurationSinceSuccess())
	recordConfig.Status = status
	recordConfig.Message = message
	if err != nil {
		if len(recordConfig.Message) == 0 {
			recordConfig.Message = err.Error()
		}
		if updateErr := u.db.Update(id, recordConfig); updateErr != nil {
			return fmt.Errorf("%s, %s", err, updateErr)
		}
		return err
	}
	if newIP != nil {
		recordConfig.History.SuccessTime = time.Now()
		recordConfig.History.IPs = append(recordConfig.History.IPs, newIP)
		if err := u.gotify.Notify("DDNS Updater", 1, message); err != nil {
			u.logger.Warn(err)
		}
	}
	return u.db.Update(id, recordConfig) // persists some data if needed (i.e new IP)
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
	case constants.PROVIDERNAMECHEAP:
		ip, err = updateNamecheap(
			u.client,
			settings.Host,
			settings.Domain,
			settings.Password,
			ip,
		)
	case constants.PROVIDERGODADDY:
		err = updateGoDaddy(
			u.client,
			settings.Host,
			settings.Domain,
			settings.Key,
			settings.Secret,
			ip,
		)
	case constants.PROVIDERDUCKDNS:
		ip, err = updateDuckDNS(
			u.client,
			settings.Domain,
			settings.Token,
			ip,
		)
	case constants.PROVIDERDREAMHOST:
		err = updateDreamhost(
			u.client,
			settings.Domain,
			settings.Key,
			settings.BuildDomainName(),
			ip,
		)
	case constants.PROVIDERCLOUDFLARE:
		err = updateCloudflare(
			u.client,
			settings.ZoneIdentifier,
			settings.Identifier,
			settings.Host,
			settings.Email,
			settings.Key,
			settings.UserServiceKey,
			settings.Proxied,
			settings.Ttl,
			ip,
		)
	case constants.PROVIDERNOIP:
		ip, err = updateNoIP(
			u.client,
			settings.BuildDomainName(),
			settings.Username,
			settings.Password,
			ip,
		)
	case constants.PROVIDERDNSPOD:
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
	if currentIP == nil {
		// first IP assigned
		message = fmt.Sprintf("%s has now IP address %s", settings.BuildDomainName(), ip.String())
	} else {
		message = fmt.Sprintf("%s changed from %s to %s", settings.BuildDomainName(), currentIP.String(), ip.String())
	}
	return constants.SUCCESS, message, ip, nil
}

func getPublicIP(client libnetwork.Client, IPMethod models.IPMethod) (ip net.IP, err error) {
	switch IPMethod {
	case constants.IPMETHODPROVIDER:
		return nil, nil
	case constants.IPMETHODGOOGLE:
		return network.GetPublicIP(client, "https://google.com/search?q=ip")
	case constants.IPMETHODOPENDNS:
		return network.GetPublicIP(client, "https://diagnostic.opendns.com/myip")
	}
	return nil, fmt.Errorf("IP method %q not supported", IPMethod)
}
