// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package shutdown_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/shutdown"
)

func TestManager_Register(t *testing.T) {
	t.Run("registers cleanup function", func(t *testing.T) {
		manager := shutdown.NewManager()

		manager.Register(context.Background(), "test-cleanup", func(ctx context.Context) error {
			return nil
		})

		if got := manager.Count(); got != 1 {
			t.Errorf("Count() = %d, want 1", got)
		}
	})

	t.Run("registers multiple cleanup functions", func(t *testing.T) {
		manager := shutdown.NewManager()

		manager.Register(context.Background(), "cleanup-1", func(ctx context.Context) error { return nil })
		manager.Register(context.Background(), "cleanup-2", func(ctx context.Context) error { return nil })
		manager.Register(context.Background(), "cleanup-3", func(ctx context.Context) error { return nil })

		if got := manager.Count(); got != 3 {
			t.Errorf("Count() = %d, want 3", got)
		}
	})
}

func TestManager_Cleanup(t *testing.T) {
	t.Run("executes cleanup functions in reverse registration order (LIFO)", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executionOrder []string

		manager.Register(context.Background(), "first", func(ctx context.Context) error {
			executionOrder = append(executionOrder, "first")
			return nil
		})
		manager.Register(context.Background(), "second", func(ctx context.Context) error {
			executionOrder = append(executionOrder, "second")
			return nil
		})
		manager.Register(context.Background(), "third", func(ctx context.Context) error {
			executionOrder = append(executionOrder, "third")
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		want := []string{"third", "second", "first"}
		if len(executionOrder) != len(want) {
			t.Fatalf("executed %d functions, want %d", len(executionOrder), len(want))
		}

		for i, v := range want {
			if executionOrder[i] != v {
				t.Errorf("executionOrder[%d] = %q, want %q", i, executionOrder[i], v)
			}
		}
	})

	t.Run("continues execution even if cleanup function fails", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executed []string

		manager.Register(context.Background(), "first", func(ctx context.Context) error {
			executed = append(executed, "first")
			return nil
		})
		manager.Register(context.Background(), "second", func(ctx context.Context) error {
			executed = append(executed, "second")
			return errors.New("intentional failure")
		})
		manager.Register(context.Background(), "third", func(ctx context.Context) error {
			executed = append(executed, "third")
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		if got := len(executed); got != 3 {
			t.Errorf("executed %d functions, want 3", got)
		}
	})

	t.Run("respects timeout and skips remaining functions", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executed []string

		manager.Register(context.Background(), "skipped", func(ctx context.Context) error {
			executed = append(executed, "skipped")
			return nil
		})
		manager.Register(context.Background(), "slow", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			executed = append(executed, "slow")
			return nil
		})
		manager.Register(context.Background(), "runs-first", func(ctx context.Context) error {
			executed = append(executed, "runs-first")
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 50*time.Millisecond)

		if got := len(executed); got > 2 {
			t.Errorf("executed %d functions, want at most 2 due to timeout", got)
		}
		if len(executed) > 0 && executed[0] != "runs-first" {
			t.Errorf("first execution was %q, want 'runs-first'", executed[0])
		}
	})
}

func TestManager_Reset(t *testing.T) {
	t.Run("clears all registered cleanup functions", func(t *testing.T) {
		manager := shutdown.NewManager()

		manager.Register(context.Background(), "cleanup-1", func(ctx context.Context) error { return nil })
		manager.Register(context.Background(), "cleanup-2", func(ctx context.Context) error { return nil })

		if got := manager.Count(); got != 2 {
			t.Fatalf("Count() = %d, want 2", got)
		}

		manager.Reset()

		if got := manager.Count(); got != 0 {
			t.Errorf("Count() after Reset() = %d, want 0", got)
		}
	})

	t.Run("allows registering new functions after reset", func(t *testing.T) {
		manager := shutdown.NewManager()

		manager.Register(context.Background(), "cleanup-1", func(ctx context.Context) error { return nil })
		manager.Reset()
		manager.Register(context.Background(), "cleanup-2", func(ctx context.Context) error { return nil })

		if got := manager.Count(); got != 1 {
			t.Errorf("Count() after Reset() and new Register() = %d, want 1", got)
		}
	})
}

func TestManager_ListenAndShutdownWithSignal(t *testing.T) {
	t.Run("executes cleanup when signal is received", func(t *testing.T) {
		manager := shutdown.NewManager()
		executed := false

		manager.Register(context.Background(), "test-cleanup", func(ctx context.Context) error {
			executed = true
			return nil
		})

		ctx := context.Background()
		sigChan := make(chan os.Signal, 1)

		done := make(chan struct{})
		go func() {
			manager.ListenAndShutdownWithSignal(ctx, 5*time.Second, sigChan)
			close(done)
		}()

		sigChan <- os.Interrupt

		select {
		case <-done:

		case <-time.After(1 * time.Second):
			t.Fatal("ListenAndShutdownWithSignal did not complete within timeout")
		}

		if !executed {
			t.Error("cleanup function was not executed")
		}
	})
}

func TestManager_Isolation(t *testing.T) {
	t.Run("multiple managers do not interfere with each other", func(t *testing.T) {
		manager1 := shutdown.NewManager()
		manager2 := shutdown.NewManager()

		manager1.Register(context.Background(), "mgr1-cleanup", func(ctx context.Context) error { return nil })
		manager2.Register(context.Background(), "mgr2-cleanup-1", func(ctx context.Context) error { return nil })
		manager2.Register(context.Background(), "mgr2-cleanup-2", func(ctx context.Context) error { return nil })

		if got := manager1.Count(); got != 1 {
			t.Errorf("manager1.Count() = %d, want 1", got)
		}
		if got := manager2.Count(); got != 2 {
			t.Errorf("manager2.Count() = %d, want 2", got)
		}
	})
}

func TestManager_PanicRecovery(t *testing.T) {
	t.Run("recovers from panic in cleanup function and continues", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executed []string

		manager.Register(context.Background(), "first", func(ctx context.Context) error {
			executed = append(executed, "first")
			return nil
		})
		manager.Register(context.Background(), "panics", func(ctx context.Context) error {
			executed = append(executed, "panics")
			panic("intentional panic for testing")
		})
		manager.Register(context.Background(), "third", func(ctx context.Context) error {
			executed = append(executed, "third")
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		if got := len(executed); got != 3 {
			t.Errorf("executed %d functions, want 3", got)
		}

		want := []string{"third", "panics", "first"}
		for i, v := range want {
			if i < len(executed) && executed[i] != v {
				t.Errorf("executionOrder[%d] = %q, want %q", i, executed[i], v)
			}
		}
	})
}

func TestManager_RegistrationDuringCleanup(t *testing.T) {
	t.Run("rejects registration during cleanup", func(t *testing.T) {
		manager := shutdown.NewManager()
		registeredDuringCleanup := false

		manager.Register(context.Background(), "slow-cleanup", func(ctx context.Context) error {
			initialCount := manager.Count()
			manager.Register(context.Background(), "late-registration", func(ctx context.Context) error {
				registeredDuringCleanup = true
				return nil
			})
			if manager.Count() != initialCount {
				t.Error("registration during cleanup should be rejected")
			}
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		if registeredDuringCleanup {
			t.Error("late-registration function should not have executed")
		}
	})
}

func TestManager_IsCleanupInProgress(t *testing.T) {
	t.Run("returns false when no cleanup is running", func(t *testing.T) {
		manager := shutdown.NewManager()

		if manager.IsCleanupInProgress() {
			t.Error("IsCleanupInProgress() should be false before cleanup starts")
		}
	})

	t.Run("returns true during cleanup", func(t *testing.T) {
		manager := shutdown.NewManager()
		wasInProgress := false

		manager.Register(context.Background(), "check-in-progress", func(ctx context.Context) error {
			wasInProgress = manager.IsCleanupInProgress()
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		if !wasInProgress {
			t.Error("IsCleanupInProgress() should be true during cleanup")
		}

		if manager.IsCleanupInProgress() {
			t.Error("IsCleanupInProgress() should be false after cleanup completes")
		}
	})
}

func TestManager_PerFunctionTimeout(t *testing.T) {
	t.Run("each function gets individual timeout from budget", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executed []string

		manager.Register(context.Background(), "first", func(ctx context.Context) error {
			executed = append(executed, "first")
			return nil
		})
		manager.Register(context.Background(), "second", func(ctx context.Context) error {
			executed = append(executed, "second")
			return nil
		})
		manager.Register(context.Background(), "third", func(ctx context.Context) error {
			executed = append(executed, "third")
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 3*time.Second)

		if len(executed) != 3 {
			t.Errorf("executed %d functions, want 3", len(executed))
		}
	})
}

func TestManager_DefaultTimeout(t *testing.T) {
	t.Run("DefaultTimeout constant is set to reasonable value", func(t *testing.T) {
		if shutdown.DefaultTimeout < 10*time.Second {
			t.Errorf("DefaultTimeout = %v, should be at least 10s for production", shutdown.DefaultTimeout)
		}

		if shutdown.DefaultTimeout > time.Minute {
			t.Errorf("DefaultTimeout = %v, should not exceed 1 minute", shutdown.DefaultTimeout)
		}
	})

	t.Run("MinFunctionTimeout constant is set", func(t *testing.T) {
		if shutdown.MinFunctionTimeout < 100*time.Millisecond {
			t.Errorf("MinFunctionTimeout = %v, should be at least 100ms", shutdown.MinFunctionTimeout)
		}
	})
}

func TestPackageLevelRegisterAndCleanup(t *testing.T) {
	t.Run("register and cleanup via package-level functions", func(t *testing.T) {
		executed := false

		shutdown.Register(context.Background(), "pkg-level-test", func(ctx context.Context) error {
			executed = true
			return nil
		})

		ctx := context.Background()
		shutdown.Cleanup(ctx, 5*time.Second)

		if !executed {
			t.Error("cleanup function registered via package-level Register was not executed")
		}
	})
}

func TestManager_Cleanup_NoFunctions(t *testing.T) {
	t.Run("completes without error when no functions registered", func(t *testing.T) {
		manager := shutdown.NewManager()
		ctx := context.Background()

		manager.Cleanup(ctx, 5*time.Second)

		if manager.Count() != 0 {
			t.Errorf("Count() = %d, want 0", manager.Count())
		}
	})
}

func TestManager_Cleanup_FunctionReceivesContextWithDeadline(t *testing.T) {
	t.Run("cleanup function receives context with deadline", func(t *testing.T) {
		manager := shutdown.NewManager()
		var receivedDeadline bool

		manager.Register(context.Background(), "check-deadline", func(ctx context.Context) error {
			_, receivedDeadline = ctx.Deadline()
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)

		if !receivedDeadline {
			t.Error("cleanup function should receive a context with a deadline")
		}
	})
}

func TestManager_Cleanup_CanBeCalledMultipleTimes(t *testing.T) {
	t.Run("cleanup can be called more than once", func(t *testing.T) {
		manager := shutdown.NewManager()
		callCount := 0

		manager.Register(context.Background(), "repeatable", func(ctx context.Context) error {
			callCount++
			return nil
		})

		ctx := context.Background()
		manager.Cleanup(ctx, 5*time.Second)
		manager.Cleanup(ctx, 5*time.Second)

		if callCount != 2 {
			t.Errorf("cleanup function called %d times, want 2", callCount)
		}
	})
}

func TestManager_ConcurrentRegistration(t *testing.T) {
	t.Run("concurrent registrations are safe", func(t *testing.T) {
		manager := shutdown.NewManager()
		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() {
				manager.Register(context.Background(), "concurrent", func(ctx context.Context) error {
					return nil
				})
			})
		}

		wg.Wait()

		if got := manager.Count(); got != 10 {
			t.Errorf("Count() = %d, want 10", got)
		}
	})
}

func TestManager_ListenAndShutdownWithSignal_LIFO(t *testing.T) {
	t.Run("signal path executes cleanup in LIFO order", func(t *testing.T) {
		manager := shutdown.NewManager()
		var order []string

		manager.Register(context.Background(), "first", func(ctx context.Context) error {
			order = append(order, "first")
			return nil
		})
		manager.Register(context.Background(), "second", func(ctx context.Context) error {
			order = append(order, "second")
			return nil
		})

		ctx := context.Background()
		sigChan := make(chan os.Signal, 1)

		done := make(chan struct{})
		go func() {
			manager.ListenAndShutdownWithSignal(ctx, 5*time.Second, sigChan)
			close(done)
		}()

		sigChan <- os.Interrupt

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("did not complete within timeout")
		}

		want := []string{"second", "first"}
		if len(order) != len(want) {
			t.Fatalf("executed %d functions, want %d", len(order), len(want))
		}
		for i, v := range want {
			if order[i] != v {
				t.Errorf("order[%d] = %q, want %q", i, order[i], v)
			}
		}
	})
}

func TestManager_ListenAndShutdownWithSignal_ContinuesOnError(t *testing.T) {
	t.Run("signal path continues after function error", func(t *testing.T) {
		manager := shutdown.NewManager()
		var executed []string

		manager.Register(context.Background(), "after-error", func(ctx context.Context) error {
			executed = append(executed, "after-error")
			return nil
		})
		manager.Register(context.Background(), "fails", func(ctx context.Context) error {
			executed = append(executed, "fails")
			return errors.New("intentional")
		})

		ctx := context.Background()
		sigChan := make(chan os.Signal, 1)

		done := make(chan struct{})
		go func() {
			manager.ListenAndShutdownWithSignal(ctx, 5*time.Second, sigChan)
			close(done)
		}()

		sigChan <- os.Interrupt

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("did not complete within timeout")
		}

		if len(executed) != 2 {
			t.Errorf("executed %d functions, want 2", len(executed))
		}
	})
}

func TestManager_Cleanup_CancelledContext(t *testing.T) {
	t.Run("cleanup runs even with already-cancelled parent context", func(t *testing.T) {
		manager := shutdown.NewManager()
		executed := false

		manager.Register(context.Background(), "check", func(ctx context.Context) error {
			executed = true
			return nil
		})

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(fmt.Errorf("test: simulating cancelled context"))

		manager.Cleanup(ctx, 5*time.Second)

		if !executed {
			t.Error("cleanup function should still execute with cancelled parent context")
		}
	})
}
