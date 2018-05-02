package yamux

import (
	"testing"
)

func BenchmarkPing(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	for i := 0; i < b.N; i++ {
		rtt, err := client.Ping()
		if err != nil {
			b.Fatalf("err: %v", err)
		}
		if rtt == 0 {
			b.Fatalf("bad: %v", rtt)
		}
	}
}

func BenchmarkAccept(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	go func() {
		for i := 0; i < b.N; i++ {
			stream, err := server.AcceptStream()
			if err != nil {
				return
			}
			stream.Close()
		}
	}()

	for i := 0; i < b.N; i++ {
		stream, err := client.Open()
		if err != nil {
			b.Fatalf("err: %v", err)
		}
		stream.Close()
	}
}

func BenchmarkSendRecv(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	sendBuf := make([]byte, 512)
	recvBuf := make([]byte, 512)

	doneCh := make(chan struct{})
	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			return
		}
		defer stream.Close()
		for i := 0; i < b.N; i++ {
			if _, err := stream.Read(recvBuf); err != nil {
				b.Fatalf("err: %v", err)
			}
		}
		close(doneCh)
	}()

	stream, err := client.Open()
	if err != nil {
		b.Fatalf("err: %v", err)
	}
	defer stream.Close()
	for i := 0; i < b.N; i++ {
		if _, err := stream.Write(sendBuf); err != nil {
			b.Fatalf("err: %v", err)
		}
	}
	<-doneCh
}

func BenchmarkSendRecvLarge(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()
	const sendSize = 512 * 1024 * 1024
	const recvSize = 4 * 1024

	sendBuf := make([]byte, sendSize)
	recvBuf := make([]byte, recvSize)

	b.ResetTimer()
	recvDone := make(chan struct{})

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			return
		}
		defer stream.Close()
		for i := 0; i < b.N; i++ {
			for j := 0; j < sendSize/recvSize; j++ {
				if _, err := stream.Read(recvBuf); err != nil {
					b.Fatalf("err: %v", err)
				}
			}
		}
		close(recvDone)
	}()

	stream, err := client.Open()
	if err != nil {
		b.Fatalf("err: %v", err)
	}
	defer stream.Close()
	for i := 0; i < b.N; i++ {
		if _, err := stream.Write(sendBuf); err != nil {
			b.Fatalf("err: %v", err)
		}
	}
	<-recvDone
}
