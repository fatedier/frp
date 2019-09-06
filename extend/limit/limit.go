package limit

import (
	"io"

	frpNet "github.com/fatedier/frp/utils/net"
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
	frpNet.Conn

	lr io.Reader
	lw io.Writer
}

func NewLimitConn(maxread, maxwrite uint64, c frpNet.Conn) LimitConn {
	return LimitConn{
		lr:   NewReaderWithLimit(c, maxread*KB),
		lw:   NewWriterWithLimit(c, maxwrite*KB),
		Conn: c,
	}
}

func (c LimitConn) Read(p []byte) (n int, err error) {
	return c.lr.Read(p)
}

func (c LimitConn) Write(p []byte) (n int, err error) {
	return c.lw.Write(p)
}
