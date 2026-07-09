package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"proxy-core/internal/models"
	"proxy-core/internal/tunnel"
)

type MinerStatsData struct {
	ID            string  `json:"id"`
	IsOffline     bool    `json:"isOffline"`
	Wallet        string  `json:"wallet"`
	Worker        string  `json:"worker"`
	ClientAgent   string  `json:"clientAgent"`
	Shares        int64   `json:"shares"`
	FeeShares     int64   `json:"feeShares"`
	ValidShares   int64   `json:"validShares"`
	InvalidShares int64   `json:"invalidShares"`
	CurrentDiff   float64 `json:"currentDiff"`
	Hashrate      string  `json:"hashrate"`
	Uptime        int64   `json:"uptime"`
	IsEncrypted   bool    `json:"isEncrypted"`
}

type GlobalMinerStats struct {
	MinerStatsData
	Port     int    `json:"port"`
	CoinName string `json:"coinName"`
}

type Server struct {
	Config                  *models.ProxyConfig
	Listener                net.Listener
	Sessions                sync.Map // map[string]*Session
	MinerLoggers            sync.Map // map[string]*MinerLogger
	ClientAgentCache        sync.Map // map[string]string (IP -> Agent)
	Quit                    chan struct{}
	ActiveConnections       int32
	FeeScheduler            *FeeScheduler
	RingBuffer              *HashrateRingBuffer
}

func NewServer(cfg *models.ProxyConfig) *Server {
	s := &Server{
		Config:     cfg,
		Quit:       make(chan struct{}),
		RingBuffer: &HashrateRingBuffer{},
	}
	s.FeeScheduler = NewFeeScheduler(s)
	go s.ReapOfflineSessions()
	go s.runRingBufferTicker()
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
	
	var l net.Listener
	var err error
	
	// Retry loop for Windows TIME_WAIT during Hot Reload
	for i := 0; i < 15; i++ {
		l, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "address already in use") && !strings.Contains(err.Error(), "bind") {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	
	if err != nil {
		return err
	}

	s.Listener = l
	log.Printf("[INFO] [%s][stratum+tcp] 添加一个代理矿池，端口 :%d", strings.ToUpper(s.Config.CoinName), s.Config.ListenPort)

	go s.acceptLoop()
	go s.logPruneLoop()
	go s.FeeScheduler.Start()
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
				// removed accept error log
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
		// removed incoming tunnel log, s.Config.ListenPort)
		tlsConn := tls.Server(conn, tlsConfig)
		
		// Perform handshake early to catch errors
		if err := tlsConn.Handshake(); err != nil {
			// removed TLS error log
			conn.Close()
			return
		}
		
		if s.Config.EnableTcpNoDelay {
			ApplyTcpNoDelay(conn)
		}
		yamuxSession, err := yamux.Server(tlsConn, getTunnelConfig())
		if err != nil {
			// removed Yamux error log
			conn.Close()
			return
		}

		// Process streams from this tunnel
		go func() {
			defer conn.Close()
			for {
				stream, err := yamuxSession.AcceptStream()
				if err != nil {
					// removed session closed log, err)
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
	currentActive := atomic.LoadInt32(&s.ActiveConnections)
	if currentActive >= 50000 {
		log.Printf("Port %d rejected connection: reached 50000 max connection limit", s.Config.ListenPort)
		conn.Close()
		return
	}

	atomic.AddInt32(&s.ActiveConnections, 1)
	defer atomic.AddInt32(&s.ActiveConnections, -1)

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
		sess.mu.Unlock()
		
		if isOffline && w == worker {
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
	if s.FeeScheduler != nil {
		s.FeeScheduler.Stop()
	}
	if s.Listener != nil {
		s.Listener.Close()
	}
	log.Printf("[INFO] [%s][stratum+tcp] 删除一个代理矿池，端口 :%d", strings.ToUpper(s.Config.CoinName), s.Config.ListenPort)
	log.Printf("[INFO] [TCP] 关闭代理, 端口 :%d, 币种: %s", s.Config.ListenPort, strings.ToUpper(s.Config.CoinName))
	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.Close()
		return true
	})
}

func (s *Server) GetStats() map[string]interface{} {
	totalShares := int64(0)
	totalFeeShares := int64(0)
	totalHashrateMHs := float64(0)
	activeMiners := 0
	activeDevFees := 0
	activeOpFees := 0

	s.Sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		sess.mu.Lock()
		isOffline := sess.IsOffline
		lastShareTime := sess.LastShareTime
		isProbe := sess.IsProbe
		sess.mu.Unlock()
		
		if isProbe {
			return true // Skip probe connections
		}
		
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
		totalHashrateMHs += sess.GetHashrateMHs()

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
		"totalHashrateMHs":   totalHashrateMHs,
		"devFeePercent":      s.Config.DevFeePercent,
		"operatorFeePercent": s.Config.OperatorFeePercent,
		"activeDevFees":      activeDevFees,
		"activeOpFees":       activeOpFees,
		// Exposed flags for UI:
		"enableAsic":      s.Config.EnableAsic,
		"enableAntiBan":   s.Config.EnableAntiBan,
	}
}

func (s *Server) GetPaginatedMiners() (int, []MinerStatsData) {
	minerMap := make(map[string]*MinerStatsData)
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
		isProbe := sess.IsProbe
		sess.mu.Unlock()

		if isProbe {
			return true // Skip probe connections
		}

		uptimeSecs := int64(now.Sub(connectedAt).Seconds())
		
		if !isOffline && time.Since(lastShareTime) > 3*time.Minute {
			isOffline = true
			if !lastShareTime.IsZero() {
				uptimeSecs = int64(lastShareTime.Sub(connectedAt).Seconds())
			} else {
				uptimeSecs = 0
			}
		}

		if shares == 0 {
			return true // Completely hide any connection (online or offline) until it submits at least 1 share. This makes scanners invisible.
		}

		if isOffline && !sess.OfflineAt.IsZero() {
			uptimeSecs = int64(sess.OfflineAt.Sub(connectedAt).Seconds())
		}
		
		rawHashrate := 0.0
		if !isOffline {
			rawHashrate = sess.GetHashrateMHs()
		}

		mapID := wallet + "." + worker

		existing, ok := minerMap[mapID]
		if !ok {
			minerMap[mapID] = &MinerStatsData{
				ID:            sess.ID,
				IsOffline:     isOffline,
				Wallet:        wallet,
				Worker:        worker,
				ClientAgent:   sess.ClientAgent,
				Shares:        shares,
				FeeShares:     feeShares,
				ValidShares:   validShares,
				InvalidShares: invalidShares,
				CurrentDiff:   currentDiff,
				Hashrate:      fmt.Sprintf("%f", rawHashrate), // Temporary raw storage
				Uptime:        uptimeSecs,
				IsEncrypted:   sess.IsEncrypted,
			}
		} else {
			// Aggregate duplicate workers
			existing.Shares += shares
			existing.FeeShares += feeShares
			existing.ValidShares += validShares
			existing.InvalidShares += invalidShares
			
			// If at least one session is online, the aggregated worker is online
			if !isOffline {
				existing.IsOffline = false
				existing.ID = sess.ID // Use ID of online session
			}
			
			// Use the maximum uptime
			if uptimeSecs > existing.Uptime {
				existing.Uptime = uptimeSecs
			}
			
			// Sum the raw hashrate
			oldRaw, _ := strconv.ParseFloat(existing.Hashrate, 64)
			existing.Hashrate = fmt.Sprintf("%f", oldRaw + rawHashrate)
			
			// Use the highest difficulty
			if currentDiff > existing.CurrentDiff {
				existing.CurrentDiff = currentDiff
			}
		}

		return true
	})

	miners := make([]MinerStatsData, 0, len(minerMap))
	for _, data := range minerMap {
		// Format the aggregated hashrate
		rawMHs, _ := strconv.ParseFloat(data.Hashrate, 64)
		if data.IsOffline {
			data.Hashrate = "0.00 TH/s"
		} else {
			data.Hashrate = FormatHashrateMHs(rawMHs, s.Config.HashrateUnit)
		}
		miners = append(miners, *data)
	}

	sort.Slice(miners, func(i, j int) bool {
		if miners[i].IsOffline != miners[j].IsOffline {
			return !miners[i].IsOffline // Online first
		}
		return miners[i].Worker < miners[j].Worker
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

func getTunnelConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 30 * time.Second
	cfg.ConnectionWriteTimeout = 5 * time.Minute // Prevent aggressive dropping on slow connections
	cfg.MaxStreamWindowSize = 1024 * 1024        // 1MB window instead of 256KB
	return cfg
}
func (s *Server) runRingBufferTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	var ticks int
	for {
		select {
		case <-ticker.C:
			ticks++
			s.RingBuffer.Tick()

			var totalMultiplier float64
			var algoCount float64

			s.Sessions.Range(func(key, value interface{}) bool {
				sess := value.(*Session)
				sess.RingBuffer.Tick()
				if totalMultiplier == 0 {
					totalMultiplier = sess.getAlgoBaseMHs()
				}
				algoCount++
				return true
			})

			// Flush port history every 5 minutes
			if ticks%5 == 0 {
				if totalMultiplier == 0 {
					// Fallback if no sessions
					totalMultiplier = 1.0 
				}
				s.RingBuffer.FlushToDB(s.Config.ListenPort, s.Config.CoinName, 5, totalMultiplier)
			}

		case <-s.Quit:
			return
		}
	}
}
