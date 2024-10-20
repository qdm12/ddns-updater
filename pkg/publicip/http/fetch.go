package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/ipextract"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

var (
	ErrNoIPFound  = errors.New("no IP address found")
	ErrTooManyIPs = errors.New("too many IP addresses")
	ErrBanned     = errors.New("we got banned")
)

func fetch(ctx context.Context, client *http.Client, url string,
	version ipversion.IPVersion,
) (publicIP netip.Addr, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return netip.Addr{}, err
	}

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		return netip.Addr{}, fmt.Errorf("%w: %d (%s)", ErrBanned,
			response.StatusCode, bodyToSingleLine(response.Body))
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, err
	}

	err = response.Body.Close()
	if err != nil {
		return netip.Addr{}, err
	}

	s := string(b)

	ipv4 := ipextract.IPv4(s)
	ipv6 := ipextract.IPv6(s)

	switch version {
	case ipversion.IP4or6:
		switch {
		case len(ipv4) == 1: // priority to IPv4
			return ipv4[0], nil
		case len(ipv6) == 1:
			return ipv6[0], nil
		case len(ipv4) > 1:
			return netip.Addr{}, fmt.Errorf("%w: found %d IPv4 addresses instead of 1",
				ErrTooManyIPs, len(ipv4))
		case len(ipv6) > 1:
			return netip.Addr{}, fmt.Errorf("%w: found %d IPv6 addresses instead of 1",
				ErrTooManyIPs, len(ipv6))
		default:
			return netip.Addr{}, fmt.Errorf("%w: from %q", ErrNoIPFound, url)
		}
	case ipversion.IP4:
		switch len(ipv4) {
		case 0:
			return netip.Addr{}, fmt.Errorf("%w: from %q for version %s", ErrNoIPFound, url, version)
		case 1:
			return ipv4[0], nil
		default:
			return netip.Addr{}, fmt.Errorf("%w: found %d IPv4 addresses instead of 1",
				ErrTooManyIPs, len(ipv4))
		}
	case ipversion.IP6:
		switch len(ipv6) {
		case 0:
			return netip.Addr{}, fmt.Errorf("%w: from %q for version %s", ErrNoIPFound, url, version)
		case 1:
			return ipv6[0], nil
		default:
			return netip.Addr{}, fmt.Errorf("%w: found %d IPv6 addresses instead of 1",
				ErrTooManyIPs, len(ipv6))
		}
	default:
		panic(fmt.Sprintf("IP version %q is not supported", version))
	}
}

func bodyToSingleLine(body io.Reader) (s string) {
	b, err := io.ReadAll(body)
	if err != nil {
		return ""
	}
	data := string(b)
	return toSingleLine(data)
}

func toSingleLine(s string) (line string) {
	line = strings.ReplaceAll(s, "\n", "")
	line = strings.ReplaceAll(line, "\r", "")
	line = strings.ReplaceAll(line, "  ", " ")
	line = strings.ReplaceAll(line, "  ", " ")
	return line
}
