package billing

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRetryUsageSaveAfterDeadlock(t *testing.T) {
	deadlock := fmt.Errorf("save model activity: %w", &pgconn.PgError{Code: "40P01"})
	attempts := 0
	settle, err := retryUsageSave(t.Context(), func() (bool, error) {
		attempts++
		if attempts == 1 {
			return false, deadlock
		}
		return true, nil
	})
	if err != nil || !settle || attempts != 2 {
		t.Fatalf("attempts=%d settle=%v err=%v", attempts, settle, err)
	}

	attempts = 0
	_, err = retryUsageSave(t.Context(), func() (bool, error) {
		attempts++
		return false, deadlock
	})
	if !errors.Is(err, deadlock) || attempts != 3 {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}

	attempts = 0
	invalid := &pgconn.PgError{Code: "23503"}
	_, err = retryUsageSave(t.Context(), func() (bool, error) {
		attempts++
		return false, invalid
	})
	if !errors.Is(err, invalid) || attempts != 1 {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
}
