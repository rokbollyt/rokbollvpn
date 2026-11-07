package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	vkturn "github.com/cacggghp/vk-turn-proxy/client"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signalChan
		log.Printf("Terminating...")
		cancel()
		select {
		case <-signalChan:
		case <-time.After(5 * time.Second):
		}
		log.Fatalf("Exit...")
	}()

	cfg := vkturn.Config{}
	flag.StringVar(&cfg.TurnHost, "turn", "", "override TURN server ip")
	flag.StringVar(&cfg.TurnPort, "port", "", "override TURN port")
	flag.StringVar(&cfg.Listen, "listen", "127.0.0.1:9000", "listen on ip:port")
	flag.StringVar(&cfg.VKLink, "vk-link", "", "VK calls invite link \"https://vk.com/call/join/...\"")
	flag.StringVar(&cfg.YandexLink, "yandex-link", "", "Yandex telemost invite link \"https://telemost.yandex.ru/j/...\"")
	flag.StringVar(&cfg.MaxLink, "max-link", "", "MAX call link \"https://max.ru/joincall/...\" or bare token")
	flag.StringVar(&cfg.MaxCallToken, "max-token", "", "MAX call auth token (localStorage __oneme_calls_auth_token)")
	flag.StringVar(&cfg.MaxDeviceID, "max-device-id", "", "MAX device id (localStorage __oneme_device_id, optional)")
	maxSessionFile := flag.String("max-session", "", "path to MAX session .txt (JSON dump from web.max.ru console)")
	flag.StringVar(&cfg.Peer, "peer", "", "peer server address (host:port)")
	flag.IntVar(&cfg.Streams, "n", 0, "connections to TURN (default 10 for VK, 1 for Yandex)")
	flag.BoolVar(&cfg.UDP, "udp", false, "connect to TURN with UDP")
	flag.BoolVar(&cfg.NoDTLS, "no-dtls", false, "connect without obfuscation. DO NOT USE")
	flag.BoolVar(&cfg.VLESSMode, "vless", false, "VLESS mode: forward TCP connections (for VLESS) instead of UDP packets")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug logging")
	flag.BoolVar(&cfg.ManualCaptcha, "manual-captcha", false, "skip auto captcha solving, use manual mode immediately")
	flag.Parse()

	// A MAX session file supplies callToken/deviceId; -max-link still names the call.
	if *maxSessionFile != "" {
		mc, err := vkturn.LoadMaxSessionFile(*maxSessionFile, cfg.MaxLink)
		if err != nil {
			log.Fatalf("failed to load MAX session: %v", err)
		}
		cfg.MaxCallToken = mc.CallToken
		if cfg.MaxDeviceID == "" {
			cfg.MaxDeviceID = mc.DeviceID
		}
		if cfg.MaxLink == "" {
			cfg.MaxLink = mc.Link
		}
		// Ensure the MAX code path is selected even if only the session was given.
		if cfg.MaxLink == "" {
			log.Fatalf("MAX session provided but -max-link is missing")
		}
	}

	if err := vkturn.Run(ctx, cfg); err != nil && ctx.Err() == nil {
		log.Fatalf("vk-turn client failed: %v", err)
	}
}
