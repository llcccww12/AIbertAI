package environment

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type officialRuntimeCatalog struct{ runtimeCatalogFixture }

func (c officialRuntimeCatalog) Current(arch string) (RuntimePackage, error) {
	return c.Get(c.latest, arch)
}
func (c officialRuntimeCatalog) Get(version, arch string) (RuntimePackage, error) {
	release, err := c.runtimeCatalogFixture.Get(version, arch)
	release.RuntimeRevision = strings.Repeat("a", 40)
	release.EngineRevision = strings.Repeat("b", 40)
	release.Files = map[string]RuntimeFile{"monkeyai-runtime": release.Files["monkeyai-runtime"]}
	return release, err
}
func TestOfficialRuntimeRequiresNewHostCapability(t *testing.T) {
	service, created, host, _ := managedEnvironment(t)
	service.WithRuntimeCatalog(officialRuntimeCatalog{runtimeCatalogFixture{latest: "26.1105.2"}})
	ctx := t.Context()
	env := created.Environment
	if _, err := service.pool.Exec(ctx, `UPDATE hosts SET report=report || '{"environment_protocol":2,"runtime_update_protocol":1}'::jsonb WHERE id=$1`, host); err != nil {
		t.Fatal(err)
	}
	syncState(t, service, host, observe(env, "running"))
	stopped, err := service.Change(ctx, &env.UserID, env.ID, "stopped", Change{Generation: 1, RequestKey: "official-stop"})
	if err != nil {
		t.Fatal(err)
	}
	observation := observe(stopped.Environment, "stopped")
	observation.RuntimeVersion = "26.1105.1"
	syncState(t, service, host, observation)
	input := RuntimeUpdate{Generation: 2, RequestKey: "official-update"}
	if _, err := service.UpdateRuntime(ctx, env.UserID, env.ID, input); err == nil {
		t.Fatal("将单文件下发给仅支持旧资源包的宿主")
	}
	if _, err := service.pool.Exec(ctx, `UPDATE hosts SET report=report || '{"runtime_update_protocol":2}'::jsonb WHERE id=$1`, host); err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateRuntime(ctx, env.UserID, env.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	var spec Spec
	if err := json.Unmarshal(updated.Environment.Spec, &spec); err != nil || spec.RuntimePackage == nil || !spec.RuntimePackage.SingleFile() || spec.RuntimePackage.Validate("amd64") != nil {
		t.Fatal(spec, err)
	}
	raw, _ := json.Marshal(Exchange{Protocol: 1})
	response, err := service.Exchange(ctx, host, raw)
	if err != nil {
		t.Fatal(err)
	}
	commands, ok := response.(Commands)
	if !ok || commands.RuntimeUpdateProtocol != 2 {
		t.Fatal(response)
	}
}
func TestSingleFileRuntimeRejectsMissingRevisionsAndExtraPaths(t *testing.T) {
	release, _ := officialRuntimeCatalog{runtimeCatalogFixture{latest: "26.1105.2"}}.Current("amd64")
	if err := release.Validate("amd64"); err != nil {
		t.Fatal(err)
	}
	release.EngineRevision = ""
	if release.Validate("amd64") == nil {
		t.Fatal("缺少官方修订仍接受单文件发行")
	}
	release.EngineRevision = strings.Repeat("b", 40)
	release.Files["../ohmyagent"] = RuntimeFile{SHA256: strings.Repeat("a", 64), Size: 1}
	if release.Validate("amd64") == nil {
		t.Fatal("接受多余文件路径")
	}
}

type aiBertSourceCatalog struct{ officialRuntimeCatalog }

func (c aiBertSourceCatalog) Current(arch string) (RuntimePackage, error) {
	return c.Get(c.latest, arch)
}

func (c aiBertSourceCatalog) Get(version, arch string) (RuntimePackage, error) {
	release, err := c.officialRuntimeCatalog.Get(version, arch)
	release.EngineRevision = ""
	release.Files = map[string]RuntimeFile{"aibertai-native": release.Files["monkeyai-runtime"]}
	return release, err
}

func TestSourceProvisioningProtectsExistingCloudIdentity(t *testing.T) {
	s, created, host, _ := managedEnvironment(t)
	e := created.Environment
	ctx := t.Context()
	binding := runtimeRequest(e)
	if err := s.approveRuntime(ctx, host, binding); err != nil {
		t.Fatal(err)
	}
	catalog := officialRuntimeCatalog{runtimeCatalogFixture{latest: "26.1105.2"}}
	s.WithSourceRuntimeProvisioning().WithRuntimeCatalog(catalog)
	in := Creation{Name: "Source instance", RequestKey: "source-instance"}
	if _, err := s.Create(ctx, e.UserID, in); err == nil {
		t.Fatal("created a new cloud environment with a legacy runtime")
	}
	s.WithRuntimeCatalog(aiBertSourceCatalog{catalog})
	if _, err := s.Create(ctx, e.UserID, in); err == nil {
		t.Fatal("assigned a source release to a host without protocol 3")
	}
	if _, err := s.pool.Exec(ctx, `UPDATE hosts SET report=report || '{"runtime_update_protocol":3}'::jsonb WHERE id=$1`, host); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.Create(ctx, e.UserID, in)
	if err != nil {
		t.Fatal(err)
	}
	var spec Spec
	if err := json.Unmarshal(fresh.Environment.Spec, &spec); err != nil || spec.RuntimePackage == nil || !spec.RuntimePackage.SourceRuntime() {
		t.Fatalf("new cloud environment has no source runtime: %+v, %v", spec.RuntimePackage, err)
	}

	syncState(t, s, host, observe(e, "running"))
	stopped, err := s.Change(ctx, &e.UserID, e.ID, "stopped", Change{Generation: e.Generation, RequestKey: "legacy-stop"})
	if err != nil {
		t.Fatal(err)
	}
	report := observe(stopped.Environment, "stopped")
	report.RuntimeVersion = "26.1105.1"
	syncState(t, s, host, report)
	_, err = s.UpdateRuntime(ctx, e.UserID, e.ID, RuntimeUpdate{Generation: stopped.Environment.Generation, RequestKey: "legacy-upgrade"})
	var rejection *failure
	if !errors.As(err, &rejection) || rejection.code != "runtime_identity_migration_required" {
		t.Fatalf("expected migration guard for legacy identity: %v", err)
	}
}
