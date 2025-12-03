package locking

import (
	"context"
	"testing"
	"time"
)

func TestAcquireReleaseQuery(t *testing.T) {
	lm := NewLockManager()
	owner1 := []byte("owner1")
	owner2 := []byte("owner2")

	// owner1 acquires write lock
	if err := lm.Acquire("file", owner1, 0, 100, F_WRLCK); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// owner2 should be blocked for overlapping write or read when owner1 holds write
	if err := lm.Acquire("file", owner2, 0, 50, F_RDLCK); err != ErrWouldBlock {
		t.Fatalf("expected ErrWouldBlock, got %v", err)
	}

	// Query should find a conflicting lock
	if info, ok := lm.Query("file", 0, 10); !ok {
		t.Fatalf("expected Query to report conflict")
	} else {
		if string(info.Owner) != string(owner1) {
			t.Fatalf("Query returned wrong owner: %s", string(info.Owner))
		}
	}

	// Release by wrong owner should fail
	if err := lm.Release("file", owner2, 0, 100); err != ErrNotOwner {
		t.Fatalf("expected ErrNotOwner on release by non-owner, got %v", err)
	}

	// Release by correct owner should succeed
	if err := lm.Release("file", owner1, 0, 100); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// after release, owner2 can acquire a read lock
	if err := lm.Acquire("file", owner2, 0, 100, F_RDLCK); err != nil {
		t.Fatalf("Acquire after release failed: %v", err)
	}
}

func TestAcquireWait(t *testing.T) {
	lm := NewLockManager()
	owner1 := []byte("o1")
	owner2 := []byte("o2")

	// owner1 acquires a write lock for a long range
	if err := lm.Acquire("key", owner1, 0, 1000, F_WRLCK); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// after a short delay, release in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = lm.Release("key", owner1, 0, 1000)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// owner2 should wait and acquire once released
	if err := lm.AcquireWait(ctx, "key", owner2, 0, 1000, F_WRLCK); err != nil {
		t.Fatalf("AcquireWait failed: %v", err)
	}
}

func TestNonOverlappingLocks(t *testing.T) {
	lm := NewLockManager()
	ownerA := []byte("A")
	ownerB := []byte("B")

	// non-overlapping ranges should allow concurrent locks
	if err := lm.Acquire("file", ownerA, 0, 10, F_WRLCK); err != nil {
		t.Fatalf("Acquire A failed: %v", err)
	}
	if err := lm.Acquire("file", ownerB, 20, 30, F_WRLCK); err != nil {
		t.Fatalf("Acquire B failed: %v", err)
	}
}
