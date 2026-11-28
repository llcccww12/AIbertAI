package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

type desktopStartupConnection struct {
	user     account
	auth     authorization
	model    selectedModelConfig
	serverID string
	device   relayDevice
	ready    relayReady
	key      invocationKey
}

func prepareDesktopStartup(ctx context.Context, source *desktopSourceAuth, dataKey, userID string) (desktopStartupConnection, error) {
	var result desktopStartupConnection
	c := source.client
	if err := withAuthRetry(ctx, source, func(ctx context.Context, auth authorization) error {
		return c.json(ctx, http.MethodGet, "/api/v1/me", auth.Access, "", nil, &result.user)
	}); err != nil {
		return result, err
	}
	if result.user.ID != userID {
		return result, errDesktopCredential
	}
	if err := withAuthRetry(ctx, source, func(ctx context.Context, auth authorization) error {
		var err error
		result.model, err = c.selectedModelConfig(ctx, auth)
		if err != nil {
			return err
		}
		_, err = c.employeeModelCatalog(ctx, auth)
		return err
	}); err != nil {
		return result, fmt.Errorf("desktop model catalog: %w", err)
	}
	if err := withAuthRetry(ctx, source, func(ctx context.Context, _ authorization) error {
		var err error
		result.serverID, _, err = c.relaySigningKeys(ctx)
		return err
	}); err != nil {
		return result, fmt.Errorf("desktop server identity: %w", err)
	}
	device, err := desktopRelayDevice(dataKey, c.server, result.serverID, userID)
	if err != nil {
		return result, err
	}
	result.device = device
	if err := withAuthRetry(ctx, source, func(ctx context.Context, auth authorization) error {
		var bridge *websocket.Conn
		var err error
		bridge, result.ready, err = c.relayHandshake(ctx, auth, result.user, result.serverID, device, "bind_identity", "control", 0)
		if err == nil {
			_ = bridge.CloseNow()
		}
		return err
	}); err != nil {
		return result, fmt.Errorf("desktop device identity: %w", err)
	}
	if err := withAuthRetry(ctx, source, func(ctx context.Context, auth authorization) error {
		result.auth = auth
		var err error
		result.key, err = c.createModelKey(ctx, auth)
		return err
	}); err != nil {
		if validUUID(result.key.ID) {
			_ = revokeDesktopModelKey(source, result.key.ID)
			result.key = invocationKey{}
		}
		return result, fmt.Errorf("desktop model authorization: %w", err)
	}
	return result, nil
}

func revokeDesktopModelKey(source *desktopSourceAuth, id string) error {
	cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	return withAuthRetry(cleanup, source, func(ctx context.Context, auth authorization) error {
		return source.client.json(ctx, http.MethodDelete, "/api/v1/api-keys/"+id, auth.Access, "", nil, nil)
	})
}
