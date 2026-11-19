package control

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Client) RegistryAuth(host, reference string) (string, error) {
	server, err := url.Parse(c.URL)
	if err != nil {
		return "", errors.New("宿主仓库身份无效")
	}
	return c.registryAuth(host, reference, server.Host)
}

// RegistryAuthForLoopback selects the explicitly configured same-host
// registry without changing the signed control-plane origin.
func (c *Client) RegistryAuthForLoopback(host, reference, address string) (string, error) {
	parsed, err := netip.ParseAddrPort(address)
	if err != nil || parsed.Addr() != netip.AddrFrom4([4]byte{127, 0, 0, 1}) || parsed.Port() == 0 {
		return "", errors.New("本机镜像仓库地址无效")
	}
	return c.registryAuth(host, reference, parsed.String())
}

func (c *Client) registryAuth(host, reference, address string) (string, error) {
	registry, repository, ok := strings.Cut(reference, "/")
	if host == "" || !ok || !strings.EqualFold(registry, address) || !strings.HasPrefix(repository, "environments/") {
		return "", errors.New("环境镜像必须从当前 server 仓库获取")
	}
	expires := time.Now().Add(30 * time.Minute).Unix()
	message := []byte("monkeyai-engine/registry/v1\n" + c.Server + "\n" + host + "\n" + strconv.FormatInt(expires, 10))
	token, err := json.Marshal(struct {
		ExpiresAt int64  `json:"expires_at"`
		Signature []byte `json:"signature"`
	}{expires, ed25519.Sign(c.Key, message)})
	if err != nil {
		return "", err
	}
	auth, err := json.Marshal(map[string]string{"username": host, "password": base64.RawURLEncoding.EncodeToString(token), "serveraddress": address})
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(auth), nil
}
