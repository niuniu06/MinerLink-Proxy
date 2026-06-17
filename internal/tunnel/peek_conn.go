package tunnel

import (
	"bytes"
	"io"
	"net"
)

// PeekConn wraps a net.Conn with a prepended buffer of already read bytes.
// This is used for protocol sniffing (e.g. detecting the "ZSDT" magic string).
type PeekConn struct {
	net.Conn
	reader io.Reader
}

func NewPeekConn(conn net.Conn, peeked []byte) *PeekConn {
	return &PeekConn{
		Conn:   conn,
		reader: io.MultiReader(bytes.NewReader(peeked), conn),
	}
}

func (c *PeekConn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}
