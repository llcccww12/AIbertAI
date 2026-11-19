package hostpaths

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigRoot       = "/etc/aibertai-engine"
	StateRoot        = "/var/lib/aibertai-engine"
	UnitName         = "aibertai-engine.service"
	LegacyConfigRoot = "/etc/monkeyai-engine"
	LegacyStateRoot  = "/var/lib/monkeyai-engine"
	LegacyUnitName   = "monkeyai-engine.service"
)

// Existing hosts can install a compatible engine before scheduling the rename.
var Config, State, Unit = active()

func active() (string, string, string) {
	if _, err := os.Lstat(filepath.Join(ConfigRoot, "config.json")); os.IsNotExist(err) {
		if _, err := os.Lstat(filepath.Join(LegacyConfigRoot, "config.json")); err == nil {
			return LegacyConfigRoot, LegacyStateRoot, LegacyUnitName
		}
	}
	return ConfigRoot, StateRoot, UnitName
}

// Resolve a saved path only through the alias created by the host migration.
func SavedPath(saved string) string {
	for _, pair := range [][2]string{{LegacyStateRoot, StateRoot}, {StateRoot, LegacyStateRoot}} {
		suffix, exists := strings.CutPrefix(saved, pair[0])
		if !exists || suffix != "" && !strings.HasPrefix(suffix, "/") {
			continue
		}
		if alias, err := os.Readlink(pair[0]); err == nil && alias == pair[1] {
			return pair[1] + suffix
		}
	}
	return saved
}
