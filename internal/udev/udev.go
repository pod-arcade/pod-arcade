package udev

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
	eventemitter "github.com/vansante/go-event-emitter"
)

type UDev struct {
	closed bool
	// internal sockets
	kernelSock *NetlinkConnection
	udevSock   *NetlinkConnection

	// Used to listen to events from udev
	UDevEvents *eventemitter.Emitter
	// Used to listen to events from the Kernel
	KernelEvents *eventemitter.Emitter

	l   zerolog.Logger
	ctx context.Context
}

// mod with number of seconds in a year. I don't know how high these sequence numbers can get...
var seqNum = time.Now().Unix() % int64((24 * time.Hour * 30 * 12).Seconds())

func NewUDev(ctx context.Context) *UDev {
	var udev = &UDev{
		l:   log.NewLogger("UDEV", nil),
		ctx: ctx,
	}

	return udev
}

func (udev *UDev) Open() error {
	udev.kernelSock = NewNetlinkConnection(udev.ctx, KERNEL)
	udev.udevSock = NewNetlinkConnection(udev.ctx, UDEV)

	if err := udev.kernelSock.Connect(); err != nil {
		return err
	}

	if err := udev.udevSock.Connect(); err != nil {
		// If we can't connect to udev, we need to also close the kernel socket
		return errors.Join(err, udev.kernelSock.Close())
	}

	udev.KernelEvents = eventemitter.NewEmitter(false)
	udev.UDevEvents = eventemitter.NewEmitter(false)

	// run forever and read from uevents
	go func() {
		for !udev.closed {
			evt, err := udev.kernelSock.ReadUEvent()
			if err != nil {
				udev.l.Warn().Msgf("Error reading kernel UEvent - %v\n", err)
			} else {
				udev.KernelEvents.EmitEvent(eventemitter.EventType(evt.Action.String()), evt)
			}
		}
	}()

	return nil
}

func (u *UDev) Close() error {
	u.closed = true
	return errors.Join(
		u.kernelSock.Close(),
		u.udevSock.Close(),
	)
}

func (u *UDev) WriteUDevEvent(evt UEvent) error {
	seqNum++
	evt.Env["SEQNUM"] = fmt.Sprint(seqNum)
	u.UDevEvents.EmitEvent(eventemitter.EventType(evt.Action.String()), evt)
	return u.udevSock.WriteUEvent(evt)
}

func ToStringUEvent(evt UEvent) string {
	envString := ""
	for k, v := range evt.Env {
		envString += "\n\t" + k + "=" + v
	}
	return strings.Join([]string{
		"Action: " + evt.Action.String(),
		"KObj: " + evt.KObj,
		"Env: " + envString,
	}, "\n")
}
