package tcputil

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"

	"github.com/xtaci/kcp-go/v5"
)

// DefaultSecretKey is the default preshared key used for KCP AES block crypt.
const DefaultSecretKey = "rokboll-turn-key-32bytes-secret!"

// RTPPacketConn wraps a net.PacketConn to prepend a 12-byte standard WebRTC RTP header
// on write and strip it on read.
//
// Mail.ru / VK Calls TURN servers inspect the first byte of incoming UDP datagrams:
//   - Packets starting with 0x14-0x19 (DTLS ApplicationData/Control) are classified
//     as control signaling and throttled to 10 pkts/s (~0.08 Mbps).
//   - Packets starting with 0x80 (WebRTC RTP/SRTP media) are classified as media streams
//     and passed at full bandwidth (~2.09 Mbps per TURN server).
type RTPPacketConn struct {
	net.PacketConn
	txSeq atomic.Uint32
}

// NewRTPPacketConn wraps a net.PacketConn with RTP framing.
func NewRTPPacketConn(pc net.PacketConn) *RTPPacketConn {
	return &RTPPacketConn{PacketConn: pc}
}

func (r *RTPPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	tmp := make([]byte, 2048)
	n, addr, err := r.PacketConn.ReadFrom(tmp)
	if err != nil {
		return 0, addr, err
	}
	// Strip 12-byte RTP header if present (starts with 0x80)
	if n >= 12 && tmp[0] == 0x80 {
		copy(b, tmp[12:n])
		return n - 12, addr, nil
	}
	copy(b, tmp[:n])
	return n, addr, nil
}

func (r *RTPPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	rtpPkt := make([]byte, 12+len(b))
	rtpPkt[0] = 0x80 // RFC 3550 RTP Version 2
	rtpPkt[1] = 0x60 // Dynamic payload type (96)
	seq := uint16(r.txSeq.Add(1))
	binary.BigEndian.PutUint16(rtpPkt[2:4], seq)
	nowTs := uint32(time.Now().UnixNano() / int64(time.Millisecond))
	binary.BigEndian.PutUint32(rtpPkt[4:8], nowTs)
	binary.BigEndian.PutUint32(rtpPkt[8:12], 0x11223344) // SSRC identifier
	copy(rtpPkt[12:], b)

	_, err := r.PacketConn.WriteTo(rtpPkt, addr)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// NewKCPOverRTP creates a KCP session over an RTP-framed PacketConn.
func NewKCPOverRTP(pc net.PacketConn, peer net.Addr, isServer bool, secretKey string) (*kcp.UDPSession, error) {
	rtpConn := NewRTPPacketConn(pc)
	block, err := kcp.NewAESBlockCrypt([]byte(secretKey))
	if err != nil {
		return nil, err
	}

	var sess *kcp.UDPSession
	if isServer {
		listener, err := kcp.ServeConn(block, 10, 3, rtpConn)
		if err != nil {
			return nil, err
		}
		sess, err = listener.AcceptKCP()
		if err != nil {
			return nil, err
		}
	} else {
		sess, err = kcp.NewConn2(peer, block, 10, 3, rtpConn)
		if err != nil {
			return nil, err
		}
	}

	mtu := envInt("KCP_MTU", 1000)
	sndWnd := envInt("KCP_SNDWND", 512)
	rcvWnd := envInt("KCP_RCVWND", 512)
	interval := envInt("KCP_INTERVAL", 20)
	resend := envInt("KCP_RESEND", 2)
	nc := envInt("KCP_NC", 0)

	sess.SetMtu(mtu)
	sess.SetNoDelay(1, interval, resend, nc)
	sess.SetWindowSize(sndWnd, rcvWnd)
	sess.SetACKNoDelay(true)
	_ = sess.SetReadBuffer(4 * 1024 * 1024)
	_ = sess.SetWriteBuffer(4 * 1024 * 1024)

	return sess, nil
}
