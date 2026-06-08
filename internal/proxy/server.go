package proxy

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"proxy-core/internal/models"
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

		session := NewSession(conn, s.Config)
		s.Sessions.Store(session.ID, session)

		go func() {
			session.Start()
			s.Sessions.Delete(session.ID)
		}()
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
