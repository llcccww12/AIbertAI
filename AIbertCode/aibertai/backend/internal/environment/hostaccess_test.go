package environment

import (
	"errors"
	"testing"

	"aibertai.local/backend/internal/environment/sqlc"
	"aibertai.local/backend/internal/resource"
	"github.com/jackc/pgx/v5"
)

func addConfiguredHost(t *testing.T, s *Service, user, source string) string {
	t.Helper()
	id := resource.ID()
	_, err := s.pool.Exec(t.Context(), `INSERT INTO hosts(id,user_id,name,architecture,version,status,last_seen_at,report,enabled,exchange_protocol,environment_synced_at,max_environments)
 SELECT $1::uuid,user_id,'second',architecture,version,'active',now(),report,true,1,environment_synced_at,max_environments FROM hosts WHERE id=$2`, id, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO host_environment_configs(host_id,resource,user_max_environments,billing) SELECT $1,resource,user_max_environments,billing FROM host_environment_configs WHERE host_id=$2`, id, source); err != nil {
		t.Fatal(err)
	}
	setHostAccess(t, s, id, user, []resource.Object{{"all_users": true}})
	return id
}
func setHostAccess(t *testing.T, s *Service, host, actor string, grants []resource.Object) {
	t.Helper()
	if err := s.transactionWithTx(t.Context(), func(_ *sqlc.Queries, tx pgx.Tx) error {
		return resource.SaveGrants(t.Context(), tx, "host", host, actor, grants, false)
	}); err != nil {
		t.Fatal(err)
	}
}
func hostSetup(t *testing.T, s *Service, host string) SetupView {
	t.Helper()
	out, err := s.setupView(t.Context(), sqlc.New(s.pool), host, s.pool)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHostSetupAndQuotasAreIndependent(t *testing.T) {
	s, user, first := fixture(t)
	second := addConfiguredHost(t, s, user, first)
	a, b := hostSetup(t, s, first), hostSetup(t, s, second)
	next := *a.Resource
	next.CPUMillis = 2000
	saved, err := s.saveSetup(t.Context(), first, Setup{Resource: next, MaxEnvironments: a.MaxEnvironments, UserMaxEnvironments: 1})
	if err != nil {
		t.Fatal(err)
	}
	after := hostSetup(t, s, second)
	if *after.Resource != *b.Resource || after.UserMaxEnvironments != b.UserMaxEnvironments || after.MaxEnvironments != b.MaxEnvironments {
		t.Fatal("修改一个宿主污染另一个宿主")
	}
	firstEnv, err := s.CreateOnHost(t.Context(), user, first, Creation{"first", "host-first"})
	if err != nil {
		t.Fatal(err)
	}
	if firstEnv.Environment.Spec == nil || firstEnv.Environment.HostID != first {
		t.Fatal("未固定请求宿主")
	}
	_, err = s.CreateOnHost(t.Context(), user, first, Creation{"too many", "host-first-full"})
	var f *failure
	if !errors.As(err, &f) || f.code != "quota_exceeded" {
		t.Fatalf("未按宿主限制配额：%v", err)
	}
	for _, key := range []string{"host-second-a", "host-second-b"} {
		if _, err = s.CreateOnHost(t.Context(), user, second, Creation{key, key}); err != nil {
			t.Fatal("另一个宿主受旧全局配额影响", err)
		}
	}
	if saved.Resource.CPUMillis != 2000 {
		t.Fatal("宿主规格未保存")
	}
}

func TestHostAccessRevocationPreservesRunningEnvironment(t *testing.T) {
	s, user, host := fixture(t)
	e, err := s.CreateOnHost(t.Context(), user, host, Creation{"retained", "access-retained"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	setHostAccess(t, s, host, user, []resource.Object{})
	syncState(t, s, host, observe(e.Environment, "running"))
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID, UserID: &user})
	if err != nil || current.Desired != "running" || current.Generation != 1 {
		t.Fatal("撤回授权打断已有环境", err)
	}
	if _, err = s.CreateOnHost(t.Context(), user, host, Creation{"denied", "host-access-denied"}); err == nil {
		t.Fatal("未授权创建宿主环境")
	}
	stopped, err := s.Change(t.Context(), &user, e.Environment.ID, "stopped", Change{1, "stop-revoked"})
	if err != nil {
		t.Fatal("撤回授权禁止了停止", err)
	}
	syncState(t, s, host, observe(stopped.Environment, "stopped"))
	if _, err = s.Change(t.Context(), &user, e.Environment.ID, "running", Change{2, "start-revoked"}); err == nil {
		t.Fatal("撤回授权仍可重新启动")
	}
	if _, err = s.Change(t.Context(), &user, e.Environment.ID, "deleted", Change{2, "delete-revoked"}); err != nil {
		t.Fatal("撤回授权禁止了删除", err)
	}
}

func TestHostScopeIncludesDescendantGroups(t *testing.T) {
	s, user, host := fixture(t)
	parent, child, other := resource.ID(), resource.ID(), resource.ID()
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO groups(id,name) VALUES($1,'parent')`, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO groups(id,name,parent_id) VALUES($1,'child',$2)`, child, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO group_users(group_id,user_id,assigned_by_user_id) VALUES($1,$2,$2)`, child, user); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO users(id,name,email) VALUES($1::uuid,'other',$1::text || '@example.com')`, other); err != nil {
		t.Fatal(err)
	}
	setHostAccess(t, s, host, user, []resource.Object{{"group_id": parent}})
	if _, err := s.CreateOnHost(t.Context(), user, host, Creation{"group", "group-allowed"}); err != nil {
		t.Fatal("父组授权未包含下级成员", err)
	}
	if _, err := s.CreateOnHost(t.Context(), other, host, Creation{"other", "group-denied"}); err == nil {
		t.Fatal("分组授权越权")
	}
}

func TestComputeGlobalPricingAppliesToAllHosts(t *testing.T) {
	s, user, first, _ := meterFixture(t, "10000")
	second := addConfiguredHost(t, s, user, first)
	for index, host := range []string{first, second} {
		if _, err := s.CreateOnHost(t.Context(), user, host, Creation{"global price", "rate-" + host}); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			setComputePrice(t, s, BillingConfig{Enabled: true, CPU: 720_000000})
		}
	}
	cost(t, s, user, "4320", 2)
	budget, _, err := s.billingBudget(t.Context(), user)
	if err != nil || budget == nil || budget.HourlyCredits != "1440.000000" {
		t.Fatalf("全局费率未应用到全部环境预算：%v %+v", err, budget)
	}
}
