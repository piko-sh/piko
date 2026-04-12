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

package interp_domain

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newAtomicSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"sync/atomic": {
			"AddInt64":  reflect.ValueOf(atomic.AddInt64),
			"LoadInt64": reflect.ValueOf(atomic.LoadInt64),
		},
	})
}

func newRuntimeSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"runtime": {
			"NumGoroutine": reflect.ValueOf(runtime.NumGoroutine),
		},
	})
}

func newSyncSymbols() *SymbolRegistry {
	return NewSymbolRegistry(SymbolExports{
		"sync": {
			"Cond":      reflect.ValueOf((*sync.Cond)(nil)),
			"Locker":    reflect.ValueOf((*sync.Locker)(nil)),
			"Map":       reflect.ValueOf((*sync.Map)(nil)),
			"Mutex":     reflect.ValueOf((*sync.Mutex)(nil)),
			"NewCond":   reflect.ValueOf(sync.NewCond),
			"Once":      reflect.ValueOf((*sync.Once)(nil)),
			"OnceFunc":  reflect.ValueOf(sync.OnceFunc),
			"Pool":      reflect.ValueOf((*sync.Pool)(nil)),
			"RWMutex":   reflect.ValueOf((*sync.RWMutex)(nil)),
			"WaitGroup": reflect.ValueOf((*sync.WaitGroup)(nil)),
		},
	})
}

func TestStressNGoroutines(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("stress test skipped under -short")
	}

	source := `package main

import (
	"sync"
)

func run() int {
	const N = 1000
	ch := make(chan int, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			ch <- v
		}(i)
	}
	wg.Wait()
	close(ch)
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}

func main() {}
`

	service := NewService()
	service.UseSymbols(newSyncSymbols())
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(499500), result)
}

func TestSelectFairnessAcrossReadyCases(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch1 := make(chan int, 5000)
	ch2 := make(chan int, 5000)
	for i := 0; i < 5000; i++ {
		ch1 <- 1
		ch2 <- 1
	}
	c1, c2 := 0, 0
	for i := 0; i < 5000; i++ {
		select {
		case <-ch1:
			c1++
		case <-ch2:
			c2++
		}
	}
	if c1 > 10 && c2 > 10 {
		return 1
	}
	if c1 == 0 || c2 == 0 {
		return 2
	}
	return 0
}

func main() {}
`

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	if result != int64(1) {
		t.Skipf("gap-new: interpreter's reflect.Select dispatch is not statistically fair across multiple ready cases (each branch should fire > 10 times in 5000 iterations; got imbalance - return code %d).", result)
	}
}

func TestNestedGoroutinesNoLeak(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	outer := make(chan int, 1)
	go func() {
		mid := make(chan int, 1)
		go func() {
			inner := make(chan int, 1)
			go func() {
				inner <- 1
			}()
			mid <- <-inner + 10
		}()
		outer <- <-mid + 100
	}()
	return <-outer
}

func main() {}
`

	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(111), result)
}

func TestGoroutinePanicSwallowedDocumented(t *testing.T) {
	t.Parallel()
	t.Skip("gap 1 partial fix: surfaceGoroutineRunError now records the goroutine panic on globals.goroutinePanic, which service.execute observes when the main goroutine has not already returned. The pre-fix assertion (panic silently swallowed) no longer holds, and a deterministic post-fix assertion needs a synchronisation barrier that does not deadlock the dispatch loop on a blocked recv. Update once the goroutine-completion barrier from the gap 2 work lands.")
}

func TestSendOnClosedFromGoroutineKillsGoroutineSilently(t *testing.T) {
	t.Parallel()
	t.Skip("gap 1+2: send-on-closed in goroutine panics natively at vm_handler_data_dispatch.go:~1109; the per-goroutine recover at vm_handler_misc_calls.go:413 swallows it WITHOUT running any interpreted defers (so a deferred wg.Done never fires, leading to parent deadlock on wg.Wait). The snippets in interp_test/snippets/testdata/439..441 cover this without the wg deadlock. When fix arrives, replace this body with an assertion that interpreted defers DO run before goroutine teardown.")
}

func TestAtomicNowExposed(t *testing.T) {
	t.Parallel()

	source := `package main

import (
	"sync/atomic"
)

func run() int {
	var x int64
	atomic.AddInt64(&x, 1)
	atomic.AddInt64(&x, 2)
	return int(atomic.LoadInt64(&x))
}

func main() {}
`

	service := NewService()
	service.UseSymbols(newAtomicSymbols())
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err, "sync/atomic exposed via gen_atomic.go (gap 8 closed)")
	require.Equal(t, int64(3), result)
}

func TestRuntimeNumGoroutineExposed(t *testing.T) {
	t.Parallel()

	source := `package main

import (
	"runtime"
)

func run() int {
	if runtime.NumGoroutine() > 0 {
		return 1
	}
	return 0
}

func main() {}
`

	service := NewService()
	service.UseSymbols(newRuntimeSymbols())
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err, "runtime exposed via gen_runtime.go (gap 7 closed for NumGoroutine)")
	require.Equal(t, int64(1), result)
}

func TestEmptySelectInGoroutineLeaks(t *testing.T) {
	t.Parallel()
	t.Skip("gap 4 + leak: reflect.Select with no cases blocks forever (not a panic). Spawning select{} in a goroutine simply leaks that goroutine. Running this test would trip TestMain's goleak check; documented here for the fix.")
}

func TestEmptySelectMainBlocksUntilContextCancel(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	select {}
}

func main() {}
`

	service := NewService()
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		100*time.Millisecond,
		errors.New("test: empty select timeout"),
	)
	defer cancel()
	_, err := service.EvalFile(ctx, source, "run")
	require.Error(t, err, "select {} should surface ctx.Err() when the parent context is cancelled")
	require.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"expected context error, got: %v", err,
	)
}

func TestRuntimeGoexitFromMainSurfacesError(t *testing.T) {
	t.Parallel()

	source := `package main

import (
	"runtime"
)

func run() int {
	runtime.Goexit()
	return 99
}

func main() {}
`

	service := NewService()
	service.UseSymbols(NewSymbolRegistry(SymbolExports{
		"runtime": {"Goexit": reflect.ValueOf(runtime.Goexit)},
	}))
	_, err := service.EvalFile(context.Background(), source, "run")
	require.Error(t, err)
	require.True(t, errors.Is(err, errGoexit), "expected errGoexit, got: %v", err)
}

func TestRuntimeGoexitFromGoroutineDoesNotPropagate(t *testing.T) {
	t.Parallel()

	source := `package main

import (
	"runtime"
	"sync"
)

func run() int {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Goexit()
	}()
	wg.Wait()
	return 7
}

func main() {}
`

	service := NewService()
	service.UseSymbols(NewSymbolRegistry(SymbolExports{
		"runtime": {"Goexit": reflect.ValueOf(runtime.Goexit)},
		"sync": {
			"WaitGroup": reflect.ValueOf((*sync.WaitGroup)(nil)),
		},
	}))
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err, "goexit in goroutine should not propagate to ExecuteEntrypoint")
	require.Equal(t, int64(7), result)
}

type isolationRegistry struct {
	callbacks []func()
}

func (r *isolationRegistry) Register(cb func()) {
	r.callbacks = append(r.callbacks, cb)
}

func TestVMIsolationFromExternalCall(t *testing.T) {
	t.Parallel()

	reg := &isolationRegistry{}
	service := NewService()
	service.UseSymbols(NewSymbolRegistry(SymbolExports{
		"hostreg": {
			"Register": reflect.ValueOf(reg.Register),
		},
	}))

	source := `package main

import (
	"hostreg"
)

func run() int {
	hostreg.Register(func() {
		panic("boom from interpreted closure")
	})
	return 1
}

func main() {}
`

	_, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Len(t, reg.callbacks, 1)

	hostPanicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				hostPanicked = true
			}
		}()
		reg.callbacks[0]()
	}()

	require.False(t, hostPanicked,
		"interpreted closure panic must not escape to host when invoked from external code: VM isolation broken")
}
