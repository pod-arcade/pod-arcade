// Code borrowed from mochi-mqtt and adapted to also serve http traffic

package listeners

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/mochi-mqtt/server/v2/listeners"
)

var (
	ErrInvalidMessage = errors.New("message type not binary")
)

var _ listeners.Listener = (*Websocket)(nil)

// Websocket is a listener for establishing websocket connections.
type Websocket struct {
	sync.RWMutex
	id        string                // the internal id of the listener
	log       *slog.Logger          // server logger
	establish listeners.EstablishFn // the server's establish connection handler
	upgrader  *websocket.Upgrader   //  upgrade the incoming http/tcp connection to a websocket compliant connection.
}

// NewWebsocket initializes and returns a new Websocket listener, listening on an address.
func NewWebsocket(id string) *Websocket {
	return &Websocket{
		id: id,
		// listen:        server,
		upgrader: &websocket.Upgrader{
			Subprotocols: []string{"mqtt"},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// ID returns the id of the listener.
func (l *Websocket) Address() string {
	return ""
}

// ID returns the id of the listener.
func (l *Websocket) ID() string {
	return l.id
}

// Protocol returns the address of the listener.
func (l *Websocket) Protocol() string {
	return "ws"
}

// Init initializes the listener.
func (l *Websocket) Init(log *slog.Logger) error {
	l.log = log

	return nil
}

// handler upgrades and handles an incoming websocket connection.
func (l *Websocket) Handler(w http.ResponseWriter, r *http.Request) {

	c, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	err = l.establish(l.id, &wsConn{Conn: c.UnderlyingConn(), c: c})
	if err != nil {
		l.log.Warn("", "error", err)
	}
}

// Serve starts waiting for new Websocket connections, and calls the connection
// establishment callback for any received.
func (l *Websocket) Serve(establish listeners.EstablishFn) {
	l.establish = establish
}

// Close closes the listener and any client connections.
func (l *Websocket) Close(closeClients listeners.CloseFn) {
	l.Lock()
	defer l.Unlock()

	closeClients(l.id)
}

// wsConn is a websocket connection which satisfies the net.Conn interface.
type wsConn struct {
	net.Conn
	c *websocket.Conn

	// reader for the current message (can be nil)
	r io.Reader
}

// Read reads the next span of bytes from the websocket connection and returns the number of bytes read.
func (ws *wsConn) Read(p []byte) (int, error) {
	if ws.r == nil {
		op, r, err := ws.c.NextReader()
		if err != nil {
			return 0, err
		}

		if op != websocket.BinaryMessage {
			err = ErrInvalidMessage
			return 0, err
		}

		ws.r = r
	}

	var n int
	for {
		// buffer is full, return what we've read so far
		if n == len(p) {
			return n, nil
		}

		br, err := ws.r.Read(p[n:])
		n += br
		if err != nil {
			// when ANY error occurs, we consider this the end of the current message (either because it really is, via
			// io.EOF, or because something bad happened, in which case we want to drop the remainder)
			ws.r = nil

			if errors.Is(err, io.EOF) {
				err = nil
			}
			return n, err
		}
	}
}

// Write writes bytes to the websocket connection.
func (ws *wsConn) Write(p []byte) (int, error) {
	err := ws.c.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close signals the underlying websocket conn to close.
func (ws *wsConn) Close() error {
	return ws.Conn.Close()
}
