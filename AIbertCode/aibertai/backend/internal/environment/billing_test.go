package environment

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"aibertai.local/backend/internal/billing"
	"aibertai.local/backend/internal/environment/sqlc"
)

func meterFixture(t *testing.T, credits string) (*Service, string, string, *time.Time) {
	t.Helper()
	s, user, host := fixture(t)
	if _, err := s.pool.Exec(t.Context(), `INSERT INTO users(id,name,email,role) VALUES(gen_random_uuid(),'计费管理员',gen_random_uuid()::text || '@example.com','admin')`); err != nil {
		t.Fatal(err)
	}
	charges := billing.NewService(s.pool)
	if err := charges.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `UPDATE settings SET value=jsonb_set(value,'{root_credits}',$1::jsonb) WHERE key='billing'`, credits); err != nil {
		t.Fatal(err)
	}
	setComputePrice(t, s, BillingConfig{Enabled: true, CPU: 3600_000000})
	at := time.Now().UTC().Truncate(time.Second)
	s.now = func() time.Time { return at }
	return s.WithBilling(charges), user, host, &at
}

func setComputePrice(t *testing.T, s *Service, pricing BillingConfig) {
	t.Helper()
	raw, err := json.Marshal(pricing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.pool.Exec(t.Context(), `UPDATE settings SET value=jsonb_set(value,'{environment}',$1::jsonb) WHERE key='billing'`, raw); err != nil {
		t.Fatal(err)
	}
}

func cost(t *testing.T, s *Service, user, want string, count int) {
	t.Helper()
	var amount string
	var n int
	if err := s.pool.QueryRow(t.Context(), `SELECT COALESCE(sum(-credit_delta),0)::text,count(*) FROM credit_ledger_entries WHERE user_id=$1 AND category='environment'`, user).Scan(&amount, &n); err != nil {
		t.Fatal(err)
	}
	a, err := billing.ParseAmount(amount)
	if err != nil || a.String() != want || n != count {
		t.Fatalf("费用=%s 笔数=%d，期望%s/%d，错误=%v", amount, n, want, count, err)
	}
}
func balance(t *testing.T, s *Service, user string) string {
	t.Helper()
	var amount string
	if err := s.pool.QueryRow(t.Context(), `SELECT balance::text FROM credit_accounts WHERE user_id=$1 ORDER BY period_start_at DESC LIMIT 1`, user).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	a, err := billing.ParseAmount(amount)
	if err != nil {
		t.Fatal(err)
	}
	return a.String()
}
func settle(t *testing.T, s *Service, closedOnly bool) {
	t.Helper()
	if err := s.settleBilling(t.Context(), closedOnly); err != nil {
		t.Fatal(err)
	}
}

func TestComputeMeterLifecycleAndIdempotency(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"计费环境", "create-meter"})
	if err != nil {
		t.Fatal(err)
	}
	cost(t, s, user, "3600", 1)
	stopped, err := s.Change(t.Context(), &user, e.Environment.ID, "stopped", Change{1, "stop-meter"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(stopped.Environment, "stopped"))
	started, err := s.Change(t.Context(), &user, e.Environment.ID, "running", Change{2, "restart-meter"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(started.Environment, "running"))
	cost(t, s, user, "3600", 1)
	*at = at.Add(time.Hour)
	syncState(t, s, host, observe(started.Environment, "running"))
	settle(t, s, false)
	cost(t, s, user, "7200", 2)
	if got := balance(t, s, user); got != "2800" {
		t.Fatalf("余额=%s", got)
	}
}

func TestComputeInsufficientBalanceStopsAtRenewal(t *testing.T) {
	s, user, host, at := meterFixture(t, "4000")
	e, err := s.Create(t.Context(), user, Creation{"余额环境", "balance-create"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	*at = at.Add(time.Hour)
	syncState(t, s, host, observe(e.Environment, "running"))
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Desired != "stopped" || current.BillingStopReason != "insufficient_credits" {
		t.Fatalf("余额不足后未停止：%v %s %s", err, current.Desired, current.BillingStopReason)
	}
	cost(t, s, user, "3600", 1)
}

func TestComputeUnknownIntervalIsNeverBackfilled(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"断线环境", "unknown-create"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	*at = at.Add(2 * time.Hour)
	settle(t, s, false)
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Desired != "stopped" || current.BillingStopReason != "billing_unavailable" {
		t.Fatalf("运行状态不明时没有停止续费：%v %s %s", err, current.Desired, current.BillingStopReason)
	}
	cost(t, s, user, "3600", 1)
}

func TestComputeConcurrentPurchaseAndRollback(t *testing.T) {
	s, user, _, _ := meterFixture(t, "10000")
	in := Creation{"并发环境", "create-concurrent"}
	if _, err := s.pool.Exec(t.Context(), `CREATE FUNCTION reject_compute() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.category='environment' THEN RAISE EXCEPTION 'test failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_compute BEFORE INSERT ON credit_ledger_entries FOR EACH ROW EXECUTE FUNCTION reject_compute()`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(t.Context(), user, in); err == nil {
		t.Fatal("未触发账务回滚")
	}
	cost(t, s, user, "0", 0)
	var environments int
	if err := s.pool.QueryRow(t.Context(), "SELECT count(*) FROM environments WHERE user_id=$1", user).Scan(&environments); err != nil || environments != 0 {
		t.Fatalf("扣费失败仍创建了环境：%d，%v", environments, err)
	}
	if _, err := s.pool.Exec(t.Context(), `DROP TRIGGER reject_compute ON credit_ledger_entries`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan Result, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			out, err := s.Create(t.Context(), user, in)
			results <- out
			errs <- err
		})
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var expected string
	for out := range results {
		if expected != "" && expected != out.Environment.ID {
			t.Fatal("重试创建了多个云电脑")
		}
		expected = out.Environment.ID
	}
	cost(t, s, user, "3600", 1)
}

func TestComputeKeepsPrepaidHourAcrossAccountCycle(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	zone, _ := time.LoadLocation("Asia/Shanghai")
	now := at.In(zone)
	monday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone).AddDate(0, 0, (8-int(now.Weekday()))%7)
	if !monday.After(*at) {
		monday = monday.AddDate(0, 0, 7)
	}
	*at = monday.Add(-30 * time.Second).UTC()
	e, err := s.Create(t.Context(), user, Creation{"跨期环境", "create-cycle"})
	if err != nil {
		t.Fatal(err)
	}
	*at = at.Add(time.Minute)
	syncState(t, s, host, observe(e.Environment, "running"))
	cost(t, s, user, "3600", 1)
	*at = at.Add(59 * time.Minute)
	syncState(t, s, host, observe(e.Environment, "running"))
	cost(t, s, user, "7200", 2)
	var accounts int
	if err = s.pool.QueryRow(t.Context(), `SELECT count(*) FROM credit_accounts WHERE user_id=$1 AND balance=6400`, user).Scan(&accounts); err != nil || accounts != 2 {
		t.Fatalf("续费未计入新周期：%d，%v", accounts, err)
	}
}

func TestComputeConfigChangeAndDisable(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"调价环境", "create-price"})
	if err != nil {
		t.Fatal(err)
	}
	setComputePrice(t, s, BillingConfig{Enabled: true, CPU: 720_000000})
	syncState(t, s, host, observe(e.Environment, "running"))
	cost(t, s, user, "3600", 1)
	*at = at.Add(time.Hour)
	syncState(t, s, host, observe(e.Environment, "running"))
	cost(t, s, user, "4320", 2)
	setComputePrice(t, s, BillingConfig{})
	*at = at.Add(time.Hour)
	syncState(t, s, host, observe(e.Environment, "running"))
	settle(t, s, false)
	cost(t, s, user, "4320", 2)
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Desired != "running" {
		t.Fatalf("关闭收费不应停止环境：%v", err)
	}
}

func TestComputeBudgetIncludesAllRunningEnvironments(t *testing.T) {
	s, user, host, _ := meterFixture(t, "8000")
	a, err := s.Create(t.Context(), user, Creation{"环境 A", "budget-a"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(t.Context(), user, Creation{"环境 B", "budget-b"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(a.Environment, "running"), observe(b.Environment, "running"))
	budget, enabled, err := s.billingBudget(t.Context(), user)
	if err != nil || !enabled || budget == nil {
		t.Fatalf("预算查询失败：%v", err)
	}
	if budget.Balance.String() != "800" || budget.RunningEnvironments != 2 || budget.RemainingSeconds == nil || *budget.RemainingSeconds != 3600 || !budget.LowBalance {
		t.Fatalf("用户合计预算错误：%+v", budget)
	}
	views := []View{a.Environment, b.Environment}
	if err = s.billingViews(t.Context(), views); err != nil {
		t.Fatal(err)
	}
	if views[0].Billing == nil || views[0].Billing.HourlyCredits.String() != "3600" || views[0].Billing.SettledSeconds != 3600 {
		t.Fatalf("计费展示信息错误：%+v", views[0].Billing)
	}
}

func TestComputePrepaidHourUsesExactDecimal(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	setComputePrice(t, s, BillingConfig{Enabled: true, CPU: 100000})
	e, err := s.Create(t.Context(), user, Creation{"小额计费", "create-small"})
	if err != nil {
		t.Fatal(err)
	}
	for range 59 {
		*at = at.Add(time.Minute)
		syncState(t, s, host, observe(e.Environment, "running"))
	}
	cost(t, s, user, "0.1", 1)
}

func TestComputeRejectsInvalidPricing(t *testing.T) {
	for _, pricing := range []BillingConfig{{Enabled: true, CPU: -1}, {Enabled: true, CPU: 1000000000000_000000}} {
		if err := pricing.validate(); err == nil {
			t.Fatal("接受非法环境价格")
		}
	}
}

func TestComputeDeletedUserStillSettlesHistoricalUsage(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"历史结算", "create-deleted-owner"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	*at = at.Add(30 * time.Second)
	syncState(t, s, host, observe(e.Environment, "running"))
	if _, err = s.pool.Exec(t.Context(), "UPDATE users SET deleted_at=now() WHERE id=$1", user); err != nil {
		t.Fatal(err)
	}
	settle(t, s, false)
	cost(t, s, user, "3600", 1)
	if got := balance(t, s, user); got != "6400" {
		t.Fatalf("删除用户导致历史费用丢失：%s", got)
	}
	if _, err = s.Create(t.Context(), user, Creation{"不应创建", "deleted-admission"}); err == nil {
		t.Fatal("已删除用户仍能新建环境")
	}
	syncState(t, s, host)
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Desired != "stopped" || current.Phase != "running" {
		t.Fatalf("删除用户未触发停止或过早释放资源：%v", err)
	}
	*at = at.Add(8 * 24 * time.Hour)
	budget, _, err := s.billingBudget(t.Context(), user)
	if err != nil || budget != nil {
		t.Fatalf("已删除用户创建了新周期预算：%v %+v", err, budget)
	}
	var accounts int
	if err = s.pool.QueryRow(t.Context(), "SELECT count(*) FROM credit_accounts WHERE user_id=$1", user).Scan(&accounts); err != nil || accounts != 1 {
		t.Fatalf("历史结算为删除用户补发了新周期账户：%d %v", accounts, err)
	}
}

func TestComputeInspectionFailureDoesNotExtendBilling(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"巡检失败", "create-inspection-failure"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	*at = at.Add(30 * time.Second)
	syncState(t, s, host, observe(e.Environment, "running"))
	*at = at.Add(10 * time.Second)
	failed := observe(e.Environment, "failed")
	failed.Reason = "inspect_failed"
	syncState(t, s, host, failed)
	settle(t, s, true)
	cost(t, s, user, "3600", 1)
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Phase != "failed" {
		t.Fatalf("不能把巡检失败当作已停止：%v", err)
	}
	views := []View{view(current)}
	if err = s.billingViews(t.Context(), views); err != nil || views[0].Billing == nil || views[0].Billing.State != "unknown" {
		t.Fatalf("巡检失败应展示未确认计费状态：%v", err)
	}
	syncState(t, s, host, observe(e.Environment, "running"))
	settle(t, s, false)
	cost(t, s, user, "3600", 1)
}

func TestComputePriceUsesExactDecimal(t *testing.T) {
	for _, tt := range []struct {
		cpu, memory, seconds        int64
		cpuPrice, memoryPrice, want string
	}{
		{1000, 2 << 30, 3600, "0.1", "0.025", "0.15"},
		{100, 1 << 28, 60, "0.1", "0.025", "0.000271"},
		{1000, 1 << 30, 1800, "0.000001", "0", "0.000001"},
	} {
		cpu, _ := billing.ParseAmount(tt.cpuPrice)
		memory, _ := billing.ParseAmount(tt.memoryPrice)
		got, err := computePrice(BillingConfig{CPU: cpu, Memory: memory}, tt.cpu, tt.memory, tt.seconds)
		if err != nil || got.String() != tt.want {
			t.Fatalf("计价=%s 期望=%s 错误=%v", got, tt.want, err)
		}
	}
}
