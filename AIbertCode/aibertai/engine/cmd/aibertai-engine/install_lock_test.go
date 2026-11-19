//go:build linux || darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockHostInstallOnFreshHost(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "control")
	unlock, err := lockHostInstall(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("control directory is not private: %v", err)
	}
	if release, err := lockHostInstall(dir); err == nil {
		release()
		t.Fatal("a second installer acquired the lock")
	}
}
