package envconfig

import "testing"

func TestLookupEnvPrefersCurrentKey(t *testing.T) {
	const key = "AIBERTAI_ENVCONFIG_TEST"
	t.Setenv("MONKEYAI_ENVCONFIG_TEST", "previous")

	if value := Getenv(key); value != "previous" {
		t.Fatalf("legacy value = %q", value)
	}

	t.Setenv(key, "current")
	if value := Getenv(key); value != "current" {
		t.Fatalf("current value = %q", value)
	}

	t.Setenv(key, "")
	if value, present := LookupEnv(key); !present || value != "" {
		t.Fatalf("empty override = %q, present = %t", value, present)
	}
}
