package config

import (
	"net"
	"strconv"

	"github.com/qdm12/golibs/params"
)

type Health struct {
	ServerAddress string
	Port          uint16 // obtained from ServerAddress
}

func (h *Health) Get(env params.Env) (warning string, err error) {
	h.ServerAddress, warning, err = env.ListeningAddress(
		"HEALTH_SERVER_ADDRESS", params.Default("127.0.0.1:9999"))
	if err != nil {
		return warning, err
	}
	_, portStr, err := net.SplitHostPort(h.ServerAddress)
	if err != nil {
		return warning, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return warning, err
	}
	h.Port = uint16(port)
	return warning, nil
}
