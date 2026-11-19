package identity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOIDCDiscoveryAndUserInfo(t *testing.T) {
	var issuer string
	var userinfo string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(r.Context(), w, 200, providerMetadata{issuer + "/authorize", issuer + "/token", issuer + "/userinfo"})
		case "/token":
			if r.Method != "POST" || r.FormValue("client_id") != "client" || r.FormValue("client_secret") != "secret" || r.FormValue("code") != "code" {
				t.Error("交换令牌参数无效")
			}
			writeJSON(r.Context(), w, 200, map[string]string{"access_token": "upstream-token"})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer upstream-token" {
				t.Error("缺少上游令牌")
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(userinfo)); err != nil {
				t.Error(err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	issuer = upstream.URL
	s := NewService(nil, nil, "https://monkeyai.example")
	connection := OAuthConnection{Provider: "oidc", IssuerURL: issuer, ClientID: "client", ClientSecret: "secret"}
	for _, tc := range []struct {
		provider string
		scopes   []string
		want     string
	}{
		{"oidc", nil, "openid profile email"},
		{"oidc", []string{"profile"}, "profile"},
	} {
		connection.Provider, connection.Scopes = tc.provider, tc.scopes
		target, err := s.upstreamAuthorizeURL(t.Context(), connection, "state")
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := url.Parse(target)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Path != "/authorize" || parsed.Query().Get("scope") != tc.want || parsed.Query().Get("state") != "state" {
			t.Fatalf("授权地址无效: %s", target)
		}
	}
	for _, tc := range []struct {
		name     string
		provider string
		fields   OAuthConnection
		body     string
		want     upstreamProfile
		wantErr  string
	}{
		{
			name:     "oidc",
			provider: "oidc",
			body:     `{"sub":"1002","name":"OIDC 用户","preferred_username":"user","email":"user@example.com","picture":"https://example.com/oidc.png"}`,
			want:     upstreamProfile{Provider: "oidc", Issuer: issuer, Subject: "1002", Username: "user", Name: "OIDC 用户", Email: "user@example.com", AvatarURL: "https://example.com/oidc.png"},
		},
		{
			name:     "oidc_custom_fields",
			provider: "oidc",
			fields:   OAuthConnection{IDField: "id", UsernameField: "display_name", AvatarField: "avatar", EmailField: "mail"},
			body:     `{"sub":"other","id":9007199254740993,"name":"默认名称","display_name":"自定义名称","picture":"https://example.com/default.png","avatar":"https://example.com/custom.png","email":"default@example.com","mail":"custom@example.com"}`,
			want:     upstreamProfile{Provider: "oidc", Issuer: issuer, Subject: "9007199254740993", Username: "自定义名称", Name: "自定义名称", Email: "custom@example.com", AvatarURL: "https://example.com/custom.png"},
		},
		{
			name:     "oidc_missing_custom_id",
			provider: "oidc",
			fields:   OAuthConnection{IDField: "id"},
			body:     `{"sub":"other"}`,
			wantErr:  "上游用户缺少 subject",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connection.Provider, userinfo = tc.provider, tc.body
			connection.IDField, connection.UsernameField = tc.fields.IDField, tc.fields.UsernameField
			connection.AvatarField, connection.EmailField = tc.fields.AvatarField, tc.fields.EmailField
			profile, err := s.exchangeUpstream(t.Context(), connection, "code")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || profile != (upstreamProfile{}) {
					t.Fatalf("无效用户信息未被拒绝: profile=%+v err=%v", profile, err)
				}
				return
			}
			if err != nil || profile != tc.want {
				t.Fatalf("上游身份解析错误: profile=%+v want=%+v err=%v", profile, tc.want, err)
			}
		})
	}
}

func TestOAuthConnectionAutoRegistrationDefault(t *testing.T) {
	disabled := false
	if !((OAuthConnection{}).autoRegistrationEnabled()) {
		t.Fatal("未配置时 SSO 应默认开启自动注册")
	}
	if (OAuthConnection{AutoRegistrationEnabled: &disabled}).autoRegistrationEnabled() {
		t.Fatal("显式关闭时 SSO 不应自动注册")
	}
}

func TestOIDCEmailBinding(t *testing.T) {
	pool := emailDatabase(t)
	s := newTestService(pool, nil, "")
	for _, role := range []string{"user", "admin"} {
		t.Run(role, func(t *testing.T) {
			profile := upstreamProfile{Provider: "oidc", Issuer: "https://identity.example", Subject: role, Name: "OIDC 测试用户", Email: role + "@example.com"}
			owner, err := s.insertUser(t.Context(), "已有用户", strings.ToUpper(profile.Email), role, "")
			if err != nil {
				t.Fatal(err)
			}
			if role == "user" {
				if _, err := s.upsertIdentity(t.Context(), profile, true, false); !errors.Is(err, ErrAdminRoleRequired) {
					t.Fatalf("普通用户不应通过管理后台登录: %v", err)
				}
			}
			again, err := s.upsertIdentity(t.Context(), profile, role == "admin", false)
			if err != nil || again.ID != owner.ID || again.Role != role || again.Email != owner.Email {
				t.Fatalf("未复用同邮箱账号: user=%+v err=%v", again, err)
			}
			if _, err := s.updateUser(t.Context(), owner.ID, owner.Name, role, "disabled", ""); err != nil {
				t.Fatal(err)
			}
			if _, err := s.upsertIdentity(t.Context(), profile, false, false); !errors.Is(err, ErrUserDisabled) {
				t.Fatalf("停用用户不应通过邮箱关联登录: %v", err)
			}
		})
	}
}

func TestExchangeUpstreamDoesNotExposeTokenResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"access_token":"private-token","password":"private-password"}`)); err != nil {
			t.Error(err)
		}
	}))
	defer upstream.Close()
	s := NewService(nil, nil, "https://monkeyai.example")
	connection := OAuthConnection{Provider: "oidc", AuthorizationURL: upstream.URL, TokenURL: upstream.URL, UserInfoURL: upstream.URL}
	_, err := s.exchangeUpstream(t.Context(), connection, "code")
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), "private-") {
		t.Fatalf("上游敏感响应不得出现在错误中: %v", err)
	}
}

type upstreamTransport func(*http.Request) (*http.Response, error)

func (f upstreamTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpstreamFailureLogPreservesOriginalError(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	s := NewService(nil, nil, "https://monkeyai.example")
	connection := OAuthConnection{
		Provider: "oidc", AuthorizationURL: "https://example.com/authorize%zz?client_secret=private-authorize",
		TokenURL: "https://example.com/token?client_secret=private-token", UserInfoURL: "https://example.com/userinfo?access_token=private-userinfo",
	}
	_, err := s.upstreamAuthorizeURL(t.Context(), connection, "state")
	if err == nil || !strings.Contains(err.Error(), "private-authorize") {
		t.Fatalf("测试应覆盖包含凭据 URL 的解析错误: %v", err)
	}
	logUpstreamFailure(t.Context(), "构造上游授权地址", "connection-1", err)
	if !strings.Contains(output.String(), "invalid_upstream_url") || !strings.Contains(output.String(), "private-authorize") {
		t.Fatalf("授权地址错误日志泄露凭据: %s", output.String())
	}

	for _, stage := range []string{"token", "userinfo"} {
		t.Run(stage, func(t *testing.T) {
			output.Reset()
			s.client = &http.Client{Transport: upstreamTransport(func(r *http.Request) (*http.Response, error) {
				if stage == "userinfo" && r.URL.Path == "/token" {
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"private-access"}`)), Header: make(http.Header)}, nil
				}
				return nil, fmt.Errorf("transport failure: %s %s", r.URL.String(), r.Header.Get("Authorization"))
			})}
			_, err := s.exchangeUpstream(t.Context(), connection, "private-code")
			if err == nil || !strings.Contains(err.Error(), "private-") {
				t.Fatalf("测试应覆盖包含凭据的外部传输错误: %v", err)
			}
			logUpstreamFailure(t.Context(), "交换上游身份", "connection-1", err)
			if !strings.Contains(output.String(), "reason=request_failed") || !strings.Contains(output.String(), "private-") {
				t.Fatalf("上游传输错误日志泄露凭据: %s", output.String())
			}
		})
	}

	output.Reset()
	logUpstreamFailure(t.Context(), "交换上游身份", "connection-1", &upstreamHTTPError{operation: "交换上游令牌", status: http.StatusUnauthorized})
	if !strings.Contains(output.String(), "status=401") || strings.Contains(output.String(), "private-") {
		t.Fatalf("上游状态码日志不安全: %s", output.String())
	}
}

type failingCloseBody struct {
	io.Reader
	err error
}

func (body failingCloseBody) Close() error { return body.err }

func TestUpstreamResponseCloseLogsOriginalError(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	s := NewService(nil, nil, "https://monkeyai.example")
	s.client = &http.Client{Transport: upstreamTransport(func(r *http.Request) (*http.Response, error) {
		var content string
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			content = `{"authorization_endpoint":"https://example.com/authorize","token_endpoint":"https://example.com/token","userinfo_endpoint":"https://example.com/userinfo"}`
		case "/token":
			content = `{"access_token":"private-access-token"}`
		case "/userinfo":
			content = `{"sub":"user-1"}`
		default:
			t.Errorf("意外的上游请求: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: failingCloseBody{
			Reader: strings.NewReader(content), err: fmt.Errorf("关闭 %s?access_token=private-close-token 失败", r.URL.Path),
		}}, nil
	})}
	if _, err := s.providerURLs(t.Context(), OAuthConnection{Provider: "oidc", IssuerURL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	connection := OAuthConnection{Provider: "oidc", AuthorizationURL: "https://example.com/authorize", TokenURL: "https://example.com/token", UserInfoURL: "https://example.com/userinfo"}
	if _, err := s.exchangeUpstream(t.Context(), connection, "code"); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"read_oidc_metadata", "exchange_token", "read_userinfo"} {
		if !strings.Contains(output.String(), operation) {
			t.Errorf("缺少 %s 的关闭失败日志: %s", operation, output.String())
		}
	}
	if strings.Count(output.String(), "关闭上游响应失败") != 3 || !strings.Contains(output.String(), "private-close-token") {
		t.Fatalf("关闭失败日志缺失或泄露凭据: %s", output.String())
	}

	output.Reset()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	closeUpstreamBody(ctx, failingCloseBody{Reader: strings.NewReader(""), err: errors.New("private-canceled")}, "读取上游用户")
	if output.Len() != 0 {
		t.Fatalf("取消上下文不应记录关闭错误: %s", output.String())
	}
}
