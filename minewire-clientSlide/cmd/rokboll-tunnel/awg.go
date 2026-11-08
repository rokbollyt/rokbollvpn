package main

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"sync"
)

// AWGConfig holds AmneziaWG obfuscation parameters.
type AWGConfig struct {
	Enabled bool
	Jc      int
	Jmin    int
	Jmax    int
	S1      int
	S2      int
	H1      uint32
	H2      uint32
	H3      uint32
	H4      uint32
}

func defaultAWGConfig() AWGConfig {
	return AWGConfig{
		Enabled: true,
		Jc:      4,
		Jmin:    40,
		Jmax:    70,
		S1:      0,
		S2:      0,
		H1:      1,
		H2:      2,
		H3:      3,
		H4:      4,
	}
}

// AWGTranslator translates packets between standard WireGuard (used by local Xray)
// and AmneziaWG (expected by the remote server).
type AWGTranslator struct {
	cfg AWGConfig
	mu  sync.Mutex
}

func NewAWGTranslator(cfg AWGConfig) *AWGTranslator {
	return &AWGTranslator{cfg: cfg}
}

// ObfuscateClientPacket converts a standard WireGuard packet from local client
// into an AmneziaWG packet to be sent to the remote peer.
// Returns optional junk packets (if Jc > 0 on initiation) and the obfuscated main packet.
func (t *AWGTranslator) ObfuscateClientPacket(pkt []byte) (junks [][]byte, out []byte) {
	if !t.cfg.Enabled || len(pkt) < 4 {
		return nil, pkt
	}

	wgType := binary.LittleEndian.Uint32(pkt[:4])

	switch wgType {
	case 1: // Handshake Initiation (148 bytes in WireGuard)
		// Generate Jc junk packets if configured
		if t.cfg.Jc > 0 && t.cfg.Jmax > t.cfg.Jmin && t.cfg.Jmin > 0 {
			diff := int64(t.cfg.Jmax - t.cfg.Jmin + 1)
			for i := 0; i < t.cfg.Jc; i++ {
				n, _ := rand.Int(rand.Reader, big.NewInt(diff))
				junkLen := t.cfg.Jmin + int(n.Int64())
				junk := make([]byte, junkLen)
				_, _ = rand.Read(junk)
				junks = append(junks, junk)
			}
		}

		// Prepend S1 random bytes and set H1
		s1 := t.cfg.S1
		out = make([]byte, s1+len(pkt))
		if s1 > 0 {
			_, _ = rand.Read(out[:s1])
		}
		copy(out[s1:], pkt)
		binary.LittleEndian.PutUint32(out[s1:s1+4], t.cfg.H1)
		return junks, out

	case 2: // Handshake Response (92 bytes in WireGuard)
		s2 := t.cfg.S2
		out = make([]byte, s2+len(pkt))
		if s2 > 0 {
			_, _ = rand.Read(out[:s2])
		}
		copy(out[s2:], pkt)
		binary.LittleEndian.PutUint32(out[s2:s2+4], t.cfg.H2)
		return nil, out

	case 3: // Cookie Reply (64 bytes in WireGuard)
		out = make([]byte, len(pkt))
		copy(out, pkt)
		binary.LittleEndian.PutUint32(out[:4], t.cfg.H3)
		return nil, out

	case 4: // Data packet (>= 32 bytes in WireGuard)
		out = make([]byte, len(pkt))
		copy(out, pkt)
		binary.LittleEndian.PutUint32(out[:4], t.cfg.H4)
		return nil, out

	default:
		return nil, pkt
	}
}

// DeobfuscateServerPacket converts an AmneziaWG packet from the remote peer
// back into a standard WireGuard packet for local Xray.
// Returns nil if packet is unrecognized junk.
func (t *AWGTranslator) DeobfuscateServerPacket(pkt []byte) []byte {
	if !t.cfg.Enabled {
		return pkt
	}

	// 1. Check Data packet with H4
	if len(pkt) >= 32 {
		h := binary.LittleEndian.Uint32(pkt[:4])
		if h == t.cfg.H4 {
			out := make([]byte, len(pkt))
			copy(out, pkt)
			binary.LittleEndian.PutUint32(out[:4], 4)
			return out
		}
	}

	// 2. Check Handshake Response with S2 prefix and H2
	s2 := t.cfg.S2
	if len(pkt) >= s2+4 {
		h := binary.LittleEndian.Uint32(pkt[s2 : s2+4])
		if h == t.cfg.H2 {
			inner := pkt[s2:]
			out := make([]byte, len(inner))
			copy(out, inner)
			binary.LittleEndian.PutUint32(out[:4], 2)
			return out
		}
	}

	// 3. Check Cookie Reply with H3
	if len(pkt) >= 4 {
		h := binary.LittleEndian.Uint32(pkt[:4])
		if h == t.cfg.H3 {
			out := make([]byte, len(pkt))
			copy(out, pkt)
			binary.LittleEndian.PutUint32(out[:4], 3)
			return out
		}
	}

	// 4. Check Handshake Initiation with S1 prefix and H1 (if server ever sends init)
	s1 := t.cfg.S1
	if len(pkt) >= s1+4 {
		h := binary.LittleEndian.Uint32(pkt[s1 : s1+4])
		if h == t.cfg.H1 {
			inner := pkt[s1:]
			out := make([]byte, len(inner))
			copy(out, inner)
			binary.LittleEndian.PutUint32(out[:4], 1)
			return out
		}
	}

	// Junk packet from server (e.g. Amnezia handshake junk), drop silently
	return nil
}

