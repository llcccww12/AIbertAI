package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const cloudOAuthClient = "monkeyai-runtime"
const cloudDeviceGrant = "urn:ietf:params:oauth:grant-type:device_code"
const cloudIdentityFile = "aibertai-cloud-identity.json"

type cloudIdentity struct {
	Server, ServerID, UserID, MachineID, Name, DataKey string
}

func (s cloudIdentity) device() (relayDevice, error) {
	key, err := base64.StdEncoding.Strict().DecodeString(s.DataKey)
	if err != nil || len(key) != ed25519.SeedSize ||
		!validUUID(s.MachineID) || !validUUID(s.ServerID) || !validUUID(s.UserID) || s.Name == "" {
		return relayDevice{}, errors.New("cloud_identity_invalid")
	}
	defer clear(key)
	boot, err := newUUID()
	if err != nil {
		return relayDevice{}, err
	}
	private := ed25519.NewKeyFromSeed(key)
	return relayDevice{machine: s.MachineID, boot: boot, name: s.Name, deploymentMode: "standalone",
		private: private, public: private.Public().(ed25519.PublicKey)}, nil
}

func readCloudIdentity(root *os.Root) (cloudIdentity, error) {
	var identity cloudIdentity
	info, err := root.Lstat(cloudIdentityFile)
	if err != nil {
		return identity, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 4096 {
		return identity, errors.New("cloud_identity_invalid")
	}
	err = readDesktopSessionRecord(root, cloudIdentityFile, &identity)
	return identity, err
}

func cloudCommand(ctx context.Context, args []string, out io.Writer) error {
	if runtime.GOOS != "linux" {
		return errors.New("cloud_runtime_requires_linux")
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("state-dir", "", "persistent cloud state")
	var server, serverID, user, name *string
	var deviceCode *bool
	if command == "login" {
		server = flags.String("server", "", "private backend origin")
		serverID = flags.String("server-id", "", "private backend identity")
		user = flags.String("user", "", "expected backend account")
		name = flags.String("name", "AIbertAI Cloud", "device name")
		deviceCode = flags.Bool("device-code", false, "use cloud approval")
	}
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || !filepath.IsAbs(*dir) || filepath.Clean(*dir) != *dir {
		return errors.New("cloud_runtime_arguments_invalid")
	}
	if command == "login" {
		if !*deviceCode || !validUUID(*serverID) || !validUUID(*user) || strings.TrimSpace(*name) == "" || len(*name) > 128 {
			return errors.New("cloud_runtime_arguments_invalid")
		}
		if err := os.MkdirAll(*dir, 0700); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(*dir)
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("installation_not_initialized")
	}
	if err != nil {
		return err
	}
	defer root.Close()
	identity, err := readCloudIdentity(root)
	if command == "login" && errors.Is(err, os.ErrNotExist) {
		// Unknown legacy credentials cannot safely be replaced by a new machine
		// identity. Allow known transcript/working data without rewriting it.
		entries, readErr := os.ReadDir(root.Name())
		if readErr != nil {
			return readErr
		}
		for _, entry := range entries {
			switch entry.Name() {
			case "runtime", "workspaces", "projects.json":
			default:
				return errors.New("cloud_existing_identity_migration_required")
			}
		}
		dataKey, keyErr := randomState()
		if keyErr != nil {
			return keyErr
		}
		seed, decodeErr := base64.RawURLEncoding.DecodeString(dataKey)
		if decodeErr != nil {
			return decodeErr
		}
		defer clear(seed)
		machine, idErr := newUUID()
		if idErr != nil {
			return idErr
		}
		identity = cloudIdentity{Server: *server, ServerID: *serverID, UserID: *user, MachineID: machine,
			Name: *name, DataKey: base64.StdEncoding.EncodeToString(seed)}
		// Verify the configured server before persisting any new identity.
		c, clientErr := newCloudBackendClient(identity.Server, os.Getenv("AIBERTAI_CLOUD_BACKEND_GATEWAY"))
		if clientErr != nil {
			return clientErr
		}
		actual, _, verifyErr := c.relaySigningKeys(ctx)
		if verifyErr != nil || actual != identity.ServerID {
			return errors.New("cloud_backend_identity_mismatch")
		}
		raw, _ := json.Marshal(identity)
		if err = persistSourceState(root, cloudIdentityFile, raw, true); err != nil {
			return err
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("installation_not_initialized")
	}
	if err != nil {
		return err
	}
	if command == "login" && (identity.Server != *server || identity.ServerID != *serverID || identity.UserID != *user) {
		return errors.New("cloud_identity_already_bound")
	}
	device, err := identity.device()
	if err != nil {
		return err
	}
	c, err := newCloudBackendClient(identity.Server, os.Getenv("AIBERTAI_CLOUD_BACKEND_GATEWAY"))
	if err != nil {
		return err
	}
	store, err := openSourceGrantStore(root.Name(), identity.DataKey, c.server, identity.UserID)
	if err != nil {
		return err
	}
	defer store.root.Close()
	saved, err := store.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	source := &desktopSourceAuth{client: c, store: store, grant: saved, oauthClient: cloudOAuthClient}
	encoder := json.NewEncoder(out)
	switch command {
	case "status":
		authorized := err == nil
		var account any
		if authorized {
			account = map[string]string{"serverUrl": identity.Server, "serverId": identity.ServerID, "userId": identity.UserID}
		}
		return encoder.Encode(map[string]any{"event": "status", "machineId": identity.MachineID,
			"publicKey": base64.RawURLEncoding.EncodeToString(device.public), "authorized": authorized, "account": account})
	case "login":
		return cloudAuthorize(ctx, source, identity, device, encoder)
	case "serve":
		return serveCloudRuntime(ctx, root.Name(), identity, device, source)
	default:
		return errors.New("cloud_runtime_arguments_invalid")
	}
}

func cloudAuthorize(ctx context.Context, source *desktopSourceAuth, identity cloudIdentity, device relayDevice, out *json.Encoder) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var approval struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		Interval   int64  `json:"interval"`
	}
	c := source.client
	serverID, _, err := c.relaySigningKeys(ctx)
	if err != nil || serverID != identity.ServerID {
		return errors.New("cloud_backend_identity_mismatch")
	}
	err = c.form(ctx, "/oauth/device/authorization", url.Values{
		"client_id": {cloudOAuthClient}, "machine_id": {device.machine}, "display_name": {device.name},
		"platform": {"linux"}, "arch": {runtime.GOARCH}, "execution_public_key": {base64.RawURLEncoding.EncodeToString(device.public)},
	}, &approval)
	if err != nil || approval.DeviceCode == "" || approval.UserCode == "" || len(approval.UserCode) > 16 ||
		approval.Interval < 1 || approval.Interval > 60 {
		return errors.New("cloud_device_request_failed")
	}
	if err := out.Encode(map[string]string{"event": "authorize", "userCode": approval.UserCode}); err != nil {
		return err
	}
	interval := time.Duration(approval.Interval) * time.Second
	fields := url.Values{"client_id": {cloudOAuthClient}, "grant_type": {cloudDeviceGrant}, "device_code": {approval.DeviceCode}}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		response, err := c.request(ctx, "POST", "/oauth/token", "", "", "", "application/x-www-form-urlencoded",
			strings.NewReader(fields.Encode()), http.StatusBadRequest)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(io.LimitReader(response.Body, 32<<10))
		if response.StatusCode == http.StatusBadRequest {
			var reply struct{ Error string }
			err = decoder.Decode(&reply)
			response.Body.Close()
			if err != nil {
				return errors.New("cloud_device_grant_invalid")
			}
			switch reply.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				interval += 5 * time.Second
				continue
			default:
				return errors.New("cloud_device_grant_rejected")
			}
		}
		var token authorization
		err = decoder.Decode(&token)
		response.Body.Close()
		var user account
		if err != nil || !validSourceAuthorization(token) || c.json(ctx, "GET", "/api/v1/me", token.Access, "", nil, &user) != nil || user.ID != identity.UserID {
			return errors.New("cloud_device_grant_identity_mismatch")
		}
		_, err = source.store.replace(token)
		return err
	}
}

func serveCloudRuntime(ctx context.Context, profile string, identity cloudIdentity, device relayDevice, source *desktopSourceAuth) error {
	startup, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if source.grant.Expires.IsZero() {
		return errors.New("cloud_device_authorization_required")
	}
	auth, err := source.access(startup)
	if err != nil {
		return err
	}
	c := source.client
	var user account
	if err := c.json(startup, "GET", "/api/v1/me", auth.Access, "", nil, &user); err != nil || user.ID != identity.UserID {
		return errors.New("cloud_device_grant_identity_mismatch")
	}
	conn, ready, err := c.relayHandshake(startup, auth, user, identity.ServerID, device, "bind_identity", "control", 0)
	if err != nil {
		return err
	}
	_ = conn.CloseNow()
	selected, err := c.selectedModelConfig(startup, auth)
	if err != nil {
		return err
	}
	live, err := openEmployeeSessions(profile, ready.ServerID, device.machine, "")
	if err != nil {
		return err
	}
	defer live.store.Close()
	defer live.workspaces.root.Close()
	// Docker mounts this directory as the environment's persistent workspace.
	const workspace = "/workspace"
	if _, err := live.workspaces.authorizeDirectory(json.RawMessage(`{"cwd":"/workspace"}`)); err != nil {
		return err
	}
	live.cloudWorkspace = workspace
	live.client, live.source, live.protocol, live.modelBudget = c, source, selected.Protocol, selected.inputBudget()
	modelID = selected.ID
	terminals, err := newDesktopTerminals()
	if err != nil {
		return err
	}
	defer terminals.shutdown()
	host := &relaySessionHost{profile: profile, live: live, terminals: terminals, terminalLifetime: ctx}
	relay := newDesktopRelay(host, device, user, ready, auth)
	if relay.uploads != nil {
		defer relay.uploads.root.Close()
	}
	if relay.stateErr != nil {
		return relay.stateErr
	}
	relay.supported = true // cloud access is managed by the environment controller.
	relay.pref.Enabled, relay.pref.PendingRevoke = true, false
	host.desktopRelay = relay
	cancel()
	done := make(chan struct{})
	defer func() { <-done }()
	go func() {
		defer close(done)
		host.superviseDesktopReports(ctx, 30*time.Second)
	}()
	relay.supervise(ctx, func() {
		relay.mu.Lock()
		online := relay.online
		relay.mu.Unlock()
		if online {
			attempt, stop := context.WithTimeout(ctx, 10*time.Second)
			defer stop()
			host.recoverDesktopReports(attempt)
		}
	})
	return nil
}

func emitCloudError(out io.Writer, err error) {
	reason := "cloud_runtime_failed"
	if err.Error() == "installation_not_initialized" || err.Error() == "cloud_existing_identity_migration_required" {
		reason = err.Error()
	}
	_ = json.NewEncoder(out).Encode(map[string]string{"event": "error", "reason": reason})
	fmt.Fprintln(os.Stderr, err)
}
