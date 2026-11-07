package tcputil

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	FrameOpen  byte = 0x01
	FrameData  byte = 0x02
	FrameClose byte = 0x03
	FrameReset byte = 0x04
	FramePing  byte = 0x05

	HeaderSize = 11
)

// EncodeFrame serializes a multipath frame into a single byte slice for atomic network write.
func EncodeFrame(fType byte, streamID uint32, seq uint32, payload []byte) []byte {
	buf := make([]byte, HeaderSize+len(payload))
	buf[0] = fType
	binary.BigEndian.PutUint32(buf[1:5], streamID)
	binary.BigEndian.PutUint32(buf[5:9], seq)
	binary.BigEndian.PutUint16(buf[9:11], uint16(len(payload)))
	if len(payload) > 0 {
		copy(buf[HeaderSize:], payload)
	}
	return buf
}

// ReadFrame reads a single frame from an io.Reader.
func ReadFrame(r io.Reader) (fType byte, streamID uint32, seq uint32, payload []byte, err error) {
	var hdr [HeaderSize]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return 0, 0, 0, nil, err
	}
	fType = hdr[0]
	streamID = binary.BigEndian.Uint32(hdr[1:5])
	seq = binary.BigEndian.Uint32(hdr[5:9])
	pLen := int(binary.BigEndian.Uint16(hdr[9:11]))
	if pLen > 0 {
		payload = make([]byte, pLen)
		if _, err = io.ReadFull(r, payload); err != nil {
			return 0, 0, 0, nil, err
		}
	}
	return fType, streamID, seq, payload, nil
}

// ReorderBuffer reorders chunks arriving across different TURN multipath sessions.
type ReorderBuffer struct {
	mu      sync.Mutex
	nextSeq uint32
	chunks  map[uint32][]byte
	closed  bool
}

// NewReorderBuffer creates an initialized ReorderBuffer starting at seq 0.
func NewReorderBuffer() *ReorderBuffer {
	return &ReorderBuffer{
		chunks: make(map[uint32][]byte),
	}
}

// Push adds a chunk at seq. Returns all contiguous chunks ready to be consumed in order.
func (rb *ReorderBuffer) Push(seq uint32, data []byte) [][]byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Duplicate or old chunk
	if seq < rb.nextSeq {
		return nil
	}

	rb.chunks[seq] = data

	var ready [][]byte
	for {
		chunk, exists := rb.chunks[rb.nextSeq]
		if !exists {
			break
		}
		delete(rb.chunks, rb.nextSeq)
		ready = append(ready, chunk)
		rb.nextSeq++
	}

	return ready
}

// Close marks the buffer as closed.
func (rb *ReorderBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.closed = true
}

// Pending returns the number of out-of-order chunks waiting in the buffer.
func (rb *ReorderBuffer) Pending() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return len(rb.chunks)
}

// WriteOpenPayload encodes target host and port into Open frame payload.
func WriteOpenPayload(host string, port int) []byte {
	b := make([]byte, 3+len(host))
	binary.BigEndian.PutUint16(b[0:2], uint16(port))
	b[2] = byte(len(host))
	copy(b[3:], host)
	return b
}

// ReadOpenPayload decodes target host and port from Open frame payload.
func ReadOpenPayload(payload []byte) (string, int, error) {
	if len(payload) < 3 {
		return "", 0, errors.New("open payload too short")
	}
	port := int(binary.BigEndian.Uint16(payload[0:2]))
	hLen := int(payload[2])
	if len(payload) < 3+hLen {
		return "", 0, errors.New("invalid host length in open payload")
	}
	host := string(payload[3 : 3+hLen])
	return fmt.Sprintf("%s:%d", host, port), port, nil
}
