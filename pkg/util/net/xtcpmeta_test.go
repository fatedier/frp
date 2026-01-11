package net

import (
	"bytes"
	"io"
	"net"
	"testing"
)

type bufRWC struct{ *bytes.Buffer }

func (b *bufRWC) Close() error { return nil }

func TestXTCPMetaRoundTrip(t *testing.T) {
	// Build header into temp buffer
	hdrBuf := &bytes.Buffer{}
	ip := net.ParseIP("203.0.113.10")
	if err := WriteXTCPClientMeta(hdrBuf, &net.TCPAddr{IP: ip, Port: 54321}); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	payload := []byte("HELLO")
	stream := append(append([]byte{}, hdrBuf.Bytes()...), payload...)
	rwc := &bufRWC{Buffer: bytes.NewBuffer(stream)}

	newRWC, host, port, ok, err := ReadOrReplayXTCPClientMeta(rwc)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if !ok {
		t.Fatalf("expected meta ok")
	}
	if host != ip.String() || port != 54321 {
		t.Fatalf("unexpected parsed addr: %s:%d", host, port)
	}
	// Remaining data must be payload only
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(newRWC, got); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: %q != %q", got, payload)
	}
}

func TestXTCPMetaAbsentReplay(t *testing.T) {
	orig := []byte("XYZDATA")
	rwc := &bufRWC{Buffer: bytes.NewBuffer(orig)}
	newRWC, host, port, ok, err := ReadOrReplayXTCPClientMeta(rwc)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if ok || host != "" || port != 0 {
		t.Fatalf("expected no meta, got ok=%v host=%s port=%d", ok, host, port)
	}
	buf := make([]byte, len(orig))
	if _, err := io.ReadFull(newRWC, buf); err != nil {
		t.Fatalf("read replay: %v", err)
	}
	if !bytes.Equal(buf, orig) {
		t.Fatalf("replayed data mismatch: %q != %q", buf, orig)
	}
}