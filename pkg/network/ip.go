package network

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

var cidrs []*net.IPNet
var regexIP = regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`).FindString

func init() {
	maxCidrBlocks := [8]string{
		"127.0.0.1/8",    // localhost
		"10.0.0.0/8",     // 24-bit block
		"172.16.0.0/12",  // 20-bit block
		"192.168.0.0/16", // 16-bit block
		"169.254.0.0/16", // link local address
		"::1/128",        // localhost IPv6
		"fc00::/7",       // unique local address IPv6
		"fe80::/10",      // link local address IPv6
	}
	for _, maxCidrBlock := range maxCidrBlocks {
		_, cidr, err := net.ParseCIDR(maxCidrBlock)
		if err != nil {
			zap.S().Fatal(err)
		}
		cidrs = append(cidrs, cidr)
	}
}

func ipIsPrivate(ip string) (bool, error) {
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return false, fmt.Errorf("address %s is not valid", ip)
	}
	for i := range cidrs {
		if cidrs[i].Contains(netIP) {
			return true, nil
		}
	}
	return false, nil
}

func checkIP(ip string) error {
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return fmt.Errorf("address %s is not valid", ip)
	}
	return nil
}

// IPHeaders contains all the raw IP headers of an HTTP request
type IPHeaders struct {
	RemoteAddress string
	XRealIP       string
	XForwardedFor string
}

func (headers *IPHeaders) String() string {
	return fmt.Sprintf("remoteAddr=%s | xRealIP=%s | xForwardedFor=%s",
		headers.RemoteAddress, headers.XRealIP, headers.XForwardedFor)
}

func getRemoteIP(remoteAddr string) (ip string, err error) {
	ip = remoteAddr
	if strings.ContainsRune(ip, ':') {
		ip, _, err = net.SplitHostPort(ip)
		if err != nil {
			return "", err
		}
	}
	return ip, nil
}

func extractPublicIPs(ips []string) (publicIPs []string) {
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		private, err := ipIsPrivate(ip)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		if !private {
			publicIPs = append(publicIPs, ip)
		}
	}
	return publicIPs
}

func getXForwardedIPs(XForwardedFor string) (ips []string) {
	if len(XForwardedFor) == 0 {
		return nil
	}
	XForwardForIPs := strings.Split(XForwardedFor, ", ")
	if len(XForwardForIPs) == 1 {
		// In case it did not work, split with separator `,`
		XForwardForIPs = strings.Split(XForwardedFor, ",")
	}
	for _, ip := range XForwardForIPs {
		ip = strings.ReplaceAll(ip, " ", "")
		err := checkIP(ip)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

// GetClientIPHeaders returns the IP related HTTP headers from a request
func GetClientIPHeaders(r *http.Request) (headers IPHeaders) {
	headers.RemoteAddress = strings.ReplaceAll(r.RemoteAddr, " ", "")
	headers.XRealIP = strings.ReplaceAll(r.Header.Get("X-Real-Ip"), " ", "")
	headers.XForwardedFor = strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", "")
	return headers
}

// GetClientIP returns one single client IP address
func GetClientIP(r *http.Request) (ip string, err error) {
	headers := GetClientIPHeaders(r)
	// Extract relevant IP data from headers
	remoteIP, err := getRemoteIP(headers.RemoteAddress)
	if err != nil {
		return "", err
	}
	// No headers so it can only be RemoteAddress
	if headers.XRealIP == "" && headers.XForwardedFor == "" {
		return remoteIP, nil
	}
	// 3. RemoteAddress is the proxy server forwarding the IP so
	// we look into the HTTP headers to get the client IP
	xForwardedIPs := getXForwardedIPs(headers.XForwardedFor)
	// TODO check number of ips to match number of proxies setup
	publicXForwardedIPs := extractPublicIPs(xForwardedIPs)
	if len(publicXForwardedIPs) > 0 {
		// first XForwardedIP should be the client IP
		return publicXForwardedIPs[0], nil
	}
	if headers.XRealIP != "" {
		err := checkIP(headers.XRealIP)
		if err == nil {
			return headers.XRealIP, nil
		}
	}
	// latest private XForwardedFor IP
	if len(xForwardedIPs) > 0 {
		return xForwardedIPs[len(xForwardedIPs)-1], nil
	}
	return remoteIP, nil
}
