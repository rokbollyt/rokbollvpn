package tcputil

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

// envInt читает целочисленный параметр из переменной окружения, возвращая
// значение по умолчанию, если переменная не задана или некорректна.
func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		// KCP_NC=0 — валидное значение, но Atoi>0 его отбросит; обработаем явно.
		if n, err := strconv.Atoi(v); err == nil && n == 0 {
			return 0
		}
	}
	return def
}

// DtlsPacketConn wraps a net.Conn (DTLS) as a net.PacketConn for KCP.
// Each DTLS Read/Write preserves message boundaries (datagram semantics).
type DtlsPacketConn struct {
	conn net.Conn
}

func NewDtlsPacketConn(conn net.Conn) *DtlsPacketConn {
	return &DtlsPacketConn{conn: conn}
}

func (d *DtlsPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := d.conn.Read(b)
	return n, d.conn.RemoteAddr(), err
}

func (d *DtlsPacketConn) WriteTo(b []byte, _ net.Addr) (int, error) {
	return d.conn.Write(b)
}

func (d *DtlsPacketConn) Close() error {
	return d.conn.Close()
}

func (d *DtlsPacketConn) LocalAddr() net.Addr {
	return d.conn.LocalAddr()
}

func (d *DtlsPacketConn) SetDeadline(t time.Time) error {
	return d.conn.SetDeadline(t)
}

func (d *DtlsPacketConn) SetReadDeadline(t time.Time) error {
	return d.conn.SetReadDeadline(t)
}

func (d *DtlsPacketConn) SetWriteDeadline(t time.Time) error {
	return d.conn.SetWriteDeadline(t)
}

// NewKCPOverDTLS creates a KCP session over a DTLS connection.
// isServer: true for server-side (listener), false for client-side (dialer).
func NewKCPOverDTLS(dtlsConn net.Conn, isServer bool) (*kcp.UDPSession, error) {
	pc := NewDtlsPacketConn(dtlsConn)

	block, err := kcp.NewNoneBlockCrypt(nil) // DTLS already encrypts
	if err != nil {
		return nil, err
	}

	var sess *kcp.UDPSession

	if isServer {
		// Server: listen on the PacketConn and accept one session
		var listener *kcp.Listener
		listener, err = kcp.ServeConn(block, 0, 0, pc)
		if err != nil {
			return nil, err
		}
		if err = listener.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return nil, err
		}
		sess, err = listener.AcceptKCP()
		if err != nil {
			return nil, err
		}
	} else {
		// Client: dial through the PacketConn
		sess, err = kcp.NewConn2(dtlsConn.RemoteAddr(), block, 0, 0, pc)
		if err != nil {
			return nil, err
		}
	}

	// Настройка KCP под TURN-канал.
	//
	// Значения подбираются экспериментально и поэтому вынесены в переменные
	// окружения: канал через TURN капризный (высокий RTT, потери, скрытый
	// лимит MTU внутри DTLS+STUN ChannelData), и оптимум сильно зависит от
	// конкретного маршрута. Дефолты — консервативные и проверенные.
	//
	//	KCP_MTU       размер сегмента KCP (по умолчанию 1000)
	//	KCP_SNDWND    окно отправки в пакетах (по умолчанию 512)
	//	KCP_RCVWND    окно приёма в пакетах (по умолчанию 512)
	//	KCP_INTERVAL  интервал внутреннего таймера, мс (по умолчанию 20)
	//	KCP_RESEND    порог быстрой повторной отправки (по умолчанию 2)
	//	KCP_NC        1 = отключить контроль перегрузки (по умолчанию 0 — включён)
	mtu := envInt("KCP_MTU", 1000)
	sndWnd := envInt("KCP_SNDWND", 512)
	rcvWnd := envInt("KCP_RCVWND", 512)
	interval := envInt("KCP_INTERVAL", 20)
	resend := envInt("KCP_RESEND", 2)
	nc := envInt("KCP_NC", 0)

	// Порядок важен: MTU влияет на расчёт окон, поэтому ставим его первым.
	sess.SetMtu(mtu)
	sess.SetNoDelay(1, interval, resend, nc)
	sess.SetWindowSize(sndWnd, rcvWnd)
	sess.SetACKNoDelay(true)
	// Буферы сокета: без них ядро отбрасывает пакеты на всплесках трафика.
	_ = sess.SetReadBuffer(4 * 1024 * 1024)
	_ = sess.SetWriteBuffer(4 * 1024 * 1024)

	return sess, nil
}

// DefaultSmuxConfig returns smux config tuned for TURN tunnel.
func DefaultSmuxConfig() *smux.Config {
	cfg := smux.DefaultConfig()
	// Версия 2 экономнее по накладным расходам и лучше делит полосу между
	// потоками, чем версия 1.
	cfg.Version = 2
	cfg.MaxReceiveBuffer = 16 * 1024 * 1024
	cfg.MaxStreamBuffer = 2 * 1024 * 1024
	cfg.KeepAliveInterval = 10 * time.Second
	cfg.KeepAliveTimeout = 30 * time.Second
	return cfg
}
