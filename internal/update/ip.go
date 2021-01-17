package update

import (
	"context"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/network"
)

const cycle = "cycle"

type IPGetter interface {
	IP(ctx context.Context) (ip net.IP, err error)
	IPv4(ctx context.Context) (ip net.IP, err error)
	IPv6(ctx context.Context) (ip net.IP, err error)
}

type ipGetter struct {
	client     *http.Client
	ipMethod   models.IPMethod
	ipv4Method models.IPMethod
	ipv6Method models.IPMethod
	cyclerIP   cycler
	cyclerIPv4 cycler
	cyclerIPv6 cycler
}

func NewIPGetter(client *http.Client, ipMethod, ipv4Method, ipv6Method models.IPMethod) IPGetter {
	ipMethods := []models.IPMethod{}
	ipv4Methods := []models.IPMethod{}
	ipv6Methods := []models.IPMethod{}
	for _, method := range constants.IPMethods() {
		switch {
		case method.IPv4 && method.IPv6:
			ipMethods = append(ipMethods, method)
		case method.IPv4:
			ipv4Methods = append(ipv4Methods, method)
		case method.IPv6:
			ipv6Methods = append(ipv6Methods, method)
		}
	}
	return &ipGetter{
		client:     client,
		ipMethod:   ipMethod,
		ipv4Method: ipv4Method,
		ipv6Method: ipv6Method,
		cyclerIP:   newCycler(ipMethods),
		cyclerIPv4: newCycler(ipv4Methods),
		cyclerIPv6: newCycler(ipv6Methods),
	}
}

func (i *ipGetter) IP(ctx context.Context) (ip net.IP, err error) {
	method := i.ipMethod
	if method.Name == cycle {
		method = i.cyclerIP.next()
	}
	return network.GetPublicIP(ctx, i.client, method.URL, constants.IPv4OrIPv6)
}

func (i *ipGetter) IPv4(ctx context.Context) (ip net.IP, err error) {
	method := i.ipv4Method
	if method.Name == cycle {
		method = i.cyclerIPv4.next()
	}
	return network.GetPublicIP(ctx, i.client, method.URL, constants.IPv4)
}

func (i *ipGetter) IPv6(ctx context.Context) (ip net.IP, err error) {
	method := i.ipv6Method
	if method.Name == cycle {
		method = i.cyclerIPv6.next()
	}
	return network.GetPublicIP(ctx, i.client, method.URL, constants.IPv6)
}
