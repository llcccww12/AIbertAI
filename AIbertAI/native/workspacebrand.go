package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	workspaceDataDir       = ".aibertai"
	legacyWorkspaceDataDir = ".monkeyai"
	workspaceBrandMarker   = ".workspace-brand-migration-v1"
	workspaceBrandReceipt  = ".monkeyai -> .aibertai\n"
	workspaceUploads       = workspaceDataDir + "/uploads/upload-"
	workspaceHistoryFiles  = workspaceDataDir + "/history-files/"
)

var workspaceBrandMu sync.Mutex

// Stored paths may refer to the old name even after the directory has moved.
func canonicalWorkspaceReference(name string) string {
	if name == legacyWorkspaceDataDir {
		return workspaceDataDir
	}
	if tail, ok := strings.CutPrefix(name, legacyWorkspaceDataDir+"/"); ok {
		return workspaceDataDir + "/" + tail
	}
	return name
}

func sameRelayFileReference(saved, requested relayFileRef) bool {
	saved.Path = canonicalWorkspaceReference(saved.Path)
	requested.Path = canonicalWorkspaceReference(requested.Path)
	return saved == requested
}

func workspaceBrandDirectory(root *os.Root, name string) (bool, error) {
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !info.IsDir() {
		return false, errors.New("workspace_brand_path_unsafe")
	}
	return true, nil
}

func workspaceBrandReceiptExists(root *os.Root, dir string) (bool, error) {
	name := filepath.Join(dir, workspaceBrandMarker)
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(workspaceBrandReceipt)) {
		return false, errors.New("workspace_brand_receipt_invalid")
	}
	file, err := openAttachmentFile(root, name, int64(len(workspaceBrandReceipt)))
	if err != nil {
		return false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(len(workspaceBrandReceipt)+1)))
	if err != nil || string(data) != workspaceBrandReceipt {
		return false, errors.New("workspace_brand_receipt_invalid")
	}
	return true, nil
}

func moveWorkspaceBrand(root *os.Root, rollback bool) error {
	workspaceBrandMu.Lock()
	defer workspaceBrandMu.Unlock()
	legacy, err := workspaceBrandDirectory(root, legacyWorkspaceDataDir)
	if err != nil {
		return err
	}
	current, err := workspaceBrandDirectory(root, workspaceDataDir)
	if err != nil {
		return err
	}
	if legacy && current {
		return errors.New("workspace_brand_conflict")
	}
	if rollback {
		dir := workspaceDataDir
		if legacy {
			dir = legacyWorkspaceDataDir
		}
		present, err := workspaceBrandReceiptExists(root, dir)
		if err != nil {
			return err
		}
		if !present {
			return errors.New("workspace_brand_rollback_unavailable")
		}
		if legacy {
			return nil
		}
		return root.Rename(workspaceDataDir, legacyWorkspaceDataDir)
	}
	if !legacy {
		return nil
	}
	present, err := workspaceBrandReceiptExists(root, legacyWorkspaceDataDir)
	if err != nil {
		return err
	}
	if !present {
		name := filepath.Join(legacyWorkspaceDataDir, workspaceBrandMarker)
		if err := persistSourceState(root, name, []byte(workspaceBrandReceipt), true); err != nil {
			return err
		}
	}
	// Rename once within the same workspace. No media is copied or overwritten.
	return root.Rename(legacyWorkspaceDataDir, workspaceDataDir)
}

func prepareWorkspaceBrand(root *os.Root, err error) (*os.Root, error) {
	if err != nil {
		return nil, err
	}
	if err := moveWorkspaceBrand(root, false); err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

func rollbackWorkspaceBrand(cwd string) error {
	if !filepath.IsAbs(cwd) {
		return errors.New("workspace_brand_absolute_path_required")
	}
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return err
	}
	defer root.Close()
	return moveWorkspaceBrand(root, true)
}
