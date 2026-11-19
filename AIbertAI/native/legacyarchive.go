package main

import (
	"crypto/cipher"
	"encoding/json"
	"errors"
	"os"
)

// The old local MCP archive stores an AES-GCM envelope under the account's
// existing profile key. Its payload is normally a JSON array of connector rows.
func readLegacyMCPArchive(root *os.Root, name string, profileKey cipher.AEAD) ([]legacyMCP, error) {
	data, err := readLegacyResource(root, name, 2<<20)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	var envelope struct{ Nonce, Ciphertext []byte }
	if err != nil || profileKey == nil || json.Unmarshal(data, &envelope) != nil || len(envelope.Nonce) != profileKey.NonceSize() {
		return nil, errors.New("legacy_mcp_archive_unreadable")
	}
	plain, err := profileKey.Open(nil, envelope.Nonce, envelope.Ciphertext, nil)
	defer clear(plain)
	if err != nil {
		return nil, errors.New("legacy_mcp_archive_unreadable")
	}
	return decodeLegacyMCP(plain)
}
