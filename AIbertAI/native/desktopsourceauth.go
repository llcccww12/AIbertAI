package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const sourceGrantFilename = "source-credentials.enc.json"
const sourceGrantScopes = "device:bind device:connect device:publish device:manage session:report session:read session:control files:read files:write resources:manage environment:read environment:manage"

var (
	errSourceLoginCancelled = errors.New("source_login_cancelled")
	errSourceGrantRejected  = errors.New("source_grant_rejected")
)

type sourceGrant struct {
	Auth    authorization
	Expires time.Time
}

type sourceGrantStore struct {
	root   *os.Root
	cipher cipher.AEAD
	aad    []byte
}

func openSourceGrantStore(profile, dataKey, server, user string) (*sourceGrantStore, error) {
	key, err := base64.StdEncoding.DecodeString(dataKey)
	defer clear(key)
	if err != nil || len(key) != 32 || !filepath.IsAbs(profile) || !validUUID(user) {
		return nil, errDesktopCredential
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	sealer, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(profile)
	if err != nil {
		return nil, err
	}
	return &sourceGrantStore{root: root, cipher: sealer, aad: []byte("aibertai/source-grant/v1\x00" + server + "\x00" + user)}, nil
}

type sealedSourceGrant struct {
	Version           int
	Nonce, Ciphertext []byte
}

func (s *sourceGrantStore) load() (sourceGrant, error) {
	var grant sourceGrant
	info, err := s.root.Lstat(sourceGrantFilename)
	if err != nil {
		return grant, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDesktopCredentials {
		return grant, errDesktopCredential
	}
	file, err := s.root.Open(sourceGrantFilename)
	if err != nil {
		return grant, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return grant, errDesktopCredential
	}
	var envelope sealedSourceGrant
	if json.NewDecoder(io.LimitReader(file, maxDesktopCredentials)).Decode(&envelope) != nil ||
		envelope.Version != 1 || len(envelope.Nonce) != s.cipher.NonceSize() {
		return grant, errDesktopCredential
	}
	plain, err := s.cipher.Open(nil, envelope.Nonce, envelope.Ciphertext, s.aad)
	if err != nil {
		return grant, errDesktopCredential
	}
	defer clear(plain)
	if json.Unmarshal(plain, &grant) != nil || !validSourceAuthorization(grant.Auth) || grant.Expires.IsZero() {
		return sourceGrant{}, errDesktopCredential
	}
	return grant, nil
}

func (s *sourceGrantStore) save(grant sourceGrant) error {
	encoded, err := json.Marshal(grant)
	if err != nil {
		return err
	}
	defer clear(encoded)
	nonce := make([]byte, s.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed, err := json.Marshal(sealedSourceGrant{Version: 1, Nonce: nonce, Ciphertext: s.cipher.Seal(nil, nonce, encoded, s.aad)})
	if err != nil {
		return err
	}
	if len(sealed) > maxDesktopCredentials {
		return errDesktopCredential
	}
	return writeSourceState(s.root, sourceGrantFilename, sealed)
}

func validSourceAuthorization(auth authorization) bool {
	scopes := strings.Fields(auth.Scope)
	return auth.Access != "" && auth.Refresh != "" && validUUID(auth.Grant) && auth.TTL > 0 &&
		auth.TTL <= 7*24*3600 && slices.Contains(scopes, "device:bind") && slices.Contains(scopes, "session:report")
}

func (s *sourceGrantStore) replace(auth authorization) (sourceGrant, error) {
	if !validSourceAuthorization(auth) {
		return sourceGrant{}, errDesktopCredential
	}
	grant := sourceGrant{Auth: auth, Expires: time.Now().Add(time.Duration(auth.TTL) * time.Second)}
	return grant, s.save(grant)
}

type desktopSourceAuth struct {
	mu          sync.Mutex
	client      *client
	store       *sourceGrantStore
	grant       sourceGrant
	oauthClient string
}

func (s *desktopSourceAuth) access(ctx context.Context) (authorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Until(s.grant.Expires) > 2*time.Minute {
		return s.grant.Auth, nil
	}
	client := s.oauthClient
	if client == "" {
		client = clientID
	}
	auth, err := s.client.refreshSourceGrant(ctx, client, s.grant.Auth.Refresh)
	if err != nil {
		return authorization{}, err
	}
	if auth.Grant != s.grant.Auth.Grant {
		return authorization{}, errDesktopCredential
	}
	grant, err := s.store.replace(auth)
	if err != nil {
		return authorization{}, err
	}
	s.grant = grant
	return grant.Auth, nil
}

func (c *client) refreshSourceGrant(ctx context.Context, oauthClient, refresh string) (authorization, error) {
	fields := url.Values{"client_id": {oauthClient}, "grant_type": {"refresh_token"}, "refresh_token": {refresh}}
	resp, err := c.request(ctx, http.MethodPost, "/oauth/token", "", "", "",
		"application/x-www-form-urlencoded", strings.NewReader(fields.Encode()), http.StatusBadRequest)
	if err != nil {
		return authorization{}, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 32<<10))
	if resp.StatusCode == http.StatusBadRequest {
		var failure struct{ Error string }
		if decoder.Decode(&failure) == nil && failure.Error == "invalid_grant" {
			return authorization{}, errSourceGrantRejected
		}
		return authorization{}, errors.New("source_grant_refresh_failed")
	}
	var auth authorization
	if err := decoder.Decode(&auth); err != nil {
		return authorization{}, err
	}
	return auth, nil
}

func desktopSourceLogin(ctx context.Context, server string, in io.Reader, out io.Writer) (err error) {
	if backendOrigin != "" && server != backendOrigin {
		return errDesktopCredential
	}
	c, err := newBackendClient(server, true)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	type command struct{ Type, ProfileDir, DataKey string }
	targets := make(chan command, 1)
	reader := bufio.NewScanner(in)
	var initial command
	if !reader.Scan() || json.Unmarshal(reader.Bytes(), &initial) != nil || initial.Type != "init" {
		return errDesktopCredential
	}
	go func() {
		var target command
		if reader.Scan() && json.Unmarshal(reader.Bytes(), &target) == nil && target.Type == "save" {
			targets <- target
		}
		close(targets)
	}()
	events := json.NewEncoder(out)
	// Until Electron identifies the profile the grant is held only in this
	// process. Canceling the sign-in closes stdin and abandons the grant.
	type loginResult struct {
		auth authorization
		user account
		err  error
	}
	results := make(chan loginResult, 1)
	go func() {
		auth, user, err := c.loginWithScopes(ctx, sourceGrantScopes, func(link string) error {
			return events.Encode(map[string]string{"event": "authorize", "url": link})
		})
		results <- loginResult{auth, user, err}
	}()
	var result loginResult
	select {
	case result = <-results:
	case <-targets:
		cancel()
		result = <-results
		result.err = errSourceLoginCancelled
	}
	saved := false
	if result.auth.Refresh != "" {
		defer func() {
			if !saved {
				cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
				defer stop()
				err = errors.Join(err, c.revoke(cleanup, result.auth))
			}
		}()
	}
	if result.err != nil {
		return result.err
	}
	if !validUUID(result.user.ID) || !validSourceAuthorization(result.auth) {
		return errDesktopCredential
	}
	if err := events.Encode(map[string]any{"event": "user", "user": result.user}); err != nil {
		return err
	}
	select {
	case target, ok := <-targets:
		if !ok {
			return errSourceLoginCancelled
		}
		store, err := openSourceGrantStore(target.ProfileDir, target.DataKey, server, result.user.ID)
		if err != nil {
			return err
		}
		defer store.root.Close()
		previous, readErr := store.load()
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			return readErr
		}
		if _, err := store.replace(result.auth); err != nil {
			return err
		}
		saved = true
		if readErr == nil && previous.Auth.Grant != result.auth.Grant {
			cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
			_ = c.revoke(cleanup, previous.Auth)
			stop()
		}
		return events.Encode(map[string]any{"event": "done", "user": result.user})
	case <-ctx.Done():
		return ctx.Err()
	}
}

func desktopSourceLogout(ctx context.Context, profile, server, user string, in io.Reader) error {
	if backendOrigin != "" && server != backendOrigin {
		return errDesktopCredential
	}
	c, err := newBackendClient(server, true)
	if err != nil {
		return err
	}
	var command struct{ Type, DataKey string }
	if json.NewDecoder(io.LimitReader(in, maxDesktopCredentials)).Decode(&command) != nil || command.Type != "init" {
		return errDesktopCredential
	}
	store, err := openSourceGrantStore(profile, command.DataKey, server, user)
	if err != nil {
		return err
	}
	defer store.root.Close()
	grant, err := store.load()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		// 凭据已经读不出来（被截断、校验失败），本地直接清掉。
		// 这里返回错误会让退出登录也失败，用户既退不出去也换不了账号。
		return store.root.Remove(sourceGrantFilename)
	}
	// 撤销是尽力而为：断网或后端 5xx 时也必须完成本地登出，否则账号被卡死。
	_ = c.revoke(ctx, grant.Auth)
	return store.root.Remove(sourceGrantFilename)
}
