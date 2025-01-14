package service

import (
	"io"
	"net"
)

// TODO: Move to a benching directory
type CountingIO struct {
	io.ReadWriteCloser

	sent uint64
	recv uint64
	conn net.Conn
}

func NewCountingIO(conn net.Conn) *CountingIO {
	var c CountingIO
	c.conn = conn
	return &c
}

func (c *CountingIO) Read(p []byte) (int, error) {
	n, err := c.conn.Read(p)
	c.recv += uint64(n)
	return n, err
}

func (c *CountingIO) Write(p []byte) (int, error) {
	n, err := c.conn.Write(p)
	c.sent += uint64(n)
	return n, err
}

func (c *CountingIO) Close() error {
	return c.conn.Close()
}

func (c *CountingIO) GetCounts() (uint64, uint64) {
	return c.sent, c.recv
}

func (c *CountingIO) Reset() {
	c.sent = 0
	c.recv = 0
}
