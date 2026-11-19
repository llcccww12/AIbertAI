package environment

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var runtimeVersion = regexp.MustCompile(`^[0-9]{1,8}\.[0-9]{1,8}\.[0-9]{1,8}$`)
var runtimeHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

type RuntimeFile struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type RuntimePackage struct {
	RuntimeRevision string                 `json:"runtime_revision,omitempty"`
	EngineRevision  string                 `json:"engine_revision,omitempty"`
	Version         string                 `json:"version"`
	Architecture    string                 `json:"architecture"`
	Files           map[string]RuntimeFile `json:"files"`
}

func (p RuntimePackage) SingleFile() bool { return len(p.Files) == 1 }
func (p RuntimePackage) SourceRuntime() bool {
	_, ok := p.Files["aibertai-native"]
	return p.SingleFile() && ok
}
func (p RuntimePackage) Protocol() int {
	if p.SourceRuntime() {
		return 3
	}
	if p.SingleFile() {
		return 2
	}
	return 1
}

func (p RuntimePackage) Validate(arch string) error {
	if !runtimeVersion.MatchString(p.Version) || p.Architecture != arch || (arch != "amd64" && arch != "arm64") || (len(p.Files) != 4 && !p.SingleFile()) {
		return errors.New("runtime 发行版本或架构无效")
	}
	names := []string{"monkeyai-runtime", "ohmyagent", "LICENSES.txt", "bundle.json"}
	if p.SingleFile() {
		names = names[:1]
		if p.SourceRuntime() {
			names = []string{"aibertai-native"}
		}
	}
	if p.SingleFile() || p.RuntimeRevision != "" || p.EngineRevision != "" {
		revisions := []string{p.RuntimeRevision, p.EngineRevision}
		if p.SourceRuntime() && p.EngineRevision == "" {
			revisions = revisions[:1]
		}
		for _, revision := range revisions {
			if len(revision) != 40 || strings.Trim(revision, "0123456789abcdef") != "" {
				return errors.New("runtime 源码修订无效")
			}
		}
	}
	for _, name := range names {
		file, ok := p.Files[name]
		limit := int64(512 << 20)
		if name == "bundle.json" {
			limit = 16 << 10
		}
		if name == "LICENSES.txt" {
			limit = 8 << 20
		}
		if !ok || !runtimeHash.MatchString(file.SHA256) || file.Size <= 0 || file.Size > limit {
			return errors.New("runtime 发行文件缺失或校验值无效")
		}
	}
	return nil
}

func newerRuntime(candidate, current string) bool {
	if !runtimeVersion.MatchString(candidate) || !runtimeVersion.MatchString(current) {
		return false
	}
	left, right := strings.Split(candidate, "."), strings.Split(current, ".")
	for index := range left {
		a, _ := strconv.Atoi(left[index])
		b, _ := strconv.Atoi(right[index])
		if a != b {
			return a > b
		}
	}
	return false
}

type RuntimeCatalog interface {
	Current(string) (RuntimePackage, error)
	Get(string, string) (RuntimePackage, error)
}

func (s *Service) WithRuntimeCatalog(catalog RuntimeCatalog) *Service {
	s.runtimeCatalog = catalog
	return s
}

func (s *Service) WithSourceRuntimeProvisioning() *Service {
	s.sourceRuntime = true
	return s
}

func (s *Service) runtimeForProvisioning(arch string) (RuntimePackage, error) {
	if s.runtimeCatalog == nil {
		return RuntimePackage{}, errors.New("云电脑缺少源码运行时发行资源")
	}
	release, err := s.runtimeCatalog.Current(arch)
	if err != nil {
		return RuntimePackage{}, err
	}
	if s.sourceRuntime && !release.SourceRuntime() {
		return RuntimePackage{}, errors.New("新建云电脑需要源码运行时")
	}
	return release, release.Validate(arch)
}
