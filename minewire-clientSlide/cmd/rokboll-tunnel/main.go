package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	vkturn "github.com/cacggghp/vk-turn-proxy/client"
	"github.com/dmitrymodder/minewire-cli/src/core"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	var args []string
	if strings.HasPrefix(cmd, "-") {
		// Implicit "run" command when invoked directly with flags
		args = os.Args[1:]
		cmd = "run"
	} else {
		args = os.Args[2:]
	}

	switch cmd {
	case "max-request-code":
		handleMaxRequestCode(args)
	case "max-verify-code":
		handleMaxVerifyCode(args)
	case "max-verify-password":
		handleMaxVerifyPassword(args)
	case "max-refresh-token":
		handleMaxRefreshToken(args)
	case "udp", "raw-udp":
		handleRawUDP(args)
	case "run":
		handleRunTunnel(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: rokboll-tunnel <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  run               Start tunnel (minewire, maxturn, vkturn, or udp)")
	fmt.Println("  udp               Direct UDP relay through MAX/VK TURN to any remote peer")
	fmt.Println("  max-request-code  Request SMS code / captcha for MAX account login")
	fmt.Println("  max-verify-code   Verify SMS code and retrieve long-term session")
	fmt.Println("  max-refresh-token Refresh call token using long-term login token")
}

func setupSignals(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Detect if parent process died (stdin closed)
	go func() {
		buf := make([]byte, 1)
		for {
			_, err := os.Stdin.Read(buf)
			if err != nil {
				cancel()
				return
			}
		}
	}()
}

func handleRawUDP(args []string) {
	fs := flag.NewFlagSet("udp", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:11590", "Local UDP listen address")
	peer := fs.String("peer", "", "Remote peer UDP address (host:port, e.g. OpenVPN/WireGuard/Hysteria2)")
	link := fs.String("link", "", "VK or MAX call link")
	loginTok := fs.String("login-token", "", "MAX login token")
	callTok := fs.String("call-token", "", "MAX call token")
	deviceID := fs.String("device-id", "68470aa6-34a4-49ad-bf2f-2a534624270b", "Device UUID")
	direct := fs.Bool("direct", false, "Direct UDP without MAX-TURN relay")

	// AmneziaWG obfuscation parameters
	awg := fs.Bool("awg", false, "Enable AmneziaWG header and junk obfuscation")
	jc := fs.Int("jc", 4, "AmneziaWG Junk packet count")
	jmin := fs.Int("jmin", 40, "AmneziaWG Junk packet min length")
	jmax := fs.Int("jmax", 70, "AmneziaWG Junk packet max length")
	s1 := fs.Int("s1", 0, "AmneziaWG Init packet junk prefix length")
	s2 := fs.Int("s2", 0, "AmneziaWG Response packet junk prefix length")
	h1 := fs.Uint("h1", 1, "AmneziaWG Init header")
	h2 := fs.Uint("h2", 2, "AmneziaWG Response header")
	h3 := fs.Uint("h3", 3, "AmneziaWG Cookie header")
	h4 := fs.Uint("h4", 4, "AmneziaWG Data header")
	fs.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignals(cancel)

	awgCfg := AWGConfig{
		Enabled: *awg,
		Jc:      *jc,
		Jmin:    *jmin,
		Jmax:    *jmax,
		S1:      *s1,
		S2:      *s2,
		H1:      uint32(*h1),
		H2:      uint32(*h2),
		H3:      uint32(*h3),
		H4:      uint32(*h4),
	}

	runRawUDP(ctx, *listen, *peer, *link, *loginTok, *callTok, *deviceID, *direct, awgCfg)
}

func handleMaxRequestCode(args []string) {
	fs := flag.NewFlagSet("max-request-code", flag.ExitOnError)
	phone := fs.String("phone", "", "Phone number in international format (+7...)")
	captcha := fs.String("captcha", "", "Captcha token if required")
	deviceID := fs.String("device-id", "", "Device UUID")
	fs.Parse(args)

	captchaURL, trackToken, codeLen, err := vkturn.RequestMaxPhoneCode(*phone, *captcha, *deviceID)
	resp := map[string]interface{}{}
	if err != nil {
		resp["error"] = err.Error()
	} else {
		resp["captcha_url"] = captchaURL
		resp["track_token"] = trackToken
		resp["code_length"] = codeLen
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func handleMaxVerifyCode(args []string) {
	fs := flag.NewFlagSet("max-verify-code", flag.ExitOnError)
	trackToken := fs.String("track", "", "Track token from request-code")
	code := fs.String("code", "", "SMS verification code")
	deviceID := fs.String("device-id", "", "Device UUID")
	fs.Parse(args)

	loginTok, callTok, err := vkturn.VerifyMaxPhoneCode(*trackToken, *code, *deviceID)
	resp := map[string]interface{}{}
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "PASSWORD_CHALLENGE:") {
			parts := strings.SplitN(errStr, ":", 3)
			resp["password_required"] = true
			if len(parts) >= 2 {
				resp["track_id"] = parts[1]
			}
			if len(parts) >= 3 {
				resp["hint"] = parts[2]
			}
		} else {
			resp["error"] = errStr
		}
	} else {
		resp["login_token"] = loginTok
		resp["call_token"] = callTok
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func handleMaxVerifyPassword(args []string) {
	fs := flag.NewFlagSet("max-verify-password", flag.ExitOnError)
	trackID := fs.String("track", "", "Track ID from password challenge")
	password := fs.String("password", "", "Cloud 2FA password")
	deviceID := fs.String("device-id", "", "Device UUID")
	fs.Parse(args)

	loginTok, callTok, err := vkturn.VerifyMaxPassword(*trackID, *password, *deviceID)
	resp := map[string]interface{}{}
	if err != nil {
		resp["error"] = err.Error()
	} else {
		resp["login_token"] = loginTok
		resp["call_token"] = callTok
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func handleMaxRefreshToken(args []string) {
	fs := flag.NewFlagSet("max-refresh-token", flag.ExitOnError)
	loginTok := fs.String("login-token", "", "Long-term login token")
	deviceID := fs.String("device-id", "", "Device UUID")
	fs.Parse(args)

	callTok, lifetime, err := vkturn.FetchMaxCallToken(*loginTok, *deviceID)
	resp := map[string]interface{}{}
	if err != nil {
		resp["error"] = err.Error()
	} else {
		resp["call_token"] = callTok
		resp["lifetime"] = lifetime
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func handleRunTunnel(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	kind := fs.String("kind", "maxturn", "Tunnel kind: minewire, maxturn, or vkturn")
	listen := fs.String("listen", "127.0.0.1:10808", "Local SOCKS5 listen address")
	peer := fs.String("peer", "150.251.143.214:56000", "Exit peer host:port for TURN")
	server := fs.String("server", "150.251.143.214:25565", "Minewire server host:port")
	password := fs.String("password", "dsadk1os01jds0k2-", "Minewire password")
	mode := fs.String("mode", "fast", "Minewire mode: fast or realistic")
	link := fs.String("link", "", "VK call link or MAX call link")
	loginTok := fs.String("login-token", "", "MAX login token")
	callTok := fs.String("call-token", "", "MAX call token")
	deviceID := fs.String("device-id", "68470aa6-34a4-49ad-bf2f-2a534624270b", "Device UUID")
	streams := fs.Int("streams", 24, "Concurrent RTP streams for TURN")
	fs.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignals(cancel)

	switch strings.ToLower(*kind) {
	case "minewire":
		runMinewire(ctx, *listen, *server, *password, *mode)
	case "maxturn":
		runMaxTurn(ctx, *listen, *peer, *link, *loginTok, *callTok, *deviceID, *streams)
	case "vkturn":
		runVKTurn(ctx, *listen, *peer, *link, *streams)
	case "udp", "raw-udp", "maxturn-udp":
		runRawUDP(ctx, *listen, *peer, *link, *loginTok, *callTok, *deviceID, false, AWGConfig{})
	default:
		fmt.Fprintf(os.Stderr, "unknown kind: %s\n", *kind)
		os.Exit(1)
	}
}

func runMinewire(ctx context.Context, listen, server, password, mode string) {
	logger := core.NewLogger(core.LevelInfo)
	cfg := core.Config{
		LocalPort:     listen,
		ServerAddress: server,
		Password:      password,
		ProxyType:     "socks5",
		Mode:          mode,
	}
	engine := core.NewEngine(cfg, logger)
	if err := engine.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Minewire start error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Minewire SOCKS5 listening on %s (upstream %s)\n", listen, server)
	<-ctx.Done()
	engine.Stop()
}

const defaultMaxLoginToken = "An_Sx6HQ9HDirOT7IdcIx40SBT-5aeYIx9PGJtju5TlgYJ7QeK56xYPECGSE_t_2PU65CI0GOY1hO6Ioq9hpPlr17llWSpx64zuLw1_vLNKld5A0ktE-G6UunofIDxkAAN9_OCN8_qfZP6OqJGUBA1hPoKUwOEn0-Au5KMNa4KXMz1c6lRgJI0v16PXhbTeg3HAw_SdZ36kQDN5NPIzTFN3_MV41JjD1VUL8IVaSIfOLy3m_IPo6iAvmSC6Ne0TlqcOxw6eJAFD-OfeAo0NYr-231cBnapE6PK_VvXT_AUeqWXudq5hMVW7Y11Y5-ApVgBefrFLcRtcdrhySBPvJ9zhop5LPR7ogiIrL52Bymj0qPUDVu0f78wZp-ZUm431Uow7AW_cMu0KUXYXFsHBQDoLnP02UtgYiWx8ipsbTmTuAMoMcaNetybiDyuVmrkoqiuAxo1pJC7c6ANeW7G1MDyVcvQca2QOsWjNmyepyRbjBkA0DwYUKpUmcSJ3lS4HSUiZjyiLLI16SY41a7iP9C4dd9vPpNcvpCHZjX04dtVPCpqlfxXWXpjF2KX-Mkwwd_Oyl5PZh_d-E_n3u5TdgFhR4F3KmFTvaVB29mjBsD57ye5Vup2gn_ulQmyB1I9f3FuUh4GpV5pVgXavUC5sXDtchCQvonw8fjsMcX3Q2U7_kJhU0L2L_T7YIsoj_85qG0XwPA9o"

func runMaxTurn(ctx context.Context, listen, peer, link, loginTok, callTok, deviceID string, streams int) {
	if link == "" {
		link = "flBY3rVQonTIxS_G81y8qMRQCbhquUQgFi4kLGDtTNQ"
	}
	if deviceID == "" {
		deviceID = "68470aa6-34a4-49ad-bf2f-2a534624270b"
	}
	if loginTok == "" && (callTok == "" || strings.HasPrefix(callTok, "$OdI")) {
		loginTok = defaultMaxLoginToken
		callTok = ""
	}

	cfg := vkturn.Config{
		Listen:        listen,
		Peer:          peer,
		MaxLink:       link,
		MaxLoginToken: loginTok,
		MaxCallToken:  callTok,
		MaxDeviceID:   deviceID,
		DirectSocks:   true,
		VLESSMode:     false,
		UDP:           true,
		Streams:       streams,
	}

	fmt.Printf("MAX-TURN SOCKS5 starting on %s (peer %s, streams %d)...\n", listen, peer, streams)
	errCh := make(chan error, 1)
	go func() {
		errCh <- vkturn.Run(ctx, cfg)
	}()

	select {
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "MAX-TURN error: %v\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}

func runVKTurn(ctx context.Context, listen, peer, link string, streams int) {
	if link == "" {
		fmt.Fprintf(os.Stderr, "VK call link is required\n")
		os.Exit(1)
	}

	cfg := vkturn.Config{
		Listen:      listen,
		Peer:        peer,
		VKLink:      link,
		DirectSocks: true,
		VLESSMode:   false,
		UDP:         true,
		Streams:     streams,
	}

	fmt.Printf("VK-TURN SOCKS5 starting on %s (peer %s, streams %d)...\n", listen, peer, streams)
	errCh := make(chan error, 1)
	go func() {
		errCh <- vkturn.Run(ctx, cfg)
	}()

	select {
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "VK-TURN error: %v\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}

func runRawUDP(ctx context.Context, listen, peer, link, loginTok, callTok, deviceID string, direct bool, awgCfg AWGConfig) {
	if peer == "" {
		fmt.Fprintf(os.Stderr, "peer address (remote host:port) is required for UDP relay\n")
		os.Exit(1)
	}

	var translator *AWGTranslator
	if awgCfg.Enabled {
		translator = NewAWGTranslator(awgCfg)
		fmt.Printf("AmneziaWG Obfuscation active: H1=%d, H2=%d, H3=%d, H4=%d, Jc=%d, S1=%d, S2=%d\n",
			awgCfg.H1, awgCfg.H2, awgCfg.H3, awgCfg.H4, awgCfg.Jc, awgCfg.S1, awgCfg.S2)
	}

	if direct {
		// Direct UDP relay (no MAX-TURN) with optional Amnezia obfuscation
		fmt.Printf("Direct RAW UDP Relay starting on %s -> %s (no MAX-TURN)...\n", listen, peer)
		laddr, err := net.ResolveUDPAddr("udp", listen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve listen %s: %v\n", listen, err)
			os.Exit(1)
		}
		raddr, err := net.ResolveUDPAddr("udp", peer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve peer %s: %v\n", peer, err)
			os.Exit(1)
		}
		lconn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "listen %s: %v\n", listen, err)
			os.Exit(1)
		}
		defer lconn.Close()
		rconn, err := net.DialUDP("udp", nil, raddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dial peer %s: %v\n", peer, err)
			os.Exit(1)
		}
		defer rconn.Close()

		var clientAddr net.Addr
		go func() {
			buf := make([]byte, 65536)
			for {
				n, addr, err := lconn.ReadFrom(buf)
				if err != nil {
					return
				}
				clientAddr = addr
				pkt := buf[:n]
				if translator != nil {
					junks, out := translator.ObfuscateClientPacket(pkt)
					for _, junk := range junks {
						_, _ = rconn.Write(junk)
					}
					if len(out) == 0 {
						continue
					}
					pkt = out
				}
				_, _ = rconn.Write(pkt)
			}
		}()

		go func() {
			buf := make([]byte, 65536)
			for {
				n, err := rconn.Read(buf)
				if err != nil {
					return
				}
				pkt := buf[:n]
				if translator != nil {
					pkt = translator.DeobfuscateServerPacket(pkt)
					if len(pkt) == 0 {
						continue
					}
				}
				if clientAddr != nil {
					_, _ = lconn.WriteTo(pkt, clientAddr)
				}
			}
		}()

		<-ctx.Done()
		return
	}

	if link == "" {
		link = "flBY3rVQonTIxS_G81y8qMRQCbhquUQgFi4kLGDtTNQ"
	}
	if deviceID == "" {
		deviceID = "68470aa6-34a4-49ad-bf2f-2a534624270b"
	}
	if loginTok == "" && (callTok == "" || strings.HasPrefix(callTok, "$OdI")) {
		loginTok = defaultMaxLoginToken
		callTok = ""
	}

	cfg := vkturn.Config{
		Listen:        listen,
		Peer:          peer,
		MaxLink:       link,
		MaxLoginToken: loginTok,
		MaxCallToken:  callTok,
		MaxDeviceID:   deviceID,
		UDP:           true,
	}

	if translator != nil {
		cfg.TransformOutbound = translator.ObfuscateClientPacket
		cfg.TransformInbound = translator.DeobfuscateServerPacket
	}

	if strings.Contains(link, "vk.me") || strings.Contains(link, "vk.com") {
		cfg.VKLink = link
		cfg.MaxLink = ""
	}

	fmt.Printf("MAX/VK-TURN RAW UDP Relay starting on %s -> %s...\n", listen, peer)
	errCh := make(chan error, 1)
	go func() {
		errCh <- vkturn.RunRawUDPRelay(ctx, cfg)
	}()

	select {
	case err := <-errCh:
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "UDP Relay error: %v\n", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}

// Ensure net is used
var _ = net.Dial
var _ = time.Second
