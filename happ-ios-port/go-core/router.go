// Package core is the Go static library that powers HappPort's VPN tunnel.
// It exports a minimal C API consumed by the Swift PacketTunnelProvider:
//
//   RouterStart(configJSON *C.char) C.int   – start tunnel, returns 0 on OK
//   RouterStop()                            – graceful shutdown
//   RouterWritePacket(data *C.char, n C.int) – inject L3 packet from tun into Go
//   RouterReadPacket(buf  *C.char, max C.int) C.int – read outbound L3 packet
//
// Architecture:
//   iOS tun fd
//     ↕ (read/write loop in Swift PacketTunnelProvider)
//   RouterWritePacket / RouterReadPacket  ← C bridge
//     ↕
//   gVisor userspace TCP/IP stack (netstack via tun2socks)
//     ↕
//   SOCKS5 → Hysteria2 client (subprocess, local :10808)
//     ↕ (UDP packets routed through MAX-TURN relay)
//   MAX-TURN relay (embedded goroutines, no subprocess)
//     ↕
//   Hysteria2 server (remote)
//
// The relay is embedded: RunRawUDPRelay runs in a goroutine and owns the
// TURN socket.  Hysteria2 is launched as a child process (darwin/arm64 binary
// bundled in app resources) pointing its upstream UDP port at the local relay
// listener.  The tun2socks adapter bridges the tun channel to the SOCKS5 port
// that Hysteria2 exposes.
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
	"sync"
	"time"
	"unsafe"

	vkturn "github.com/cacggghp/vk-turn-proxy/client"
)

// ---------- packet channel -----------------------------------------------

const chanSize = 4096
const maxPkt = 65536

var (
	toGo   = make(chan []byte, chanSize) // Swift → Go (inbound from device)
	fromGo = make(chan []byte, chanSize) // Go → Swift (outbound to device)
)

// ---------- global state --------------------------------------------------

var (
	mu       sync.Mutex
	cancel   context.CancelFunc
	running  bool
	hystCmd  *exec.Cmd
)

// ---------- config JSON ---------------------------------------------------

// RouterConfig is the JSON structure passed from Swift via RouterStart.
type RouterConfig struct {
	// Hysteria2 share URI, e.g. hysteria2://...
	HysteriaURI string `json:"hysteriaURI"`

	// MAX-TURN credentials
	MaxCallToken  string `json:"maxCallToken"`
	MaxLoginToken string `json:"maxLoginToken"`
	MaxDeviceID   string `json:"maxDeviceID"`
	MaxLink       string `json:"maxLink"`

	// Optional override: address of Hysteria2 server for TURN peer.
	// If empty, parsed from HysteriaURI.
	PeerAddr string `json:"peerAddr"`

	// Local ports (defaults used if 0)
	Socks5Port int `json:"socks5Port"` // Hysteria2 SOCKS5 output, default 10808
	RelayPort  int `json:"relayPort"`  // Local UDP port that relay listens on, default 10900

	// Path to hysteria binary inside app bundle.
	// If empty, looks for "hysteria" next to the process executable.
	HysteriaBin string `json:"hysteriaBin"`

	// If true, skip MAX-TURN relay and connect Hysteria2 directly.
	DirectMode bool `json:"directMode"`
}

func defaultConfig() RouterConfig {
	return RouterConfig{
		Socks5Port: 10808,
		RelayPort:  10900,
	}
}

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
	socksAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Socks5Port)
	relayAddr := fmt.Sprintf("127.0.0.1:%d", cfg.RelayPort)

	// --- parse peer addr from URI if not supplied ---
	peer := cfg.PeerAddr
	if peer == "" && cfg.HysteriaURI != "" {
		p, err := parseHy2Peer(cfg.HysteriaURI)
		if err != nil {
			return fmt.Errorf("parse hysteria2 URI: %w", err)
		}
		peer = p
	}
	if peer == "" && !cfg.DirectMode {
		return fmt.Errorf("no peer address and not direct mode")
	}

	// --- start MAX-TURN relay (unless direct mode) ---
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
				log.Printf("[ROUTER] relay exited: %v", err)
			}
		}()
		// Wait until relay listener is ready (up to 5 s)
		if err := waitPort(ctx, relayAddr, 5*time.Second); err != nil {
			return fmt.Errorf("relay did not start: %w", err)
		}
		log.Printf("[ROUTER] relay ready at %s", relayAddr)
	}

	// --- build Hysteria2 config and start process ---
	hystBin := cfg.HysteriaBin
	if hystBin == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
		hystBin = filepath.Join(filepath.Dir(exe), "hysteria")
	}

	// Build the hysteria2 client config JSON
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

	// Wait for SOCKS5 port to be reachable
	if err := waitPort(ctx, socksAddr, 10*time.Second); err != nil {
		return fmt.Errorf("hysteria SOCKS5 did not start: %w", err)
	}
	log.Printf("[ROUTER] hysteria2 SOCKS5 ready at %s", socksAddr)

	// --- start tun2socks bridge ---
	go runTun2Socks(ctx, socksAddr)

	return nil
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

func parseHy2Peer(uri string) (string, error) {
	// hysteria2://auth@host:port?...
	// We need host:port as the UDP peer.
	const prefix = "hysteria2://"
	s := uri
	if len(s) < len(prefix) {
		return "", fmt.Errorf("too short")
	}
	s = s[len(prefix):]
	// strip auth (up to @)
	if at := findLast(s, '@'); at >= 0 {
		s = s[at+1:]
	}
	// strip query
	if q := indexOf(s, '?'); q >= 0 {
		s = s[:q]
	}
	// strip fragment
	if f := indexOf(s, '#'); f >= 0 {
		s = s[:f]
	}
	// strip path
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

// buildHy2Config generates a Hysteria2 client JSON config.
// When MAX-TURN is active, the server address is replaced with the local relay addr.
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
	if !cfg.DirectMode && cfg.PeerAddr == "" {
		// Route through relay: replace remote host:port with local relay
		server = relayAddr
	} else if !cfg.DirectMode && cfg.PeerAddr != "" {
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
	// hysteria2://auth@host:port/?sni=...&obfs=salamander&obfs-password=...
	const prefix = "hysteria2://"
	if len(uri) < len(prefix) {
		return nil, fmt.Errorf("not a hysteria2 URI")
	}
	s := uri[len(prefix):]

	out := &hy2Parsed{}

	// strip fragment
	if f := indexOf(s, '#'); f >= 0 {
		s = s[:f]
	}

	// split query
	query := ""
	if q := indexOf(s, '?'); q >= 0 {
		query = s[q+1:]
		s = s[:q]
	}

	// strip trailing /
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	// auth@host:port
	if at := findLast(s, '@'); at >= 0 {
		out.auth = s[:at]
		out.server = s[at+1:]
	} else {
		out.server = s
	}

	// parse query params
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

// main is required for c-archive but never called.
func main() {}

