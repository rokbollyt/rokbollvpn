package vkturn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
)

// region MAX (web.max.ru) Credentials Layer
//
// MAX video calls run on the OK/VK calls backend (calls.okcdn.ru/fb.do,
// ok.calls.sdk.js). Unlike VK's anonymous flow, MAX requires a per-account
// "call auth token" taken from a logged-in web.max.ru session
// (localStorage key __oneme_calls_auth_token). Because this uses a REAL
// personal account, this layer is deliberately conservative:
//   - a single conversation join is performed and reused;
//   - TURN credentials are cached aggressively and refreshed rarely;
//   - no browser automation is used (tls-client with a Chrome profile only).
//
// Flow (all POST form-encoded to https://calls.okcdn.ru/fb.do):
//  1. auth.anonymLogin { session_data:{device_id, client_version, client_type:"SDK_JS",
//       auth_token:<callToken>, version:3}, application_key } -> session_key
//  2. vchat.joinConversationByLink { joinLink:<token>, isVideo:false,
//       protocolVersion:5, capabilities:2F7F, session_key, application_key }
//       -> { endpoint(wss .../ws2?...conversationId=<id>...) }
//  3. vchat.getConversationParams { conversationId:<id>, session_key, application_key }
//       -> turn_server:{urls, username, credential}

const (
	maxAPIBase        = "https://calls.okcdn.ru/fb.do"
	maxApplicationKey = "CNHIJPLGDIHBABABA"
	maxClientVersion  = "26.8.10"
	// TURN creds live long; refresh well before any plausible expiry but rarely
	// to avoid looking like a reconnecting bot on a personal account.
	maxCredLifetime = 8 * time.Minute
)

// MaxConfig holds the MAX-specific inputs needed to obtain TURN credentials.
type MaxConfig struct {
	// LoginToken is the long-term session token from phone login (opcode 18 tokenAttrs.LOGIN.token).
	// If present and CallToken is expired or empty, it will be used to automatically fetch a fresh CallToken.
	LoginToken string
	// CallToken is localStorage["__oneme_calls_auth_token"] from web.max.ru.
	CallToken string
	// DeviceID is localStorage["__oneme_device_id"]; a stable UUID for the
	// session. If empty a random one is generated (less consistent, slightly
	// more suspicious - prefer passing the real device id).
	DeviceID string
	// Link is the MAX call link or bare token
	// (https://max.ru/joincall/<token> or just <token>).
	Link string
}

// maxCredState caches the derived MAX session across getCreds calls so we only
// join the conversation once and re-poll getConversationParams occasionally.
type maxCredState struct {
	mu             sync.Mutex
	client         tlsclient.HttpClient
	sessionKey     string
	conversationID string
	loginToken     string
	callToken      string
	link           string
	deviceIDValue  string
	// cached TURN creds
	turnUser  string
	turnPass  string
	turnAddr  string
	turnAddrs []string
	expiresAt time.Time
}

var maxTurnClusterServers = []string{
	"155.212.193.58:19302",
	"155.212.193.59:19302",
	"155.212.193.60:19302",
	"155.212.197.18:19302",
	"155.212.197.19:19302",
	"155.212.197.40:19302",
	"155.212.197.41:19302",
	"155.212.197.42:19302",
	"155.212.197.43:19302",
	"155.212.199.137:19302",
	"155.212.199.138:19302",
	"155.212.199.139:19302",
	"155.212.199.185:19302",
	"155.212.199.186:19302",
	"155.212.205.68:19302",
	"155.212.205.69:19302",
	"155.212.205.70:19302",
	"155.212.205.71:19302",
	"155.212.205.97:19302",
	"155.212.205.98:19302",
	"155.212.205.114:19302",
	"155.212.205.115:19302",
	"155.212.207.36:19302",
	"155.212.207.37:19302",
	"155.212.207.51:19302",
	"155.212.207.52:19302",
}

type maxCredPool struct {
	state *maxCredState
}

func newMaxCredPool(cfg MaxConfig) *maxCredPool {
	return &maxCredPool{
		state: newMaxCredState(cfg),
	}
}

func (p *maxCredPool) getCreds(ctx context.Context, _ string, streamID int) (string, string, string, error) {
	return p.state.getMaxCreds(ctx, "", streamID)
}

// newMaxCredState builds a single MAX credential state from MaxConfig.
func newMaxCredState(cfg MaxConfig) *maxCredState {
	return &maxCredState{
		loginToken:    cfg.LoginToken,
		callToken:     cfg.CallToken,
		link:          cfg.Link,
		deviceIDValue: cfg.DeviceID,
	}
}

func newMaxBrowserProfile() Profile {
	return Profile{
		UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
		SecChUa:         `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`,
		SecChUaMobile:   "?0",
		SecChUaPlatform: `"Windows"`,
	}
}

func (s *maxCredState) ensureClient() error {
	if s.client != nil {
		return nil
	}
	c, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(),
		tlsclient.WithTimeoutSeconds(20),
		tlsclient.WithClientProfile(profiles.Chrome_146),
		tlsclient.WithCookieJar(tlsclient.NewCookieJar()),
		tlsclient.WithDialer(getCustomNetDialer()),
		tlsclient.WithInsecureSkipVerify(),
	)
	if err != nil {
		return fmt.Errorf("init tls_client: %w", err)
	}
	s.client = c
	return nil
}

// maxDoRequest posts a form-encoded fb.do request and decodes the JSON body.
func (s *maxCredState) maxDoRequest(ctx context.Context, profile Profile, params map[string]string) (map[string]interface{}, error) {
	form := neturl.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	body := form.Encode()

	req, err := fhttp.NewRequestWithContext(ctx, "POST", maxAPIBase, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Host = "calls.okcdn.ru"
	applyBrowserProfileFhttp(req, profile)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://web.max.ru")
	req.Header.Set("Referer", "https://web.max.ru/")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode fb.do response: %w (body: %s)", err, truncate(string(raw), 300))
	}
	if code, ok := out["error_code"].(float64); ok {
		msg, _ := out["error_msg"].(string)
		return out, fmt.Errorf("MAX API error_code=%d: %s", int(code), msg)
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// MaxSession is the shape of the JSON blob copied from the web.max.ru console:
//   JSON.stringify({
//     callToken: localStorage.getItem('__oneme_calls_auth_token'),
//     auth:      localStorage.getItem('__oneme_auth'),
//     deviceId:  localStorage.getItem('__oneme_device_id'),
//   })
// The value pasted into the file may itself be a JSON string (i.e. the outer
// quotes are included) — LoadMaxSessionFile handles both.
type MaxSession struct {
	CallToken string `json:"callToken"`
	Auth      string `json:"auth"`
	DeviceID  string `json:"deviceId"`
}

// LoadMaxSessionFile reads a MAX session dump (see MaxSession) from a file and
// returns a MaxConfig ready for the client. The call link is provided
// separately because it is not part of the session dump.
func LoadMaxSessionFile(path, link string) (MaxConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MaxConfig{}, fmt.Errorf("read session file: %w", err)
	}
	s := strings.TrimSpace(string(raw))
	// The console prints the JSON wrapped in single quotes; strip them.
	s = strings.Trim(s, "'")
	// If the whole thing is a quoted JSON string, unquote it once.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var unq string
		if json.Unmarshal([]byte(s), &unq) == nil {
			s = unq
		}
	}
	var sess MaxSession
	if err := json.Unmarshal([]byte(s), &sess); err != nil {
		return MaxConfig{}, fmt.Errorf("parse session JSON: %w", err)
	}
	if sess.CallToken == "" {
		return MaxConfig{}, fmt.Errorf("session file has empty callToken (__oneme_calls_auth_token)")
	}
	return MaxConfig{
		CallToken: sess.CallToken,
		DeviceID:  sess.DeviceID,
		Link:      link,
	}, nil
}

// normalizeMaxLink extracts the bare call token from a MAX call link.
func normalizeMaxLink(link string) string {
	l := strings.TrimSpace(link)
	if idx := strings.Index(l, "joincall/"); idx != -1 {
		l = l[idx+len("joincall/"):]
	}
	if idx := strings.IndexAny(l, "/?#"); idx != -1 {
		l = l[:idx]
	}
	return l
}

// login performs auth.anonymLogin and stores the session key.
func (s *maxCredState) login(ctx context.Context, profile Profile) error {
	deviceID := s.deviceID()
	sessData, _ := json.Marshal(map[string]interface{}{
		"device_id":      deviceID,
		"client_version": maxClientVersion,
		"client_type":    "SDK_JS",
		"auth_token":     s.callToken,
		"version":        3,
	})
	resp, err := s.maxDoRequest(ctx, profile, map[string]string{
		"method":          "auth.anonymLogin",
		"format":          "JSON",
		"application_key": maxApplicationKey,
		"session_data":    string(sessData),
	})
	if err != nil {
		return fmt.Errorf("anonymLogin: %w", err)
	}
	sk, ok := resp["session_key"].(string)
	if !ok || sk == "" {
		return fmt.Errorf("anonymLogin: missing session_key in %v", resp)
	}
	s.sessionKey = sk
	return nil
}

var maxDeviceID string
var maxDeviceIDOnce sync.Once

func (s *maxCredState) deviceID() string {
	if s.deviceIDValue != "" {
		return s.deviceIDValue
	}
	maxDeviceIDOnce.Do(func() { maxDeviceID = uuid.New().String() })
	return maxDeviceID
}

// join performs vchat.joinConversationByLink and extracts the conversationId
// from the returned websocket endpoint URL.
func (s *maxCredState) join(ctx context.Context, profile Profile) error {
	token := normalizeMaxLink(s.link)
	resp, err := s.maxDoRequest(ctx, profile, map[string]string{
		"method":          "vchat.joinConversationByLink",
		"format":          "JSON",
		"application_key": maxApplicationKey,
		"session_key":     s.sessionKey,
		"joinLink":        token,
		// Присоединяемся как ВИДЕО-участник. Это принципиально для скорости:
		// при isVideo=false TURN-аллокация получает лимит аудиозвонка (порядка
		// 64 кбит/с — измерено ~8 КБ/с), а видеозвонку выделяется в разы
		// большая полоса.
		"isVideo":         "true",
		"protocolVersion": "5",
		"capabilities":    "2F7F",
	})
	if err != nil {
		return fmt.Errorf("joinConversationByLink: %w", err)
	}
	endpoint, ok := resp["endpoint"].(string)
	if !ok || endpoint == "" {
		return fmt.Errorf("joinConversationByLink: missing endpoint in %v", resp)
	}
	u, err := neturl.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse endpoint %q: %w", endpoint, err)
	}
	cid := u.Query().Get("conversationId")
	if cid == "" {
		return fmt.Errorf("joinConversationByLink: no conversationId in endpoint %q", endpoint)
	}
	s.conversationID = cid
	return nil
}

// fetchParams performs vchat.getConversationParams and parses TURN creds.
func (s *maxCredState) fetchParams(ctx context.Context, profile Profile) (string, string, []string, error) {
	resp, err := s.maxDoRequest(ctx, profile, map[string]string{
		"method":          "vchat.getConversationParams",
		"format":          "JSON",
		"application_key": maxApplicationKey,
		"session_key":     s.sessionKey,
		"conversationId":  s.conversationID,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("getConversationParams: %w", err)
	}
	ts, ok := resp["turn_server"].(map[string]interface{})
	if !ok {
		return "", "", nil, fmt.Errorf("getConversationParams: missing turn_server in %v", resp)
	}
	user, _ := ts["username"].(string)
	pass, _ := ts["credential"].(string)
	urlsRaw, ok := ts["urls"].([]interface{})
	if !ok || len(urlsRaw) == 0 {
		return "", "", nil, fmt.Errorf("getConversationParams: empty turn_server.urls")
	}
	if user == "" || pass == "" {
		return "", "", nil, fmt.Errorf("getConversationParams: missing username/credential")
	}

	var addrs []string
	for _, uRaw := range urlsRaw {
		if urlStr, ok := uRaw.(string); ok && urlStr != "" {
			clean := strings.Split(urlStr, "?")[0]
			addr := strings.TrimPrefix(strings.TrimPrefix(clean, "turn:"), "turns:")
			if addr != "" {
				addrs = append(addrs, addr)
			}
		}
	}
	if len(addrs) == 0 {
		return "", "", nil, fmt.Errorf("getConversationParams: no valid turn addresses found in %v", urlsRaw)
	}
	return user, pass, addrs, nil
}

// getMaxCreds is the getCredsFunc for MAX. It logs in and joins the
// conversation once, then serves cached TURN credentials, re-polling
// getConversationParams only after the cache expires.
func (s *maxCredState) getMaxCreds(ctx context.Context, _ string, streamID int) (string, string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.turnAddrs) > 0 && time.Now().Before(s.expiresAt) {
		selectedAddr := s.turnAddrs[streamID%len(s.turnAddrs)]
		return s.turnUser, s.turnPass, selectedAddr, nil
	}

	if err := s.ensureClient(); err != nil {
		return "", "", "", err
	}
	profile := newMaxBrowserProfile()

	// Establish the session + conversation once; refresh only creds afterwards.
	if s.callToken == "" && s.loginToken != "" {
		log.Printf("[STREAM %d] [MAX Auth] call_token is empty, fetching fresh one via login_token...", streamID)
		freshTok, _, ferr := FetchMaxCallToken(s.loginToken, s.deviceID())
		if ferr == nil && freshTok != "" {
			s.callToken = freshTok
		}
	}
	if s.sessionKey == "" {
		log.Printf("[STREAM %d] [MAX Auth] logging in...", streamID)
		if err := s.login(ctx, profile); err != nil {
			if s.loginToken != "" && (strings.Contains(err.Error(), "Token is outdated") || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "AUTH_LOGIN")) {
				log.Printf("[STREAM %d] [MAX Auth] call_token outdated (%v), refreshing automatically via login_token...", streamID, err)
				freshTok, _, ferr := FetchMaxCallToken(s.loginToken, s.deviceID())
				if ferr == nil && freshTok != "" {
					s.callToken = freshTok
					if lerr := s.login(ctx, profile); lerr != nil {
						return "", "", "", lerr
					}
				} else {
					return "", "", "", fmt.Errorf("login failed (%v) and refresh via login_token failed: %w", err, ferr)
				}
			} else {
				return "", "", "", err
			}
		}
	}
	if s.conversationID == "" {
		log.Printf("[STREAM %d] [MAX Auth] joining conversation...", streamID)
		if err := s.join(ctx, profile); err != nil {
			// A session may have expired; retry login once.
			if strings.Contains(err.Error(), "error_code=10") || strings.Contains(err.Error(), "SESSION") {
				s.sessionKey = ""
				if lerr := s.login(ctx, profile); lerr != nil {
					return "", "", "", lerr
				}
				if jerr := s.join(ctx, profile); jerr != nil {
					return "", "", "", jerr
				}
			} else {
				return "", "", "", err
			}
		}
		log.Printf("[STREAM %d] [MAX Auth] conversation=%s", streamID, s.conversationID)
	}

	user, pass, addrs, err := s.fetchParams(ctx, profile)
	if err != nil {
		// Session or conversation went stale: reset and let next call rebuild.
		if strings.Contains(err.Error(), "error_code=10") || strings.Contains(err.Error(), "error_code=103") {
			s.sessionKey = ""
			s.conversationID = ""
		}
		return "", "", "", err
	}

	s.turnUser, s.turnPass, s.turnAddrs = user, pass, addrs
	s.turnAddr = addrs[0]
	s.expiresAt = time.Now().Add(maxCredLifetime)
	selectedAddr := s.turnAddrs[streamID%len(s.turnAddrs)]
	log.Printf("[STREAM %d] [MAX Auth] TURN ready: %s (slot %d of %d servers, user=%s)", streamID, selectedAddr, streamID%len(s.turnAddrs), len(s.turnAddrs), user)
	return user, pass, selectedAddr, nil
}
