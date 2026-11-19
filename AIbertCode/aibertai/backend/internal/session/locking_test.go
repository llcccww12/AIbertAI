package session

import (
	"context"
	"testing"
	"time"

	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/session/sqlc"
)

func TestSessionUpdateAllowsChildReference(t *testing.T) {
	s, _, user, _ := sessionDBFixture(t)
	parent := resource.ID()
	if err := s.EnsureSession(t.Context(), user, parent, "", ""); err != nil {
		t.Fatal(err)
	}
	update, err := s.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer update.Rollback(context.Background())
	if _, err := readSession(t.Context(), update, parent, true); err != nil {
		t.Fatal(err)
	}
	child, err := s.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer child.Rollback(context.Background())
	if _, err := child.Exec(t.Context(), "SET LOCAL lock_timeout = '1s'"); err != nil {
		t.Fatal(err)
	}
	if err := sqlc.New(child).InsertPlaceholder(t.Context(), sqlc.InsertPlaceholderParams{
		ID: resource.ID(), OwnerUserID: user, ParentSessionID: parent, StartedAt: time.Now(),
	}); err != nil {
		t.Fatalf("更新会话阻塞子会话外键检查: %v", err)
	}
	if err := child.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestTurnMirrorPreservesSessionLockCompatibility(t *testing.T) {
	s, _, user, _ := sessionDBFixture(t)
	started := time.Now().UTC()
	id := turnSeedSession(t, s, user, started)
	report, err := s.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer report.Rollback(context.Background())
	if _, err := report.Exec(t.Context(), "SET LOCAL lock_timeout = '1s'"); err != nil {
		t.Fatal(err)
	}
	if _, err := readSession(t.Context(), report, id, true); err != nil {
		t.Fatal(err)
	}
	settlement, err := s.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer settlement.Rollback(context.Background())
	if _, err := settlement.Exec(t.Context(), "SET LOCAL lock_timeout = '1s'"); err != nil {
		t.Fatal(err)
	}
	// Inserting model_calls takes this mode through its session foreign key.
	if _, err := settlement.Exec(t.Context(), "SELECT id FROM sessions WHERE id=$1 FOR KEY SHARE", id); err != nil {
		t.Fatal(err)
	}
	if _, err := report.Exec(t.Context(), `INSERT INTO session_turns (
		session_id,turn_index,facts_version,report_hash,input_seq,started_at,ended_at,stop_reason,resources_snapshot_id,input_kind
	) VALUES($1,1,1,'hash',1,$2,$2,'complete','snapshot','prompt')`, id, started); err != nil {
		t.Fatalf("轮次镜像因会话外键检查而阻塞: %v", err)
	}
	if err := report.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := settlement.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	turnInvariant(t, s, false)
}
