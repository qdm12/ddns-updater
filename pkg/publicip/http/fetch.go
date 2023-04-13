package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

var (
	ErrNoIPFound   = errors.New("no IP address found")
	ErrTooManyIPs  = errors.New("too many IP addresses")
	ErrIPMalformed = errors.New("IP address malformed")
	ErrBanned      = errors.New("we got banned")
)

var (
	ipv4Regex = regexp.MustCompile(`(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]|[0-9])`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             //nolint:lll
	ipv6Regex = regexp.MustCompile(`(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`) //nolint:lll
)

func fetch(ctx context.Context, client *http.Client, url string, version ipversion.IPVersion) (
	publicIP net.IP, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %d (%s)", ErrBanned,
			response.StatusCode, bodyToSingleLine(response.Body))
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = response.Body.Close()
	if err != nil {
		return nil, err
	}

	s := string(b)

	ipv4Strings := ipv4Regex.FindAllString(s, -1)
	ipv6Strings := ipv6Regex.FindAllString(s, -1)

	var ipString string
	switch version {
	case ipversion.IP4or6:
		switch {
		case len(ipv4Strings) == 1: // priority to IPv4
			ipString = ipv4Strings[0]
		case len(ipv6Strings) == 1:
			ipString = ipv6Strings[0]
		case len(ipv4Strings) > 1:
			return nil, fmt.Errorf("%w: found %d IPv4 addresses instead of 1",
				ErrTooManyIPs, len(ipv4Strings))
		case len(ipv6Strings) > 1:
			return nil, fmt.Errorf("%w: found %d IPv6 addresses instead of 1",
				ErrTooManyIPs, len(ipv6Strings))
		default:
			return nil, fmt.Errorf("%w: from %q", ErrNoIPFound, url)
		}
	case ipversion.IP4:
		switch len(ipv4Strings) {
		case 0:
			return nil, fmt.Errorf("%w: from %q for version %s", ErrNoIPFound, url, version)
		case 1:
			ipString = ipv4Strings[0]
		default:
			return nil, fmt.Errorf("%w: found %d IPv4 addresses instead of 1",
				ErrTooManyIPs, len(ipv4Strings))
		}
	case ipversion.IP6:
		switch len(ipv6Strings) {
		case 0:
			return nil, fmt.Errorf("%w: from %q for version %s", ErrNoIPFound, url, version)
		case 1:
			ipString = ipv6Strings[0]
		default:
			return nil, fmt.Errorf("%w: found %d IPv6 addresses instead of 1",
				ErrTooManyIPs, len(ipv6Strings))
		}
	}

	publicIP = net.ParseIP(ipString)
	if publicIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPMalformed, ipString)
	}

	return publicIP, nil
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
