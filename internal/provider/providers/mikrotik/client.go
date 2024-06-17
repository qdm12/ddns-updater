package mikrotik

import (
	"io"
	"net"
	"net/netip"
	"sync"
)

type client struct {
	conn    io.Closer
	reader  *reader
	writer  *writer
	closing bool
	mutex   sync.Mutex
}

func newClient(address netip.AddrPort) (c *client, err error) {
	conn, err := net.Dial("tcp", address.String())
	if err != nil {
		return nil, err
	}
	return &client{
		conn:   conn,
		reader: newReader(conn),
		writer: newWriter(conn),
	}, nil
}

func (c *client) Close() {
	c.mutex.Lock()
	if c.closing {
		c.mutex.Unlock()
		return
	}
	c.closing = true
	c.mutex.Unlock()
	c.conn.Close()
}
