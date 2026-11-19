package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func newUUID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	data[6] = data[6]&0x0f | 0x40
	return formatUUID(data), nil
}

func formatUUID(data [16]byte) string {
	data[8] = data[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", data[:4], data[4:6], data[6:8], data[8:10], data[10:])
}

func newRelayDevice(name string) (relayDevice, error) {
	machine, err := newUUID()
	if err != nil {
		return relayDevice{}, err
	}
	boot, err := newUUID()
	if err != nil {
		return relayDevice{}, err
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return relayDevice{}, err
	}
	return relayDevice{machine: machine, boot: boot, name: name, public: public, private: private}, nil
}

func (c *client) bindTestDevice(ctx context.Context, auth authorization, user account) (string, error) {
	device, err := newRelayDevice("AIbertAI Runtime Acceptance")
	if err != nil {
		return "", err
	}
	conn, _, err := c.relayHandshake(ctx, auth, user, "", device, "bind_identity", "control", 0)
	if err != nil {
		return "", err
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return device.machine, nil
}

type relayDevice struct {
	machine, boot, name string
	public              ed25519.PublicKey
	private             ed25519.PrivateKey
	restoreRevoked      bool
	deploymentMode      string
}

type relayReady struct {
	Type                string `json:"type"`
	Code                string `json:"code"`
	ServerID            string `json:"server_id"`
	MachineID           string `json:"machine_id"`
	RegistrationVersion int64  `json:"registration_version"`
	GrantVersion        int64  `json:"grant_version"`
	Epoch               int64  `json:"lease_epoch"`
	Bound               bool   `json:"bound"`
}

func (c *client) relayHandshake(ctx context.Context, auth authorization, user account, serverID string, device relayDevice, intent, channel string, epoch int64) (*websocket.Conn, relayReady, error) {
	var ready relayReady
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	publicKey := base64.RawURLEncoding.EncodeToString(device.public)
	supported := []int{1}
	mode := device.deploymentMode
	if mode == "" {
		mode = "desktop_managed"
	}
	hello := map[string]any{
		"type": "hello", "tunnel_version": 1, "channel_kind": channel,
		"intent": intent, "machine_id": device.machine, "execution_public_key": publicKey,
		"boot_id": device.boot, "display_name": device.name, "platform": platform, "arch": runtime.GOARCH,
		"deployment_mode": mode, "lease_epoch": epoch, "protocols": map[string][]int{"acp": supported, "runtime_api": supported},
		"restore_revoked": device.restoreRevoked,
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	endpoint := "ws" + strings.TrimPrefix(c.server, "http") + "/api/v1/endpoint-bridge?channel_kind=" + channel + "&tunnel_version=1"
	conn, resp, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{
		HTTPClient: c.http,
		HTTPHeader: http.Header{"Authorization": {"Bearer " + auth.Access}},
	})
	if err != nil {
		if resp != nil {
			return nil, ready, fmt.Errorf("设备证明连接失败，HTTP %d", resp.StatusCode)
		}
		return nil, ready, fmt.Errorf("设备证明连接失败: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = conn.CloseNow()
		}
	}()
	conn.SetReadLimit(64 << 10)
	if err := wsjson.Write(ctx, conn, hello); err != nil {
		return nil, ready, err
	}
	var challenge struct {
		Type                string    `json:"type"`
		Code                string    `json:"code"`
		ServerID            string    `json:"server_id"`
		OwnerUserID         string    `json:"owner_user_id"`
		ChallengeID         string    `json:"challenge_id"`
		Nonce               string    `json:"nonce"`
		TTLMS               int64     `json:"ttl_ms"`
		ServerTime          time.Time `json:"server_time"`
		GrantID             string    `json:"grant_id"`
		RegistrationVersion int64     `json:"registration_version"`
	}
	if err := wsjson.Read(ctx, conn, &challenge); err != nil {
		return nil, ready, err
	}
	if challenge.Type == "error" && knowledgeErrorCode.MatchString(challenge.Code) {
		return nil, ready, errors.New(challenge.Code)
	}
	if challenge.Type != "challenge" || challenge.OwnerUserID != user.ID || challenge.GrantID != auth.Grant ||
		challenge.ServerID == "" || (serverID != "" && challenge.ServerID != serverID) || challenge.ChallengeID == "" || challenge.Nonce == "" || challenge.TTLMS <= 0 {
		return nil, ready, errors.New("设备证明挑战与当前用户或授权不匹配")
	}
	payload, err := json.Marshal([]any{
		"monkeyai.endpoint.proof.v1",
		[]any{1, channel, intent, device.machine, publicKey, device.boot, device.name, platform, runtime.GOARCH, mode, epoch, device.restoreRevoked, supported, supported},
		[]any{challenge.ServerID, challenge.OwnerUserID, challenge.ChallengeID, challenge.Nonce, challenge.TTLMS,
			challenge.ServerTime, challenge.GrantID, challenge.RegistrationVersion},
	})
	if err != nil {
		return nil, ready, err
	}
	if err := wsjson.Write(ctx, conn, map[string]string{
		"type": "proof", "challenge_id": challenge.ChallengeID,
		"signature": base64.RawURLEncoding.EncodeToString(ed25519.Sign(device.private, payload)),
	}); err != nil {
		return nil, ready, err
	}
	if err := wsjson.Read(ctx, conn, &ready); err != nil {
		return nil, ready, err
	}
	if ready.Type == "error" && knowledgeErrorCode.MatchString(ready.Code) {
		return nil, ready, errors.New(ready.Code)
	}
	if ready.Type != "ready" || !ready.Bound || ready.ServerID != challenge.ServerID ||
		ready.MachineID != device.machine || ready.RegistrationVersion < 1 || (intent != "bind_identity" && ready.GrantVersion < 1) ||
		(intent == "bind_identity" && ready.Epoch != 0) || (intent == "publish" && ready.Epoch < 1) || (intent == "attach" && ready.Epoch != epoch) {
		return nil, ready, errors.New("私有后端未确认绑定验收设备")
	}
	ack := map[string]any{"type": "ready_ack", "registration_version": ready.RegistrationVersion}
	if ready.Epoch > 0 {
		ack["lease_epoch"] = ready.Epoch
	}
	if err := wsjson.Write(ctx, conn, ack); err != nil {
		return nil, ready, err
	}
	complete = true
	return conn, ready, nil
}
