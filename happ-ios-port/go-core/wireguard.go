package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"
)

// chanTun implements tun.Device by piping packets directly to/from
// the Swift NEPacketTunnelProvider via toGo and fromGo channels.
type chanTun struct {
	events chan tun.Event
	closed atomic.Bool
	mtu    int
}

func newChanTun(mtu int) *chanTun {
	if mtu <= 0 {
		mtu = 1420
	}
	return &chanTun{
		events: make(chan tun.Event, 10),
		mtu:    mtu,
	}
}

func (t *chanTun) File() *os.File {
	return nil
}

func (t *chanTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	if t.closed.Load() {
		return 0, io.EOF
	}

	select {
	case pkt, ok := <-toGo:
		if !ok || t.closed.Load() {
			return 0, io.EOF
		}
		if len(pkt) == 0 {
			return 0, nil
		}
		n := len(pkt)
		if n > len(bufs[0])-offset {
			n = len(bufs[0]) - offset
		}
		copy(bufs[0][offset:], pkt[:n])
		sizes[0] = n
		return 1, nil
	case <-time.After(50 * time.Millisecond):
		return 0, nil
	}
}

func (t *chanTun) Write(bufs [][]byte, offset int) (int, error) {
	if t.closed.Load() {
		return 0, io.EOF
	}

	for _, b := range bufs {
		if offset >= len(b) {
			continue
		}
		payload := b[offset:]
		if len(payload) == 0 {
			continue
		}
		pkt := make([]byte, len(payload))
		copy(pkt, payload)
		select {
		case fromGo <- pkt:
		default:
		}
	}
	return len(bufs), nil
}

func (t *chanTun) MTU() (int, error) {
	return t.mtu, nil
}

func (t *chanTun) Name() (string, error) {
	return "happ-tun", nil
}

func (t *chanTun) Events() <-chan tun.Event {
	return t.events
}

func (t *chanTun) Close() error {
	t.closed.Store(true)
	close(t.events)
	return nil
}

func (t *chanTun) BatchSize() int {
	return 1
}

// runWireGuard creates and runs an AmneziaWG/WireGuard device.
func runWireGuard(ctx context.Context, cfg RouterConfig, endpoint string) (*device.Device, error) {
	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = 1420
	}
	tunDev := newChanTun(mtu)

	devLogger := device.NewLogger(device.LogLevelError, "[AWG] ")
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), devLogger)

	// Format UAPI configuration string
	var sb strings.Builder

	privHex := keyToHex(cfg.PrivateKey)
	pubHex := keyToHex(cfg.PublicKey)

	sb.WriteString(fmt.Sprintf("private_key=%s\n", privHex))
	sb.WriteString(fmt.Sprintf("public_key=%s\n", pubHex))
	sb.WriteString(fmt.Sprintf("endpoint=%s\n", endpoint))
	sb.WriteString("allowed_ip=0.0.0.0/0\n")
	sb.WriteString("allowed_ip=::/0\n")
	sb.WriteString("persistent_keepalive_interval=25\n")

	if cfg.PresharedKey != "" {
		pskHex := keyToHex(cfg.PresharedKey)
		sb.WriteString(fmt.Sprintf("preshared_key=%s\n", pskHex))
	}

	// AmneziaWG obfuscation parameters
	if cfg.H1 > 0 || cfg.H2 > 0 || cfg.H3 > 0 || cfg.H4 > 0 || cfg.Jc > 0 {
		sb.WriteString(fmt.Sprintf("jc=%d\n", cfg.Jc))
		sb.WriteString(fmt.Sprintf("jmin=%d\n", cfg.Jmin))
		sb.WriteString(fmt.Sprintf("jmax=%d\n", cfg.Jmax))
		sb.WriteString(fmt.Sprintf("s1=%d\n", cfg.S1))
		sb.WriteString(fmt.Sprintf("s2=%d\n", cfg.S2))
		sb.WriteString(fmt.Sprintf("h1=%d\n", cfg.H1))
		sb.WriteString(fmt.Sprintf("h2=%d\n", cfg.H2))
		sb.WriteString(fmt.Sprintf("h3=%d\n", cfg.H3))
		sb.WriteString(fmt.Sprintf("h4=%d\n", cfg.H4))
	}

	uapi := sb.String()
	if err := dev.IpcSet(uapi); err != nil {
		dev.Close()
		return nil, fmt.Errorf("IpcSet error: %w", err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("device Up error: %w", err)
	}

	log.Printf("[AWG] WireGuard/AmneziaWG device active, endpoint: %s", endpoint)

	go func() {
		<-ctx.Done()
		dev.Close()
	}()

	return dev, nil
}

// keyToHex converts base64 key or raw hex string to 64-character hex string.
func keyToHex(k string) string {
	k = strings.TrimSpace(k)
	// If already 64-char hex
	if len(k) == 64 {
		if _, err := hex.DecodeString(k); err == nil {
			return k
		}
	}
	// Try base64 decode
	b, err := hexOrBase64ToBytes(k)
	if err == nil && len(b) == 32 {
		return hex.EncodeToString(b)
	}
	return k
}

func hexOrBase64ToBytes(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	// Standard base64
	pad := (4 - (len(s) % 4)) % 4
	s = s + strings.Repeat("=", pad)
	var b [32]byte
	n, err := hexOrBase64Decode(s, b[:])
	if err == nil && n == 32 {
		return b[:], nil
	}
	return nil, fmt.Errorf("invalid key encoding")
}

func hexOrBase64Decode(s string, out []byte) (int, error) {
	const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var buf [4]byte
	bufIdx := 0
	outIdx := 0

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '=' {
			break
		}
		pos := strings.IndexByte(b64, c)
		if pos < 0 {
			if c == '-' {
				pos = 62
			} else if c == '_' {
				pos = 63
			} else {
				continue
			}
		}
		buf[bufIdx] = byte(pos)
		bufIdx++
		if bufIdx == 4 {
			if outIdx+3 > len(out) {
				return outIdx, nil
			}
			out[outIdx] = (buf[0] << 2) | (buf[1] >> 4)
			out[outIdx+1] = (buf[1] << 4) | (buf[2] >> 2)
			out[outIdx+2] = (buf[2] << 6) | buf[3]
			outIdx += 3
			bufIdx = 0
		}
	}
	if bufIdx > 1 && outIdx < len(out) {
		out[outIdx] = (buf[0] << 2) | (buf[1] >> 4)
		outIdx++
		if bufIdx > 2 && outIdx < len(out) {
			out[outIdx] = (buf[1] << 4) | (buf[2] >> 2)
			outIdx++
		}
	}
	return outIdx, nil
}
