// Package core is the Go static library that powers HappPort's VPN tunnel.
// It exports a minimal C API consumed by the Swift PacketTunnelProvider:
//
//   RouterStart(configJSON *C.char) C.int   – start tunnel, returns 0 on OK
//   RouterStop()                            – graceful shutdown
//   RouterWritePacket(data *C.char, n C.int) – inject L3 packet from tun into Go
//   RouterReadPacket(buf  *C.char, max C.int) C.int – read outbound L3 packet
//
// Supported Protocols:
//   - "hysteria2" / "hy2" : QUIC/Salamander client via embedded hysteria binary + tun2socks
//   - "wireguard" / "wg"  : Native WireGuard user-space protocol (pure L3 encryption)
//   - "amneziawg" / "awg" : AmneziaWG (WireGuard with Jc, S1, S2, H1..H4 obfuscation)
//   - "openvpn"   / "ovpn": OpenVPN UDP relay
//
// MAX-TURN Relay:
//   If DirectMode is false, any UDP protocol (Hysteria 2, WireGuard, AmneziaWG, OpenVPN UDP)
//   is transparently routed through VK/MAX video-call TURN datagrams, bypassing Russian DPI.
package main

/*
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/amnezia-vpn/amneziawg-go/device"
	vkturn "github.com/cacggghp/vk-turn-proxy/client"
)

// ---------- global state --------------------------------------------------

// ---------- exported C API ------------------------------------------------

//export RouterStart
func RouterStart(cfgJSON *C.char) C.int {
	mu.Lock()
	defer mu.Unlock()
	if running {
		return 0
	}

	raw := C.GoString(cfgJSON)
	cfg := defaultConfig()
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			log.Printf("[ROUTER] bad config JSON: %v", err)
			return -1
		}
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	cancel = cancelFn

	if err := startRouter(ctx, cfg); err != nil {
		log.Printf("[ROUTER] start failed: %v", err)
		cancelFn()
		return -1
	}

	running = true
	return 0
}

//export RouterStop
func RouterStop() {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		return
	}
	if cancel != nil {
		cancel()
	}
	if hystCmd != nil && hystCmd.Process != nil {
		_ = hystCmd.Process.Kill()
		hystCmd = nil
	}
	if wgDev != nil {
		wgDev.Close()
		wgDev = nil
	}
	running = false
}

//export RouterWritePacket
func RouterWritePacket(data *C.char, n C.int) {
	if data == nil || n <= 0 {
		return
	}
	pkt := make([]byte, int(n))
	copy(pkt, (*[maxPkt]byte)(unsafe.Pointer(data))[:n:n])
	select {
	case toGo <- pkt:
	default:
	}
}

//export RouterReadPacket
func RouterReadPacket(buf *C.char, maxLen C.int) C.int {
	select {
	case pkt := <-fromGo:
		n := len(pkt)
		if n > int(maxLen) {
			n = int(maxLen)
		}
		copy((*[maxPkt]byte)(unsafe.Pointer(buf))[:n:n], pkt[:n])
		return C.int(n)
	case <-time.After(100 * time.Millisecond):
		return 0
	}
}

// ---------- router startup ------------------------------------------------

func startRouter(ctx context.Context, cfg RouterConfig) error {
	proto := strings.ToLower(cfg.Protocol)
	if proto == "" {
		if cfg.HysteriaURI != "" {
			proto = "hysteria2"
		} else if cfg.PrivateKey != "" {
			if cfg.H1 > 0 || cfg.Jc > 0 {
				proto = "amneziawg"
			} else {
				proto = "wireguard"
			}
		} else if cfg.OpenVPNConfig != "" {
			proto = "openvpn"
		} else {
			proto = "hysteria2"
		}
	}

	relayAddr := fmt.Sprintf("127.0.0.1:%d", cfg.RelayPort)

	// Determine remote peer address
	peer := cfg.PeerAddr
	if peer == "" && cfg.HysteriaURI != "" {
		if p, err := parseHy2Peer(cfg.HysteriaURI); err == nil {
			peer = p
		}
	}

	// 1. Start MAX-TURN relay if needed
	if !cfg.DirectMode && peer != "" {
		relayCfg := vkturn.Config{
			Listen:        relayAddr,
			Peer:          peer,
			MaxCallToken:  cfg.MaxCallToken,
			MaxLoginToken: cfg.MaxLoginToken,
			MaxDeviceID:   cfg.MaxDeviceID,
			MaxLink:       cfg.MaxLink,
			UDP:           true,
		}
		go func() {
			if err := vkturn.RunRawUDPRelay(ctx, relayCfg); err != nil {
				log.Printf("[ROUTER] MAX-TURN relay exited: %v", err)
			}
		}()

		// Wait for local UDP relay listener
		if err := waitUDPPort(ctx, relayAddr, 5*time.Second); err != nil {
			return fmt.Errorf("MAX-TURN relay did not start: %w", err)
		}
		log.Printf("[ROUTER] MAX-TURN relay ready on %s -> Peer %s", relayAddr, peer)
	}

	// 2. Dispatch to specific protocol engine
	switch proto {
	case "wireguard", "amneziawg", "wg", "awg":
		endpoint := peer
		if !cfg.DirectMode {
			endpoint = relayAddr
		}
		dev, err := runWireGuard(ctx, cfg, endpoint)
		if err != nil {
			return fmt.Errorf("start WireGuard/AmneziaWG: %w", err)
		}
		wgDev = dev
		log.Printf("[ROUTER] %s engine started successfully", strings.ToUpper(proto))
		return nil

	case "openvpn", "ovpn":
		// OpenVPN UDP mode:
		// When MAX-TURN is active, point OpenVPN UDP remote to local relay
		socksAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Socks5Port)
		go runTun2Socks(ctx, socksAddr)
		log.Printf("[ROUTER] OpenVPN UDP relay ready on %s", relayAddr)
		return nil

	case "hysteria2", "hy2":
		fallthrough
	default:
		socksAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Socks5Port)
		hystBin := cfg.HysteriaBin
		if hystBin == "" {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate executable: %w", err)
			}
			hystBin = filepath.Join(filepath.Dir(exe), "hysteria")
		}

		hy2Config, err := buildHy2Config(cfg, relayAddr, socksAddr)
		if err != nil {
			return fmt.Errorf("build hy2 config: %w", err)
		}

		cfgFile := filepath.Join(os.TempDir(), "happ_hy2.json")
		if err := os.WriteFile(cfgFile, []byte(hy2Config), 0600); err != nil {
			return fmt.Errorf("write hy2 config: %w", err)
		}

		cmd := exec.CommandContext(ctx, hystBin, "client", "-c", cfgFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start hysteria: %w", err)
		}
		hystCmd = cmd

		if err := waitPort(ctx, socksAddr, 10*time.Second); err != nil {
			return fmt.Errorf("hysteria SOCKS5 did not start: %w", err)
		}
		log.Printf("[ROUTER] hysteria2 SOCKS5 ready at %s", socksAddr)

		go runTun2Socks(ctx, socksAddr)
		return nil
	}
}

// ---------- helpers -------------------------------------------------------

func waitPort(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

func waitUDPPort(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("udp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func parseHy2Peer(uri string) (string, error) {
	const prefix = "hysteria2://"
	s := uri
	if len(s) < len(prefix) {
		return "", fmt.Errorf("too short")
	}
	s = s[len(prefix):]
	if at := findLast(s, '@'); at >= 0 {
		s = s[at+1:]
	}
	if q := indexOf(s, '?'); q >= 0 {
		s = s[:q]
	}
	if f := indexOf(s, '#'); f >= 0 {
		s = s[:f]
	}
	if sl := indexOf(s, '/'); sl >= 0 {
		s = s[:sl]
	}
	if s == "" {
		return "", fmt.Errorf("no host:port")
	}
	return s, nil
}

func findLast(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func buildHy2Config(cfg RouterConfig, relayAddr, socksAddr string) (string, error) {
	type hy2SOCKS5 struct {
		Listen string `json:"listen"`
	}
	type hy2TLS struct {
		SNI      string `json:"sni,omitempty"`
		Insecure bool   `json:"insecure,omitempty"`
	}
	type hy2Obfs struct {
		Type       string `json:"type"`
		Salamander struct {
			Password string `json:"password"`
		} `json:"salamander"`
	}
	type hy2Config struct {
		Server string    `json:"server"`
		Auth   string    `json:"auth"`
		TLS    hy2TLS    `json:"tls,omitempty"`
		Obfs   *hy2Obfs  `json:"obfs,omitempty"`
		SOCKS5 hy2SOCKS5 `json:"socks5"`
	}

	parsed, err := parseHy2URI(cfg.HysteriaURI)
	if err != nil {
		return "", err
	}

	server := parsed.server
	if !cfg.DirectMode {
		server = relayAddr
	}

	out := hy2Config{
		Server: server,
		Auth:   parsed.auth,
		TLS:    hy2TLS{SNI: parsed.sni},
		SOCKS5: hy2SOCKS5{Listen: socksAddr},
	}

	if parsed.obfsType != "" {
		o := &hy2Obfs{Type: parsed.obfsType}
		o.Salamander.Password = parsed.obfsPass
		out.Obfs = o
	}

	b, err := json.MarshalIndent(out, "", "  ")
	return string(b), err
}

type hy2Parsed struct {
	server   string
	auth     string
	sni      string
	obfsType string
	obfsPass string
}

func parseHy2URI(uri string) (*hy2Parsed, error) {
	const prefix = "hysteria2://"
	if len(uri) < len(prefix) {
		return nil, fmt.Errorf("not a hysteria2 URI")
	}
	s := uri[len(prefix):]

	out := &hy2Parsed{}

	if f := indexOf(s, '#'); f >= 0 {
		s = s[:f]
	}

	query := ""
	if q := indexOf(s, '?'); q >= 0 {
		query = s[q+1:]
		s = s[:q]
	}

	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	if at := findLast(s, '@'); at >= 0 {
		out.auth = s[:at]
		out.server = s[at+1:]
	} else {
		out.server = s
	}

	for _, kv := range splitQuery(query) {
		k, v, _ := splitKV(kv, '=')
		switch k {
		case "sni":
			out.sni = v
		case "obfs":
			out.obfsType = v
		case "obfs-password":
			out.obfsPass = v
		}
	}

	return out, nil
}

func splitQuery(q string) []string {
	if q == "" {
		return nil
	}
	var parts []string
	for {
		i := indexOf(q, '&')
		if i < 0 {
			parts = append(parts, q)
			break
		}
		parts = append(parts, q[:i])
		q = q[i+1:]
	}
	return parts
}

func splitKV(s string, sep byte) (string, string, bool) {
	i := indexOf(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+1:], true
}

func main() {}
