package yamux

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type logCapture struct{ bytes.Buffer }

func (l *logCapture) logs() []string {
	return strings.Split(strings.TrimSpace(l.String()), "\n")
}

func (l *logCapture) match(expect []string) bool {
	return reflect.DeepEqual(l.logs(), expect)
}

func captureLogs(s *Session) *logCapture {
	buf := new(logCapture)
	s.logger = log.New(buf, "", 0)
	return buf
}

type pipeConn struct {
	reader       *io.PipeReader
	writer       *io.PipeWriter
	writeBlocker sync.Mutex
}

func (p *pipeConn) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

func (p *pipeConn) Write(b []byte) (int, error) {
	p.writeBlocker.Lock()
	defer p.writeBlocker.Unlock()
	return p.writer.Write(b)
}

func (p *pipeConn) Close() error {
	p.reader.Close()
	return p.writer.Close()
}

func testConn() (io.ReadWriteCloser, io.ReadWriteCloser) {
	read1, write1 := io.Pipe()
	read2, write2 := io.Pipe()
	conn1 := &pipeConn{reader: read1, writer: write2}
	conn2 := &pipeConn{reader: read2, writer: write1}
	return conn1, conn2
}

func testConf() *Config {
	conf := DefaultConfig()
	conf.AcceptBacklog = 64
	conf.KeepAliveInterval = 100 * time.Millisecond
	conf.ConnectionWriteTimeout = 250 * time.Millisecond
	return conf
}

func testConfNoKeepAlive() *Config {
	conf := testConf()
	conf.EnableKeepAlive = false
	return conf
}

func testClientServer() (*Session, *Session) {
	return testClientServerConfig(testConf())
}

func testClientServerConfig(conf *Config) (*Session, *Session) {
	conn1, conn2 := testConn()
	client, _ := Client(conn1, conf)
	server, _ := Server(conn2, conf)
	return client, server
}

func TestPing(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	rtt, err := client.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}

	rtt, err = server.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}
}

func TestPing_Timeout(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	// Prevent the client from responding
	clientConn := client.conn.(*pipeConn)
	clientConn.writeBlocker.Lock()

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Ping() // Ping via the server session
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != ErrTimeout {
			t.Fatalf("err: %v", err)
		}
	case <-time.After(client.config.ConnectionWriteTimeout * 2):
		t.Fatalf("failed to timeout within expected %v", client.config.ConnectionWriteTimeout)
	}

	// Verify that we recover, even if we gave up
	clientConn.writeBlocker.Unlock()

	go func() {
		_, err := server.Ping() // Ping via the server session
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	case <-time.After(client.config.ConnectionWriteTimeout):
		t.Fatalf("timeout")
	}
}

func TestCloseBeforeAck(t *testing.T) {
	cfg := testConf()
	cfg.AcceptBacklog = 8
	client, server := testClientServerConfig(cfg)

	defer client.Close()
	defer server.Close()

	for i := 0; i < 8; i++ {
		s, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}

	for i := 0; i < 8; i++ {
		s, err := server.AcceptStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}()

	select {
	case <-done:
	case <-time.After(time.Second * 5):
		t.Fatal("timed out trying to open stream")
	}
}

func TestAccept(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}

	wg := &sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 1 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 2 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := server.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 2 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 1 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(time.Second):
		panic("timeout")
	}
}

func TestNonNilInterface(t *testing.T) {
	_, server := testClientServer()
	server.Close()

	conn, err := server.Accept()
	if err != nil && conn != nil {
		t.Error("bad: accept should return a connection of nil value")
	}

	conn, err = server.Open()
	if err != nil && conn != nil {
		t.Error("bad: open should return a connection of nil value")
	}
}

func TestSendData_Small(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if server.NumStreams() != 1 {
			t.Fatalf("bad")
		}

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			n, err := stream.Read(buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short read: %d", n)
			}
			if string(buf) != "test" {
				t.Fatalf("bad: %s", buf)
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if client.NumStreams() != 1 {
			t.Fatalf("bad")
		}

		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte("test"))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short write %d", n)
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		panic("timeout")
	}

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}
}

func TestSendData_Large(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	const (
		sendSize = 250 * 1024 * 1024
		recvSize = 4 * 1024
	)

	data := make([]byte, sendSize)
	for idx := range data {
		data[idx] = byte(idx % 256)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		var sz int
		buf := make([]byte, recvSize)
		for i := 0; i < sendSize/recvSize; i++ {
			n, err := stream.Read(buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != recvSize {
				t.Fatalf("short read: %d", n)
			}
			sz += n
			for idx := range buf {
				if buf[idx] != byte(idx%256) {
					t.Fatalf("bad: %v %v %v", i, idx, buf[idx])
				}
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}

		t.Logf("cap=%d, n=%d\n", stream.recvBuf.Cap(), sz)
	}()

	go func() {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		n, err := stream.Write(data)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != len(data) {
			t.Fatalf("short write %d", n)
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		panic("timeout")
	}
}

func TestGoAway(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	if err := server.GoAway(); err != nil {
		t.Fatalf("err: %v", err)
	}

	_, err := client.Open()
	if err != ErrRemoteGoAway {
		t.Fatalf("err: %v", err)
	}
}

func TestManyStreams(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}

	acceptor := func(i int) {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 512)
		for {
			n, err := stream.Read(buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n == 0 {
				t.Fatalf("err: %v", err)
			}
		}
	}
	sender := func(i int) {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		msg := fmt.Sprintf("%08d", i)
		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte(msg))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != len(msg) {
				t.Fatalf("short write %d", n)
			}
		}
	}

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go acceptor(i)
		go sender(i)
	}

	wg.Wait()
}

func TestManyStreams_PingPong(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}

	ping := []byte("ping")
	pong := []byte("pong")

	acceptor := func(i int) {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for {
			// Read the 'ping'
			n, err := stream.Read(buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, ping) {
				t.Fatalf("bad: %s", buf)
			}

			// Shrink the internal buffer!
			stream.Shrink()

			// Write out the 'pong'
			n, err = stream.Write(pong)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
		}
	}
	sender := func(i int) {
		defer wg.Done()
		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			// Send the 'ping'
			n, err := stream.Write(ping)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short write %d", n)
			}

			// Read the 'pong'
			n, err = stream.Read(buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, pong) {
				t.Fatalf("bad: %s", buf)
			}

			// Shrink the buffer
			stream.Shrink()
		}
	}

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go acceptor(i)
		go sender(i)
	}

	wg.Wait()
}

func TestHalfClose(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err = stream.Write([]byte("a")); err != nil {
		t.Fatalf("err: %v", err)
	}

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	stream2.Close() // Half close

	buf := make([]byte, 4)
	n, err := stream2.Read(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 1 {
		t.Fatalf("bad: %v", n)
	}

	// Send more
	if _, err = stream.Write([]byte("bcd")); err != nil {
		t.Fatalf("err: %v", err)
	}
	stream.Close()

	// Read after close
	n, err = stream2.Read(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 3 {
		t.Fatalf("bad: %v", n)
	}

	// EOF after close
	n, err = stream2.Read(buf)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}
	if n != 0 {
		t.Fatalf("bad: %v", n)
	}
}

func TestReadDeadline(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 4)
	if _, err := stream.Read(buf); err != ErrTimeout {
		t.Fatalf("err: %v", err)
	}
}

func TestWriteDeadline(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 512)
	for i := 0; i < int(initialStreamWindow); i++ {
		_, err := stream.Write(buf)
		if err != nil && err == ErrTimeout {
			return
		} else if err != nil {
			t.Fatalf("err: %v", err)
		}
	}
	t.Fatalf("Expected timeout")
}

func TestBacklogExceeded(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	// Fill the backlog
	max := client.config.AcceptBacklog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Attempt to open a new stream
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Open()
		errCh <- err
	}()

	// Shutdown the server
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatalf("open should fail")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestKeepAlive(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	time.Sleep(200 * time.Millisecond)

	// Ping value should increase
	client.pingLock.Lock()
	defer client.pingLock.Unlock()
	if client.pingID == 0 {
		t.Fatalf("should ping")
	}

	server.pingLock.Lock()
	defer server.pingLock.Unlock()
	if server.pingID == 0 {
		t.Fatalf("should ping")
	}
}

func TestKeepAlive_Timeout(t *testing.T) {
	conn1, conn2 := testConn()

	clientConf := testConf()
	clientConf.ConnectionWriteTimeout = time.Hour // We're testing keep alives, not connection writes
	clientConf.EnableKeepAlive = false            // Just test one direction, so it's deterministic who hangs up on whom
	client, _ := Client(conn1, clientConf)
	defer client.Close()

	server, _ := Server(conn2, testConf())
	defer server.Close()

	_ = captureLogs(client) // Client logs aren't part of the test
	serverLogs := captureLogs(server)

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Accept() // Wait until server closes
		errCh <- err
	}()

	// Prevent the client from responding
	clientConn := client.conn.(*pipeConn)
	clientConn.writeBlocker.Lock()

	select {
	case err := <-errCh:
		if err != ErrKeepAliveTimeout {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for timeout")
	}

	if !server.IsClosed() {
		t.Fatalf("server should have closed")
	}

	if !serverLogs.match([]string{"[ERR] yamux: keepalive failed: i/o deadline reached"}) {
		t.Fatalf("server log incorect: %v", serverLogs.logs())
	}
}

func TestLargeWindow(t *testing.T) {
	conf := DefaultConfig()
	conf.MaxStreamWindowSize *= 2

	client, server := testClientServerConfig(conf)
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	stream.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	buf := make([]byte, conf.MaxStreamWindowSize)
	n, err := stream.Write(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("short write: %d", n)
	}
}

type UnlimitedReader struct{}

func (u *UnlimitedReader) Read(p []byte) (int, error) {
	runtime.Gosched()
	return len(p), nil
}

func TestSendData_VeryLarge(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	var n int64 = 1 * 1024 * 1024 * 1024
	var workers int = 16

	wg := &sync.WaitGroup{}
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			stream, err := server.AcceptStream()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()

			buf := make([]byte, 4)
			_, err = stream.Read(buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, []byte{0, 1, 2, 3}) {
				t.Fatalf("bad header")
			}

			recv, err := io.Copy(ioutil.Discard, stream)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if recv != n {
				t.Fatalf("bad: %v", recv)
			}
		}()
	}
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			stream, err := client.Open()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()

			_, err = stream.Write([]byte{0, 1, 2, 3})
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			unlimited := &UnlimitedReader{}
			sent, err := io.Copy(stream, io.LimitReader(unlimited, n))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if sent != n {
				t.Fatalf("bad: %v", sent)
			}
		}()
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(20 * time.Second):
		panic("timeout")
	}
}

func TestBacklogExceeded_Accept(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	max := 5 * client.config.AcceptBacklog
	go func() {
		for i := 0; i < max; i++ {
			stream, err := server.Accept()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()
		}
	}()

	// Fill the backlog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}

func TestSession_WindowUpdateWriteDuringRead(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Choose a huge flood size that we know will result in a window update.
	flood := int64(client.config.MaxStreamWindowSize) - 1

	// The server will accept a new stream and then flood data to it.
	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		n, err := stream.Write(make([]byte, flood))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if int64(n) != flood {
			t.Fatalf("short write: %d", n)
		}
	}()

	// The client will open a stream, block outbound writes, and then
	// listen to the flood from the server, which should time out since
	// it won't be able to send the window update.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		conn := client.conn.(*pipeConn)
		conn.writeBlocker.Lock()

		_, err = stream.Read(make([]byte, flood))
		if err != ErrConnectionWriteTimeout {
			t.Fatalf("err: %v", err)
		}
	}()

	wg.Wait()
}

func TestSession_PartialReadWindowUpdate(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	// Choose a huge flood size that we know will result in a window update.
	flood := int64(client.config.MaxStreamWindowSize)
	var wr *Stream

	// The server will accept a new stream and then flood data to it.
	go func() {
		defer wg.Done()

		var err error
		wr, err = server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer wr.Close()

		if wr.sendWindow != client.config.MaxStreamWindowSize {
			t.Fatalf("sendWindow: exp=%d, got=%d", client.config.MaxStreamWindowSize, wr.sendWindow)
		}

		n, err := wr.Write(make([]byte, flood))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if int64(n) != flood {
			t.Fatalf("short write: %d", n)
		}
		if wr.sendWindow != 0 {
			t.Fatalf("sendWindow: exp=%d, got=%d", 0, wr.sendWindow)
		}
	}()

	stream, err := client.OpenStream()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	wg.Wait()

	_, err = stream.Read(make([]byte, flood/2+1))

	if exp := uint32(flood/2 + 1); wr.sendWindow != exp {
		t.Errorf("sendWindow: exp=%d, got=%d", exp, wr.sendWindow)
	}
}

func TestSession_sendNoWait_Timeout(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()
	}()

	// The client will open the stream and then block outbound writes, we'll
	// probe sendNoWait once it gets into that state.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		conn := client.conn.(*pipeConn)
		conn.writeBlocker.Lock()

		hdr := header(make([]byte, headerSize))
		hdr.encode(typePing, flagACK, 0, 0)
		for {
			err = client.sendNoWait(hdr)
			if err == nil {
				continue
			} else if err == ErrConnectionWriteTimeout {
				break
			} else {
				t.Fatalf("err: %v", err)
			}
		}
	}()

	wg.Wait()
}

func TestSession_PingOfDeath(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	var doPingOfDeath sync.Mutex
	doPingOfDeath.Lock()

	// This is used later to block outbound writes.
	conn := server.conn.(*pipeConn)

	// The server will accept a stream, block outbound writes, and then
	// flood its send channel so that no more headers can be queued.
	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		conn.writeBlocker.Lock()
		for {
			hdr := header(make([]byte, headerSize))
			hdr.encode(typePing, 0, 0, 0)
			err = server.sendNoWait(hdr)
			if err == nil {
				continue
			} else if err == ErrConnectionWriteTimeout {
				break
			} else {
				t.Fatalf("err: %v", err)
			}
		}

		doPingOfDeath.Unlock()
	}()

	// The client will open a stream and then send the server a ping once it
	// can no longer write. This makes sure the server doesn't deadlock reads
	// while trying to reply to the ping with no ability to write.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		// This ping will never unblock because the ping id will never
		// show up in a response.
		doPingOfDeath.Lock()
		go func() { client.Ping() }()

		// Wait for a while to make sure the previous ping times out,
		// then turn writes back on and make sure a ping works again.
		time.Sleep(2 * server.config.ConnectionWriteTimeout)
		conn.writeBlocker.Unlock()
		if _, err = client.Ping(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	wg.Wait()
}

func TestSession_ConnectionWriteTimeout(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()
	}()

	// The client will open the stream and then block outbound writes, we'll
	// tee up a write and make sure it eventually times out.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		conn := client.conn.(*pipeConn)
		conn.writeBlocker.Lock()

		// Since the write goroutine is blocked then this will return a
		// timeout since it can't get feedback about whether the write
		// worked.
		n, err := stream.Write([]byte("hello"))
		if err != ErrConnectionWriteTimeout {
			t.Fatalf("err: %v", err)
		}
		if n != 0 {
			t.Fatalf("lied about writes: %d", n)
		}
	}()

	wg.Wait()
}
