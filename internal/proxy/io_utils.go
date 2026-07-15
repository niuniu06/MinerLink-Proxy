package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"proxy-core/internal/tunnel"
)



func pearlSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if data[0] == '{' || data[0] == '[' {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	if data[0] == '\n' || data[0] == '\r' {
		return 1, data[:1], nil
	}

	return 1, data[:1], nil
}

func parseEthProxyTargetToDiff(targetHex string) float64 {
	targetHex = strings.TrimPrefix(targetHex, "0x")
	tInt, ok := new(big.Int).SetString(targetHex, 16)
	if !ok || tInt.Sign() == 0 {
		return 1.0
	}
	tFloat := new(big.Float).SetInt(tInt)
	maxT := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil))
	hashFloat := new(big.Float).Quo(maxT, tFloat)
	diffFloat := new(big.Float).Quo(hashFloat, big.NewFloat(4294967296.0))
	diff, _ := diffFloat.Float64()
	if diff <= 0 {
		return 1.0
	}
	return diff
}

func extractTCPConn(conn net.Conn) *net.TCPConn {
	for conn != nil {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			return tcpConn
		}

		if peekConn, ok := conn.(*tunnel.PeekConn); ok {
			conn = peekConn.Conn
			continue
		}

		if snappyConn, ok := conn.(*tunnel.SnappyConn); ok {
			conn = snappyConn.Conn
			continue
		}

		if tlsConn, ok := conn.(*tls.Conn); ok {
			conn = tlsConn.NetConn() // Available in Go 1.15+
			continue
		}

		break
	}
	return nil
}

func safeWrite(conn net.Conn, data []byte, timeout time.Duration) (int, error) {
	if conn == nil {
		return 0, fmt.Errorf("nil connection")
	}
	conn.SetWriteDeadline(time.Now().Add(timeout))
	n, err := conn.Write(data)
	conn.SetWriteDeadline(time.Time{})
	return n, err
}

// safeFprintf formats according to a format specifier and writes to the connection with a timeout
func safeFprintf(conn net.Conn, timeout time.Duration, format string, a ...interface{}) (int, error) {
	if conn == nil {
		return 0, fmt.Errorf("nil connection")
	}
	return safeWrite(conn, []byte(fmt.Sprintf(format, a...)), timeout)
}
