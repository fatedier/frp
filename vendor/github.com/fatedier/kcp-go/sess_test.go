package kcp

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const portEcho = "127.0.0.1:9999"
const portSink = "127.0.0.1:19999"
const portTinyBufferEcho = "127.0.0.1:29999"
const portListerner = "127.0.0.1:9998"
const salt = "kcptest"

var key = []byte("testkey")
var fec = 4
var pass = pbkdf2.Key(key, []byte(portSink), 4096, 32, sha1.New)

func init() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	go echoServer()
	go sinkServer()
	go tinyBufferEchoServer()
	println("beginning tests, encryption:salsa20, fec:10/3")
}

func dialEcho() (*UDPSession, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	sess, err := DialWithOptions(portEcho, block, 10, 3)
	if err != nil {
		panic(err)
	}

	sess.SetStreamMode(true)
	sess.SetStreamMode(false)
	sess.SetStreamMode(true)
	sess.SetWindowSize(4096, 4096)
	sess.SetReadBuffer(4 * 1024 * 1024)
	sess.SetWriteBuffer(4 * 1024 * 1024)
	sess.SetStreamMode(true)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetMtu(1400)
	sess.SetMtu(1600)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)
	sess.SetDeadline(time.Now().Add(time.Minute))
	return sess, err
}

func dialSink() (*UDPSession, error) {
	sess, err := DialWithOptions(portSink, nil, 0, 0)
	if err != nil {
		panic(err)
	}

	sess.SetStreamMode(true)
	sess.SetWindowSize(4096, 4096)
	sess.SetReadBuffer(4 * 1024 * 1024)
	sess.SetWriteBuffer(4 * 1024 * 1024)
	sess.SetStreamMode(true)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)
	sess.SetDeadline(time.Now().Add(time.Minute))
	return sess, err
}

func dialTinyBufferEcho() (*UDPSession, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	sess, err := DialWithOptions(portTinyBufferEcho, block, 10, 3)
	if err != nil {
		panic(err)
	}
	return sess, err
}

//////////////////////////
func listenEcho() (net.Listener, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	return ListenWithOptions(portEcho, block, 10, 3)
}
func listenTinyBufferEcho() (net.Listener, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	block, _ := NewSalsa20BlockCrypt(pass)
	return ListenWithOptions(portTinyBufferEcho, block, 10, 3)
}

func listenSink() (net.Listener, error) {
	return ListenWithOptions(portSink, nil, 0, 0)
}

func echoServer() {
	l, err := listenEcho()
	if err != nil {
		panic(err)
	}

	go func() {
		kcplistener := l.(*Listener)
		kcplistener.SetReadBuffer(4 * 1024 * 1024)
		kcplistener.SetWriteBuffer(4 * 1024 * 1024)
		kcplistener.SetDSCP(46)
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}

			// coverage test
			s.(*UDPSession).SetReadBuffer(4 * 1024 * 1024)
			s.(*UDPSession).SetWriteBuffer(4 * 1024 * 1024)
			go handleEcho(s.(*UDPSession))
		}
	}()
}

func sinkServer() {
	l, err := listenSink()
	if err != nil {
		panic(err)
	}

	go func() {
		kcplistener := l.(*Listener)
		kcplistener.SetReadBuffer(4 * 1024 * 1024)
		kcplistener.SetWriteBuffer(4 * 1024 * 1024)
		kcplistener.SetDSCP(46)
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}

			go handleSink(s.(*UDPSession))
		}
	}()
}

func tinyBufferEchoServer() {
	l, err := listenTinyBufferEcho()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}
			go handleTinyBufferEcho(s.(*UDPSession))
		}
	}()
}

///////////////////////////

func handleEcho(conn *UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetDSCP(46)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Hour))
	conn.SetWriteDeadline(time.Now().Add(time.Hour))
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		conn.Write(buf[:n])
	}
}

func handleSink(conn *UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetDSCP(46)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Hour))
	conn.SetWriteDeadline(time.Now().Add(time.Hour))
	buf := make([]byte, 65536)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
	}
}

func handleTinyBufferEcho(conn *UDPSession) {
	conn.SetStreamMode(true)
	buf := make([]byte, 2)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		conn.Write(buf[:n])
	}
}

///////////////////////////

func TestTimeout(t *testing.T) {
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 10)

	//timeout
	cli.SetDeadline(time.Now().Add(time.Second))
	<-time.After(2 * time.Second)
	n, err := cli.Read(buf)
	if n != 0 || err == nil {
		t.Fail()
	}
	cli.Close()
}

func TestSendRecv(t *testing.T) {
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}
	cli.SetWriteDelay(true)
	cli.SetDUP(1)
	const N = 100
	buf := make([]byte, 10)
	for i := 0; i < N; i++ {
		msg := fmt.Sprintf("hello%v", i)
		cli.Write([]byte(msg))
		if n, err := cli.Read(buf); err == nil {
			if string(buf[:n]) != msg {
				t.Fail()
			}
		} else {
			panic(err)
		}
	}
	cli.Close()
}

func TestTinyBufferReceiver(t *testing.T) {
	cli, err := dialTinyBufferEcho()
	if err != nil {
		panic(err)
	}
	const N = 100
	snd := byte(0)
	fillBuffer := func(buf []byte) {
		for i := 0; i < len(buf); i++ {
			buf[i] = snd
			snd++
		}
	}

	rcv := byte(0)
	check := func(buf []byte) bool {
		for i := 0; i < len(buf); i++ {
			if buf[i] != rcv {
				return false
			}
			rcv++
		}
		return true
	}
	sndbuf := make([]byte, 7)
	rcvbuf := make([]byte, 7)
	for i := 0; i < N; i++ {
		fillBuffer(sndbuf)
		cli.Write(sndbuf)
		if n, err := io.ReadFull(cli, rcvbuf); err == nil {
			if !check(rcvbuf[:n]) {
				t.Fail()
			}
		} else {
			panic(err)
		}
	}
	cli.Close()
}

func TestClose(t *testing.T) {
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 10)

	cli.Close()
	if cli.Close() == nil {
		t.Fail()
	}
	n, err := cli.Write(buf)
	if n != 0 || err == nil {
		t.Fail()
	}
	n, err = cli.Read(buf)
	if n != 0 || err == nil {
		t.Fail()
	}
	cli.Close()
}

func TestParallel1024CLIENT_64BMSG_64CNT(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1024)
	for i := 0; i < 1024; i++ {
		go parallel_client(&wg)
	}
	wg.Wait()
}

func parallel_client(wg *sync.WaitGroup) (err error) {
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}

	err = echo_tester(cli, 64, 64)
	wg.Done()
	return
}

func BenchmarkEchoSpeed4K(b *testing.B) {
	speedclient(b, 4096)
}

func BenchmarkEchoSpeed64K(b *testing.B) {
	speedclient(b, 65536)
}

func BenchmarkEchoSpeed512K(b *testing.B) {
	speedclient(b, 524288)
}

func BenchmarkEchoSpeed1M(b *testing.B) {
	speedclient(b, 1048576)
}

func speedclient(b *testing.B, nbytes int) {
	b.ReportAllocs()
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}

	if err := echo_tester(cli, nbytes, b.N); err != nil {
		b.Fail()
	}
	b.SetBytes(int64(nbytes))
}

func BenchmarkSinkSpeed4K(b *testing.B) {
	sinkclient(b, 4096)
}

func BenchmarkSinkSpeed64K(b *testing.B) {
	sinkclient(b, 65536)
}

func BenchmarkSinkSpeed256K(b *testing.B) {
	sinkclient(b, 524288)
}

func BenchmarkSinkSpeed1M(b *testing.B) {
	sinkclient(b, 1048576)
}

func sinkclient(b *testing.B, nbytes int) {
	b.ReportAllocs()
	cli, err := dialSink()
	if err != nil {
		panic(err)
	}

	sink_tester(cli, nbytes, b.N)
	b.SetBytes(int64(nbytes))
}

func echo_tester(cli net.Conn, msglen, msgcount int) error {
	buf := make([]byte, msglen)
	for i := 0; i < msgcount; i++ {
		// send packet
		if _, err := cli.Write(buf); err != nil {
			return err
		}

		// receive packet
		nrecv := 0
		for {
			n, err := cli.Read(buf)
			if err != nil {
				return err
			} else {
				nrecv += n
				if nrecv == msglen {
					break
				}
			}
		}
	}
	return nil
}

func sink_tester(cli *UDPSession, msglen, msgcount int) error {
	// sender
	buf := make([]byte, msglen)
	for i := 0; i < msgcount; i++ {
		if _, err := cli.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

func TestSNMP(t *testing.T) {
	t.Log(DefaultSnmp.Copy())
	t.Log(DefaultSnmp.Header())
	t.Log(DefaultSnmp.ToSlice())
	DefaultSnmp.Reset()
	t.Log(DefaultSnmp.ToSlice())
}

func TestListenerClose(t *testing.T) {
	l, err := ListenWithOptions(portListerner, nil, 10, 3)
	if err != nil {
		t.Fail()
	}
	l.SetReadDeadline(time.Now().Add(time.Second))
	l.SetWriteDeadline(time.Now().Add(time.Second))
	l.SetDeadline(time.Now().Add(time.Second))
	time.Sleep(2 * time.Second)
	if _, err := l.Accept(); err == nil {
		t.Fail()
	}

	l.Close()
	if l.closeSession(sessionKey{
		addr:   "127.0.0.1:1111",
		convID: 1234,
	}) {
		t.Fail()
	}
}
