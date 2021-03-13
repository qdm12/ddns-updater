package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
)

var (
	ErrNoIPFound   = errors.New("no IP address found")
	ErrTooManyIPs  = errors.New("too many IP addresses")
	ErrIPMalformed = errors.New("IP address malformed")
)

var (
	ipv4Regex = regexp.MustCompile(`(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]|[0-9])`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             //nolint:lll
	ipv6Regex = regexp.MustCompile(`(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`) //nolint:lll
)

func fetch(ctx context.Context, client *http.Client, url string) (
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

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := response.Body.Close(); err != nil {
		return nil, err
	}

	s := string(b)

	ipv4Strings := ipv4Regex.FindAllString(s, -1)
	ipv6Strings := ipv6Regex.FindAllString(s, -1)

	ipv4Count := len(ipv4Strings)
	ipv6Count := len(ipv6Strings)

	switch {
	case ipv4Count+ipv6Count == 0:
		return nil, ErrNoIPFound
	case ipv4Count > 0 && ipv6Count > 0:
		return nil, fmt.Errorf(
			"%w: found %d IPv4 addresses and %d IPv6 addresses, instead of a single one",
			ErrTooManyIPs, ipv4Count, ipv6Count)
	case ipv4Count+ipv6Count > 1:
		return nil, fmt.Errorf(
			"%w: found %d IP addresses instead of a single one",
			ErrTooManyIPs, ipv4Count+ipv6Count)
	}

	var ipString string
	if ipv4Count > 0 {
		ipString = ipv4Strings[0]
	} else {
		ipString = ipv6Strings[0]
	}

	publicIP = net.ParseIP(ipString)
	if publicIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPMalformed, ipString)
	}

	return publicIP, nil
}
