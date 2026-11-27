package identity

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPTrustsForwardedHeaderOnlyFromProxy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		remote  string
		headers []string
		want    string
	}{
		{name: "内网代理转发", remote: "127.0.0.1:5555", headers: []string{"203.0.113.9"}, want: "203.0.113.9"},
		{name: "内网代理转发 IPv6", remote: "192.168.1.5:5555", headers: []string{"2001:db8::7"}, want: "2001:db8::7"},
		{name: "公网直连不采信", remote: "203.0.113.9:5555", headers: []string{"198.51.100.4"}, want: "203.0.113.9"},
		{name: "多个转发头不采信", remote: "127.0.0.1:5555", headers: []string{"203.0.113.9", "198.51.100.4"}, want: "127.0.0.1"},
		{name: "没有转发头", remote: "10.0.0.8:5555", want: "10.0.0.8"},
		{name: "转发头不是地址", remote: "127.0.0.1:5555", headers: []string{"not-an-ip"}, want: "127.0.0.1"},
		{name: "v4 映射地址归一", remote: "[::ffff:10.0.0.8]:5555", want: "10.0.0.8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = tc.remote
			for _, value := range tc.headers {
				r.Header.Add("X-Real-IP", value)
			}
			if got := clientIP(r); got != tc.want {
				t.Fatalf("clientIP = %q, want %q", got, tc.want)
			}
		})
	}
}
