package udev

import (
	"io"
	"syscall"

	netlink "github.com/pilebones/go-udev/netlink"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
)

type UDevNetlinkConnection struct {
	closed bool
	netlink.UEventConn
	l zerolog.Logger
}

func (c *UDevNetlinkConnection) Write(event netlink.UEvent) (err error) {
	data := event.Bytes()
	err = syscall.Sendto(c.Fd, data, 0, &c.Addr)
	// If the underlying socket has been closed with Reader.Close()
	// syscall.Read() returns a -1 and an EBADF error.
	// This Read() function is called by bufio.Reader.ReadString() that
	// panics if a negative number of read bytes is returned.
	// Since the EBADF errors could either mean that the file
	// descriptor is not valid or not open for reading we keep track
	// if it's actually closed or not and return an io.EOF.
	if c.closed {
		return io.EOF
	}
	return
}

func (c *UDevNetlinkConnection) Close() error {
	if c.closed {
		// Already closed, nothing to do
		return nil
	}
	c.closed = true
	return syscall.Close(c.Fd)
}

func NewUdevNetlink(mode netlink.Mode) (*UDevNetlinkConnection, error) {
	conn := &UDevNetlinkConnection{
		l: logger.CreateLogger(map[string]string{
			"Component": "Netlink",
		}),
	}
	err := conn.Connect(mode)
	return conn, err
}
