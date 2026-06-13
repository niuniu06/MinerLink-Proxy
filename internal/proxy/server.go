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
	Config                  *models.ProxyConfig
	Listener                net.Listener
	Sessions                sync.Map // map[string]*Session
	MinerLoggers            sync.Map // map[string]*MinerLogger
	ClientAgentCache        sync.Map // map[string]string (IP -> Agent)
	Quit                    chan struct{}
}

func NewServer(cfg *models.ProxyConfig) *Server {
	s := &Server{
		Config: cfg,
		Quit:   make(chan struct{}),
	}
	go s.ReapOfflineSessions()
	return s
}

func (s *Server) GetLogger(worker string) *MinerLogger {
	if worker == "" {
		worker = "default"
	}
	val, ok := s.MinerLoggers.Load(worker)
	if ok {
		return val.(*MinerLogger)
	}
	logger := NewMinerLogger(worker)
	s.MinerLoggers.Store(worker, logger)
	return logger
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
	go s.logPruneLoop()
	return nil
}

func (s *Server) logPruneLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.Quit:
			return
		case <-ticker.C:
			s.MinerLoggers.Range(func(key, value interface{}) bool {
				logger := value.(*MinerLogger)
				logger.Prune()
				return true
			})
		}
	}
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
	// If TLS is not configured, we don't need to sniff for ZSDT tunnel protocol.
	// This is critical for miners that wait for a server challenge before sending data (like AlphaMiner).
	if tlsConfig == nil {
		s.startSession(conn, false)
		return
	}

	// Sniff the first 4 bytes to detect protocol
	buf := make([]byte, 4)
	// Use a very short deadline so we don't delay standard miners that wait for server challenge
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, _ := io.ReadFull(conn, buf)
	_ = conn.SetReadDeadline(time.Time{})

	if n < 4 {
		// Not enough bytes for ZSDT, must be a standard connection.
		// If n=0, it means it's a miner waiting for server challenge.
		if n > 0 {
			peekConn := tunnel.NewPeekConn(conn, buf[:n])
			if s.Config.EnableTcpNoDelay {
				ApplyTcpNoDelay(conn)
			}
			s.startSession(peekConn, false)
		} else {
			if s.Config.EnableTcpNoDelay {
				ApplyTcpNoDelay(conn)
			}
			s.startSession(conn, false)
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
		
		if s.Config.EnableTcpNoDelay {
			ApplyTcpNoDelay(conn)
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
				// yamux streams don't support TCP optimizations directly
				go s.startSession(snappyConn, true)
			}
		}()
	} else {
		// Normal Stratum Miner
		if s.Config.EnableTcpNoDelay {
			ApplyTcpNoDelay(conn)
		}
		peekConn := tunnel.NewPeekConn(conn, buf)
		s.startSession(peekConn, false)
	}
}

func (s *Server) startSession(conn net.Conn, isEncrypted bool) {
	session := NewSession(conn, s.Config, isEncrypted)
	session.Server = s // Link server to session so session can write logs
	s.Sessions.Store(session.ID, session)

	session.Start()
	
	// Check if this was just a probe connection (0 shares)
	session.mu.Lock()
	shares := session.Stats.Shares
	session.mu.Unlock()

	if shares == 0 {
		s.DeleteSession(session)
		session.LogError("Miner probe connection dropped with 0 shares, deleted immediately")
		return
	}

	// When session ends, do NOT delete immediately. Mark it as offline.
	session.mu.Lock()
	session.IsOffline = true
	session.OfflineAt = time.Now()
	session.mu.Unlock()
	session.LogError("Miner connection dropped, marked as offline (10 minute retention started)")
}

func (s *Server) DeleteSession(sess *Session) {
	s.Sessions.Delete(sess.ID)
}

func (s *Server) CleanOfflineWorker(worker string, remoteIP string) *Session {
	if worker == "" { return nil }
	var oldSession *Session
	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.mu.Lock()
		isOffline := sess.IsOffline
		w := sess.MinerWorker
		var ip string
		if sess.MinerConn != nil {
			if tcpAddr, ok := sess.MinerConn.RemoteAddr().(*net.TCPAddr); ok {
				ip = tcpAddr.IP.String()
			}
		}
		sess.mu.Unlock()
		
		if isOffline && w == worker && ip == remoteIP {
			oldSession = sess
			s.DeleteSession(sess)
		}
		return true
	})
	return oldSession
}

func (s *Server) ReapOfflineSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.Quit:
			return
		case <-ticker.C:
			now := time.Now()
			s.Sessions.Range(func(key, value interface{}) bool {
				sess := value.(*Session)
				sess.mu.Lock()
				isOffline := sess.IsOffline
				offlineAt := sess.OfflineAt
				sess.mu.Unlock()
				
				if isOffline && now.Sub(offlineAt) > 10*time.Minute {
					s.DeleteSession(sess)
				}
				return true
			})
		}
	}
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
	totalShares := int64(0)
	totalFeeShares := int64(0)
	activeMiners := 0
	activeDevFees := 0
	activeOpFees := 0

	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.mu.Lock()
		isOffline := sess.IsOffline
		lastShareTime := sess.LastShareTime
		sess.mu.Unlock()
		if isOffline || time.Since(lastShareTime) > 3*time.Minute {
			return true // Skip offline miners for active stats
		}
		
		activeMiners++

		sess.mu.Lock()
		shares := sess.Stats.Shares
		feeShares := sess.Stats.FeeShares
		feeMode := sess.CurrentFeeMode
		sess.mu.Unlock()

		totalShares += shares
		totalFeeShares += feeShares

		if feeMode == FeeModeDev {
			activeDevFees++
		} else if feeMode == FeeModeOperator {
			activeOpFees++
		}
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
		// Exposed flags for UI:
		"enableSmoothFee": s.Config.EnableSmoothFee,
		"enableAsic":      s.Config.EnableAsic,
		"enableAntiBan":   s.Config.EnableAntiBan,
	}
}

func (s *Server) GetPaginatedMiners(page, limit int) (int, []map[string]interface{}) {
	miners := make([]map[string]interface{}, 0)
	now := time.Now()

	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.mu.Lock()
		isOffline := sess.IsOffline
		lastShareTime := sess.LastShareTime
		shares := sess.Stats.Shares
		feeShares := sess.Stats.FeeShares
		validShares := sess.Stats.ValidShares
		invalidShares := sess.Stats.InvalidShares
		currentDiff := sess.CurrentDiff
		connectedAt := sess.Stats.ConnectedAt
		wallet := sess.MinerWallet
		worker := sess.MinerWorker
		sess.mu.Unlock()

		uptimeSecs := int64(now.Sub(connectedAt).Seconds())
		
		if !isOffline && time.Since(lastShareTime) > 3*time.Minute {
			isOffline = true
			if !lastShareTime.IsZero() {
				uptimeSecs = int64(lastShareTime.Sub(connectedAt).Seconds())
			} else {
				uptimeSecs = 0
			}
		}

		if isOffline && !sess.OfflineAt.IsZero() {
			uptimeSecs = int64(sess.OfflineAt.Sub(connectedAt).Seconds())
		}

		hashrateStr := sess.FormatHashrate()
		if isOffline {
			hashrateStr = "0.00 TH/s"
		}

		miners = append(miners, map[string]interface{}{
			"id":            sess.ID,
			"isOffline":     isOffline,
			"wallet":        wallet,
			"worker":        worker,
			"clientAgent":   sess.ClientAgent,
			"shares":        shares,
			"feeShares":     feeShares,
			"validShares":   validShares,
			"invalidShares": invalidShares,
			"currentDiff":   currentDiff,
			"hashrate":      hashrateStr,
			"uptime":        uptimeSecs,
			"isEncrypted":   sess.IsEncrypted,
		})
		return true
	})
	return len(miners), miners
}

func (s *Server) GetMinerLogs(worker string) ([]LogEntry, []LogEntry) {
	if worker == "" {
		worker = "default"
	}
	val, ok := s.MinerLoggers.Load(worker)
	if !ok {
		return []LogEntry{}, []LogEntry{}
	}
	logger := val.(*MinerLogger)
	logger.mu.RLock()
	defer logger.mu.RUnlock()
	
	genLogs := make([]LogEntry, len(logger.GeneralLogs))
	copy(genLogs, logger.GeneralLogs)
	errLogs := make([]LogEntry, len(logger.ErrorLogs))
	copy(errLogs, logger.ErrorLogs)
	
	return genLogs, errLogs
}

func ApplyTcpNoDelay(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(3 * time.Minute)
	}
}
