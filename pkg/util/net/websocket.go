package net

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

var ErrWebsocketListenerClosed = errors.New("websocket listener closed")

const (
	FrpWebsocketPath = "/~!frp"
)

type WebsocketListener struct {
	ln       net.Listener
	acceptCh chan net.Conn

	// paths are the URL paths accepted for websocket upgrade.
	paths []string
	// wsServer handles a single websocket upgrade on an already-prepared
	// connection. It is used both by the plaintext listener and by wss after
	// TLS termination (see HandleConn).
	wsServer *websocket.Server
	server   *http.Server
}

// IsWebsocketRequest reports whether the initial bytes read from a connection
// look like a websocket upgrade request targeting one of the given paths.
func IsWebsocketRequest(data []byte, paths []string) bool {
	const prefix = "GET "
	if len(data) < len(prefix) {
		return false
	}
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return false
	}
	rest := data[len(prefix):]
	for _, p := range paths {
		if len(rest) >= len(p) && bytes.Equal(rest[:len(p)], []byte(p)) {
			// Make sure the matched path is not a prefix of a longer one,
			// i.e. it is followed by a space (end of request line) or the
			// data we have read ends exactly at the path.
			if len(rest) == len(p) || rest[len(p)] == ' ' {
				return true
			}
		}
	}
	return false
}

// maxWebsocketPrefixLen returns the number of bytes that must be read so the
// muxer (or a peek) can decide whether a request targets one of paths.
func maxWebsocketPrefixLen(paths []string) int {
	n := len("GET ") + len(FrpWebsocketPath)
	for _, p := range paths {
		if l := len("GET ") + len(p); l > n {
			n = l
		}
	}
	return n
}

// NewWebsocketListener to handle websocket connections
// ln: tcp listener for websocket connections
// paths: the URL paths that are accepted for websocket upgrade. When empty,
// the default FrpWebsocketPath is used.
func NewWebsocketListener(ln net.Listener, paths ...string) (wl *WebsocketListener) {
	wl = &WebsocketListener{
		ln:       ln,
		acceptCh: make(chan net.Conn),
	}

	if len(paths) == 0 {
		paths = []string{FrpWebsocketPath}
	}
	wl.paths = paths

	handler := func(c *websocket.Conn) {
		// Reject requests whose path is not one of the configured paths.
		if !pathAllowed(c.Request().URL.Path, wl.paths) {
			c.Close()
			return
		}
		// The tunnel payload is a raw byte stream (yamux), not UTF-8 text.
		// Send it as binary frames; otherwise RFC 6455-compliant intermediaries
		// (e.g. API gateways/reverse proxies) UTF-8-validate the default text
		// frames and close the connection on invalid bytes.
		c.PayloadType = websocket.BinaryFrame
		notifyCh := make(chan struct{})
		// *websocket.Conn.RemoteAddr() reports the websocket origin URL rather
		// than the real client address, so surface the actual peer address
		// (recorded on the upgrade request) to frps.
		ra := remoteAddrFromRequest(c.Request(), c.RemoteAddr())
		conn := WrapCloseNotifyConn(&wsRemoteAddrConn{Conn: c, remoteAddr: ra}, func(_ error) {
			close(notifyCh)
		})
		wl.acceptCh <- conn
		<-notifyCh
	}

	wl.wsServer = &websocket.Server{
		Handler: websocket.Handler(handler),
		// Populate config.Origin during the handshake. *websocket.Conn's
		// RemoteAddr() returns the origin URL, so without this it would be
		// nil and RemoteAddr().String() would panic. This mirrors the
		// default checkOrigin behaviour used by websocket.Handler (which the
		// plaintext ws path relies on), while keeping frp's token-based auth
		// as the real access control.
		Handshake: func(config *websocket.Config, req *http.Request) error {
			origin, err := websocket.Origin(config, req)
			if err != nil {
				return err
			}
			if origin == nil {
				return errors.New("null origin")
			}
			config.Origin = origin
			return nil
		},
	}
	wl.server = &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           wl.wsServer,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		_ = wl.server.Serve(ln)
	}()
	return
}

// HandleConn upgrades an already-prepared connection (e.g. one whose TLS has
// already been terminated by the caller) to a websocket connection and hands
// it to the listener. It is used to support wss (websocket over TLS) where the
// TLS termination happens before the websocket muxer can see the plaintext
// "GET" request.
func (wl *WebsocketListener) HandleConn(c net.Conn) error {
	br := bufio.NewReader(c)
	req, err := http.ReadRequest(br)
	if err != nil {
		return err
	}
	// http.Server normally sets Request.RemoteAddr before invoking the
	// handler, but here we drive ServeHTTP directly, so record the real
	// client address (the underlying connection peer) ourselves. frps uses
	// it as the client address instead of the websocket origin URL.
	if req.RemoteAddr == "" {
		req.RemoteAddr = c.RemoteAddr().String()
	}
	w := &hijackResponseWriter{conn: c, buf: br}
	wl.wsServer.ServeHTTP(w, req)
	return nil
}

// wsRemoteAddrConn overrides RemoteAddr so that frps sees the real client
// address (from the upgrade request) instead of the websocket origin URL that
// *websocket.Conn.RemoteAddr would otherwise report.
type wsRemoteAddrConn struct {
	*websocket.Conn
	remoteAddr net.Addr
}

func (w *wsRemoteAddrConn) RemoteAddr() net.Addr { return w.remoteAddr }

// remoteAddrFromRequest returns the real client network address recorded on
// the websocket upgrade request. When it is unavailable, fallback (the
// websocket origin) is returned to avoid breaking existing behaviour.
func remoteAddrFromRequest(req *http.Request, fallback net.Addr) net.Addr {
	if req != nil && req.RemoteAddr != "" {
		if tcp, err := net.ResolveTCPAddr("tcp", req.RemoteAddr); err == nil {
			return tcp
		}
		return stringAddr(req.RemoteAddr)
	}
	return fallback
}

type stringAddr string

func (a stringAddr) Network() string { return "tcp" }
func (a stringAddr) String() string { return string(a) }

func (p *WebsocketListener) Accept() (net.Conn, error) {
	c, ok := <-p.acceptCh
	if !ok {
		return nil, ErrWebsocketListenerClosed
	}
	return c, nil
}

func (p *WebsocketListener) Close() error {
	return p.server.Close()
}

func (p *WebsocketListener) Addr() net.Addr {
	return p.ln.Addr()
}

// TryWebsocket peeks the initial bytes of a (TLS-terminated) connection to
// decide whether it is a websocket upgrade request for one of the given paths.
// It always returns a connection with the peeked bytes preserved so the caller
// can keep using it regardless of the result. The boolean reports whether the
// connection looks like a websocket request.
func TryWebsocket(c net.Conn, paths []string) (net.Conn, bool) {
	n := maxWebsocketPrefixLen(paths)
	if n < 256 {
		n = 256
	}
	buf := make([]byte, n)
	total := 0
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	for total < n {
		m, err := c.Read(buf[total:])
		if m > 0 {
			total += m
		}
		if err != nil {
			break
		}
		if bytes.Contains(buf[:total], []byte("\n")) {
			break
		}
	}
	_ = c.SetReadDeadline(time.Time{})
	data := buf[:total]
	restored := &peekedConn{Conn: c, buf: bytes.NewBuffer(data)}
	return restored, IsWebsocketRequest(data, paths)
}

func pathAllowed(path string, allowed []string) bool {
	for _, p := range allowed {
		if p == path {
			return true
		}
	}
	return false
}

// peekedConn wraps a net.Conn and replays previously read bytes before reading
// from the underlying connection.
type peekedConn struct {
	net.Conn
	buf *bytes.Buffer
}

func (p *peekedConn) Read(b []byte) (int, error) {
	if p.buf.Len() > 0 {
		return p.buf.Read(b)
	}
	return p.Conn.Read(b)
}

// hijackResponseWriter is a minimal http.ResponseWriter/http.Hijacker used to
// drive websocket.Server.ServeHTTP against a raw connection.
type hijackResponseWriter struct {
	conn net.Conn
	buf  *bufio.Reader
}

func (h *hijackResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (h *hijackResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (h *hijackResponseWriter) WriteHeader(int) {}

func (h *hijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.conn, bufio.NewReadWriter(h.buf, bufio.NewWriter(h.conn)), nil
}
