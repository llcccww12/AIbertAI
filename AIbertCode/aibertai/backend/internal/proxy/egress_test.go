package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"testing"
)

// The package's other tests talk to httptest servers on 127.0.0.1, which the
// egress guard rejects by default. Widening the allow list here keeps those
// tests on their real sockets; each test below sets the list it needs.
func TestMain(m *testing.M) {
	if os.Getenv("AIBERTAI_MODEL_ALLOWED_CIDRS") == "" {
		os.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", defaultUpstreamCIDRs+",127.0.0.0/8,::1/128")
	}
	os.Exit(m.Run())
}

func TestAllowedUpstreamIP(t *testing.T) {
	def := parseUpstreamCIDRs(defaultUpstreamCIDRs)
	loopbackAllowed := parseUpstreamCIDRs("127.0.0.0/8,::1/128")
	for _, tc := range []struct {
		name  string
		ip    string
		allow []netip.Prefix
		want  bool
	}{
		{name: "部署在机房里的模型", ip: "192.168.157.200", allow: def, want: true},
		{name: "十段内网", ip: "10.1.2.3", allow: def, want: true},
		{name: "172.16 内网", ip: "172.16.5.4", allow: def, want: true},
		{name: "公网", ip: "8.8.8.8", allow: def, want: true},
		{name: "公网 IPv6", ip: "2001:db8::1", allow: def, want: true},
		{name: "IPv6 内网", ip: "fc00::1", allow: def, want: true},
		{name: "回环", ip: "127.0.0.1", allow: def, want: false},
		{name: "回环别名", ip: "127.0.0.53", allow: def, want: false},
		{name: "IPv6 回环", ip: "::1", allow: def, want: false},
		{name: "v4 映射回环", ip: "::ffff:127.0.0.1", allow: def, want: false},
		{name: "云元数据", ip: "169.254.169.254", allow: def, want: false},
		{name: "链路本地 IPv6", ip: "fe80::1", allow: def, want: false},
		{name: "未指定", ip: "0.0.0.0", allow: def, want: false},
		{name: "运营商级 NAT", ip: "100.64.0.1", allow: def, want: false},
		{name: "基准测试网段", ip: "198.18.0.1", allow: def, want: false},
		{name: "组播", ip: "224.0.0.1", allow: def, want: false},
		{name: "无效地址", ip: "not-an-ip", allow: def, want: false},
		// 运维显式加白名单时，白名单优先；这属于运维的自主决定，不是用户能触碰的开关。
		{name: "白名单放行回环", ip: "127.0.0.1", allow: loopbackAllowed, want: true},
		{name: "白名单之外的公网仍然放行", ip: "1.1.1.1", allow: loopbackAllowed, want: true},
		{name: "白名单之外的内网被拒", ip: "10.1.2.3", allow: loopbackAllowed, want: false},
		{name: "白名单不能放行云元数据", ip: "169.254.169.254", allow: parseUpstreamCIDRs(defaultUpstreamCIDRs + ",169.254.0.0/16"), want: false},
		{name: "白名单不能放行运营商级 NAT", ip: "100.64.0.1", allow: parseUpstreamCIDRs("100.64.0.0/10"), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ip, err := netip.ParseAddr(tc.ip)
			if err != nil && tc.ip != "not-an-ip" {
				t.Fatalf("解析测试地址 %q: %v", tc.ip, err)
			}
			if got := allowedUpstreamIP(ip, tc.allow); got != tc.want {
				t.Fatalf("allowedUpstreamIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestUpstreamCIDRsFallsBackToDefault(t *testing.T) {
	t.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", "")
	if got := upstreamCIDRs(); got != defaultUpstreamCIDRs {
		t.Fatalf("未配置时应当使用默认网段，得到 %q", got)
	}
	t.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", "192.168.0.0/16")
	if got := upstreamCIDRs(); got != "192.168.0.0/16" {
		t.Fatalf("配置的网段应当覆盖默认值，得到 %q", got)
	}
}

func TestDialUpstreamBlocksLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", defaultUpstreamCIDRs)
	_, err := dialUpstream(context.Background(), "tcp", server.Listener.Addr().String())
	var denied *egressDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("回环地址应当被拒绝，得到 %v", err)
	}
	if code := denied.DiagnosticCode(); code != "upstream_address_denied" {
		t.Fatalf("诊断码 = %q", code)
	}

	t.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", defaultUpstreamCIDRs+",127.0.0.0/8")
	conn, err := dialUpstream(context.Background(), "tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("白名单放行后应当能连上：%v", err)
	}
	conn.Close()
}

func TestDialUpstreamBlocksLocalhostName(t *testing.T) {
	t.Setenv("AIBERTAI_MODEL_ALLOWED_CIDRS", defaultUpstreamCIDRs)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_, err = dialUpstream(context.Background(), "tcp", "localhost:"+port)
	var denied *egressDeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("解析到回环的主机名应当被拒绝，得到 %v", err)
	}
}
