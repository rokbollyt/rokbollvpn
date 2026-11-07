package vkturn

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cacggghp/vk-turn-proxy/tcputil"
	"github.com/xtaci/smux"
)

type clientTrunk struct {
	stream    *smux.Stream
	ch        chan []byte
	closed    atomic.Bool
	closeOnce sync.Once
}

func newClientTrunk(s *smux.Stream) *clientTrunk {
	ct := &clientTrunk{
		stream: s,
		ch:     make(chan []byte, 2048),
	}
	go ct.sendLoop()
	return ct
}

func (ct *clientTrunk) close() {
	ct.closeOnce.Do(func() {
		ct.closed.Store(true)
		_ = ct.stream.Close()
	})
}

func (ct *clientTrunk) sendLoop() {
	for chunk := range ct.ch {
		if ct.closed.Load() {
			return
		}
		_, err := ct.stream.Write(chunk)
		if err != nil {
			ct.close()
			return
		}
	}
}

type clientTrunkManager struct {
	mu     sync.RWMutex
	trunks []*clientTrunk
	rrIdx  uint32
}

func (cm *clientTrunkManager) addTrunk(s *smux.Stream) *clientTrunk {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ct := newClientTrunk(s)
	cm.trunks = append(cm.trunks, ct)
	log.Printf("[Multipath] Active trunks: %d", len(cm.trunks))
	return ct
}

func (cm *clientTrunkManager) removeTrunk(ct *clientTrunk) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ct.close()
	for i, tr := range cm.trunks {
		if tr == ct {
			cm.trunks = append(cm.trunks[:i], cm.trunks[i+1:]...)
			break
		}
	}
	log.Printf("[Multipath] Active trunks after remove: %d", len(cm.trunks))
}

func (cm *clientTrunkManager) getTrunk() *clientTrunk {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	n := len(cm.trunks)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := atomic.AddUint32(&cm.rrIdx, 1) % uint32(n)
		t := cm.trunks[idx]
		if !t.closed.Load() {
			return t
		}
	}
	return nil
}

func (cm *clientTrunkManager) sendRR(frame []byte) {
	for {
		cm.mu.RLock()
		n := len(cm.trunks)
		if n == 0 {
			cm.mu.RUnlock()
			time.Sleep(2 * time.Millisecond)
			continue
		}
		startIdx := atomic.AddUint32(&cm.rrIdx, 1)
		for i := 0; i < n; i++ {
			t := cm.trunks[(startIdx+uint32(i))%uint32(n)]
			if !t.closed.Load() {
				select {
				case t.ch <- frame:
					cm.mu.RUnlock()
					return
				default:
				}
			}
		}
		cm.mu.RUnlock()
		time.Sleep(50 * time.Microsecond)
	}
}

func (cm *clientTrunkManager) count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.trunks)
}

type clientStreamState struct {
	userConn   net.Conn
	reorderBuf *tcputil.ReorderBuffer
	upSeq      uint32
	closed     atomic.Bool
	closeOnce  sync.Once
}

type clientStreamTable struct {
	mu sync.RWMutex
	m  map[uint32]*clientStreamState
}

func (t *clientStreamTable) get(id uint32) *clientStreamState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.m[id]
}

func (t *clientStreamTable) register(id uint32, s *clientStreamState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.m[id] = s
}

func (t *clientStreamTable) remove(id uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.m, id)
}

var (
	globalStreamIDGen atomic.Uint32
)

// runDirectSocksMode runs high-speed Multipath SOCKS5 proxy across all TURN sessions.
func runDirectSocksMode(ctx context.Context, tp *turnParams, peer *net.UDPAddr, listenAddr string, numSessions int) error {
	cm := &clientTrunkManager{}
	stTable := &clientStreamTable{m: make(map[uint32]*clientStreamState)}
	clientSessionID := uint32(time.Now().UnixNano())

	var wgMaint sync.WaitGroup
	for i := 0; i < numSessions; i++ {
		wgMaint.Add(1)
		go func(id int) {
			defer wgMaint.Done()
			if id > 0 {
				delay := time.Duration(id) * 80 * time.Millisecond
				select {
				case <-ctx.Done():
					return
				case <-time.After(delay):
				}
			}
			maintainMultipathSession(ctx, tp, peer, id, cm, clientSessionID, stTable)
		}(i)
	}

	log.Printf("direct SOCKS mode: ожидание сессий (всего %d)...", numSessions)
	for {
		select {
		case <-ctx.Done():
			wgMaint.Wait()
			return nil
		case <-time.After(100 * time.Millisecond):
		}
		if cm.count() > 0 {
			break
		}
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("SOCKS5 listen %q: %w", listenAddr, err)
	}
	context.AfterFunc(ctx, func() { _ = listener.Close() })
	log.Printf("direct SOCKS mode: SOCKS5 на %s (Multipath striping по %d сессиям)", listenAddr, numSessions)

	var wgConn sync.WaitGroup
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				wgConn.Wait()
				wgMaint.Wait()
				return nil
			default:
			}
			continue
		}
		wgConn.Add(1)
		go func(c net.Conn) {
			defer wgConn.Done()
			defer func() { _ = c.Close() }()
			handleDirectSocksMultipath(ctx, c, cm, stTable)
		}(conn)
	}
}

func maintainMultipathSession(ctx context.Context, tp *turnParams, peer *net.UDPAddr, id int, cm *clientTrunkManager, sessID uint32, stTable *clientStreamTable) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[session %d] recovered panic in maintenance: %v", id, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		smuxSess, cleanup, err := createSmuxSession(ctx, tp, peer, id)
		if err != nil {
			log.Printf("[session %d] setup error: %s, retrying...", id, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		stream, err := smuxSess.OpenStream()
		if err != nil {
			cleanup()
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		// Send session ID for automatic old trunk purging on server
		var sessBuf [4]byte
		binary.BigEndian.PutUint32(sessBuf[:], sessID)
		if _, err := stream.Write(sessBuf[:]); err != nil {
			_ = stream.Close()
			cleanup()
			continue
		}

		tr := cm.addTrunk(stream)
		log.Printf("[session %d] multipath trunk active (total: %d)", id, cm.count())

		// Read inbound frames from this trunk
		go func() {
			defer func() {
				cm.removeTrunk(tr)
				_ = stream.Close()
				cleanup()
			}()

			hdr := make([]byte, tcputil.HeaderSize)
			for {
				_, err := io.ReadFull(stream, hdr)
				if err != nil {
					return
				}
				fType := hdr[0]
				streamID := binary.BigEndian.Uint32(hdr[1:5])
				seq := binary.BigEndian.Uint32(hdr[5:9])
				pLen := int(binary.BigEndian.Uint16(hdr[9:11]))
				payload := make([]byte, pLen)
				if pLen > 0 {
					if _, err := io.ReadFull(stream, payload); err != nil {
						return
					}
				}

				st := stTable.get(streamID)
				if st == nil {
					continue
				}

				switch fType {
				case tcputil.FrameData:
					if !st.closed.Load() {
						readyChunks := st.reorderBuf.Push(seq, payload)
						for _, chunk := range readyChunks {
							_, _ = st.userConn.Write(chunk)
						}
					}
				case tcputil.FrameClose, tcputil.FrameReset:
					st.closeOnce.Do(func() {
						st.closed.Store(true)
						_ = st.userConn.Close()
						stTable.remove(streamID)
					})
				}
			}
		}()

		// Wait until smux session closes
		for !smuxSess.IsClosed() {
			select {
			case <-ctx.Done():
				cm.removeTrunk(tr)
				cleanup()
				return
			case <-time.After(1 * time.Second):
			}
		}

		cm.removeTrunk(tr)
		cleanup()
		log.Printf("[session %d] disconnected, reconnecting...", id)

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func handleDirectSocksMultipath(ctx context.Context, c net.Conn, cm *clientTrunkManager, stTable *clientStreamTable) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered panic in handleDirectSocksMultipath: %v", r)
		}
	}()
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))

	host, port, err := readSocks5Request(c)
	if err != nil {
		return
	}

	if cm.count() == 0 {
		_, _ = c.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	streamID := globalStreamIDGen.Add(1)
	st := &clientStreamState{
		userConn:   c,
		reorderBuf: tcputil.NewReorderBuffer(),
	}
	stTable.register(streamID, st)
	defer func() {
		st.closeOnce.Do(func() {
			st.closed.Store(true)
			_ = c.Close()
			stTable.remove(streamID)
			cm.sendRR(tcputil.EncodeFrame(tcputil.FrameClose, streamID, 0, nil))
		})
	}()

	// Send FrameOpen
	openPayload := tcputil.WriteOpenPayload(host, port)
	openFrame := tcputil.EncodeFrame(tcputil.FrameOpen, streamID, 0, openPayload)
	cm.sendRR(openFrame)

	// Reply SOCKS5 success immediately
	if _, err := c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	_ = c.SetDeadline(time.Time{})

	// Uplink loop: read from client and stripe chunks round-robin across all trunks
	upBuf := make([]byte, 32768)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := c.Read(upBuf)
		if n > 0 {
			seq := atomic.AddUint32(&st.upSeq, 1) - 1
			frame := tcputil.EncodeFrame(tcputil.FrameData, streamID, seq, upBuf[:n])
			cm.sendRR(frame)
		}
		if err != nil {
			break
		}
	}
}

// readSocks5Request выполняет рукопожатие SOCKS5 и возвращает адрес цели.
func readSocks5Request(c net.Conn) (string, int, error) {
	buf := make([]byte, 262)
	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return "", 0, err
	}
	if buf[0] != 0x05 {
		return "", 0, fmt.Errorf("не SOCKS5")
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(c, buf[:nMethods]); err != nil {
		return "", 0, err
	}
	if _, err := c.Write([]byte{0x05, 0x00}); err != nil { // без аутентификации
		return "", 0, err
	}

	if _, err := io.ReadFull(c, buf[:4]); err != nil {
		return "", 0, err
	}
	if buf[1] != 0x01 { // только CONNECT
		_, _ = c.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return "", 0, fmt.Errorf("команда %d не поддерживается", buf[1])
	}

	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(c, buf[:4]); err != nil {
			return "", 0, err
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // доменное имя
		if _, err := io.ReadFull(c, buf[:1]); err != nil {
			return "", 0, err
		}
		l := int(buf[0])
		if _, err := io.ReadFull(c, buf[:l]); err != nil {
			return "", 0, err
		}
		host = string(buf[:l])
	case 0x04: // IPv6
		_, _ = c.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return "", 0, fmt.Errorf("IPv6 не поддерживается")
	default:
		return "", 0, fmt.Errorf("неизвестный тип адреса %d", buf[3])
	}

	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return "", 0, err
	}
	port := int(buf[0])<<8 | int(buf[1])
	if port == 0 {
		return "", 0, fmt.Errorf("нулевой порт")
	}
	return host, port, nil
}

var _ = strconv.Itoa
