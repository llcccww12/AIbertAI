package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

func workspaceDesign(root *os.Root, sessionID string) json.RawMessage {
	if !validUUID(sessionID) {
		return nil
	}
	base := filepath.Join(workspaceDataDir, "design")
	markerDir := filepath.Join(base, "sessions")
	for _, name := range []string{workspaceDataDir, base, markerDir} {
		if info, err := root.Lstat(name); err != nil || !info.IsDir() {
			return nil
		}
	}
	marker := readDesignJSON(root, filepath.Join(markerDir, sessionID+".json"), 64<<10)
	var design struct{ CanvasID, LocalDesignPath string }
	if json.Unmarshal(marker, &design) != nil || !validUUID(design.CanvasID) ||
		canonicalWorkspaceReference(design.LocalDesignPath) != workspaceDataDir+"/design/"+design.CanvasID {
		return nil
	}
	dir := filepath.Join(base, design.CanvasID)
	if info, err := root.Lstat(dir); err != nil || !info.IsDir() {
		return nil
	}
	var context struct{ DesignSessionID, CanvasID string }
	if json.Unmarshal(readDesignJSON(root, filepath.Join(dir, "design-context.json"), maxDesktopSessionBytes), &context) != nil ||
		context.DesignSessionID != sessionID || context.CanvasID != design.CanvasID {
		return nil
	}
	if info, err := root.Lstat(filepath.Join(dir, "canvas.json")); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if design.LocalDesignPath != canonicalWorkspaceReference(design.LocalDesignPath) {
		var fields map[string]json.RawMessage
		if json.Unmarshal(marker, &fields) != nil {
			return nil
		}
		fields["localDesignPath"], _ = json.Marshal(canonicalWorkspaceReference(design.LocalDesignPath))
		marker, _ = json.Marshal(fields)
	}
	return marker
}

func readDesignJSON(root *os.Root, name string, limit int64) []byte {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil
	}
	file, err := root.Open(name)
	if err != nil {
		return nil
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil
	}
	return data
}
