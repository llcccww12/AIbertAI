package model

import (
	"errors"
	"net/url"
	"strings"
)

// CompatibleEndpoint also accepts an already-complete endpoint URL.
func CompatibleEndpoint(raw, operation string) (*url.URL, error) {
	base, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" || base.User != nil || base.Fragment != "" {
		return nil, errors.New("模型地址无效")
	}
	path := strings.TrimRight(base.Path, "/")
	if !strings.HasSuffix(path, "/"+operation) {
		path += "/" + operation
	}
	base.Path = path
	base.RawPath = ""
	return base, nil
}
