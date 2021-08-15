package update

import (
	"context"
	"net"
	"strconv"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/logging"
)

type getIPFunc func(ctx context.Context) (ip net.IP, err error)

func tryAndRepeatGettingIP(ctx context.Context, getIPFunc getIPFunc,
	logger logging.Logger, version ipversion.IPVersion) (ip net.IP, err error) {
	const tries = 3
	logMessagePrefix := "obtaining " + version.String() + " address"
	for try := 0; try < tries; try++ {
		ip, err = getIPFunc(ctx)
		if err != nil {
			logger.Warn(logMessagePrefix + ": try " + strconv.Itoa(try+1) + " of " +
				strconv.Itoa(tries) + ": " + err.Error())
			continue
		}
		if try > 0 {
			logger.Info(logMessagePrefix + ": succeeded after " +
				strconv.Itoa(try+1) + " tries")
		}
		break
	}
	return ip, err
}
