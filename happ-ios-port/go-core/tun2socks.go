package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/net/proxy"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

// runTun2Socks creates a gVisor userspace netstack that terminates TCP/UDP
// from the device and forwards everything through the local Hysteria2 SOCKS5 proxy.
func runTun2Socks(ctx context.Context, socksAddr string) {
	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})

	// Optimize TCP buffers
	rcv := tcpip.TCPReceiveBufferSizeRangeOption{Min: 1024 * 1024, Default: 4 * 1024 * 1024, Max: 16 * 1024 * 1024}
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &rcv)
	snd := tcpip.TCPSendBufferSizeRangeOption{Min: 1024 * 1024, Default: 4 * 1024 * 1024, Max: 16 * 1024 * 1024}
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &snd)

	ep := channel.New(1024, 1500, "")
	nicID := tcpip.NICID(1)
	if err := s.CreateNIC(nicID, ep); err != nil {
		log.Printf("[TUN2SOCKS] CreateNIC error: %v", err)
		return
	}

	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)
	s.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: nicID})

	// SOCKS5 dialer to local Hysteria2 instance
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		log.Printf("[TUN2SOCKS] SOCKS5 dialer setup error: %v", err)
		return
	}

	// TCP handler
	tcpFwd := tcp.NewForwarder(s, 0, 4096, func(r *tcp.ForwarderRequest) {
		id := r.ID()
		dest := fmt.Sprintf("%s:%d", id.LocalAddress.String(), id.LocalPort)

		var wq waiter.Queue
		connEP, tErr := r.CreateEndpoint(&wq)
		if tErr != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)

		local := gonet.NewTCPConn(&wq, connEP)
		go func() {
			defer local.Close()
			remote, err := dialer.Dial("tcp", dest)
			if err != nil {
				return
			}
			defer remote.Close()

			done := make(chan struct{}, 2)
			go func() {
				_, _ = io.Copy(remote, local)
				done <- struct{}{}
			}()
			go func() {
				_, _ = io.Copy(local, remote)
				done <- struct{}{}
			}()
			<-done
		}()
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)

	// UDP handler (primarily DNS resolution on port 53)
	udpFwd := udp.NewForwarder(s, func(r *udp.ForwarderRequest) bool {
		id := r.ID()
		if id.LocalPort != 53 {
			// Non-DNS UDP: drop or let QUIC/Hysteria handle via SOCKS if needed
			return false
		}

		var wq waiter.Queue
		connEP, err := r.CreateEndpoint(&wq)
		if err != nil {
			return true
		}
		conn := gonet.NewUDPConn(&wq, connEP)
		dest := fmt.Sprintf("%s:53", id.LocalAddress.String())

		go func() {
			defer conn.Close()
			buf := make([]byte, 1500)
			for {
				_ = conn.SetReadDeadline(time.Now().Add(8 * time.Second))
				n, err := conn.Read(buf)
				if err != nil || n == 0 {
					return
				}

				resp, err := dnsOverTCPThroughSOCKS5(dialer, dest, buf[:n])
				if err != nil {
					return
				}
				if _, err := conn.Write(resp); err != nil {
					return
				}
			}
		}()
		return true
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	// Device -> gVisor loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case pkt := <-toGo:
				if len(pkt) == 0 {
					continue
				}
				pktBuf := stack.NewPacketBuffer(stack.PacketBufferOptions{
					Payload: buffer.MakeWithData(append([]byte{}, pkt...)),
				})
				ep.InjectInbound(ipv4.ProtocolNumber, pktBuf)
				pktBuf.DecRef()
			}
		}
	}()

	// gVisor -> Device loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			p := ep.ReadContext(ctx)
			if p == nil {
				return
			}
			data := p.ToView().ToSlice()
			p.DecRef()

			select {
			case fromGo <- data:
			default:
			}
		}
	}()

	<-ctx.Done()
	ep.Close()
	s.Close()
}

func dnsOverTCPThroughSOCKS5(dialer proxy.Dialer, dest string, query []byte) ([]byte, error) {
	// Fallback to standard 1.1.1.1:53 if local dest is internal or loopback
	remoteDNS := dest
	host, _, _ := net.SplitHostPort(dest)
	if host == "127.0.0.1" || host == "198.18.0.1" || host == "0.0.0.0" {
		remoteDNS = "1.1.1.1:53"
	}

	c, err := dialer.Dial("tcp", remoteDNS)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(6 * time.Second))

	var lp [2]byte
	binary.BigEndian.PutUint16(lp[:], uint16(len(query)))
	if _, err := c.Write(append(lp[:], query...)); err != nil {
		return nil, err
	}

	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return nil, err
	}
	respLen := binary.BigEndian.Uint16(hdr)
	resp := make([]byte, respLen)
	if _, err := io.ReadFull(c, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

