//go:build linux

package udev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
)

type ConnectionMode int

const (
	KERNEL ConnectionMode = 1
	UDEV   ConnectionMode = 2
)

type NetlinkConnection struct {
	Fd   int
	Addr syscall.SockaddrNetlink
	Mode ConnectionMode

	ctx context.Context
	l   zerolog.Logger
}

func NewNetlinkConnection(ctx context.Context, mode ConnectionMode) *NetlinkConnection {
	return &NetlinkConnection{
		Mode: mode,
		l:    log.NewLogger("NetlinkConnection", nil),
		ctx:  ctx,
	}
}

func (c *NetlinkConnection) Connect() error {
	c.l.Trace().Msgf("Connecting to netlink in %v mode", c.Mode)

	// Create a socket we can use to listen for events
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return err
	}

	c.Fd = fd

	c.Addr = syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: uint32(c.Mode),
	}

	if err := syscall.Bind(c.Fd, &c.Addr); err != nil {
		syscall.Close(c.Fd)
		return err
	}

	return err
}

// Close allow to close file descriptor and socket bound
func (c *NetlinkConnection) Close() error {
	return syscall.Close(c.Fd)
}

func (c *NetlinkConnection) msgPeek() (int, *[]byte, error) {
	var n int
	var err error
	buf := make([]byte, os.Getpagesize())
	for {
		// Just read how many bytes are available in the socket
		// Warning: syscall.MSG_PEEK is a blocking call
		if n, _, err = syscall.Recvfrom(c.Fd, buf, syscall.MSG_PEEK); err != nil {
			return n, &buf, err
		}

		// If all message could be store inside the buffer : break
		if n < len(buf) {
			break
		}

		// Increase size of buffer if not enough
		buf = make([]byte, len(buf)+os.Getpagesize())
	}
	return n, &buf, err
}

func (c *NetlinkConnection) msgRead(buf *[]byte) error {
	if buf == nil {
		return errors.New("empty buffer")
	}

	n, _, err := syscall.Recvfrom(c.Fd, *buf, 0)
	if err != nil {
		return err
	}

	// Extract only real data from buffer and return that
	*buf = (*buf)[:n]

	return nil
}

// ReadMsg allow to read an entire uevent msg
func (c *NetlinkConnection) ReadMsg() (msg []byte, err error) {
	// Just read how many bytes are available in the socket
	_, buf, err := c.msgPeek()
	if err != nil {
		return nil, err
	}

	// Now read complete data
	err = c.msgRead(buf)

	return *buf, err
}

// ReadMsg allow to read an entire uevent msg
func (c *NetlinkConnection) ReadUEvent() (*UEvent, error) {
	if c.Mode != KERNEL {
		return nil, fmt.Errorf("ReadUEvent is only available in KernelEvent mode")
	}
	msg, err := c.ReadMsg()
	if err != nil {
		return nil, err
	}

	return ParseUEvent(msg)
}

func (c *NetlinkConnection) WriteUEvent(event UEvent) (err error) {
	if c.Mode != UDEV {
		return fmt.Errorf("WriteUEvent is only available in UdevEvent mode")
	}
	data := event.Bytes()
	err = syscall.Sendto(c.Fd, data, 0, &c.Addr)
	if err != nil {
		return err
	}
	return
}
