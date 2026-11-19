// Package envconfig reads AIbertAI deployment settings and supports existing environments.
package envconfig

import (
	"os"
	"strings"
)

func LookupEnv(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	}
	return os.LookupEnv("MONKEYAI_" + strings.TrimPrefix(key, "AIBERTAI_"))
}

func Getenv(key string) string {
	value, _ := LookupEnv(key)
	return value
}
