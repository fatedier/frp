package limit

import (
	"io"
	"net"
)

const (
	B uint64 = 1 << (10 * (iota))
	KB
	MB
	GB
	TB
	PB
	EB
)

const burstLimit = 1024 * 1024 * 1024

type LimitConn struct {
	net.Conn

	lr io.Reader
	lw io.Writer
}

func NewLimitConn(maxread, maxwrite uint64, c net.Conn) LimitConn {
	// 这里不知道为什么要 49 才能对的上真实速度
	// 49 是根据 wget 速度来取的，测试了 512、1024、2048、4096、8192 等多种速度下都很准确
	return LimitConn{
		lr:   NewReaderWithLimit(c, maxread*49),
		lw:   NewWriterWithLimit(c, maxwrite*49),
		Conn: c,
	}
}

func (c LimitConn) Read(p []byte) (n int, err error) {
	return c.lr.Read(p)
}

func (c LimitConn) Write(p []byte) (n int, err error) {
	return c.lw.Write(p)
}
