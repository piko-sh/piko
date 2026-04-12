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

package stablepool

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

type testObj struct {
	Link
	Value int
	Buf   [64]byte
}

func newTestPool(t testing.TB, capacity int) *Pool[testObj] {
	t.Helper()
	initFn := func(o *testObj) { o.Value = 0 }
	cleanFn := func(o *testObj) { o.Value = 0 }
	p, err := New(initFn, cleanFn, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNew(t *testing.T) {
	p := newTestPool(t, 100)
	if p.Cap() != 100 {
		t.Errorf("Cap() = %d, want 100", p.Cap())
	}
}

func TestNewErrors(t *testing.T) {
	cases := []struct {
		name string
		fn   func() (*Pool[testObj], error)
		want error
	}{
		{"nil initialise", func() (*Pool[testObj], error) {
			return New[testObj](nil, func(*testObj) {}, 10)
		}, ErrNoInitialiser},
		{"zero capacity", func() (*Pool[testObj], error) {
			return New(func(*testObj) {}, func(*testObj) {}, 0)
		}, ErrBadCapacity},
		{"negative capacity", func() (*Pool[testObj], error) {
			return New(func(*testObj) {}, func(*testObj) {}, -5)
		}, ErrBadCapacity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.fn()
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPrewarmExactCount(t *testing.T) {
	var initCount, cleanCount atomic.Int64
	initFn := func(*testObj) { initCount.Add(1) }
	cleanFn := func(*testObj) { cleanCount.Add(1) }

	p, err := New(initFn, cleanFn, 1000)
	if err != nil {
		t.Fatal(err)
	}

	if got := initCount.Load(); got != 1000 {
		t.Errorf("initCount after New = %d, want 1000", got)
	}
	if got := cleanCount.Load(); got != 0 {
		t.Errorf("cleanCount after New = %d, want 0", got)
	}

	objs := make([]*testObj, 1000)
	for i := range objs {
		objs[i] = p.Get()
		if objs[i] == nil {
			t.Fatalf("Get %d returned nil", i)
		}
	}
	if got := initCount.Load(); got != 1000 {
		t.Errorf("initCount after 1000 Gets = %d, want 1000", got)
	}

	for _, obj := range objs {
		p.Put(obj)
	}
	if got := cleanCount.Load(); got != 1000 {
		t.Errorf("cleanCount after 1000 Puts = %d, want 1000", got)
	}
}

func TestGetPutCleaner(t *testing.T) {
	p := newTestPool(t, 10)
	obj := p.Get()
	if obj == nil {
		t.Fatal("Get returned nil")
	}
	obj.Value = 42
	p.Put(obj)
	if obj.Value != 0 {
		t.Errorf("cleaner did not reset Value; got %d", obj.Value)
	}
}

func TestPersistentGCSurvivalSlabStable(t *testing.T) {
	p := newTestPool(t, 100)

	if len(p.slabs) != 1 {
		t.Fatalf("expected 1 slab, got %d", len(p.slabs))
	}
	slab := p.slabs[0]
	preGCAddrs := make([]*testObj, len(slab))
	for i := range slab {
		preGCAddrs[i] = slab[i]
		slab[i].Value = i + 1
	}

	for range 5 {

		runtime.GC()
	}

	postSlab := p.slabs[0]
	for i := range postSlab {
		if postSlab[i] != preGCAddrs[i] {
			t.Errorf("slab[%d] address changed: %p → %p", i, preGCAddrs[i], postSlab[i])
		}
		if postSlab[i].Value != i+1 {
			t.Errorf("slab[%d].Value = %d after GC; want %d", i, postSlab[i].Value, i+1)
		}
	}
}

func TestPersistentGCSurvivalGetRoundtrip(t *testing.T) {
	const capacity = 100
	p := newTestPool(t, capacity)

	objs := make([]*testObj, capacity)
	seen := make(map[*testObj]struct{}, capacity)
	for i := range objs {
		obj := p.Get()
		if obj == nil {
			t.Fatalf("Get %d returned nil before GC", i)
		}
		objs[i] = obj
		seen[obj] = struct{}{}
	}
	if len(seen) != capacity {
		t.Fatalf("only %d distinct objects pre-GC; want %d", len(seen), capacity)
	}
	for _, obj := range objs {
		p.Put(obj)
	}

	for range 5 {

		runtime.GC()
	}

	remaining := p.Drain()
	if len(remaining) != capacity {
		t.Errorf("post-GC Drain returned %d objects; want %d", len(remaining), capacity)
	}
	drained := make(map[*testObj]struct{}, capacity)
	for _, obj := range remaining {
		if _, ok := seen[obj]; !ok {
			t.Errorf("drained object not in pre-GC set: %p", obj)
		}
		drained[obj] = struct{}{}
	}
	if len(drained) != capacity {
		t.Errorf("post-GC Drain returned %d distinct objects; want %d", len(drained), capacity)
	}
}

func TestDrain(t *testing.T) {
	p := newTestPool(t, 5)
	objs := make([]*testObj, 5)
	for i := range objs {
		objs[i] = p.Get()
		if objs[i] == nil {
			t.Fatalf("Get %d returned nil", i)
		}
	}
	if obj := p.Get(); obj != nil {
		t.Errorf("Get past capacity returned %p; want nil", obj)
	}

	p.Put(objs[0])
	if obj := p.Get(); obj == nil {
		t.Error("Get after Put returned nil")
	}
}

func TestPutCleanerRunsOutsidePin(t *testing.T) {
	var rescheduled atomic.Bool
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {
		runtime.Gosched()
		rescheduled.Store(true)
	}

	p, err := New(initFn, cleanFn, 10)
	if err != nil {
		t.Fatal(err)
	}

	obj := p.Get()
	p.Put(obj)
	if !rescheduled.Load() {
		t.Error("cleaner Gosched did not flag - possibly run under pin")
	}
}

func TestConcurrentGetPut(t *testing.T) {
	const goroutines = 16
	const iters = 10000
	p := newTestPool(t, 1024)

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for j := range iters {
				obj := p.Get()
				if obj == nil {
					continue
				}
				obj.Value = j
				p.Put(obj)
			}
		})
	}
	wg.Wait()
}

func TestWithShardCount(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 10, WithShardCount[testObj](4))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(p.shards); got != 4 {
		t.Errorf("len(shards) = %d, want 4", got)
	}
}

func TestWithShardCountInvalid(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	_, err := New(initFn, cleanFn, 10, WithShardCount[testObj](0))
	if !errors.Is(err, ErrBadShardCount) {
		t.Errorf("err = %v, want ErrBadShardCount", err)
	}
}

func TestGrowthDisabledByDefault(t *testing.T) {
	p := newTestPool(t, 4)
	objs := make([]*testObj, 4)
	for i := range objs {
		objs[i] = p.Get()
		if objs[i] == nil {
			t.Fatalf("Get %d returned nil at fixed-capacity boundary", i)
		}
	}
	if obj := p.Get(); obj != nil {
		t.Errorf("Get past capacity with no growth returned %p; want nil", obj)
	}
}

func TestGrowthExpands(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 4, WithGrowth[testObj](16))
	if err != nil {
		t.Fatal(err)
	}

	objs := make([]*testObj, 0, 16)
	for i := range 16 {
		obj := p.Get()
		if obj == nil {
			t.Fatalf("Get %d returned nil before growth cap reached", i)
		}
		objs = append(objs, obj)
	}
	if got := p.Cap(); got < 16 {
		t.Errorf("Cap after growth = %d, want >= 16", got)
	}

	if obj := p.Get(); obj != nil {
		t.Errorf("Get past growth cap returned %p; want nil", obj)
	}

	for _, obj := range objs {
		p.Put(obj)
	}
}

func TestGrowthRejectsSmallerMax(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	_, err := New(initFn, cleanFn, 10, WithGrowth[testObj](5))
	if !errors.Is(err, ErrBadGrowth) {
		t.Errorf("err = %v, want ErrBadGrowth", err)
	}
}

func TestCrossShardSteal(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 24, WithShardCount[testObj](8))
	if err != nil {
		t.Fatal(err)
	}

	got := 0
	for {
		obj := p.Get()
		if obj == nil {
			break
		}
		got++
	}
	if got != 24 {
		t.Errorf("drained %d objects across 8 shards; want 24", got)
	}
}

func TestGCAwareGetPut(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 10, WithMode[testObj](ModeGCAware))
	if err != nil {
		t.Fatal(err)
	}

	obj := p.Get()
	if obj == nil {
		t.Fatal("Get returned nil for GCAware mode")
	}
	obj.Value = 42
	p.Put(obj)
}

func TestGCAwareAllocatesOnDemand(t *testing.T) {
	var initCount atomic.Int64
	initFn := func(*testObj) { initCount.Add(1) }
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 5, WithMode[testObj](ModeGCAware))
	if err != nil {
		t.Fatal(err)
	}

	if got := initCount.Load(); got < 5 {
		t.Errorf("init count after construction = %d, want >= 5", got)
	}

	for i := range 100 {
		obj := p.Get()
		if obj == nil {
			t.Fatalf("Get %d returned nil; GCAware should always allocate", i)
		}
	}
}

func TestGCAwareSurvivesGCDifferentlyThanPersistent(t *testing.T) {
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}

	persistent, err := New(initFn, cleanFn, 100)
	if err != nil {
		t.Fatal(err)
	}
	gcAware, err := New(initFn, cleanFn, 100, WithMode[testObj](ModeGCAware))
	if err != nil {
		t.Fatal(err)
	}

	if persistent.Cap() != 100 {
		t.Errorf("persistent.Cap() = %d, want 100", persistent.Cap())
	}
	if gcAware.Cap() != 0 {
		t.Errorf("gcAware.Cap() = %d, want 0 (sync.Pool size is opaque)", gcAware.Cap())
	}

	for range 3 {

		runtime.GC()
	}

	if persistent.Cap() != 100 {
		t.Errorf("persistent.Cap() after GC = %d, want still 100", persistent.Cap())
	}
}

func TestABATorture(t *testing.T) {
	if testing.Short() {
		t.Skip("ABA torture is slow under -short")
	}
	const goroutines = 32
	const ops = 50_000
	const capacity = 256

	initFn := func(o *testObj) { o.Value = 0 }
	cleanFn := func(o *testObj) { o.Value = 0 }
	p, err := New(initFn, cleanFn, capacity)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range ops {
				obj := p.Get()
				if obj == nil {
					continue
				}
				runtime.Gosched()
				obj.Value++
				runtime.Gosched()
				p.Put(obj)
				runtime.Gosched()
			}
		})
	}
	wg.Wait()

	drained := p.Drain()
	seen := make(map[*testObj]struct{}, len(drained))
	for _, obj := range drained {
		if _, dup := seen[obj]; dup {
			t.Errorf("duplicate object in drain: %p (ABA bug detected)", obj)
		}
		seen[obj] = struct{}{}
	}
	if len(drained) > capacity {
		t.Errorf("drain returned %d objects; capacity is %d (over-count)", len(drained), capacity)
	}
}

func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("memory leak check is slow under -short")
	}
	initFn := func(*testObj) {}
	cleanFn := func(*testObj) {}
	p, err := New(initFn, cleanFn, 1024)
	if err != nil {
		t.Fatal(err)
	}

	for range 10_000 {
		obj := p.Get()
		p.Put(obj)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	for range 1_000_000 {
		obj := p.Get()
		p.Put(obj)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	growth := int64(after.HeapInuse) - int64(before.HeapInuse)
	if growth > 1<<20 {
		t.Errorf("HeapInuse grew by %d bytes over 1M ops; want < 1MB", growth)
	}
}

func TestGOMAXPROCSChange(t *testing.T) {
	if testing.Short() {
		t.Skip("flaky under race detector with oversubscribed shards")
	}
	orig := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(orig)

	runtime.GOMAXPROCS(2)
	p := newTestPool(t, 64)

	runtime.GOMAXPROCS(orig)

	const ops = 5_000
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range ops {
				obj := p.Get()
				if obj != nil {
					p.Put(obj)
				}
			}
		})
	}
	wg.Wait()
}

func TestConcurrentGCDuringOps(t *testing.T) {
	const goroutines = 8
	const iters = 5000
	p := newTestPool(t, 256)

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for j := range iters {
				obj := p.Get()
				if obj == nil {
					continue
				}
				obj.Value = j
				p.Put(obj)
			}
		})
	}
	wg.Wait()
	close(stop)
}

type linkSecond struct {
	X int
	Link
}

type linkMissing struct {
	X int
	Y int
}

func TestLinkPlacementCorrect(t *testing.T) {
	_, err := New(func(*testObj) {}, func(*testObj) {}, 10)
	if err != nil {
		t.Fatalf("New for Link-first type failed: %v", err)
	}
}

func TestLinkPlacementWrong(t *testing.T) {
	_, err := New(func(*linkSecond) {}, func(*linkSecond) {}, 10)
	if !errors.Is(err, ErrBadLink) {
		t.Errorf("err = %v, want ErrBadLink for Link not at offset 0", err)
	}
}

func TestLinkMissing(t *testing.T) {
	_, err := New(func(*linkMissing) {}, func(*linkMissing) {}, 10)
	if !errors.Is(err, ErrBadLink) {
		t.Errorf("err = %v, want ErrBadLink for type with no Link", err)
	}
}

func TestLinkPlacementSkippedForGCAware(t *testing.T) {

	_, err := New(func(*linkMissing) {}, func(*linkMissing) {}, 10, WithMode[linkMissing](ModeGCAware))
	if err != nil {
		t.Errorf("New(GCAware) for non-Link type failed: %v; expected pass", err)
	}
}

func TestMustGetWhenDrained(t *testing.T) {
	var initCount atomic.Int64
	initFn := func(o *testObj) { initCount.Add(1); o.Value = -1 }
	p, err := New(initFn, func(*testObj) {}, 2)
	if err != nil {
		t.Fatal(err)
	}

	a := p.Get()
	b := p.Get()
	if a == nil || b == nil {
		t.Fatal("Get returned nil before drain")
	}
	if p.Get() != nil {
		t.Fatal("expected pool to be drained")
	}

	pre := initCount.Load()
	fresh := p.MustGet()
	if fresh == nil {
		t.Fatal("MustGet returned nil")
	}
	if fresh.Value != -1 {
		t.Errorf("MustGet did not run initialise; Value=%d, want -1", fresh.Value)
	}
	if got := initCount.Load() - pre; got != 1 {
		t.Errorf("MustGet ran initialise %d times; want 1", got)
	}
}

func TestMustGetWhenAvailable(t *testing.T) {
	var initCount atomic.Int64
	initFn := func(*testObj) { initCount.Add(1) }
	p, err := New(initFn, func(*testObj) {}, 4)
	if err != nil {
		t.Fatal(err)
	}
	pre := initCount.Load()
	obj := p.MustGet()
	if obj == nil {
		t.Fatal("MustGet returned nil")
	}
	if got := initCount.Load() - pre; got != 0 {
		t.Errorf("MustGet ran initialise %d times when pool had stock; want 0", got)
	}
}

func TestWithMetricsInFlight(t *testing.T) {
	p, err := New(func(*testObj) {}, func(*testObj) {}, 8, WithMetrics[testObj]())
	if err != nil {
		t.Fatal(err)
	}
	if p.InFlight() != 0 {
		t.Errorf("initial InFlight = %d, want 0", p.InFlight())
	}

	a := p.Get()
	if p.InFlight() != 1 {
		t.Errorf("after one Get, InFlight = %d, want 1", p.InFlight())
	}
	b := p.Get()
	if p.InFlight() != 2 {
		t.Errorf("after two Gets, InFlight = %d, want 2", p.InFlight())
	}
	p.Put(a)
	if p.InFlight() != 1 {
		t.Errorf("after one Put, InFlight = %d, want 1", p.InFlight())
	}
	p.Put(b)
	if p.InFlight() != 0 {
		t.Errorf("after two Puts, InFlight = %d, want 0", p.InFlight())
	}

	fresh := p.MustGet()
	if p.InFlight() != 1 {
		t.Errorf("after MustGet, InFlight = %d, want 1", p.InFlight())
	}
	p.Put(fresh)
}

func TestWithoutMetricsInFlightZero(t *testing.T) {
	p, err := New(func(*testObj) {}, func(*testObj) {}, 8)
	if err != nil {
		t.Fatal(err)
	}
	a := p.Get()
	_ = p.Get()
	if p.InFlight() != 0 {
		t.Errorf("InFlight = %d without WithMetrics; want 0", p.InFlight())
	}
	p.Put(a)
	if p.InFlight() != 0 {
		t.Errorf("InFlight = %d after Put without WithMetrics; want 0", p.InFlight())
	}
}

func TestNilCleanerAllowed(t *testing.T) {
	p, err := New[testObj](func(*testObj) {}, nil, 4)
	if err != nil {
		t.Fatalf("New with nil cleaner failed: %v", err)
	}
	obj := p.Get()
	if obj == nil {
		t.Fatal("Get returned nil")
	}
	obj.Value = 99
	p.Put(obj)

	if obj.Value != 99 {
		t.Errorf("nil cleaner ran or value mutated: Value = %d, want 99", obj.Value)
	}
}

func TestFactoryInit(t *testing.T) {
	var factoryCount atomic.Int64
	factory := func() *testObj {
		factoryCount.Add(1)
		return &testObj{Value: 42}
	}
	p, err := New(FactoryInit(factory), func(*testObj) {}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := factoryCount.Load(); got != 5 {
		t.Errorf("factory called %d times during pre-warm; want 5", got)
	}
	obj := p.Get()
	if obj == nil {
		t.Fatal("Get returned nil")
	}
	if obj.Value != 42 {
		t.Errorf("FactoryInit did not copy factory result; Value = %d, want 42", obj.Value)
	}
}
