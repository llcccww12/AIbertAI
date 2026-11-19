package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHostUpgradeRestoresProgramConfigurationAndUnit(t *testing.T) {
	dir := t.TempDir()
	up := hostUpgrade{
		config: filepath.Join(dir, "config.json"), unit: filepath.Join(dir, "engine.service"),
		program: filepath.Join(dir, "engine"), journal: filepath.Join(dir, "source-upgrade.json"),
	}
	original := map[string]string{
		up.config:                           `{"host_id":"existing-host","version":"1","extra":"retain"}`,
		up.unit:                             "[Service]\nExecStart=/existing/engine serve\n",
		up.program:                          "previous executable",
		filepath.Join(dir, "host.key"):      "original identity",
		filepath.Join(dir, "workspace.txt"): "original workspace",
	}
	for name, value := range original {
		if err := os.WriteFile(name, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	restarts := 0
	up.restart = func(context.Context) error {
		restarts++
		return nil
	}
	config := []byte(`{"host_id":"existing-host","version":"2","extra":"retain"}`)
	unit := []byte("[Service]\nExecStart=/next/engine serve\n")
	program := []byte("new executable")
	missing := errors.New("missing signed heartbeat")
	if err := up.apply(t.Context(), config, unit, program, func(context.Context) error { return missing }); !errors.Is(err, missing) {
		t.Fatalf("heartbeat failure did not roll back: %v", err)
	}
	for name, value := range original {
		got, err := os.ReadFile(name)
		if err != nil || string(got) != value {
			t.Fatalf("failed to preserve %s: %v", filepath.Base(name), err)
		}
	}
	if restarts != 2 {
		t.Fatalf("expected restart and rollback restart, got %d", restarts)
	}
	for _, name := range []string{up.journal, up.journal + ".program"} {
		if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("finished rollback left %s: %v", filepath.Base(name), err)
		}
	}
	confirm := func(context.Context) error { return nil }
	if err := up.apply(t.Context(), config, unit, program, confirm); err != nil {
		t.Fatal(err)
	}
	if err := up.apply(t.Context(), config, unit, program, confirm); err != nil || restarts != 3 {
		t.Fatalf("same version restarted the service: %d, %v", restarts, err)
	}
}
