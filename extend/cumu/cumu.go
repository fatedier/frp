package cumu

import (
	"net"
	"sync"
)

// Conn 速度累计
type Conn struct {
	net.Conn

	inCount  int64
	outCount int64

	inCountLock  sync.Mutex
	outCountLock sync.Mutex
}

// NewCumuConn ...
func NewCumuConn(conn net.Conn) *Conn {
	return &Conn{
		Conn:     conn,
		inCount:  0,
		outCount: 0,
	}
}

func (c *Conn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	if err != nil {
		return
	}
	c.outCountLock.Lock()
	defer c.outCountLock.Unlock()
	c.outCount += int64(n)
	return
}

func (c *Conn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	if err != nil {
		return
	}
	c.inCountLock.Lock()
	defer c.inCountLock.Unlock()
	c.inCount += int64(n)
	return
}

// InCount get in bound byte count
func (c *Conn) InCount() int64 {
	c.inCountLock.Lock()
	defer c.inCountLock.Unlock()
	in := c.inCount
	c.inCount = 0
	return in
}

// OutCount get out bound byte count
func (c *Conn) OutCount() int64 {
	c.outCountLock.Lock()
	defer c.outCountLock.Unlock()
	out := c.outCount
	c.outCount = 0
	return out
}
