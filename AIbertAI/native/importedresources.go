package main

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
)

const importedDesktopResourcesFile = "imported-resources.enc.json"

type importedDesktopResources struct {
	Revision int
	Local    map[string]legacyMCP
	Experts  map[string]importedExpertRuntime
	Filters  map[string]builtinToolPreferences
}

func (h *relaySessionHost) runtimeResources() *importedDesktopResources {
	h.importedResourcesMu.RLock()
	defer h.importedResourcesMu.RUnlock()
	return h.importedResources
}

type importedExpertRuntime struct {
	LocalTools  []string
	DesignTools []string
	Builtin     builtinToolPreferences
}

func loadImportedDesktopResources(root *os.Root, key cipher.AEAD, server string) (*importedDesktopResources, error) {
	resources := &importedDesktopResources{Local: map[string]legacyMCP{}, Experts: map[string]importedExpertRuntime{}, Filters: map[string]builtinToolPreferences{}}
	data, err := readLegacyResource(root, importedDesktopResourcesFile, 2<<20)
	if errors.Is(err, os.ErrNotExist) {
		return resources, nil
	}
	var sealed sealedSourceGrant
	if err != nil || key == nil || json.Unmarshal(data, &sealed) != nil || sealed.Version != 1 || len(sealed.Nonce) != key.NonceSize() {
		return nil, errors.New("imported_runtime_resources_unreadable")
	}
	plain, err := key.Open(nil, sealed.Nonce, sealed.Ciphertext, importedDesktopResourcesAAD(server))
	defer clear(plain)
	if err != nil || json.Unmarshal(plain, resources) != nil || resources.Local == nil || resources.Experts == nil || resources.Filters == nil {
		return nil, errors.New("imported_runtime_resources_unreadable")
	}
	return resources, nil
}

func (r *importedDesktopResources) save(root *os.Root, key cipher.AEAD, server string) error {
	if key == nil {
		return errors.New("imported_runtime_resources_key_unavailable")
	}
	plain, err := json.Marshal(r)
	defer clear(plain)
	if err != nil || len(plain) > 1<<20 {
		return errors.New("imported_runtime_resources_too_large")
	}
	nonce := make([]byte, key.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	data, err := json.Marshal(sealedSourceGrant{Version: 1, Nonce: nonce, Ciphertext: key.Seal(nil, nonce, plain, importedDesktopResourcesAAD(server))})
	if err != nil {
		return err
	}
	return writeSourceState(root, importedDesktopResourcesFile, data)
}

func importedDesktopResourcesAAD(server string) []byte {
	return []byte("aibertai/imported-resources/v1\x00" + server)
}

func localMCPToolID(name string) string {
	sum := sha256.Sum256([]byte("aibertai/local-mcp/v1\x00" + name))
	var id [16]byte
	copy(id[:], sum[:16])
	id[6] = id[6]&0x0f | 0x80
	id[8] = id[8]&0x3f | 0x80
	return formatUUID(id)
}
