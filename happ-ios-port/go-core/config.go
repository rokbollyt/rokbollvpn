package main

// RouterConfig is the JSON structure passed from Swift via RouterStart.
type RouterConfig struct {
	// Protocol type: "hysteria2" | "wireguard" | "amneziawg" | "openvpn"
	Protocol string `json:"protocol"`

	// Target server UDP peer address: host:port
	PeerAddr string `json:"peerAddr"`

	// If true, connect directly without MAX-TURN relay. If false, wrap over MAX-TURN.
	DirectMode bool `json:"directMode"`

	// MAX-TURN credentials
	MaxCallToken  string `json:"maxCallToken"`
	MaxLoginToken string `json:"maxLoginToken"`
	MaxDeviceID   string `json:"maxDeviceID"`
	MaxLink       string `json:"maxLink"`

	// Local ports
	Socks5Port int `json:"socks5Port"` // default 10808
	RelayPort  int `json:"relayPort"`  // default 10900

	// --- Hysteria 2 ---
	HysteriaURI string `json:"hysteriaURI"`
	HysteriaBin string `json:"hysteriaBin"`

	// --- WireGuard & AmneziaWG ---
	PrivateKey   string `json:"privateKey"`
	PublicKey    string `json:"publicKey"`
	PresharedKey string `json:"presharedKey"`
	ClientIP     string `json:"clientIP"`
	DNS          string `json:"dns"`
	MTU          int    `json:"mtu"`

	// AmneziaWG Obfuscation
	Jc   int `json:"jc"`
	Jmin int `json:"jmin"`
	Jmax int `json:"jmax"`
	S1   int `json:"s1"`
	S2   int `json:"s2"`
	H1   int `json:"h1"`
	H2   int `json:"h2"`
	H3   int `json:"h3"`
	H4   int `json:"h4"`

	// --- OpenVPN ---
	OpenVPNConfig string `json:"openvpnConfig"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

func defaultConfig() RouterConfig {
	return RouterConfig{
		Protocol:   "hysteria2",
		Socks5Port: 10808,
		RelayPort:  10900,
		MTU:        1420,
	}
}

