package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	vkturn "github.com/cacggghp/vk-turn-proxy/client"
)

// config mirrors the fields in config.yml placed next to the exe.
type config struct {
	MaxLink     string
	MaxToken    string
	MaxDeviceID string
	SessionFile string
	Peer        string
	Listen      string
	// UDP: подключаться к TURN-серверу по UDP вместо TCP. TCP к TURN даёт
	// «TCP поверх TCP» (наш KCP внутри TCP-соединения), из-за чего при малейших
	// потерях скорость падает почти до нуля. UDP обычно ощутимо быстрее.
	UDP bool
	// Streams: число параллельных TURN-сессий. Для личного аккаунта MAX
	// безопасно 1; больше — выше скорость, но выше и риск бана.
	Streams int
	// DirectSocks: прямой режим (сервер должен быть запущен с -direct-socks).
	// Экономит два round-trip'а на соединение, но требует обновлённого сервера.
	DirectSocks bool
}

// parseSimpleYAML parses a minimal "key: value" config (one pair per line,
// '#' comments, optional quotes). No external YAML dependency needed.
func parseSimpleYAML(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if c := strings.Index(val, " #"); c >= 0 {
			val = strings.TrimSpace(val[:c])
		}
		val = strings.Trim(val, "\"'")
		out[strings.ToLower(key)] = val
	}
	return out
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}

func loadConfig() (config, error) {
	dir := exeDir()
	var path string
	for _, name := range []string{"config.yml", "config.yaml", "config.txt"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		return config{}, fmt.Errorf("no config.yml/config.yaml/config.txt found next to the exe (%s)", dir)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	m := parseSimpleYAML(string(raw))
	cfg := config{
		MaxLink:     firstNonEmpty(m["max_link"], m["link"]),
		MaxToken:    firstNonEmpty(m["max_token"], m["token"], m["call_token"]),
		MaxDeviceID: firstNonEmpty(m["max_device_id"], m["device_id"]),
		SessionFile: m["session_file"],
		Peer:        firstNonEmpty(m["peer"], m["server"]),
		Listen:      firstNonEmpty(m["listen"], m["local"]),
		UDP:         firstNonEmpty(m["udp"], "true") == "true",
		DirectSocks: firstNonEmpty(m["direct_socks"], "false") == "true",
	}
	if n := firstNonEmpty(m["streams"], m["n"]); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			cfg.Streams = v
		}
	}
	if cfg.Streams <= 0 {
		cfg.Streams = 1
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:1080"
	}
	// Resolve a relative session file against the exe directory.
	if cfg.SessionFile != "" && !filepath.IsAbs(cfg.SessionFile) {
		cfg.SessionFile = filepath.Join(dir, cfg.SessionFile)
	}
	return cfg, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func main() {
	fmt.Println("=== MAX TURN proxy ===")

	cfg, err := loadConfig()
	if err != nil {
		fatal("config error: %v", err)
	}
	if cfg.MaxLink == "" {
		fatal("config: 'max_link' is required (https://max.ru/joincall/...)")
	}
	if cfg.Peer == "" {
		fatal("config: 'peer' is required (your VPS host:port, e.g. rokboll.xyz:56000)")
	}

	// Token can be given inline or via a session file dumped from the console.
	if cfg.SessionFile != "" {
		mc, err := vkturn.LoadMaxSessionFile(cfg.SessionFile, cfg.MaxLink)
		if err != nil {
			fatal("failed to load session file: %v", err)
		}
		if cfg.MaxToken == "" {
			cfg.MaxToken = mc.CallToken
		}
		if cfg.MaxDeviceID == "" {
			cfg.MaxDeviceID = mc.DeviceID
		}
	}
	if cfg.MaxToken == "" {
		fatal("config: 'max_token' (or a 'session_file') is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nshutting down...")
		cancel()
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}()

	run := vkturn.Config{
		Listen:       cfg.Listen,
		Peer:         cfg.Peer,
		MaxLink:      cfg.MaxLink,
		MaxCallToken: cfg.MaxToken,
		MaxDeviceID:  cfg.MaxDeviceID,
		// Прямой режим быстрее на два round-trip'а, но требует сервера с флагом
		// -direct-socks. Если он выключен — работаем в обычном VLESS-режиме.
		DirectSocks: cfg.DirectSocks,
		VLESSMode:   !cfg.DirectSocks,
		UDP:         cfg.UDP,
		Streams:     cfg.Streams,
	}

	host, port := splitHostPort(cfg.Listen)
	fmt.Printf("call link : %s\n", cfg.MaxLink)
	fmt.Printf("exit node : %s\n", cfg.Peer)
	fmt.Println("connecting through MAX TURN, please wait ~10-20s...")
	fmt.Println(strings.Repeat("-", 52))
	fmt.Println("  Firefox → Settings → Network Settings → Manual proxy")
	fmt.Printf("  SOCKS Host: %s   Port: %s   (SOCKS v5)\n", host, port)
	fmt.Println("  ✓ Proxy DNS when using SOCKS v5")
	fmt.Println(strings.Repeat("-", 52))
	fmt.Println("Keep this window open. Press Ctrl+C to stop.")

	if err := vkturn.Run(ctx, run); err != nil && ctx.Err() == nil {
		fatal("proxy failed: %v", err)
	}
}

func splitHostPort(addr string) (string, string) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return addr, ""
	}
	return addr[:i], addr[i+1:]
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	fmt.Fprintln(os.Stderr, "\nPress Enter to exit...")
	fmt.Scanln()
	os.Exit(1)
}
