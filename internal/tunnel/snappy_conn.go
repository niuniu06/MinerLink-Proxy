package tunnel

import (
	"net"

	"github.com/golang/snappy"
)

// SnappyConn wraps a net.Conn with Snappy compression for both reading and writing.
type SnappyConn struct {
	net.Conn
	reader *snappy.Reader
	writer *snappy.Writer
}

func NewSnappyConn(conn net.Conn) *SnappyConn {
	return &SnappyConn{
		Conn:   conn,
		reader: snappy.NewReader(conn),
		writer: snappy.NewBufferedWriter(conn),
	}
}

func (s *SnappyConn) Read(b []byte) (n int, err error) {
	return s.reader.Read(b)
}

func (s *SnappyConn) Write(b []byte) (n int, err error) {
	n, err = s.writer.Write(b)
	if err == nil {
		// Must flush immediately for interactive protocols like Stratum
		err = s.writer.Flush()
	}
	return n, err
}
