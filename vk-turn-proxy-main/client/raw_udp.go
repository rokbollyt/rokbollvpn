package vkturn

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/bschaatsbergen/dnsdialer"
	"github.com/pion/logging"
	"github.com/pion/turn/v5"
)

func init() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var d net.Dialer
			dnsServers := []string{"77.88.8.8:53", "77.88.8.1:53", "8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53"}
			var lastErr error
			for _, dns := range dnsServers {
				conn, err := d.DialContext(ctx, "udp", dns)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}
}

// RunRawUDPRelay establishes a transparent, bi-directional UDP relay between
// a local UDP address (e.g. 127.0.0.1:1590) and any target remote UDP peer
// (WireGuard, AmneziaWG, Hysteria 2, OpenVPN UDP) routed directly through the
// MAX (or VK) TURN infrastructure without DTLS/KCP/smux and without a custom VPS exit.
func RunRawUDPRelay(ctx context.Context, cfg Config) error {
	if cfg.Peer == "" {
		return fmt.Errorf("peer address is required")
	}
	peerAddr, err := net.ResolveUDPAddr("udp", cfg.Peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", cfg.Peer, err)
	}
	if cfg.Listen == "" {
		cfg.Listen = fmt.Sprintf("127.0.0.1:%d", peerAddr.Port)
	}

	listenConn, err := net.ListenPacket("udp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen UDP %q: %w", cfg.Listen, err)
	}
	defer listenConn.Close()
	context.AfterFunc(ctx, func() { _ = listenConn.Close() })

	log.Printf("[RAW-UDP-RELAY] Local listener active on %s -> Remote Peer %s (via MAX-TURN)", cfg.Listen, cfg.Peer)

	var getCreds getCredsFunc
	if cfg.MaxLink != "" || cfg.MaxCallToken != "" || cfg.MaxLoginToken != "" {
		pool := newMaxCredPool(MaxConfig{
			LoginToken: cfg.MaxLoginToken,
			CallToken:  cfg.MaxCallToken,
			DeviceID:   cfg.MaxDeviceID,
			Link:       cfg.MaxLink,
		})
		getCreds = pool.getCreds
	} else if cfg.VKLink != "" {
		dialer := dnsdialer.New(
			dnsdialer.WithResolvers("77.88.8.8:53", "77.88.8.1:53", "8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53"),
			dnsdialer.WithStrategy(dnsdialer.Fallback{}),
			dnsdialer.WithCache(100, 10*time.Hour, 10*time.Hour),
		)
		getCreds = func(ctx context.Context, link string, streamID int) (string, string, string, error) {
			return getVkCredsCached(ctx, link, streamID, dialer)
		}
	} else {
		return fmt.Errorf("either MAX link/tokens or VK link is required")
	}

	targetLink := cfg.MaxLink
	if targetLink == "" {
		targetLink = cfg.VKLink
	}

	tp := &turnParams{
		host:     cfg.TurnHost,
		port:     cfg.TurnPort,
		link:     targetLink,
		udp:      cfg.UDP,
		getCreds: getCreds,
		cfg:      &cfg,
	}

	var activeClientAddr atomic.Value // holds net.Addr

	inboundChan := make(chan []byte, 4096)
	go func() {
		buf := make([]byte, 65535)
		for {
			n, clientAddr, err := listenConn.ReadFrom(buf)
			if err != nil {
				return
			}
			activeClientAddr.Store(clientAddr)
			pkt := make([]byte, n)
			copy(pkt, buf[:n])
			select {
			case inboundChan <- pkt:
			default:
				// Drop packet if queue is saturated
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := runOneRawUDPSession(ctx, tp, peerAddr, listenConn, inboundChan, &activeClientAddr)
		if err != nil && ctx.Err() == nil {
			log.Printf("[RAW-UDP-RELAY] Relay session ended (%v), reconnecting in 2s...", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func runOneRawUDPSession(ctx context.Context, tp *turnParams, peer *net.UDPAddr, listenConn net.PacketConn, inboundChan <-chan []byte, activeClientAddr *atomic.Value) error {
	user, pass, rawURL, err := tp.getCreds(ctx, tp.link, 0)
	if err != nil {
		return fmt.Errorf("get TURN credentials: %w", err)
	}

	urlhost, urlport, err := net.SplitHostPort(rawURL)
	if err != nil {
		return fmt.Errorf("parse TURN addr: %w", err)
	}
	if tp.host != "" {
		urlhost = tp.host
	}
	if tp.port != "" {
		urlport = tp.port
	}
	turnServerAddr := net.JoinHostPort(urlhost, urlport)
	turnServerUDPAddr, err := net.ResolveUDPAddr("udp", turnServerAddr)
	if err != nil {
		return fmt.Errorf("resolve TURN addr: %w", err)
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()

	var turnConn net.PacketConn
	if tp.udp {
		c, err := net.DialUDP("udp", nil, turnServerUDPAddr)
		if err != nil {
			return fmt.Errorf("dial TURN (UDP): %w", err)
		}
		defer c.Close()
		turnConn = &connectedUDPConn{c}
	} else {
		var d net.Dialer
		c, err := d.DialContext(dialCtx, "tcp", turnServerAddr)
		if err != nil {
			return fmt.Errorf("dial TURN (TCP): %w", err)
		}
		defer c.Close()
		turnConn = turn.NewSTUNConn(c)
	}

	var addrFamily turn.RequestedAddressFamily = turn.RequestedAddressFamilyIPv4
	if peer.IP.To4() == nil {
		addrFamily = turn.RequestedAddressFamilyIPv6
	}

	turnClient, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr:         turnServerAddr,
		TURNServerAddr:         turnServerAddr,
		Conn:                   turnConn,
		Net:                    newDirectNet(),
		Username:               user,
		Password:               pass,
		RequestedAddressFamily: addrFamily,
		LoggerFactory:          logging.NewDefaultLoggerFactory(),
	})
	if err != nil {
		return fmt.Errorf("create TURN client: %w", err)
	}
	defer turnClient.Close()

	if err := turnClient.Listen(); err != nil {
		return fmt.Errorf("TURN listen: %w", err)
	}

	relayConn, err := turnClient.Allocate()
	if err != nil {
		return fmt.Errorf("TURN allocate: %w", err)
	}
	defer relayConn.Close()

	log.Printf("[RAW-UDP-RELAY] Allocation OK (relayed %s) <-> Target Peer %s", relayConn.LocalAddr(), peer)

	// Establish TURN permissions for the target peer cleanly without corrupting the datagram stream
	if err := turnClient.CreatePermission(peer); err != nil {
		log.Printf("[RAW-UDP-RELAY] Warning: CreatePermission(%s): %v", peer, err)
	} else {
		log.Printf("[RAW-UDP-RELAY] CreatePermission OK for peer %s", peer)
	}

	sessCtx, sessCancel := context.WithCancel(ctx)
	defer sessCancel()

	errChan := make(chan error, 2)

	// TX: forward local client packets to remote peer through TURN
	go func() {
		for {
			select {
			case <-sessCtx.Done():
				return
			case pkt := <-inboundChan:
				_, err := relayConn.WriteTo(pkt, peer)
				if err != nil {
					errChan <- fmt.Errorf("relay WriteTo: %w", err)
					return
				}
			}
		}
	}()

	// RX: forward response packets from remote peer back to local client
	go func() {
		rxBuf := make([]byte, 65535)
		for {
			select {
			case <-sessCtx.Done():
				return
			default:
			}
			n, _, err := relayConn.ReadFrom(rxBuf)
			if err != nil {
				errChan <- fmt.Errorf("relay ReadFrom: %w", err)
				return
			}
			if n > 0 {
				client := activeClientAddr.Load()
				if client != nil {
					_, _ = listenConn.WriteTo(rxBuf[:n], client.(net.Addr))
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}

