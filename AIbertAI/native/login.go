package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type authorization struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Grant   string `json:"grant_id"`
	TTL     int64  `json:"expires_in,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

func randomState() (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data[:]), nil
}

func (c *client) login(ctx context.Context, showAuthorization func(string) error) (authorization, account, error) {
	return c.loginWithScopes(ctx, "device:bind session:report", showAuthorization)
}

func (c *client) loginWithScopes(ctx context.Context, scopes string, showAuthorization func(string) error) (authorization, account, error) {
	var result authorization
	var user account
	var metadata struct {
		Issuer    string `json:"issuer"`
		Authorize string `json:"authorization_endpoint"`
		Token     string `json:"token_endpoint"`
	}
	if err := c.json(ctx, "GET", "/.well-known/oauth-authorization-server", "", "", nil, &metadata); err != nil {
		return result, user, err
	}
	if metadata.Issuer != c.server || metadata.Authorize != c.server+"/oauth/authorize" || metadata.Token != c.server+"/oauth/token" {
		return result, user, errors.New("OAuth 发现地址与当前登录的后端不一致")
	}
	state, err := randomState()
	if err != nil {
		return result, user, err
	}
	verifier, err := randomState()
	if err != nil {
		return result, user, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return result, user, err
	}
	defer listener.Close()
	callback := "http://" + listener.Addr().String() + "/oauth/callback"
	type callbackResult struct{ code, failure string }
	received := make(chan callbackResult, 1)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		MaxHeaderBytes:    8 << 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			query := r.URL.Query()
			if r.Method != http.MethodGet || r.Host != listener.Addr().String() || r.URL.Path != "/oauth/callback" ||
				len(query["state"]) != 1 || subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 {
				http.Error(w, "Invalid authorization response", http.StatusBadRequest)
				return
			}
			code, failure := query.Get("code"), query.Get("error")
			if (code == "") == (failure == "") || len(code) > 1024 {
				http.Error(w, "Invalid authorization response", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "可以关闭此页面，返回 AIbertAI。\n")
			select {
			case received <- callbackResult{code, failure}:
			default:
			}
		}),
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	go server.Serve(listener)
	challenge := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"client_id": {clientID}, "response_type": {"code"}, "redirect_uri": {callback},
		"state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"}, "scope": {scopes},
	}
	if err := showAuthorization(metadata.Authorize + "?" + params.Encode()); err != nil {
		return result, user, err
	}
	select {
	case <-ctx.Done():
		return result, user, ctx.Err()
	case response := <-received:
		if response.failure != "" {
			return result, user, errors.New("浏览器授权未完成")
		}
		err = c.form(ctx, "/oauth/token", url.Values{
			"client_id": {clientID}, "grant_type": {"authorization_code"},
			"redirect_uri": {callback}, "code_verifier": {verifier}, "code": {response.code},
		}, &result)
		if err != nil {
			return result, user, err
		}
	}
	if result.Access == "" || result.Refresh == "" || result.Grant == "" {
		return result, user, errors.New("私有后端没有签发完整的 OAuth 凭据")
	}
	err = c.json(ctx, "GET", "/api/v1/me", result.Access, "", nil, &user)
	if err == nil && user.ID == "" {
		err = errors.New("私有后端没有返回登录用户")
	}
	return result, user, err
}

func (c *client) revoke(ctx context.Context, auth authorization) error {
	return c.form(ctx, "/oauth/revoke", url.Values{
		"client_id": {clientID}, "token": {auth.Refresh}, "token_type_hint": {"refresh_token"},
	}, nil)
}
