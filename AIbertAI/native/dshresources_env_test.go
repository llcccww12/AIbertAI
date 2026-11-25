package main

import (
	"slices"
	"testing"
)

func TestDSHProcessEnvironmentKeepsTemp(t *testing.T) {
	t.Setenv("TEMP", "/tmp/a")
	t.Setenv("TMP", "/tmp/b")
	t.Setenv("AIBERTAI_SECRET", "x")
	env := dshProcessEnvironment()
	if !slices.Contains(env, "TEMP=/tmp/a") || !slices.Contains(env, "TMP=/tmp/b") {
		t.Fatalf("temporary folder missing: %v", env)
	}
	for _, e := range env {
		if e == "AIBERTAI_SECRET=x" {
			t.Fatal("unrelated variables must not reach DSH")
		}
	}
}
