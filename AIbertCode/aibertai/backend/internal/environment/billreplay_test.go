package environment

import (
	"encoding/json"
	"testing"
	"time"

	"aibertai.local/backend/internal/environment/sqlc"
)

func TestRepeatedExchangeCannotAdvanceUsageBillingOrReopenPrices(t *testing.T) {
	s, user, host, at := meterFixture(t, "10000")
	e, err := s.Create(t.Context(), user, Creation{"replay", "replay-create"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Exchange{Protocol: 1, Results: []Observation{observe(e.Environment, "running")}})
	stamp := time.Now().UTC()
	if _, err := s.ExchangeAt(t.Context(), host, raw, stamp); err != nil {
		t.Fatal(err)
	}
	*at = at.Add(30 * time.Second)
	stamp = stamp.Add(time.Millisecond)
	if _, err := s.ExchangeAt(t.Context(), host, raw, stamp); err != nil {
		t.Fatal(err)
	}
	*at = at.Add(30 * time.Second)
	if _, err := s.ExchangeAt(t.Context(), host, raw, stamp); err == nil {
		t.Fatal("重复报告被接受")
	}
	settle(t, s, false)
	cost(t, s, user, "3600", 1)
	setComputePrice(t, s, BillingConfig{Enabled: true, CPU: 720_000000})
	*at = at.Add(time.Hour)
	if _, err := s.ExchangeAt(t.Context(), host, raw, stamp); err == nil {
		t.Fatal("旧报告可开启新价格时段")
	}
	cost(t, s, user, "3600", 1)
	current, err := sqlc.New(s.pool).GetEnvironment(t.Context(), sqlc.GetEnvironmentParams{ID: e.Environment.ID})
	if err != nil || current.Desired != "running" {
		t.Fatal("拒绝重放改变环境目标", err)
	}
}
