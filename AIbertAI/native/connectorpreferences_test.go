package main

import (
	"encoding/json"
	"testing"
)

func TestConnectorPreferencesNormalizeBuiltinTools(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{`{}`, `{}`},
		{`{"builtin_tools":null}`, `{}`},
		{`{"builtin_tools":{"blacklist":["Read"]}}`, `{"blacklist":["Read"]}`},
	} {
		var preferences connectorPreferences
		if err := json.Unmarshal([]byte(tc.input), &preferences); err != nil {
			t.Fatal(err)
		}
		if got := string(preferences.normalized().Builtin); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.input, got, tc.want)
		}
	}
}
