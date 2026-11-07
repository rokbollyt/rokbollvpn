package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cacggghp/vk-turn-proxy/tcputil"
	"github.com/pion/dtls/v3"
	"github.com/pion/dtls/v3/pkg/crypto/selfsign"
	"github.com/xtaci/smux"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:56000", "listen on ip:port")
	connect := flag.String("connect", "", "connect to ip:port")
	vlessMode := flag.Bool("vless", false, "VLESS mode: forward TCP connections (for VLESS) instead of UDP packets")
	directSocks := flag.Bool("direct-socks", false, "прямой режим: адрес цели приходит в заголовке стрима, сервер подключается сам (microsocks не нужен)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChan
		log.Printf("Terminating...\n")
		cancel()
		<-signalChan
		log.Fatalf("Exit...\n")
	}()

	addr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		panic(err)
	}
	if len(*connect) == 0 && !*directSocks {
		log.Panicf("server address is required")
	}
	// Generate a certificate and private key to secure the connection
	certificate, genErr := selfsign.GenerateSelfSigned()
	if genErr != nil {
		panic(genErr)
	}

	//
	// Everything below is the pion-DTLS API! Thanks for using it ❤️.
	//

	// Connect to a DTLS server
	listener, err := dtls.ListenWithOptions(
		"udp",
		addr,
		dtls.WithCertificates(certificate),
		dtls.WithExtendedMasterSecret(dtls.RequireExtendedMasterSecret),
		dtls.WithCipherSuites(dtls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
		dtls.WithConnectionIDGenerator(dtls.RandomCIDGenerator(8)),
	)
	if err != nil {
		panic(err)
	}
	context.AfterFunc(ctx, func() {
		if err = listener.Close(); err != nil {
			panic(err)
		}
	})

	fmt.Println("Listening")

	wg1 := sync.WaitGroup{}
	for {
		select {
		case <-ctx.Done():
			wg1.Wait()
			return
		default:
		}
		// Wait for a connection.
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		wg1.Add(1)
		go func(conn net.Conn) {
			defer wg1.Done()
			defer func() {
				if closeErr := conn.Close(); closeErr != nil {
					log.Printf("failed to close incoming connection: %s", closeErr)
				}
			}()
			log.Printf("Connection from %s\n", conn.RemoteAddr())

			// Perform the handshake with a 30-second timeout
			ctx1, cancel1 := context.WithTimeout(ctx, 30*time.Second)
			defer cancel1()

			dtlsConn, ok := conn.(*dtls.Conn)
			if !ok {
				log.Println("Type error: expected *dtls.Conn")
				return
			}
			log.Println("Start handshake")
			if err := dtlsConn.HandshakeContext(ctx1); err != nil {
				log.Printf("Handshake failed: %v", err)
				return
			}
			log.Println("Handshake done")

			switch {
			case *directSocks:
				handleDirectSocksConnection(ctx, dtlsConn)
			case *vlessMode:
				handleVLESSConnection(ctx, dtlsConn, *connect)
			default:
				handleUDPConnection(ctx, conn, *connect)
			}

			log.Printf("Connection closed: %s\n", conn.RemoteAddr())
		}(conn)
	}
}

// handleUDPConnection forwards DTLS packets to a UDP backend (WireGuard).
func handleUDPConnection(ctx context.Context, conn net.Conn, connectAddr string) {
	serverConn, err := net.Dial("udp", connectAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err = serverConn.Close(); err != nil {
			log.Printf("failed to close outgoing connection: %s", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	ctx2, cancel2 := context.WithCancel(ctx)
	context.AfterFunc(ctx2, func() {
		if err := conn.SetDeadline(time.Now()); err != nil {
			log.Printf("failed to set incoming deadline: %s", err)
		}
		if err := serverConn.SetDeadline(time.Now()); err != nil {
			log.Printf("failed to set outgoing deadline: %s", err)
		}
	})
	go func() {
		defer wg.Done()
		defer cancel2()
		buf := make([]byte, 1600)
		for {
			select {
			case <-ctx2.Done():
				return
			default:
			}
			if err1 := conn.SetReadDeadline(time.Now().Add(time.Minute * 30)); err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
			n, err1 := conn.Read(buf)
			if err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}

			if err1 = serverConn.SetWriteDeadline(time.Now().Add(time.Minute * 30)); err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
			_, err1 = serverConn.Write(buf[:n])
			if err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		defer cancel2()
		buf := make([]byte, 1600)
		for {
			select {
			case <-ctx2.Done():
				return
			default:
			}
			if err1 := serverConn.SetReadDeadline(time.Now().Add(time.Minute * 30)); err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
			n, err1 := serverConn.Read(buf)
			if err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}

			if err1 = conn.SetWriteDeadline(time.Now().Add(time.Minute * 30)); err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
			_, err1 = conn.Write(buf[:n])
			if err1 != nil {
				log.Printf("Failed: %s", err1)
				return
			}
		}
	}()
	wg.Wait()
}

// handleVLESSConnection creates a KCP+smux session over DTLS and forwards
// each smux stream as a TCP connection to the backend (Xray/VLESS).
func handleVLESSConnection(ctx context.Context, dtlsConn net.Conn, connectAddr string) {
	// 1. Create KCP session over DTLS
	kcpSess, err := tcputil.NewKCPOverDTLS(dtlsConn, true)
	if err != nil {
		log.Printf("KCP session error: %s", err)
		return
	}
	defer func() {
		if err := kcpSess.Close(); err != nil {
			log.Printf("failed to close KCP session: %v", err)
		}
	}()
	log.Printf("KCP session established (server)")

	// 2. Create smux server session over KCP
	smuxSess, err := smux.Server(kcpSess, tcputil.DefaultSmuxConfig())
	if err != nil {
		log.Printf("smux server error: %s", err)
		return
	}
	defer func() {
		if err := smuxSess.Close(); err != nil {
			log.Printf("failed to close smux session: %v", err)
		}
	}()
	log.Printf("smux session established (server)")

	// 3. Accept smux streams and forward to backend via TCP
	var wg sync.WaitGroup
	for {
		stream, err := smuxSess.AcceptStream()
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				log.Printf("smux accept error: %s", err)
			}
			break
		}

		wg.Add(1)
		go func(s *smux.Stream) {
			defer wg.Done()

			defer func() {
				if err := s.Close(); err != nil && err != smux.ErrGoAway {
					log.Printf("failed to close smux stream: %v", err)
				}
			}()

			// Connect to backend (Xray/VLESS)
			backendConn, err := net.DialTimeout("tcp", connectAddr, 10*time.Second)
			if err != nil {
				log.Printf("backend dial error: %s", err)
				return
			}
			defer func() {
				if err := backendConn.Close(); err != nil {
					log.Printf("failed to close backend connection: %v", err)
				}
			}()

			// Bidirectional copy
			pipeConn(ctx, s, backendConn)
		}(stream)
	}
	wg.Wait()
}

// handleDirectSocksConnection обслуживает «прямой» режим: каждый smux-стрим
// начинается с заголовка адреса цели, и сервер сам открывает к ней TCP.
//
// Так из пути пропадает промежуточный SOCKS5-прокси (microsocks), а вместе с
// ним и два round-trip'а сквозь туннель на каждое соединение — при высоком RTT
// это основная часть задержки открытия страниц.
func handleDirectSocksConnection(ctx context.Context, dtlsConn net.Conn) {
	kcpSess, err := tcputil.NewKCPOverDTLS(dtlsConn, true)
	if err != nil {
		log.Printf("KCP session error: %s", err)
		return
	}
	defer func() { _ = kcpSess.Close() }()

	smuxSess, err := smux.Server(kcpSess, tcputil.DefaultSmuxConfig())
	if err != nil {
		log.Printf("smux server error: %s", err)
		return
	}
	defer func() { _ = smuxSess.Close() }()
	log.Printf("direct-socks: сессия установлена")

	var wg sync.WaitGroup
	for {
		stream, err := smuxSess.AcceptStream()
		if err != nil {
			break
		}
		wg.Add(1)
		go func(s *smux.Stream) {
			defer wg.Done()
			defer func() { _ = s.Close() }()

			_ = s.SetReadDeadline(time.Now().Add(20 * time.Second))
			target, err := tcputil.ReadStreamTarget(s)
			if err != nil {
				return
			}
			_ = s.SetReadDeadline(time.Time{})

			backend, err := net.DialTimeout("tcp", target, 12*time.Second)
			if err != nil {
				return
			}
			defer func() { _ = backend.Close() }()

			pipeConn(ctx, s, backend)
		}(stream)
	}
	wg.Wait()
}

// isBenignPipeError: обычное закрытие соединения, а не сбой. Логировать такое
// не нужно — иначе лог забивается на каждом закрытом запросе.
func isBenignPipeError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{
		"use of closed network connection",
		"connection reset by peer",
		"broken pipe",
		"read/write on closed pipe",
		"i/o timeout",
		"timeout",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// pipeConn copies data bidirectionally between two connections.
func pipeConn(ctx context.Context, c1, c2 net.Conn) {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	context.AfterFunc(ctx2, func() {
		_ = c1.SetDeadline(time.Now())
		_ = c2.SetDeadline(time.Now())
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(c1, c2); err != nil && !isBenignPipeError(err) {
			log.Printf("pipeConn: c1<-c2 copy error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := io.Copy(c2, c1); err != nil && !isBenignPipeError(err) {
			log.Printf("pipeConn: c2<-c1 copy error: %v", err)
		}
	}()

	wg.Wait()

	// Reset deadlines
	_ = c1.SetDeadline(time.Time{})
	_ = c2.SetDeadline(time.Time{})
}
