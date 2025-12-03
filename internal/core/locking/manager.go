package locking

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrWouldBlock = errors.New("lock would block")
	ErrNotOwner   = errors.New("not lock owner")
)

type LockType int16

// LockInfo describes an active lock
type LockInfo struct {
	Owner []byte
	Start int64
	End   int64
	Type  LockType // F_RDLCK / F_WRLCK
	PID   int
}

const (
	F_RDLCK LockType = 1
	F_WRLCK LockType = 2
	F_UNLCK LockType = 3
)

type lockEntry struct {
	info LockInfo
}

type lockList struct {
	mu    sync.Mutex
	cond  *sync.Cond
	locks []lockEntry
}

type LockManager struct {
	mu    sync.Mutex
	table map[string]*lockList
}

func NewLockManager() *LockManager {
	return &LockManager{table: make(map[string]*lockList)}
}

func (lm *LockManager) getList(key string) *lockList {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	l, ok := lm.table[key]
	if !ok {
		l = &lockList{}
		l.cond = sync.NewCond(&l.mu)
		lm.table[key] = l
	}
	return l
}

func overlap(aStart, aEnd, bStart, bEnd int64) bool {
	if aEnd <= 0 {
		aEnd = int64(^uint64(0) >> 1) // large
	}
	if bEnd <= 0 {
		bEnd = int64(^uint64(0) >> 1)
	}
	return aStart < bEnd && bStart < aEnd
}

// Acquire tries to acquire a lock non-blocking. Returns ErrWouldBlock if conflict.
func (lm *LockManager) Acquire(key string, owner []byte, start, end int64, lockType LockType) error {
	l := lm.getList(key)
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, e := range l.locks {
		if overlap(start, end, e.info.Start, e.info.End) {
			// shared vs shared ok
			if e.info.Type == F_WRLCK || lockType == F_WRLCK {
				return ErrWouldBlock
			}
		}
	}

	le := lockEntry{info: LockInfo{Owner: append([]byte(nil), owner...), Start: start, End: end, Type: lockType, PID: -1}}
	l.locks = append(l.locks, le)
	return nil
}

// AcquireWait waits until the lock can be acquired or context is cancelled.
func (lm *LockManager) AcquireWait(ctx context.Context, key string, owner []byte, start, end int64, lockType LockType) error {
	l := lm.getList(key)
	l.mu.Lock()
	defer l.mu.Unlock()

	for {
		conflict := false
		for _, e := range l.locks {
			if overlap(start, end, e.info.Start, e.info.End) {
				if e.info.Type == F_WRLCK || lockType == F_WRLCK {
					conflict = true
					break
				}
			}
		}
		if !conflict {
			le := lockEntry{info: LockInfo{Owner: append([]byte(nil), owner...), Start: start, End: end, Type: lockType, PID: -1}}
			l.locks = append(l.locks, le)
			return nil
		}

		// wait with context
		done := make(chan struct{})
		go func() {
			l.cond.Wait()
			close(done)
		}()

		select {
		case <-ctx.Done():
			// Wake up any sleepers and return
			l.cond.Broadcast()
			return ctx.Err()
		case <-done:
			// try again
		case <-time.After(5 * time.Second):
			// periodic wake to recheck context
		}
	}
}

// Release releases locks that match owner and overlapping range. Returns ErrNotOwner if no matching lock.
func (lm *LockManager) Release(key string, owner []byte, start, end int64) error {
	l := lm.getList(key)
	l.mu.Lock()
	defer l.mu.Unlock()

	removed := false
	var remaining []lockEntry
	for _, e := range l.locks {
		if string(e.info.Owner) == string(owner) && overlap(start, end, e.info.Start, e.info.End) {
			removed = true
			continue
		}
		remaining = append(remaining, e)
	}
	if !removed {
		return ErrNotOwner
	}
	l.locks = remaining
	l.cond.Broadcast()
	return nil
}

// Query returns one conflicting lock (if any). Returns (info, true) if conflict exists.
func (lm *LockManager) Query(key string, start, end int64) (LockInfo, bool) {
	l := lm.getList(key)
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, e := range l.locks {
		if overlap(start, end, e.info.Start, e.info.End) {
			return e.info, true
		}
	}
	return LockInfo{}, false
}
