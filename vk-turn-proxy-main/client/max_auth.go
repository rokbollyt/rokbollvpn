package vkturn

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pierrec/lz4/v4"
	"github.com/vmihailenco/msgpack/v5"
)

type MsgpackExt struct {
	Type int8
	Data string
}

var registerExtOnce sync.Once

func initMsgpackExt() {
	registerExtOnce.Do(func() {
		for i := -128; i <= 127; i++ {
			id := int8(i)
			if id == -1 {
				continue
			}
			currID := id
			msgpack.RegisterExtDecoder(currID, MsgpackExt{}, func(d *msgpack.Decoder, v reflect.Value, extLen int) error {
				b := make([]byte, extLen)
				if err := d.ReadFull(b); err != nil {
					return err
				}
				v.Set(reflect.ValueOf(MsgpackExt{Type: currID, Data: string(b)}))
				return nil
			})
		}
	})
}

func init() {
	initMsgpackExt()
}

func decodeMsgpackPayload(payload []byte) (map[string]interface{}, error) {
	initMsgpackExt()
	dec := msgpack.NewDecoder(bytes.NewReader(payload))
	dec.SetMapDecoder(func(d *msgpack.Decoder) (interface{}, error) {
		return d.DecodeUntypedMap()
	})
	var raw interface{}
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("msgpack unmarshal: %w", err)
	}
	return toStringMap(raw), nil
}

func toStringMap(v interface{}) map[string]interface{} {
	switch m := v.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{}, len(m))
		for k, val := range m {
			res[k] = normalizeValue(val)
		}
		return res
	case map[interface{}]interface{}:
		res := make(map[string]interface{}, len(m))
		for k, val := range m {
			res[fmt.Sprint(k)] = normalizeValue(val)
		}
		return res
	default:
		return map[string]interface{}{"value": v}
	}
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return toStringMap(val)
	case map[interface{}]interface{}:
		return toStringMap(val)
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, el := range val {
			res[i] = normalizeValue(el)
		}
		return res
	case MsgpackExt:
		return string(val.Data)
	default:
		return v
	}
}

func extractString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case MsgpackExt:
		return string(val.Data)
	default:
		return fmt.Sprint(v)
	}
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case uint:
		return int(val), true
	case uint32:
		return int(val), true
	case uint64:
		return int(val), true
	case float64:
		return int(val), true
	}
	return 0, false
}

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case uint64:
		return int64(val), true
	case float64:
		return int64(val), true
	}
	return 0, false
}

const (
	oneMeWSURL  = "wss://api.oneme.ru/websocket"
	oneMeOrigin = "https://web.max.ru"
)

type oneMePacket struct {
	ver     uint8
	cmd     uint8
	seq     int16
	opcode  int16
	comp    uint8
	payload []byte
}

type OneMeClient struct {
	conn     *websocket.Conn
	seq      int16
	deviceID string
	mu       sync.Mutex
}

func NewOneMeClient(deviceID string) (*OneMeClient, error) {
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	header := http.Header{}
	header.Set("Origin", oneMeOrigin)
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	conn, _, err := dialer.Dial(oneMeWSURL, header)
	if err != nil {
		return nil, fmt.Errorf("dial oneme ws: %w", err)
	}

	c := &OneMeClient{
		conn:     conn,
		seq:      1,
		deviceID: deviceID,
	}

	// Handshake: ClientHello (opcode 6)
	helloPayload := map[string]interface{}{
		"userAgent": map[string]interface{}{
			"deviceType":      "WEB",
			"pushDeviceType":  "WEBPUSH",
			"locale":          "ru",
			"deviceLocale":    "ru",
			"osVersion":       "Windows 10",
			"deviceName":      "Chrome",
			"headerUserAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
			"appVersion":      "26.8.10",
		},
		"deviceId": deviceID,
	}

	if _, err := c.sendAndRecv(6, helloPayload); err != nil {
		c.Close()
		return nil, fmt.Errorf("client hello failed: %w", err)
	}

	return c, nil
}

func (c *OneMeClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *OneMeClient) sendAndRecv(opcode int16, payload interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("websocket closed")
	}

	pBytes, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("msgpack marshal: %w", err)
	}

	curSeq := c.seq
	c.seq++

	pLen := len(pBytes)
	hdr := make([]byte, 10)
	hdr[0] = 10 // protocol ver 10
	hdr[1] = 0  // cmd: request
	binary.BigEndian.PutUint16(hdr[2:4], uint16(curSeq))
	binary.BigEndian.PutUint16(hdr[4:6], uint16(opcode))
	hdr[6] = 0 // no compression on request
	hdr[7] = byte(pLen >> 16)
	hdr[8] = byte(pLen >> 8)
	hdr[9] = byte(pLen)

	packet := append(hdr, pBytes...)
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := c.conn.WriteMessage(websocket.BinaryMessage, packet); err != nil {
		return nil, fmt.Errorf("ws write: %w", err)
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("ws read: %w", err)
		}
		if msgType != websocket.BinaryMessage || len(data) < 10 {
			continue
		}

		respSeq := int16(binary.BigEndian.Uint16(data[2:4]))
		respOpcode := int16(binary.BigEndian.Uint16(data[4:6]))
		comp := data[6]
		dataLen := int(data[7])<<16 | int(data[8])<<8 | int(data[9])
		if len(data) < 10+dataLen {
			return nil, fmt.Errorf("incomplete packet")
		}

		payloadData := data[10 : 10+dataLen]
		if comp > 0 {
			uncompBuf := make([]byte, dataLen*int(comp))
			n, err := lz4.UncompressBlock(payloadData, uncompBuf)
			if err != nil {
				return nil, fmt.Errorf("lz4 uncompress: %w", err)
			}
			payloadData = uncompBuf[:n]
		}

		res, err := decodeMsgpackPayload(payloadData)
		if err != nil {
			return nil, err
		}

		// Check for error in payload
		if errStr, ok := res["error"].(string); ok && errStr != "" {
			msg, _ := res["localizedMessage"].(string)
			if msg == "" {
				msg, _ = res["message"].(string)
			}
			return res, fmt.Errorf("MAX API [%s]: %s", errStr, msg)
		}

		if respSeq == curSeq || respOpcode == opcode {
			return res, nil
		}
	}
}

// RequestMaxPhoneCode initiates phone login:
// 1. If captchaToken is empty, queries Opcode 224 for the VK Captcha link.
// 2. If captchaToken is provided (or if no captcha required), sends Opcode 17 to request SMS code.
func RequestMaxPhoneCode(phone, captchaToken, deviceID string) (captchaURL, authTrackToken string, codeLength int, err error) {
	client, err := NewOneMeClient(deviceID)
	if err != nil {
		return "", "", 0, err
	}
	defer client.Close()

	if captchaToken == "" {
		// Opcode 224: query captcha requirement
		res, err := client.sendAndRecv(224, map[string]interface{}{
			"source":     "auth",
			"identifier": phone,
		})
		if err == nil {
			if link, ok := res["link"].(string); ok && link != "" {
				return link, "", 0, nil
			}
		}
	}

	// Opcode 17: Request code with or without captchaToken
	req := map[string]interface{}{
		"phone":    phone,
		"type":     "START_AUTH",
		"language": "ru",
	}
	if captchaToken != "" {
		req["captchaToken"] = captchaToken
	}

	res, err := client.sendAndRecv(17, req)
	if err != nil {
		return "", "", 0, err
	}

	token := extractString(res["token"])
	if token == "" {
		token = extractString(res["trackToken"])
	}
	if token == "" {
		token = extractString(res["authTrackToken"])
	}
	if token == "" {
		token = extractString(res["track_token"])
	}

	cLen := 6
	if cl, ok := toInt(res["codeLength"]); ok && cl > 0 {
		cLen = cl
	}

	return "", token, cLen, nil
}

// VerifyMaxPhoneCode submits the SMS verification code (Opcode 18),
// retrieves the long-term login_token, and immediately fetches a fresh call_token (Opcodes 19 + 158).
func VerifyMaxPhoneCode(trackToken, smsCode, deviceID string) (loginToken, callToken string, err error) {
	client, err := NewOneMeClient(deviceID)
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	// Opcode 18: verify code
	res, err := client.sendAndRecv(18, map[string]interface{}{
		"token":         trackToken,
		"verifyCode":    smsCode,
		"authTokenType": "CHECK_CODE",
	})
	if err != nil {
		return "", "", fmt.Errorf("verify code failed: %w", err)
	}

	// Check if account has 2FA cloud password enabled
	if pwc, ok := res["passwordChallenge"].(map[string]interface{}); ok && pwc != nil {
		trackID := extractString(pwc["trackId"])
		hint := extractString(pwc["hint"])
		return "", "", fmt.Errorf("PASSWORD_CHALLENGE:%s:%s", trackID, hint)
	}

	var loginTok string
	if tokenAttrs, ok := res["tokenAttrs"].(map[string]interface{}); ok {
		if loginObj, ok := tokenAttrs["LOGIN"].(map[string]interface{}); ok {
			loginTok = extractString(loginObj["token"])
		}
	}
	if loginTok == "" {
		loginTok = extractString(res["token"])
	}
	if loginTok == "" {
		loginTok = extractString(res["loginToken"])
	}
	if loginTok == "" {
		loginTok = extractString(res["login_token"])
	}
	if loginTok == "" {
		return "", "", fmt.Errorf("no login token in response: %v", res)
	}

	// Login and fetch call token
	cTok, _, err := client.fetchCallTokenInternal(loginTok)
	if err != nil {
		return loginTok, "", nil // login successful, call token can be retrieved later
	}

	return loginTok, cTok, nil
}

// VerifyMaxPassword submits the 2FA cloud password (Opcode 115) following a passwordChallenge.
func VerifyMaxPassword(trackID, password, deviceID string) (loginToken, callToken string, err error) {
	client, err := NewOneMeClient(deviceID)
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	res, err := client.sendAndRecv(115, map[string]interface{}{
		"trackId":  trackID,
		"password": password,
	})
	if err != nil {
		return "", "", fmt.Errorf("verify password failed: %w", err)
	}

	var loginTok string
	if tokenAttrs, ok := res["tokenAttrs"].(map[string]interface{}); ok {
		if loginObj, ok := tokenAttrs["LOGIN"].(map[string]interface{}); ok {
			loginTok = extractString(loginObj["token"])
		}
	}
	if loginTok == "" {
		loginTok = extractString(res["token"])
	}
	if loginTok == "" {
		loginTok = extractString(res["loginToken"])
	}
	if loginTok == "" {
		loginTok = extractString(res["login_token"])
	}
	if loginTok == "" {
		return "", "", fmt.Errorf("no login token in 2fa response: %v", res)
	}

	cTok, _, err := client.fetchCallTokenInternal(loginTok)
	if err != nil {
		return loginTok, "", nil
	}

	return loginTok, cTok, nil
}

// FetchMaxCallToken uses a long-term login_token to obtain a fresh __oneme_calls_auth_token.
func FetchMaxCallToken(loginToken, deviceID string) (callToken string, lifetime int64, err error) {
	client, err := NewOneMeClient(deviceID)
	if err != nil {
		return "", 0, err
	}
	defer client.Close()

	return client.fetchCallTokenInternal(loginToken)
}

func (c *OneMeClient) fetchCallTokenInternal(loginToken string) (string, int64, error) {
	// Opcode 19: ChatSync / session login
	_, err := c.sendAndRecv(19, map[string]interface{}{
		"token":        loginToken,
		"interactive":  false,
		"chatsCount":   40,
		"chatsSync":    0,
		"contactsSync": 0,
		"presenceSync": 0,
	})
	if err != nil {
		return "", 0, fmt.Errorf("session login (op 19) failed: %w", err)
	}

	// Opcode 158: CallTokenRequest
	res, err := c.sendAndRecv(158, map[string]interface{}{})
	if err != nil {
		return "", 0, fmt.Errorf("call token request (op 158) failed: %w", err)
	}

	cTok := extractString(res["token"])
	if cTok == "" {
		cTok = extractString(res["callToken"])
	}
	if cTok == "" {
		cTok = extractString(res["call_token"])
	}
	if cTok == "" {
		return "", 0, fmt.Errorf("missing token in op 158 response: %v", res)
	}

	var lifetime int64
	if lf, ok := toInt64(res["token_lifetime_ts"]); ok {
		lifetime = lf
	} else if lf, ok := toInt64(res["lifetime"]); ok {
		lifetime = lf
	}

	return cTok, lifetime, nil
}

var (
	globalCallTokenExpiry atomic.Int64
)
