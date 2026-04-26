package vault

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

// TestWithStoreLockCreatesFile verifies that WithStoreLock creates the
// .hulak/.lock file on first call.
func TestWithStoreLockCreatesFile(t *testing.T) {
	projectDir := setupHulakProject(t)

	if err := WithStoreLock(func() error { return nil }); err != nil {
		t.Fatalf("WithStoreLock() error: %v", err)
	}

	lockPath := filepath.Join(projectDir, utils.HiddenProjectName, LockFile)
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("lock file not created: %v", err)
	}
}

// TestWithStoreLockSerializes verifies that concurrent calls do not
// overlap inside fn. We hold each call for a short duration and check
// that no two calls were active at the same time.
func TestWithStoreLockSerializes(t *testing.T) {
	setupHulakProject(t)

	const goroutines = 8
	var (
		wg          sync.WaitGroup
		active      atomic.Int32
		maxActive   atomic.Int32
		invocations atomic.Int32
	)

	for range goroutines {
		wg.Go(func() {
			err := WithStoreLock(func() error {
				cur := active.Add(1)
				// Track high-water mark of concurrent fn invocations.
				for {
					prev := maxActive.Load()
					if cur <= prev || maxActive.CompareAndSwap(prev, cur) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				invocations.Add(1)
				active.Add(-1)
				return nil
			})
			if err != nil {
				t.Errorf("WithStoreLock() error: %v", err)
			}
		})
	}
	wg.Wait()

	if got := invocations.Load(); got != goroutines {
		t.Errorf("invocations = %d, want %d", got, goroutines)
	}
	if got := maxActive.Load(); got != 1 {
		t.Errorf("maxActive = %d, want 1 (lock allowed concurrent execution)", got)
	}
}

// TestWithStoreLockPropagatesError verifies the user's error from fn is
// returned unchanged.
func TestWithStoreLockPropagatesError(t *testing.T) {
	setupHulakProject(t)

	want := os.ErrPermission
	got := WithStoreLock(func() error { return want })
	if got != want {
		t.Errorf("WithStoreLock() error = %v, want %v", got, want)
	}
}
