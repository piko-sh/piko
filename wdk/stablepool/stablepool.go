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
	"fmt"
	"math/bits"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// gcAwareDrainAttempts caps the number of consecutive nil Get() results the GC-aware
	// Drain tolerates before declaring the pool empty. sync.Pool has no drain primitive so
	// we sample heuristically; four consecutive misses signal a sufficiently exhausted pool.
	gcAwareDrainAttempts = 4
)

// Mode selects the pool's storage and reclamation strategy.
type Mode int

const (
	// ModePersistent holds objects in a pool-owned slab that survives every GC cycle.
	//
	// Memory usage equals the slab capacity; objects are only released when the Pool itself
	// becomes unreachable. Use for long-lived services where post-GC latency must remain
	// flat.
	ModePersistent Mode = iota

	// ModeGCAware mirrors sync.Pool reclamation so pooled objects may be dropped under
	// memory pressure.
	//
	// Pre-warming still runs at construction, but pre-warmed objects may be dropped on the
	// first GC cycle. Use for sporadic or bursty workloads where the memory floor matters
	// more than post-GC latency.
	ModeGCAware
)

// Link is the intrusive free-list header embedded as the first field of pooled types.
//
// It MUST be embedded as the first field of any type stored in a ModePersistent Pool.
// ModeGCAware does not require Link; the field can be left embedded but is unused. Eight
// bytes wide; the pool manipulates the next pointer when an object is on the free stack.
type Link struct {
	// next holds the successor pointer when this object sits on a shard free stack.
	next unsafe.Pointer //nolint:unused // accessed via unsafe.Pointer arithmetic from shard.go
}

var (
	// ErrNoInitialiser indicates New was called without an initialise function.
	ErrNoInitialiser = errors.New("stablepool: initialise function required")

	// ErrBadCapacity indicates New was called with a non-positive capacity.
	ErrBadCapacity = errors.New("stablepool: capacity must be > 0")

	// ErrBadGrowth indicates WithGrowth was configured with a maximum below the initial
	// capacity.
	ErrBadGrowth = errors.New("stablepool: growth maximum must be >= initial capacity")

	// ErrBadShardCount indicates WithShardCount was given a non-positive value.
	ErrBadShardCount = errors.New("stablepool: shard count must be > 0")

	// ErrBadLink indicates the pooled type does not embed Link as its first field.
	ErrBadLink = errors.New("stablepool: T must embed Link as its first field")
)

// Option customises pool construction.
type Option[T any] func(*config[T])

// config holds the resolved construction parameters; not exposed publicly.
type config[T any] struct {
	// shards holds the requested number of shards before rounding to a power of two.
	shards int

	// maxCapacity holds the growth ceiling; zero disables growth.
	maxCapacity int

	// mode selects the storage and reclamation strategy.
	mode Mode

	// metrics enables the in-flight counter when true.
	metrics bool
}

// WithMode selects the storage strategy.
//
// Takes m (Mode) which is the storage strategy to install on the Pool; defaults to
// ModePersistent.
//
// Returns an Option that records the chosen mode in the configuration.
func WithMode[T any](m Mode) Option[T] {
	return func(c *config[T]) { c.mode = m }
}

// WithShardCount overrides the number of shards.
//
// The default is runtime.GOMAXPROCS(0). The actual shard count is rounded up to the next
// power of two so the modulo can be replaced with a one-cycle AND.
//
// Takes n (int) which is the requested shard count before rounding.
//
// Returns an Option that records the requested shard count in the configuration.
func WithShardCount[T any](n int) Option[T] {
	return func(c *config[T]) { c.shards = n }
}

// WithGrowth permits the pool to allocate additional slabs on demand.
//
// Pass 0 (the default) to disable growth; pass a value less than the initial capacity to
// disable as well, which causes New to return ErrBadGrowth. Each growth event appends a
// new slab; existing slab addresses remain stable.
//
// Takes maxCapacity (int) which is the upper bound on total pooled objects across all
// slabs.
//
// Returns an Option that records the growth ceiling in the configuration.
func WithGrowth[T any](maxCapacity int) Option[T] {
	return func(c *config[T]) { c.maxCapacity = maxCapacity }
}

// WithMetrics enables the in-flight counter queryable via Pool.InFlight.
//
// Costs one atomic.Int64.Add per Get and per Put. Default is off (zero overhead). Use for
// leak detection or capacity tuning during integration.
//
// Returns an Option that records the metrics opt-in in the configuration.
func WithMetrics[T any]() Option[T] {
	return func(c *config[T]) { c.metrics = true }
}

// FactoryInit adapts a func() *T factory to the in-place initialise form expected by New.
//
// Each call allocates a fresh T via factory() and copies it into the destination slot,
// leaving the factory's result as garbage. Prefer a direct in-place initialiser when the
// factory's heap allocation matters; FactoryInit exists for ergonomic migration of
// existing code that already exposes a factory.
//
// Takes factory (func() *T) which is a constructor returning a freshly allocated *T.
//
// Returns a function suitable for use as the initialise argument to New.
func FactoryInit[T any](factory func() *T) func(*T) {
	return func(dst *T) { *dst = *factory() }
}

// Pool is a typed, pre-warmed object pool. Storage behaviour is selected at construction
// time by Mode (see WithMode).
type Pool[T any] struct {
	// initialise runs once per slot at construction and on growth to set up a fresh T.
	initialise func(*T)

	// cleaner runs on every Put to reset user-visible state before reuse.
	cleaner func(*T)

	// gcInner backs ModeGCAware operation with a standard sync.Pool.
	gcInner *sync.Pool

	// shards holds the per-shard fast slot and free stack for ModePersistent.
	shards []shard[T]

	// slabs records every allocated slab so the pre-warmed memory stays reachable.
	slabs [][]*T

	// mode records the storage strategy chosen at construction.
	mode Mode

	// capacity holds the current total slab capacity in objects.
	capacity int

	// maxCapacity holds the growth ceiling; zero disables growth.
	maxCapacity int

	// inFlight tracks objects handed out but not yet returned when metrics are enabled.
	inFlight atomic.Int64

	// slabsMu guards slabs and capacity during growth.
	slabsMu sync.Mutex

	// shardMask equals len(shards) - 1 and replaces modulo with a one-cycle AND.
	shardMask uint32

	// metrics toggles the in-flight counter on Get and Put.
	metrics bool
}

// validateLinkPlacement reflect-checks that T's first declared field is stablepool.Link.
//
// Runs once at New, never on the hot path.
//
// Returns ErrBadLink wrapped with the offending type detail when the placement is wrong,
// otherwise nil.
func validateLinkPlacement[T any]() error {
	var t T
	typ := reflect.TypeOf(t)
	if typ == nil || typ.Kind() != reflect.Struct {
		return fmt.Errorf("%w: T is not a struct", ErrBadLink)
	}
	if typ.NumField() == 0 {
		return fmt.Errorf("%w: T has no fields", ErrBadLink)
	}
	field := typ.Field(0)
	if field.Type != reflect.TypeFor[Link]() {
		return fmt.Errorf("%w: first field of %s is %s at offset %d (expected stablepool.Link at 0)",
			ErrBadLink, typ.String(), field.Type, field.Offset)
	}
	return nil
}

// nextPow2 returns the smallest power of two greater than or equal to n.
//
// Used to round the shard count up so the runtime can replace pid % numShards with pid &
// shardMask on every Get and Put.
//
// Takes n (int) which is a non-negative integer to round up.
//
// Returns the smallest power of two greater than or equal to n; returns 1 when n is at
// most 1.
func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

// New creates and pre-warms a Pool[T] with the given capacity.
//
// The initialise function runs once per slab slot at construction and on growth events,
// so it never executes on the steady-state hot path. The clean function runs on every Put
// outside any internal pinning, so it may yield or allocate.
//
// Takes initialise (func(*T)) which is the per-slot constructor; must be non-nil.
// Takes clean (func(*T)) which is an optional reset function run on every Put; may be
// nil.
// Takes capacity (int) which is the initial slab capacity; must be positive.
// Takes options (...Option[T]) which is zero or more Option values to customise
// construction.
//
// Returns the constructed Pool when configuration succeeds, and an error of
// ErrNoInitialiser, ErrBadCapacity, ErrBadShardCount, ErrBadGrowth, or ErrBadLink when
// the configuration is rejected.
func New[T any](initialise func(*T), clean func(*T), capacity int, options ...Option[T]) (*Pool[T], error) {
	cfg, err := resolvePoolConfig(initialise, capacity, options)
	if err != nil {
		return nil, err
	}
	if cfg.mode == ModePersistent {
		if err := validateLinkPlacement[T](); err != nil {
			return nil, err
		}
	}
	p := &Pool[T]{
		mode:        cfg.mode,
		initialise:  initialise,
		cleaner:     clean,
		maxCapacity: cfg.maxCapacity,
		metrics:     cfg.metrics,
	}
	if err := p.bootstrapStorage(cfg, initialise, capacity); err != nil {
		return nil, err
	}
	return p, nil
}

// resolvePoolConfig validates the public constructor arguments and merges caller-supplied
// options.
//
// Takes initialise (func(*T)) which is the constructor passed to New; must be non-nil.
// Takes capacity (int) which is the requested initial capacity; must be positive.
// Takes options ([]Option[T]) which are zero or more Option values applied after
// defaults.
//
// Returns the resolved configuration when every value passes validation, and an error of
// ErrNoInitialiser, ErrBadCapacity, ErrBadShardCount, or ErrBadGrowth on invalid input.
func resolvePoolConfig[T any](initialise func(*T), capacity int, options []Option[T]) (config[T], error) {
	if initialise == nil {
		return config[T]{}, ErrNoInitialiser
	}
	if capacity <= 0 {
		return config[T]{}, ErrBadCapacity
	}
	cfg := config[T]{shards: runtime.GOMAXPROCS(0)}
	for _, opt := range options {
		opt(&cfg)
	}
	if cfg.shards <= 0 {
		return config[T]{}, ErrBadShardCount
	}
	if cfg.maxCapacity > 0 && cfg.maxCapacity < capacity {
		return config[T]{}, ErrBadGrowth
	}
	return cfg, nil
}

// bootstrapStorage initialises per-mode storage and pre-warms the GC-aware inner pool.
//
// Allocates the first slab and shards for ModePersistent or builds the inner sync.Pool
// for ModeGCAware.
//
// Takes cfg (config[T]) which is the resolved configuration.
// Takes initialise (func(*T)) which is the per-object constructor used by the inner
// sync.Pool.
// Takes capacity (int) which is the initial capacity to pre-warm.
//
// Returns nil on success, otherwise the error from the initial slab allocation.
func (p *Pool[T]) bootstrapStorage(cfg config[T], initialise func(*T), capacity int) error {
	switch cfg.mode {
	case ModePersistent:
		shardN := nextPow2(cfg.shards)
		p.shards = make([]shard[T], shardN)
		p.shardMask = safeconv.IntToUint32(shardN - 1)
		return p.appendSlab(capacity, 0)
	case ModeGCAware:
		p.gcInner = &sync.Pool{
			New: func() any {
				var obj T
				initialise(&obj)
				return &obj
			},
		}
		p.prewarmGCInner(capacity)
	}
	return nil
}

// prewarmGCInner allocates capacity objects so the inner sync.Pool per-P caches are
// populated.
//
// Objects may be reclaimed by the next GC; the warm-up still reduces first-request
// latency for the initial workload.
//
// Takes capacity (int) which is the number of objects to pre-allocate and return to the
// pool.
func (p *Pool[T]) prewarmGCInner(capacity int) {
	warm := make([]*T, capacity)
	for i := range warm {
		warm[i] = p.popGCInner()
	}
	for _, obj := range warm {
		p.gcInner.Put(obj)
	}
}

// popGCInner pops one object from the inner sync.Pool and asserts its type.
//
// Returns the popped *T, or nil when sync.Pool itself returns nil.
func (p *Pool[T]) popGCInner() *T {
	value := p.gcInner.Get()
	if value == nil {
		return nil
	}
	if obj, ok := value.(*T); ok {
		return obj
	}
	obj := new(T)
	p.initialise(obj)
	return obj
}

// appendSlab allocates one slab, initialises each object, and distributes them across
// shards.
//
// Each T is heap-allocated individually so the GC sees per-object pointer bitmaps,
// sidestepping the GC bitmap pessimisation for arrays of large pointer-bearing values.
// Caller must hold slabsMu, or be in single-goroutine construction.
//
// Takes size (int) which is the number of objects to allocate in this slab.
// Takes preferShard (int) which is the shard index that receives the first object;
// further objects round-robin from there.
//
// Returns nil on success.
func (p *Pool[T]) appendSlab(size int, preferShard int) error {
	objects := make([]*T, size)
	for i := range objects {
		objects[i] = new(T)
		p.initialise(objects[i])
	}
	p.slabs = append(p.slabs, objects)
	p.capacity += size

	mask := safeconv.IntToUint32(len(p.shards) - 1)
	for i := range objects {
		shardIdx := uint32(preferShard+i) & mask //nolint:gosec // bounded by power-of-two mask
		p.shards[shardIdx].pushFree(objects[i])
	}
	return nil
}

// Get returns an object from the pool, or nil when the pool is drained and growth is
// exhausted.
//
// In ModeGCAware, Get always succeeds because it allocates via the initialise function on
// cache miss. For callers that prefer sync.Pool-style always-succeeds semantics, and can
// tolerate occasional heap allocation on drain, use MustGet.
//
// Returns a pooled *T, or nil when the persistent pool is drained and growth is
// exhausted.
func (p *Pool[T]) Get() *T {
	if p.mode == ModeGCAware {
		obj := p.popGCInner()
		if p.metrics && obj != nil {
			p.inFlight.Add(1)
		}
		return obj
	}

	obj := p.getPersistent()
	if p.metrics && obj != nil {
		p.inFlight.Add(1)
	}
	return obj
}

// getPersistent is the ModePersistent fast path used by Get.
//
// Split out so Get can wrap the result with metric accounting without inflating the
// inlined body.
//
// Returns a pooled *T, or nil when every shard is empty and growth is exhausted.
func (p *Pool[T]) getPersistent() *T {
	pid := runtimeProcPin()
	shardID := uint32(pid) & p.shardMask //nolint:gosec // pid is a non-negative P index, masked to bank size
	s := &p.shards[shardID]

	if obj := s.slot.tryTake(); obj != nil {
		runtimeProcUnpin()
		return obj
	}
	if obj := s.popFree(); obj != nil {
		runtimeProcUnpin()
		return obj
	}
	runtimeProcUnpin()

	if obj := p.steal(int(shardID)); obj != nil {
		return obj
	}
	return p.grow(int(shardID))
}

// MustGet returns an object from the pool, allocating a fresh one when the pool is
// drained.
//
// Mirrors sync.Pool.Get semantics and never returns nil. The heap-allocated fallback
// object joins the pool's free stack on Put, growing the effective working set past Cap.
// Use Get when the capacity bound must be honoured.
//
// Returns a pooled *T, or a freshly initialised heap object when the pool is drained.
func (p *Pool[T]) MustGet() *T {
	if obj := p.Get(); obj != nil {
		return obj
	}
	obj := new(T)
	p.initialise(obj)
	if p.metrics {
		p.inFlight.Add(1)
	}
	return obj
}

// Put returns an object to the pool.
//
// The cleaner, when configured, runs before any internal pinning so it may yield,
// allocate, or panic without breaking the pool.
//
// Takes obj (*T) which is the object to return; a nil value is ignored.
func (p *Pool[T]) Put(obj *T) {
	if obj == nil {
		return
	}
	if p.cleaner != nil {
		p.cleaner(obj)
	}
	if p.metrics {
		p.inFlight.Add(-1)
	}

	if p.mode == ModeGCAware {
		p.gcInner.Put(obj)
		return
	}

	pid := runtimeProcPin()
	shardID := uint32(pid) & p.shardMask //nolint:gosec // pid is a non-negative P index, masked to bank size
	s := &p.shards[shardID]

	if s.slot.tryPut(obj) {
		runtimeProcUnpin()
		return
	}
	runtimeProcUnpin()
	s.pushFree(obj)
}

// InFlight returns the number of objects handed out by Get but not yet returned by Put.
//
// Returns the current in-flight count, or zero when metrics are disabled because
// WithMetrics was not enabled; opt-in to avoid the per-call atomic add on the hot path.
func (p *Pool[T]) InFlight() int64 {
	return p.inFlight.Load()
}

// steal walks neighbour shards in increasing order from start+1 and returns the first
// object found.
//
// Takes start (int) which is the index of the originating shard; the walk skips this
// shard.
//
// Returns the first object popped from a neighbour shard, or nil when every other shard
// is empty.
func (p *Pool[T]) steal(start int) *T {
	numShards := len(p.shards)
	mask := safeconv.IntToUint32(numShards - 1)
	for i := 1; i < numShards; i++ {
		idx := uint32(start+i) & mask //nolint:gosec // bounded by power-of-two mask
		if obj := p.shards[idx].popFree(); obj != nil {
			return obj
		}
	}
	return nil
}

// grow allocates a new slab and returns one of its objects when growth is permitted.
//
// The new slab's objects are distributed across all shards' free stacks, starting at
// preferShard.
//
// Takes preferShard (int) which is the shard that receives the first slot of the new
// slab.
//
// Returns a freshly initialised *T when growth succeeds, or nil when growth is disabled
// or the maxCapacity bound is reached.
//
// Concurrency: Safe for concurrent use; serialised by slabsMu while a new slab is
// appended.
func (p *Pool[T]) grow(preferShard int) *T {
	if p.maxCapacity == 0 {
		return nil
	}

	p.slabsMu.Lock()
	defer p.slabsMu.Unlock()

	if obj := p.steal(preferShard); obj != nil {
		return obj
	}
	if obj := p.shards[preferShard].popFree(); obj != nil {
		return obj
	}

	if p.capacity >= p.maxCapacity {
		return nil
	}

	chunk := min(p.capacity, p.maxCapacity-p.capacity)
	if chunk <= 0 {
		return nil
	}
	if err := p.appendSlab(chunk, preferShard); err != nil {
		return nil
	}

	if obj := p.shards[preferShard].popFree(); obj != nil {
		return obj
	}
	return p.steal(preferShard)
}

// Close releases pool resources.
//
// The pool owns no background goroutines, so Close performs no work; slabs are released
// when the Pool itself becomes unreferenced.
func (*Pool[T]) Close() {}

// Cap returns the current total slab capacity for ModePersistent.
//
// The value grows as slabs are appended. In ModeGCAware the result is zero because
// sync.Pool does not expose its current cache size.
//
// Returns the persistent slab capacity in objects, or zero in ModeGCAware.
//
// Concurrency: Safe for concurrent use; guarded by slabsMu.
func (p *Pool[T]) Cap() int {
	if p.mode == ModeGCAware {
		return 0
	}
	p.slabsMu.Lock()
	defer p.slabsMu.Unlock()
	return p.capacity
}

// MaxCap returns the configured growth ceiling.
//
// Returns the maximum capacity in objects, or zero when growth is disabled.
func (p *Pool[T]) MaxCap() int { return p.maxCapacity }

// Drain removes every object from the pool and returns them in unspecified order.
//
// The walk covers every per-P private slot and every entry on every shard's free stack.
// Intended for shutdown and testing; using Drain in steady state defeats the purpose of a
// pool. Callers may re-Put the returned objects to refill.
//
// Drain MUST NOT be called concurrently with Get or Put. It bypasses procPin so
// private-slot access is unguarded.
//
// In ModeGCAware, Drain calls Get repeatedly until sync.Pool's New starts allocating
// fresh objects. The returned slice is best-effort because sync.Pool does not expose
// drain semantics, so there is no guarantee every cached object is returned.
//
// Returns a slice of objects removed from the pool; empty when the pool is already
// drained.
func (p *Pool[T]) Drain() []*T {
	if p.mode == ModeGCAware {
		out := make([]*T, 0)
		var hits int
		for hits < gcAwareDrainAttempts {
			obj := p.popGCInner()
			if obj == nil {
				hits++
				continue
			}
			out = append(out, obj)
		}
		return out
	}

	out := make([]*T, 0, p.capacity)
	for i := range p.shards {
		if obj := p.shards[i].slot.drainSlot(); obj != nil {
			out = append(out, obj)
		}
	}
	for i := range p.shards {
		for {
			obj := p.shards[i].popFree()
			if obj == nil {
				break
			}
			out = append(out, obj)
		}
	}
	return out
}
