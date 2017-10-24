package kcp

import (
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/ipv4"
)

type errTimeout struct {
	error
}

func (errTimeout) Timeout() bool   { return true }
func (errTimeout) Temporary() bool { return true }
func (errTimeout) Error() string   { return "i/o timeout" }

const (
	// 16-bytes magic number for each packet
	nonceSize = 16

	// 4-bytes packet checksum
	crcSize = 4

	// overall crypto header size
	cryptHeaderSize = nonceSize + crcSize

	// maximum packet size
	mtuLimit = 1500

	// FEC keeps rxFECMulti* (dataShard+parityShard) ordered packets in memory
	rxFECMulti = 3

	// accept backlog
	acceptBacklog = 128

	// prerouting(to session) queue
	qlen = 128
)

const (
	errBrokenPipe       = "broken pipe"
	errInvalidOperation = "invalid operation"
)

var (
	// global packet buffer
	// shared among sending/receiving/FEC
	xmitBuf sync.Pool
)

func init() {
	xmitBuf.New = func() interface{} {
		return make([]byte, mtuLimit)
	}
}

type (
	// UDPSession defines a KCP session implemented by UDP
	UDPSession struct {
		updaterIdx int            // record slice index in updater
		conn       net.PacketConn // the underlying packet connection
		kcp        *KCP           // KCP ARQ protocol
		l          *Listener      // point to the Listener if it's accepted by Listener
		block      BlockCrypt     // block encryption

		// kcp receiving is based on packets
		// recvbuf turns packets into stream
		recvbuf []byte
		bufptr  []byte
		// extended output buffer(with header)
		ext []byte

		// FEC
		fecDecoder *fecDecoder
		fecEncoder *fecEncoder

		// settings
		remote     net.Addr  // remote peer address
		rd         time.Time // read deadline
		wd         time.Time // write deadline
		headerSize int       // the overall header size added before KCP frame
		ackNoDelay bool      // send ack immediately for each incoming packet
		writeDelay bool      // delay kcp.flush() for Write() for bulk transfer
		dup        int       // duplicate udp packets

		// notifications
		die          chan struct{} // notify session has Closed
		chReadEvent  chan struct{} // notify Read() can be called without blocking
		chWriteEvent chan struct{} // notify Write() can be called without blocking
		chErrorEvent chan error    // notify Read() have an error

		isClosed bool // flag the session has Closed
		mu       sync.Mutex
	}

	setReadBuffer interface {
		SetReadBuffer(bytes int) error
	}

	setWriteBuffer interface {
		SetWriteBuffer(bytes int) error
	}
)

// newUDPSession create a new udp session for client or server
func newUDPSession(conv uint32, dataShards, parityShards int, l *Listener, conn net.PacketConn, remote net.Addr, block BlockCrypt) *UDPSession {
	sess := new(UDPSession)
	sess.die = make(chan struct{})
	sess.chReadEvent = make(chan struct{}, 1)
	sess.chWriteEvent = make(chan struct{}, 1)
	sess.chErrorEvent = make(chan error, 1)
	sess.remote = remote
	sess.conn = conn
	sess.l = l
	sess.block = block
	sess.recvbuf = make([]byte, mtuLimit)

	// FEC initialization
	sess.fecDecoder = newFECDecoder(rxFECMulti*(dataShards+parityShards), dataShards, parityShards)
	if sess.block != nil {
		sess.fecEncoder = newFECEncoder(dataShards, parityShards, cryptHeaderSize)
	} else {
		sess.fecEncoder = newFECEncoder(dataShards, parityShards, 0)
	}

	// calculate header size
	if sess.block != nil {
		sess.headerSize += cryptHeaderSize
	}
	if sess.fecEncoder != nil {
		sess.headerSize += fecHeaderSizePlus2
	}

	// only allocate extended packet buffer
	// when the extra header is required
	if sess.headerSize > 0 {
		sess.ext = make([]byte, mtuLimit)
	}

	sess.kcp = NewKCP(conv, func(buf []byte, size int) {
		if size >= IKCP_OVERHEAD {
			sess.output(buf[:size])
		}
	})
	sess.kcp.SetMtu(IKCP_MTU_DEF - sess.headerSize)

	// add current session to the global updater,
	// which periodically calls sess.update()
	updater.addSession(sess)

	if sess.l == nil { // it's a client connection
		go sess.readLoop()
		atomic.AddUint64(&DefaultSnmp.ActiveOpens, 1)
	} else {
		atomic.AddUint64(&DefaultSnmp.PassiveOpens, 1)
	}
	currestab := atomic.AddUint64(&DefaultSnmp.CurrEstab, 1)
	maxconn := atomic.LoadUint64(&DefaultSnmp.MaxConn)
	if currestab > maxconn {
		atomic.CompareAndSwapUint64(&DefaultSnmp.MaxConn, maxconn, currestab)
	}

	return sess
}

// Read implements net.Conn
func (s *UDPSession) Read(b []byte) (n int, err error) {
	for {
		s.mu.Lock()
		if len(s.bufptr) > 0 { // copy from buffer into b
			n = copy(b, s.bufptr)
			s.bufptr = s.bufptr[n:]
			s.mu.Unlock()
			return n, nil
		}

		if s.isClosed {
			s.mu.Unlock()
			return 0, errors.New(errBrokenPipe)
		}

		if size := s.kcp.PeekSize(); size > 0 { // peek data size from kcp
			atomic.AddUint64(&DefaultSnmp.BytesReceived, uint64(size))
			if len(b) >= size { // direct write to b
				s.kcp.Recv(b)
				s.mu.Unlock()
				return size, nil
			}

			// resize kcp receive buffer
			// to make sure recvbuf has enough capacity
			if cap(s.recvbuf) < size {
				s.recvbuf = make([]byte, size)
			}

			// resize recvbuf slice length
			s.recvbuf = s.recvbuf[:size]
			s.kcp.Recv(s.recvbuf)
			n = copy(b, s.recvbuf)   // copy to b
			s.bufptr = s.recvbuf[n:] // update pointer
			s.mu.Unlock()
			return n, nil
		}

		// read deadline
		var timeout *time.Timer
		var c <-chan time.Time
		if !s.rd.IsZero() {
			if time.Now().After(s.rd) {
				s.mu.Unlock()
				return 0, errTimeout{}
			}

			delay := s.rd.Sub(time.Now())
			timeout = time.NewTimer(delay)
			c = timeout.C
		}
		s.mu.Unlock()

		// wait for read event or timeout
		select {
		case <-s.chReadEvent:
		case <-c:
		case <-s.die:
		case err = <-s.chErrorEvent:
			if timeout != nil {
				timeout.Stop()
			}
			return n, err
		}

		if timeout != nil {
			timeout.Stop()
		}
	}
}

// Write implements net.Conn
func (s *UDPSession) Write(b []byte) (n int, err error) {
	for {
		s.mu.Lock()
		if s.isClosed {
			s.mu.Unlock()
			return 0, errors.New(errBrokenPipe)
		}

		// api flow control
		if s.kcp.WaitSnd() < int(s.kcp.snd_wnd) {
			n = len(b)
			for {
				if len(b) <= int(s.kcp.mss) {
					s.kcp.Send(b)
					break
				} else {
					s.kcp.Send(b[:s.kcp.mss])
					b = b[s.kcp.mss:]
				}
			}

			if !s.writeDelay {
				s.kcp.flush(false)
			}
			s.mu.Unlock()
			atomic.AddUint64(&DefaultSnmp.BytesSent, uint64(n))
			return n, nil
		}

		// write deadline
		var timeout *time.Timer
		var c <-chan time.Time
		if !s.wd.IsZero() {
			if time.Now().After(s.wd) {
				s.mu.Unlock()
				return 0, errTimeout{}
			}
			delay := s.wd.Sub(time.Now())
			timeout = time.NewTimer(delay)
			c = timeout.C
		}
		s.mu.Unlock()

		// wait for write event or timeout
		select {
		case <-s.chWriteEvent:
		case <-c:
		case <-s.die:
		}

		if timeout != nil {
			timeout.Stop()
		}
	}
}

// Close closes the connection.
func (s *UDPSession) Close() error {
	// remove this session from updater & listener(if necessary)
	updater.removeSession(s)
	if s.l != nil { // notify listener
		s.l.closeSession(sessionKey{
			addr:   s.remote.String(),
			convID: s.kcp.conv,
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isClosed {
		return errors.New(errBrokenPipe)
	}
	close(s.die)
	s.isClosed = true
	atomic.AddUint64(&DefaultSnmp.CurrEstab, ^uint64(0))
	if s.l == nil { // client socket close
		return s.conn.Close()
	}
	return nil
}

// LocalAddr returns the local network address. The Addr returned is shared by all invocations of LocalAddr, so do not modify it.
func (s *UDPSession) LocalAddr() net.Addr { return s.conn.LocalAddr() }

// RemoteAddr returns the remote network address. The Addr returned is shared by all invocations of RemoteAddr, so do not modify it.
func (s *UDPSession) RemoteAddr() net.Addr { return s.remote }

// SetDeadline sets the deadline associated with the listener. A zero time value disables the deadline.
func (s *UDPSession) SetDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rd = t
	s.wd = t
	return nil
}

// SetReadDeadline implements the Conn SetReadDeadline method.
func (s *UDPSession) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rd = t
	return nil
}

// SetWriteDeadline implements the Conn SetWriteDeadline method.
func (s *UDPSession) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wd = t
	return nil
}

// SetWriteDelay delays write for bulk transfer until the next update interval
func (s *UDPSession) SetWriteDelay(delay bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDelay = delay
}

// SetWindowSize set maximum window size
func (s *UDPSession) SetWindowSize(sndwnd, rcvwnd int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kcp.WndSize(sndwnd, rcvwnd)
}

// SetMtu sets the maximum transmission unit(not including UDP header)
func (s *UDPSession) SetMtu(mtu int) bool {
	if mtu > mtuLimit {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.kcp.SetMtu(mtu - s.headerSize)
	return true
}

// SetStreamMode toggles the stream mode on/off
func (s *UDPSession) SetStreamMode(enable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if enable {
		s.kcp.stream = 1
	} else {
		s.kcp.stream = 0
	}
}

// SetACKNoDelay changes ack flush option, set true to flush ack immediately,
func (s *UDPSession) SetACKNoDelay(nodelay bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ackNoDelay = nodelay
}

// SetDUP duplicates udp packets for kcp output, for testing purpose only
func (s *UDPSession) SetDUP(dup int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dup = dup
}

// SetNoDelay calls nodelay() of kcp
// https://github.com/skywind3000/kcp/blob/master/README.en.md#protocol-configuration
func (s *UDPSession) SetNoDelay(nodelay, interval, resend, nc int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kcp.NoDelay(nodelay, interval, resend, nc)
}

// SetDSCP sets the 6bit DSCP field of IP header, no effect if it's accepted from Listener
func (s *UDPSession) SetDSCP(dscp int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.l == nil {
		if nc, ok := s.conn.(*connectedUDPConn); ok {
			return ipv4.NewConn(nc.UDPConn).SetTOS(dscp << 2)
		} else if nc, ok := s.conn.(net.Conn); ok {
			return ipv4.NewConn(nc).SetTOS(dscp << 2)
		}
	}
	return errors.New(errInvalidOperation)
}

// SetReadBuffer sets the socket read buffer, no effect if it's accepted from Listener
func (s *UDPSession) SetReadBuffer(bytes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.l == nil {
		if nc, ok := s.conn.(setReadBuffer); ok {
			return nc.SetReadBuffer(bytes)
		}
	}
	return errors.New(errInvalidOperation)
}

// SetWriteBuffer sets the socket write buffer, no effect if it's accepted from Listener
func (s *UDPSession) SetWriteBuffer(bytes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.l == nil {
		if nc, ok := s.conn.(setWriteBuffer); ok {
			return nc.SetWriteBuffer(bytes)
		}
	}
	return errors.New(errInvalidOperation)
}

// output pipeline entry
// steps for output data processing:
// 0. Header extends
// 1. FEC
// 2. CRC32
// 3. Encryption
// 4. WriteTo kernel
func (s *UDPSession) output(buf []byte) {
	var ecc [][]byte

	// 0. extend buf's header space(if necessary)
	ext := buf
	if s.headerSize > 0 {
		ext = s.ext[:s.headerSize+len(buf)]
		copy(ext[s.headerSize:], buf)
	}

	// 1. FEC encoding
	if s.fecEncoder != nil {
		ecc = s.fecEncoder.encode(ext)
	}

	// 2&3. crc32 & encryption
	if s.block != nil {
		io.ReadFull(rand.Reader, ext[:nonceSize])
		checksum := crc32.ChecksumIEEE(ext[cryptHeaderSize:])
		binary.LittleEndian.PutUint32(ext[nonceSize:], checksum)
		s.block.Encrypt(ext, ext)

		for k := range ecc {
			io.ReadFull(rand.Reader, ecc[k][:nonceSize])
			checksum := crc32.ChecksumIEEE(ecc[k][cryptHeaderSize:])
			binary.LittleEndian.PutUint32(ecc[k][nonceSize:], checksum)
			s.block.Encrypt(ecc[k], ecc[k])
		}
	}

	// 4. WriteTo kernel
	nbytes := 0
	npkts := 0
	for i := 0; i < s.dup+1; i++ {
		if n, err := s.conn.WriteTo(ext, s.remote); err == nil {
			nbytes += n
			npkts++
		}
	}

	for k := range ecc {
		if n, err := s.conn.WriteTo(ecc[k], s.remote); err == nil {
			nbytes += n
			npkts++
		}
	}
	atomic.AddUint64(&DefaultSnmp.OutPkts, uint64(npkts))
	atomic.AddUint64(&DefaultSnmp.OutBytes, uint64(nbytes))
}

// kcp update, returns interval for next calling
func (s *UDPSession) update() (interval time.Duration) {
	s.mu.Lock()
	s.kcp.flush(false)
	if s.kcp.WaitSnd() < int(s.kcp.snd_wnd) {
		s.notifyWriteEvent()
	}
	interval = time.Duration(s.kcp.interval) * time.Millisecond
	s.mu.Unlock()
	return
}

// GetConv gets conversation id of a session
func (s *UDPSession) GetConv() uint32 { return s.kcp.conv }

func (s *UDPSession) notifyReadEvent() {
	select {
	case s.chReadEvent <- struct{}{}:
	default:
	}
}

func (s *UDPSession) notifyWriteEvent() {
	select {
	case s.chWriteEvent <- struct{}{}:
	default:
	}
}

func (s *UDPSession) kcpInput(data []byte) {
	var kcpInErrors, fecErrs, fecRecovered, fecParityShards uint64

	if s.fecDecoder != nil {
		f := s.fecDecoder.decodeBytes(data)
		s.mu.Lock()
		if f.flag == typeData {
			if ret := s.kcp.Input(data[fecHeaderSizePlus2:], true, s.ackNoDelay); ret != 0 {
				kcpInErrors++
			}
		}

		if f.flag == typeData || f.flag == typeFEC {
			if f.flag == typeFEC {
				fecParityShards++
			}

			recovers := s.fecDecoder.decode(f)
			for _, r := range recovers {
				if len(r) >= 2 { // must be larger than 2bytes
					sz := binary.LittleEndian.Uint16(r)
					if int(sz) <= len(r) && sz >= 2 {
						if ret := s.kcp.Input(r[2:sz], false, s.ackNoDelay); ret == 0 {
							fecRecovered++
						} else {
							kcpInErrors++
						}
					} else {
						fecErrs++
					}
				} else {
					fecErrs++
				}
			}
		}

		// notify reader
		if n := s.kcp.PeekSize(); n > 0 {
			s.notifyReadEvent()
		}
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		if ret := s.kcp.Input(data, true, s.ackNoDelay); ret != 0 {
			kcpInErrors++
		}
		// notify reader
		if n := s.kcp.PeekSize(); n > 0 {
			s.notifyReadEvent()
		}
		s.mu.Unlock()
	}

	atomic.AddUint64(&DefaultSnmp.InPkts, 1)
	atomic.AddUint64(&DefaultSnmp.InBytes, uint64(len(data)))
	if fecParityShards > 0 {
		atomic.AddUint64(&DefaultSnmp.FECParityShards, fecParityShards)
	}
	if kcpInErrors > 0 {
		atomic.AddUint64(&DefaultSnmp.KCPInErrors, kcpInErrors)
	}
	if fecErrs > 0 {
		atomic.AddUint64(&DefaultSnmp.FECErrs, fecErrs)
	}
	if fecRecovered > 0 {
		atomic.AddUint64(&DefaultSnmp.FECRecovered, fecRecovered)
	}
}

func (s *UDPSession) receiver(ch chan<- []byte) {
	for {
		data := xmitBuf.Get().([]byte)[:mtuLimit]
		if n, _, err := s.conn.ReadFrom(data); err == nil && n >= s.headerSize+IKCP_OVERHEAD {
			select {
			case ch <- data[:n]:
			case <-s.die:
				return
			}
		} else if err != nil {
			s.chErrorEvent <- err
			return
		} else {
			atomic.AddUint64(&DefaultSnmp.InErrs, 1)
		}
	}
}

// read loop for client session
func (s *UDPSession) readLoop() {
	chPacket := make(chan []byte, qlen)
	go s.receiver(chPacket)

	for {
		select {
		case data := <-chPacket:
			raw := data
			dataValid := false
			if s.block != nil {
				s.block.Decrypt(data, data)
				data = data[nonceSize:]
				checksum := crc32.ChecksumIEEE(data[crcSize:])
				if checksum == binary.LittleEndian.Uint32(data) {
					data = data[crcSize:]
					dataValid = true
				} else {
					atomic.AddUint64(&DefaultSnmp.InCsumErrors, 1)
				}
			} else if s.block == nil {
				dataValid = true
			}

			if dataValid {
				s.kcpInput(data)
			}
			xmitBuf.Put(raw)
		case <-s.die:
			return
		}
	}
}

type (
	sessionKey struct {
		addr   string
		convID uint32
	}

	// Listener defines a server listening for connections
	Listener struct {
		block        BlockCrypt     // block encryption
		dataShards   int            // FEC data shard
		parityShards int            // FEC parity shard
		fecDecoder   *fecDecoder    // FEC mock initialization
		conn         net.PacketConn // the underlying packet connection

		sessions        map[sessionKey]*UDPSession // all sessions accepted by this Listener
		chAccepts       chan *UDPSession           // Listen() backlog
		chSessionClosed chan sessionKey            // session close queue
		headerSize      int                        // the overall header size added before KCP frame
		die             chan struct{}              // notify the listener has closed
		rd              atomic.Value               // read deadline for Accept()
		wd              atomic.Value
	}

	// incoming packet
	inPacket struct {
		from net.Addr
		data []byte
	}
)

// monitor incoming data for all connections of server
func (l *Listener) monitor() {
	// cache last session
	var lastKey sessionKey
	var lastSession *UDPSession

	chPacket := make(chan inPacket, qlen)
	go l.receiver(chPacket)
	for {
		select {
		case p := <-chPacket:
			raw := p.data
			data := p.data
			from := p.from
			dataValid := false
			if l.block != nil {
				l.block.Decrypt(data, data)
				data = data[nonceSize:]
				checksum := crc32.ChecksumIEEE(data[crcSize:])
				if checksum == binary.LittleEndian.Uint32(data) {
					data = data[crcSize:]
					dataValid = true
				} else {
					atomic.AddUint64(&DefaultSnmp.InCsumErrors, 1)
				}
			} else if l.block == nil {
				dataValid = true
			}

			if dataValid {
				var conv uint32
				convValid := false
				if l.fecDecoder != nil {
					isfec := binary.LittleEndian.Uint16(data[4:])
					if isfec == typeData {
						conv = binary.LittleEndian.Uint32(data[fecHeaderSizePlus2:])
						convValid = true
					}
				} else {
					conv = binary.LittleEndian.Uint32(data)
					convValid = true
				}

				if convValid {
					key := sessionKey{
						addr:   from.String(),
						convID: conv,
					}
					var s *UDPSession
					var ok bool

					// packets received from an address always come in batch.
					// cache the session for next packet, without querying map.
					if key == lastKey {
						s, ok = lastSession, true
					} else if s, ok = l.sessions[key]; ok {
						lastSession = s
						lastKey = key
					}

					if !ok { // new session
						if len(l.chAccepts) < cap(l.chAccepts) && len(l.sessions) < 4096 { // do not let new session overwhelm accept queue and connection count
							s := newUDPSession(conv, l.dataShards, l.parityShards, l, l.conn, from, l.block)
							s.kcpInput(data)
							l.sessions[key] = s
							l.chAccepts <- s
						}
					} else {
						s.kcpInput(data)
					}
				}
			}

			xmitBuf.Put(raw)
		case key := <-l.chSessionClosed:
			if key == lastKey {
				lastKey = sessionKey{}
			}
			delete(l.sessions, key)
		case <-l.die:
			return
		}
	}
}

func (l *Listener) receiver(ch chan<- inPacket) {
	for {
		data := xmitBuf.Get().([]byte)[:mtuLimit]
		if n, from, err := l.conn.ReadFrom(data); err == nil && n >= l.headerSize+IKCP_OVERHEAD {
			select {
			case ch <- inPacket{from, data[:n]}:
			case <-l.die:
				return
			}
		} else if err != nil {
			return
		} else {
			atomic.AddUint64(&DefaultSnmp.InErrs, 1)
		}
	}
}

// SetReadBuffer sets the socket read buffer for the Listener
func (l *Listener) SetReadBuffer(bytes int) error {
	if nc, ok := l.conn.(setReadBuffer); ok {
		return nc.SetReadBuffer(bytes)
	}
	return errors.New(errInvalidOperation)
}

// SetWriteBuffer sets the socket write buffer for the Listener
func (l *Listener) SetWriteBuffer(bytes int) error {
	if nc, ok := l.conn.(setWriteBuffer); ok {
		return nc.SetWriteBuffer(bytes)
	}
	return errors.New(errInvalidOperation)
}

// SetDSCP sets the 6bit DSCP field of IP header
func (l *Listener) SetDSCP(dscp int) error {
	if nc, ok := l.conn.(net.Conn); ok {
		return ipv4.NewConn(nc).SetTOS(dscp << 2)
	}
	return errors.New(errInvalidOperation)
}

// Accept implements the Accept method in the Listener interface; it waits for the next call and returns a generic Conn.
func (l *Listener) Accept() (net.Conn, error) {
	return l.AcceptKCP()
}

// AcceptKCP accepts a KCP connection
func (l *Listener) AcceptKCP() (*UDPSession, error) {
	var timeout <-chan time.Time
	if tdeadline, ok := l.rd.Load().(time.Time); ok && !tdeadline.IsZero() {
		timeout = time.After(tdeadline.Sub(time.Now()))
	}

	select {
	case <-timeout:
		return nil, &errTimeout{}
	case c := <-l.chAccepts:
		return c, nil
	case <-l.die:
		return nil, errors.New(errBrokenPipe)
	}
}

// SetDeadline sets the deadline associated with the listener. A zero time value disables the deadline.
func (l *Listener) SetDeadline(t time.Time) error {
	l.SetReadDeadline(t)
	l.SetWriteDeadline(t)
	return nil
}

// SetReadDeadline implements the Conn SetReadDeadline method.
func (l *Listener) SetReadDeadline(t time.Time) error {
	l.rd.Store(t)
	return nil
}

// SetWriteDeadline implements the Conn SetWriteDeadline method.
func (l *Listener) SetWriteDeadline(t time.Time) error {
	l.wd.Store(t)
	return nil
}

// Close stops listening on the UDP address. Already Accepted connections are not closed.
func (l *Listener) Close() error {
	close(l.die)
	return l.conn.Close()
}

// closeSession notify the listener that a session has closed
func (l *Listener) closeSession(key sessionKey) bool {
	select {
	case l.chSessionClosed <- key:
		return true
	case <-l.die:
		return false
	}
}

// Addr returns the listener's network address, The Addr returned is shared by all invocations of Addr, so do not modify it.
func (l *Listener) Addr() net.Addr { return l.conn.LocalAddr() }

// Listen listens for incoming KCP packets addressed to the local address laddr on the network "udp",
func Listen(laddr string) (net.Listener, error) { return ListenWithOptions(laddr, nil, 0, 0) }

// ListenWithOptions listens for incoming KCP packets addressed to the local address laddr on the network "udp" with packet encryption,
// dataShards, parityShards defines Reed-Solomon Erasure Coding parameters
func ListenWithOptions(laddr string, block BlockCrypt, dataShards, parityShards int) (*Listener, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.ResolveUDPAddr")
	}
	conn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.ListenUDP")
	}

	return ServeConn(block, dataShards, parityShards, conn)
}

// ServeConn serves KCP protocol for a single packet connection.
func ServeConn(block BlockCrypt, dataShards, parityShards int, conn net.PacketConn) (*Listener, error) {
	l := new(Listener)
	l.conn = conn
	l.sessions = make(map[sessionKey]*UDPSession)
	l.chAccepts = make(chan *UDPSession, acceptBacklog)
	l.chSessionClosed = make(chan sessionKey)
	l.die = make(chan struct{})
	l.dataShards = dataShards
	l.parityShards = parityShards
	l.block = block
	l.fecDecoder = newFECDecoder(rxFECMulti*(dataShards+parityShards), dataShards, parityShards)

	// calculate header size
	if l.block != nil {
		l.headerSize += cryptHeaderSize
	}
	if l.fecDecoder != nil {
		l.headerSize += fecHeaderSizePlus2
	}

	go l.monitor()
	return l, nil
}

// Dial connects to the remote address "raddr" on the network "udp"
func Dial(raddr string) (net.Conn, error) { return DialWithOptions(raddr, nil, 0, 0) }

// DialWithOptions connects to the remote address "raddr" on the network "udp" with packet encryption
func DialWithOptions(raddr string, block BlockCrypt, dataShards, parityShards int) (*UDPSession, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.ResolveUDPAddr")
	}

	udpconn, err := net.DialUDP("udp", nil, udpaddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.DialUDP")
	}

	return NewConn(raddr, block, dataShards, parityShards, &connectedUDPConn{udpconn})
}

// NewConn establishes a session and talks KCP protocol over a packet connection.
func NewConn(raddr string, block BlockCrypt, dataShards, parityShards int, conn net.PacketConn) (*UDPSession, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.ResolveUDPAddr")
	}

	var convid uint32
	binary.Read(rand.Reader, binary.LittleEndian, &convid)
	return newUDPSession(convid, dataShards, parityShards, nil, conn, udpaddr, block), nil
}

func NewConnEx(convid uint32, connected bool, raddr string, block BlockCrypt, dataShards, parityShards int, conn *net.UDPConn) (*UDPSession, error) {
	udpaddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, errors.Wrap(err, "net.ResolveUDPAddr")
	}

	var pConn net.PacketConn = conn
	if connected {
		pConn = &connectedUDPConn{conn}
	}

	return newUDPSession(convid, dataShards, parityShards, nil, pConn, udpaddr, block), nil
}

// returns current time in milliseconds
func currentMs() uint32 { return uint32(time.Now().UnixNano() / int64(time.Millisecond)) }

// connectedUDPConn is a wrapper for net.UDPConn which converts WriteTo syscalls
// to Write syscalls that are 4 times faster on some OS'es. This should only be
// used for connections that were produced by a net.Dial* call.
type connectedUDPConn struct{ *net.UDPConn }

// WriteTo redirects all writes to the Write syscall, which is 4 times faster.
func (c *connectedUDPConn) WriteTo(b []byte, addr net.Addr) (int, error) { return c.Write(b) }
