package identity

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
)

// 默认部署由内网 Nginx 转发，直连地址永远是代理自己。限流如果只看
// RemoteAddr，全站用户共用一个额度：一个人发够次数，其他人就发不了。
// 只有内网或本机代理转发过来的请求才采信 X-Real-IP；同名头出现多次时
// 不做猜测，直接用连接地址。
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		slog.WarnContext(r.Context(), "操作失败", "module", "identity", "operation", "clientIP", "error", err)
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !peer.IsPrivate() && !peer.IsLoopback() {
		return peer.String()
	}
	values := r.Header.Values("X-Real-IP")
	if len(values) != 1 {
		return peer.String()
	}
	forwarded, err := netip.ParseAddr(values[0])
	if err != nil {
		slog.WarnContext(r.Context(), "操作失败", "module", "identity", "operation", "clientIP", "error", err)
		return peer.String()
	}
	return forwarded.WithZone("").Unmap().String()
}
