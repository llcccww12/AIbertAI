package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxDesktopCredentials = 64 << 10

var errDesktopCredential = errors.New("desktop_credential_unavailable")

type desktopKnowledgeInput struct {
	Server, UserID, ProfileDir, DataKey, Method string
	Params                                      struct {
		ID            string `json:"id"`
		DocumentID    string `json:"documentId"`
		CredentialID  string `json:"credentialId"`
		ServerID      string `json:"serverId"`
		MachineID     string `json:"machineId"`
		ConnectionID  string `json:"connectionId"`
		WorkspaceRoot string `json:"workspaceRoot"`
		Path          string `json:"path"`

		Owner            string            `json:"owner"`
		Query            string            `json:"q"`
		Status           string            `json:"status"`
		Page             int               `json:"page"`
		Size             int               `json:"page_size"`
		Tags             []string          `json:"tag_ids"`
		ETag             string            `json:"etag"`
		Input            json.RawMessage   `json:"input"`
		Grants           []json.RawMessage `json:"grants"`
		ExpectedRevision *int64            `json:"expectedRevision"`
	}
}

func desktopAuthorization(in desktopKnowledgeInput) (authorization, error) {
	var none authorization
	key, err := base64.StdEncoding.DecodeString(in.DataKey)
	if err != nil || len(key) != 32 || !filepath.IsAbs(in.ProfileDir) {
		return none, errDesktopCredential
	}
	defer clear(key)
	file, err := os.Open(filepath.Join(in.ProfileDir, "credentials.enc.json"))
	if err != nil {
		return none, errDesktopCredential
	}
	defer file.Close()
	var stored struct {
		Version           int
		Nonce, Ciphertext []byte
	}
	if json.NewDecoder(io.LimitReader(file, maxDesktopCredentials)).Decode(&stored) != nil || stored.Version != 1 {
		return none, errDesktopCredential
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return none, errDesktopCredential
	}
	sealer, err := cipher.NewGCM(block)
	if err != nil || len(stored.Nonce) != sealer.NonceSize() {
		return none, errDesktopCredential
	}
	plain, err := sealer.Open(nil, stored.Nonce, stored.Ciphertext, nil)
	if err != nil {
		return none, errDesktopCredential
	}
	defer clear(plain)
	var payload struct {
		Server, UserID, ClientID string
		Tokens                   struct{ AccessToken string }
	}
	if json.Unmarshal(plain, &payload) != nil || payload.Server != in.Server || payload.UserID != in.UserID ||
		payload.ClientID != clientID || payload.Tokens.AccessToken == "" {
		return none, errDesktopCredential
	}
	return authorization{Access: payload.Tokens.AccessToken}, nil
}

func desktopKnowledge(ctx context.Context, reader io.Reader) (any, error) {
	var in desktopKnowledgeInput
	if err := json.NewDecoder(io.LimitReader(reader, 8<<20)).Decode(&in); err != nil {
		return nil, errors.New("invalid_request")
	}
	if in.Method == "_monkeyai/workspace/stat" {
		return statWorkspaceFile(in.Params.WorkspaceRoot, in.Params.Path)
	}
	c, err := newClient(in.Server)
	if err != nil || (backendOrigin != "" && in.Server != backendOrigin) {
		return nil, errDesktopCredential
	}
	auth, err := desktopAuthorization(in)
	if err != nil {
		return nil, err
	}
	return c.employeeKnowledge(ctx, auth, in)
}

func (c *client) employeeKnowledge(ctx context.Context, auth authorization, in desktopKnowledgeInput) (any, error) {
	p := in.Params
	filter := knowledgeFilter{Owner: p.Owner, Query: p.Query, Status: p.Status, Page: p.Page, Size: p.Size, Tags: p.Tags}
	switch in.Method {
	case "_monkeyai/kb/list":
		return c.knowledgeBases(ctx, auth, filter)
	case "_monkeyai/kb/documents/list":
		return c.knowledgeDocuments(ctx, auth, p.ID, filter)
	case "_monkeyai/kb/documents/get":
		return c.knowledgeDocument(ctx, auth, p.ID, p.DocumentID)
	case "_monkeyai/devices/connect":
		return c.remoteConnection(ctx, auth, p.ServerID, p.MachineID, p.ConnectionID)
	case "_monkeyai/resources/connectors/tools":
		return c.connectorTools(ctx, auth, p.ID, p.CredentialID)
	default:
		op, err := desktopKnowledgeOperation(in)
		if err != nil {
			return nil, err
		}
		return c.knowledgeChange(ctx, auth, op)
	}
}

func writeDesktopKnowledge(ctx context.Context, input io.Reader, output io.Writer) {
	result, err := desktopKnowledge(ctx, input)
	response := map[string]any{"result": result}
	if err != nil {
		response = map[string]any{"error": err.Error()}
	}
	_ = json.NewEncoder(output).Encode(response)
}
