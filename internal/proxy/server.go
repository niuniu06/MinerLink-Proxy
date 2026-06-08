package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"proxy-core/internal/models"
	"proxy-core/internal/tunnel"
)

type Server struct {
	Config   *models.ProxyConfig
	Listener net.Listener
	Sessions sync.Map // map[string]*Session
	Quit     chan struct{}
}

func NewServer(cfg *models.ProxyConfig) *Server {
	return &Server{
		Config: cfg,
		Quit:   make(chan struct{}),
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.Config.ListenPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.Listener = l
	log.Printf("Proxy server started on port %d for %s", s.Config.ListenPort, s.Config.CoinName)

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	// Generate TLS Config for Tunnel
	tlsConfig, err := tunnel.GenerateTLSConfig()
	if err != nil {
		log.Printf("Failed to generate TLS config for tunnel on port %d: %v", s.Config.ListenPort, err)
	}

	for {
		select {
		case <-s.Quit:
			return
		default:
		}

		conn, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-s.Quit:
				return
			default:
				log.Printf("Accept error on port %d: %v", s.Config.ListenPort, err)
				continue
			}
		}

		go s.handleNewConnection(conn, tlsConfig)
	}
}

func (s *Server) handleNewConnection(conn net.Conn, tlsConfig *tls.Config) {
	// Sniff the first 4 bytes to detect protocol
	buf := make([]byte, 4)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := io.ReadFull(conn, buf)
	conn.SetReadDeadline(time.Time{})

	if err != nil {
		if n > 0 {
			peekConn := tunnel.NewPeekConn(conn, buf[:n])
			s.startSession(peekConn, false)
		} else {
			conn.Close()
		}
		return
	}

	if string(buf) == "ZSDT" && tlsConfig != nil {
		log.Printf("Incoming ZSDT Tunnel connection from %v on port %d", conn.RemoteAddr(), s.Config.ListenPort)
		tlsConn := tls.Server(conn, tlsConfig)
		
		// Perform handshake early to catch errors
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("Tunnel TLS handshake failed: %v", err)
			conn.Close()
			return
		}

		yamuxSession, err := yamux.Server(tlsConn, yamux.DefaultConfig())
		if err != nil {
			log.Printf("Tunnel Yamux server failed: %v", err)
			conn.Close()
			return
		}

		// Process streams from this tunnel
		go func() {
			defer conn.Close()
			for {
				stream, err := yamuxSession.AcceptStream()
				if err != nil {
					log.Printf("Tunnel session closed for %v: %v", conn.RemoteAddr(), err)
					return
				}
				
				// Wrap stream with Snappy compression before starting session
				snappyConn := tunnel.NewSnappyConn(stream)
				go s.startSession(snappyConn, true)
			}
		}()
	} else {
		// Normal Stratum Miner
		peekConn := tunnel.NewPeekConn(conn, buf)
		s.startSession(peekConn, false)
	}
}

func (s *Server) startSession(conn net.Conn, isEncrypted bool) {
	session := NewSession(conn, s.Config, isEncrypted)
	s.Sessions.Store(session.ID, session)

	session.Start()
	s.Sessions.Delete(session.ID)
}

func (s *Server) Stop() {
	close(s.Quit)
	if s.Listener != nil {
		s.Listener.Close()
	}
	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.Close()
		return true
	})
	log.Printf("Proxy server on port %d stopped", s.Config.ListenPort)
}

func (s *Server) GetStats() map[string]interface{} {
	activeMiners := 0
	totalShares := int64(0)
	totalFeeShares := int64(0)
	activeDevFees := 0
	activeOpFees := 0

	miners := make([]map[string]interface{}, 0)
	now := time.Now()

	s.Sessions.Range(func(key, value interface{}) bool {
		activeMiners++
		sess := value.(*Session)

		totalShares += sess.Stats.Shares
		totalFeeShares += sess.Stats.FeeShares

		if sess.CurrentFeeMode == FeeModeDev {
			activeDevFees++
		} else if sess.CurrentFeeMode == FeeModeOperator {
			activeOpFees++
		}

		uptimeSecs := int64(now.Sub(sess.Stats.ConnectedAt).Seconds())

		miners = append(miners, map[string]interface{}{
			"id":            sess.ID,
			"wallet":        sess.MinerWallet,
			"worker":        sess.MinerWorker,
			"shares":        sess.Stats.Shares,
			"feeShares":     sess.Stats.FeeShares,
			"validShares":   sess.Stats.ValidShares,
			"invalidShares": sess.Stats.InvalidShares,
			"currentDiff":   sess.CurrentDiff,
			"hashrate":      sess.FormatHashrate(),
			"uptime":        uptimeSecs,
			"isEncrypted":   sess.IsEncrypted,
		})
		return true
	})

	return map[string]interface{}{
		"coinName":           s.Config.CoinName,
		"listenPort":         s.Config.ListenPort,
		"activeMiners":       activeMiners,
		"totalShares":        totalShares,
		"totalFeeShares":     totalFeeShares,
		"devFeePercent":      s.Config.DevFeePercent,
		"operatorFeePercent": s.Config.OperatorFeePercent,
		"activeDevFees":      activeDevFees,
		"activeOpFees":       activeOpFees,
		"miners":             miners,
		// Exposed flags for UI:
		"enableSmoothFee": s.Config.EnableSmoothFee,
		"enableAsic":      s.Config.EnableAsic,
		"enableAntiBan":   s.Config.EnableAntiBan,
	}
}
