package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
	"proxy-core/internal/tunnel"
)


type Mapping struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
}

type EmbedConfig struct {
	Mappings []Mapping `json:"mappings"`
}

func parseEmbedded() *EmbedConfig {
	exePath, err := os.Executable()
	if err != nil { return nil }
	f, err := os.Open(exePath)
	if err != nil { return nil }
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.Size() < 16 { return nil }

	_, _ = f.Seek(-16, io.SeekEnd)
	trailer := make([]byte, 16)
	_, _ = f.Read(trailer)

	if string(trailer[8:]) != "ZSDT_CFG" { return nil }

	length, err := strconv.Atoi(string(trailer[:8]))
	if err != nil || length <= 0 || int64(length) > stat.Size()-16 { return nil }

	_, _ = f.Seek(-16-int64(length), io.SeekEnd)
	jsonBytes := make([]byte, length)
	_, _ = f.Read(jsonBytes)

	var cfg EmbedConfig
	if err := json.Unmarshal(jsonBytes, &cfg); err != nil { return nil }
	return &cfg
}

func main() {
	localAddr := flag.String("local", "", "Local address to listen on")
	remoteAddr := flag.String("remote", "", "Remote server address (e.g. 8.8.8.8:10130)")
	flag.Parse()

	var mappings []Mapping

	embedded := parseEmbedded()
	if embedded != nil && len(embedded.Mappings) > 0 {
		mappings = embedded.Mappings
		log.Printf("Loaded customized embedded configuration with %d mappings.", len(mappings))
	} else {
		// Fallback to command line arguments
		lAddr := *localAddr
		rAddr := *remoteAddr
		if lAddr == "" {
			lAddr = ":3333"
		}
		if rAddr == "" {
			log.Fatalf("Please specify the remote address using -remote or use a customized client.")
		}
		mappings = append(mappings, Mapping{Local: lAddr, Remote: rAddr})
	}

	for _, m := range mappings {
		go startTunnelMapping(m.Local, m.Remote)
	}

	// Wait for exit signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
}

func startTunnelMapping(localAddr, remoteAddr string) {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", localAddr, err)
	}
	defer listener.Close()

	log.Printf("Local Tunnel Client started on %s", localAddr)
	log.Printf("Forwarding securely to %s", remoteAddr)

	var session *yamux.Session
	var mu sync.Mutex

	// Background goroutine to maintain the tunnel session
	go func() {
		for {
			mu.Lock()
			if session == nil || session.IsClosed() {
				log.Printf("[%s] Connecting to remote tunnel at %s...", localAddr, remoteAddr)
				conn, err := net.DialTimeout("tcp", remoteAddr, 10*time.Second)
				if err != nil {
					log.Printf("[%s] Dial failed: %v", localAddr, err)
					mu.Unlock()
					time.Sleep(3 * time.Second)
					continue
				}

				// Send the ZSDT magic bytes
				_, err = conn.Write([]byte("ZSDT"))
				if err != nil {
					conn.Close()
					mu.Unlock()
					continue
				}

				// Start TLS client
				tlsConn := tls.Client(conn, &tls.Config{
					InsecureSkipVerify: true, // We trust our own tunnel server unconditionally
				})
				if err := tlsConn.Handshake(); err != nil {
					log.Printf("[%s] TLS handshake failed: %v", localAddr, err)
					conn.Close()
					mu.Unlock()
					time.Sleep(3 * time.Second)
					continue
				}

				// Start Yamux Client
				ySession, err := yamux.Client(tlsConn, getTunnelConfig())
				if err != nil {
					log.Printf("[%s] Yamux client setup failed: %v", localAddr, err)
					conn.Close()
					mu.Unlock()
					time.Sleep(3 * time.Second)
					continue
				}

				session = ySession
				log.Printf("[%s] Tunnel connection established successfully.", localAddr)
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}
	}()

	// Accept local miners
	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("[%s] Accept error: %v", localAddr, err)
			continue
		}

		go handleMiner(localConn, &session, &mu)
	}
}

func handleMiner(localConn net.Conn, sessionPtr **yamux.Session, mu *sync.Mutex) {
	defer localConn.Close()

	mu.Lock()
	session := *sessionPtr
	mu.Unlock()

	if session == nil || session.IsClosed() {
		log.Printf("Dropping connection, tunnel is currently offline.")
		return
	}

	stream, err := session.Open()
	if err != nil {
		log.Printf("Failed to open multiplexed stream: %v", err)
		return
	}
	defer stream.Close()

	// Wrap stream with Snappy compression
	snappyConn := tunnel.NewSnappyConn(stream)

	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(snappyConn, localConn)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(localConn, snappyConn)
		errCh <- err
	}()

	<-errCh
}

func getTunnelConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.EnableKeepAlive = true
	cfg.KeepAliveInterval = 30 * time.Second
	cfg.ConnectionWriteTimeout = 5 * time.Minute
	cfg.MaxStreamWindowSize = 1024 * 1024
	return cfg
}