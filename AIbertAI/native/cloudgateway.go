package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
)

func newCloudBackendClient(origin, gateway string) (*client, error) {
	c, err := newBackendClient(origin, true)
	if err != nil || gateway == "" {
		return c, err
	}
	address, err := netip.ParseAddrPort(gateway)
	if err != nil || !address.Addr().Is4() || !address.Addr().IsPrivate() || address.Port() == 0 {
		return nil, errors.New("invalid_cloud_backend_gateway")
	}
	public, _ := url.Parse(c.server)
	if public.Scheme != "http" {
		return nil, errors.New("cloud_backend_gateway_requires_http")
	}
	port := public.Port()
	if port == "" {
		port = "80"
	}
	expected := net.JoinHostPort(public.Hostname(), port)
	transport := c.http.Transport.(*http.Transport)
	dial := transport.DialContext
	transport.DialContext = func(ctx context.Context, network, endpoint string) (net.Conn, error) {
		if endpoint != expected {
			return nil, errors.New("cloud_backend_gateway_unexpected_destination")
		}
		return dial(ctx, network, address.String())
	}
	return c, nil
}
