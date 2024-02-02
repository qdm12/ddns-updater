package update

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type getIPFunc func(ctx context.Context) (ip netip.Addr, err error)

var (
	ErrIPv6NotSupported = errors.New("IPv6 is not supported on this system")
)

func tryAndRepeatGettingIP(ctx context.Context, getIPFunc getIPFunc,
	logger Logger, version ipversion.IPVersion) (ip netip.Addr, err error) {
	const tries = 3
	logMessagePrefix := "obtaining " + version.String() + " address"
	errs := make([]error, 0, tries)
	for try := 0; try < tries; try++ {
		ip, err = getIPFunc(ctx)
		if err != nil {
			errs = append(errs, err)
			logger.Debug(logMessagePrefix + ": try " + strconv.Itoa(try+1) + " of " +
				strconv.Itoa(tries) + " failed: " + err.Error())
			continue
		} else if try == 0 {
			return ip, nil
		}

		tryWord := "try"
		if try > 1 {
			tryWord = "tries"
		}
		logger.Info(logMessagePrefix + " succeeded after " +
			strconv.Itoa(try) + " failed " + tryWord)
		return ip, nil
	}

	allErrorsAreIPv6NotSupported := true
	for _, err := range errs {
		const ipv6NotSupportedMessage = "connect: cannot assign requested address"
		if !strings.Contains(err.Error(), ipv6NotSupportedMessage) {
			allErrorsAreIPv6NotSupported = false
			break
		}
	}

	err = &joinedErrors{errs: errs}
	if allErrorsAreIPv6NotSupported {
		return ip, fmt.Errorf("%w: after %d tries, errors were: %w", ErrIPv6NotSupported, tries, err)
	}
	return ip, fmt.Errorf("%s: after %d tries, errors were: %w", logMessagePrefix, tries, err)
}
