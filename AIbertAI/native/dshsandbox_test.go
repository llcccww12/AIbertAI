package main

import "testing"

func TestDSHSandboxEscalationMatchesDSHContract(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode, why any
		allow     bool
	}{
		{"plain call", nil, nil, true},
		{"read-only", "read-only", "inspect", true},
		{"workspace-write", "workspace-write", "write the report", true},
		{"full access", "danger-full-access", "network install", true},
		{"missing justification", "workspace-write", nil, false},
		{"blank justification", "danger-full-access", "  ", false},
		{"unknown mode", "root", "why", false},
		{"justification alone", nil, "why", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dshSandboxEscalationValid(tc.mode, tc.why); got != tc.allow {
				t.Fatalf("allow=%v, want %v", got, tc.allow)
			}
		})
	}
}
