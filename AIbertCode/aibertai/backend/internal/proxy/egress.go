package proxy

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"aibertai.local/backend/internal/envconfig"
)

// 模型地址由用户自建模型自己填写，后端会带着服务端身份去请求它。
// 不做校验时，任何登录用户都能把它指向内网或云厂商元数据地址，
// 拿后端当跳板。默认放行私有网段（部署在机房里的模型就是这样访问的），
// 回环、链路本地、云元数据这些地址一律拒绝。
const defaultUpstreamCIDRs = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

type egressDeniedError struct{ host string }

func (e *egressDeniedError) Error() string {
	return fmt.Sprintf("模型地址 %s 指向内网保留地址，已拒绝连接", e.host)
}

func (e *egressDeniedError) DiagnosticCode() string { return "upstream_address_denied" }

func upstreamCIDRs() string {
	cidrs := strings.TrimSpace(envconfig.Getenv("AIBERTAI_MODEL_ALLOWED_CIDRS"))
	if cidrs == "" {
		return defaultUpstreamCIDRs
	}
	return cidrs
}

func parseUpstreamCIDRs(raw string) []netip.Prefix {
	var prefixes []netip.Prefix
	for value := range strings.SplitSeq(raw, ",") {
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(value)); err == nil {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

func allowedUpstreamIP(ip netip.Addr, allow []netip.Prefix) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.Zone() != "" {
		return false
	}
	// Ranges where cloud metadata services and host-local daemons live stay out
	// of reach even when an operator widens the allow list by mistake.
	for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "198.18.0.0/15", "169.254.0.0/16", "fe80::/10"} {
		if netip.MustParsePrefix(cidr).Contains(ip) {
			return false
		}
	}
	for _, prefix := range allow {
		if prefix.Contains(ip) {
			return true
		}
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback()
}

func dialUpstream(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	allow := parseUpstreamCIDRs(upstreamCIDRs())
	var dialErr error
	for _, ip := range ips {
		if !allowedUpstreamIP(ip, allow) {
			dialErr = &egressDeniedError{host: host}
			continue
		}
		conn, err := (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).
			DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	if dialErr == nil {
		dialErr = &egressDeniedError{host: host}
	}
	return nil, dialErr
}
