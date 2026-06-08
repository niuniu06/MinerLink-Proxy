package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
	"proxy-core/internal/tunnel"
)

func main() {
	localAddr := flag.String("local", ":3333", "Local address to listen on")
	remoteAddr := flag.String("remote", "", "Remote server address (e.g. 8.8.8.8:10130)")
	flag.Parse()

	if *remoteAddr == "" {
		log.Fatalf("Please specify the remote address using -remote")
	}

	listener, err := net.Listen("tcp", *localAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *localAddr, err)
	}
	defer listener.Close()

	log.Printf("Local Tunnel Client started on %s", *localAddr)
	log.Printf("Forwarding securely to %s", *remoteAddr)

	var session *yamux.Session
	var mu sync.Mutex

	// Background goroutine to maintain the tunnel session
	go func() {
		for {
			mu.Lock()
			if session == nil || session.IsClosed() {
				log.Printf("Connecting to remote tunnel at %s...", *remoteAddr)
				conn, err := net.DialTimeout("tcp", *remoteAddr, 10*time.Second)
				if err != nil {
					log.Printf("Dial failed: %v", err)
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
					log.Printf("TLS handshake failed: %v", err)
					conn.Close()
					mu.Unlock()
					time.Sleep(3 * time.Second)
					continue
				}

				// Start Yamux Client
				ySession, err := yamux.Client(tlsConn, yamux.DefaultConfig())
				if err != nil {
					log.Printf("Yamux client setup failed: %v", err)
					conn.Close()
					mu.Unlock()
					time.Sleep(3 * time.Second)
					continue
				}

				session = ySession
				log.Printf("Tunnel connection established successfully.")
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}
	}()

	// Accept local miners
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("Accept error: %v", err)
				continue
			}

			go handleMiner(localConn, &session, &mu)
		}
	}()

	// Wait for exit signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
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
