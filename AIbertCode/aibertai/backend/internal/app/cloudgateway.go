package app

import (
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strings"
	"time"
)

func cloudImageRegistryURL(registry, listener, public string) (string, error) {
	if registry == "" {
		return public, nil
	}
	address, err := netip.ParseAddrPort(registry)
	if err != nil || address.Addr() != netip.AddrFrom4([4]byte{127, 0, 0, 1}) || address.Port() == 0 || registry != listener {
		return "", errors.New("云电脑镜像仓库必须使用同机后台的回环监听地址")
	}
	return "http://" + address.String(), nil
}

func cloudGateway(bind, origin string, backend http.Handler) (*http.Server, error) {
	address, err := netip.ParseAddrPort(bind)
	if err != nil || !address.Addr().Is4() || !address.Addr().IsPrivate() || address.Port() == 0 {
		return nil, errors.New("云电脑网关必须绑定到指定的私有 IPv4 地址和端口")
	}
	public, err := url.Parse(origin)
	if err != nil || public.Scheme != "http" || public.Host == "" || public.User != nil ||
		(public.Path != "" && public.Path != "/") || public.RawQuery != "" || public.Fragment != "" {
		return nil, errors.New("当前云电脑内网网关要求 HTTP 后端源地址")
	}
	return &http.Server{
		Addr:              address.String(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != public.Host || !cloudGatewayPath(r) {
				http.NotFound(w, r)
				return
			}
			// Cloud runtimes use device grants, never a browser's admin session.
			r.Header.Del("Cookie")
			backend.ServeHTTP(w, r)
		}),
	}, nil
}

func cloudGatewayPath(r *http.Request) bool {
	p := r.URL.Path
	if r.URL.IsAbs() || r.URL.RawPath != "" || p != path.Clean(p) || strings.Contains(p, "\\") {
		return false
	}
	if r.Method == http.MethodConnect || r.Method == http.MethodTrace {
		return false
	}
	switch p {
	case "/.well-known/oauth-authorization-server":
		return r.Method == http.MethodGet
	case "/oauth/device/authorization", "/oauth/token":
		return r.Method == http.MethodPost
	case "/api/v1/environments":
		return r.Method == http.MethodGet
	}
	// Employee and model routes still enforce the backend's device grant scopes.
	return strings.HasPrefix(p, "/api/v1/") && !strings.HasPrefix(p, "/api/v1/environments/") ||
		strings.HasPrefix(p, "/v1/")
}
