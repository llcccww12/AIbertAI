package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var errDSHSharedProfileChanged = errors.New("dsh_shared_plugin_profile_changed_during_startup")

// The sdk profile and home overlay are shared by this employee's SDK children.
// The invocation overlay and session journals are private and intentionally excluded.
var dshSharedFiles = [...]string{"cordis.patch.yml", "profiles/sdk/cordis.patch.yml", "profiles/sdk/package.json", "profiles/sdk/pnpm-lock.yaml", "profiles/sdk/pnpm-workspace.yaml", "profiles/sdk/compatibility.json"}

type dshSharedProfile [len(dshSharedFiles)]struct {
	Digest   [sha256.Size]byte
	Modified int64
}

func readDSHSharedProfile(ctx context.Context, home string) (dshSharedProfile, error) {
	// The upstream plugin manager serializes install/remove/enable operations with
	// this PID lock. Do not boot against partially published packages or steal a writer's lock.
	lock := filepath.Join(home, "profiles", "sdk", "package.json.lock")
	wait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		var profile dshSharedProfile
		if _, err := os.Lstat(lock); errors.Is(err, os.ErrNotExist) {
			for i, name := range dshSharedFiles {
				path := filepath.Join(home, filepath.FromSlash(name))
				stat, err := os.Stat(path)
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					return profile, err
				}
				if !stat.Mode().IsRegular() {
					return profile, fmt.Errorf("invalid DSH profile file %s", name)
				}
				file, err := os.Open(path)
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					return profile, err
				}
				digest := sha256.New()
				_, readErr := io.Copy(digest, file)
				file.Close()
				if readErr != nil {
					return profile, readErr
				}
				copy(profile[i].Digest[:], digest.Sum(nil))
				profile[i].Modified = stat.ModTime().UnixNano()
			}
			if _, err = os.Lstat(lock); errors.Is(err, os.ErrNotExist) {
				return profile, nil
			} else if err != nil {
				return profile, err
			}
		} else if err != nil {
			return profile, err
		}
		select {
		case <-wait.Done():
			return profile, fmt.Errorf("waiting for the DSH plugin manager: %w", wait.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}
