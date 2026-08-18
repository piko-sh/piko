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

package gopls_bridge

import (
	"context"
	"errors"
	"sync"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// defaultMaxGoplsChildren caps the number of concurrent gopls processes.
	defaultMaxGoplsChildren = 8

	// defaultMaxOverlaysPerChild caps how many virtual Go documents one gopls child holds.
	defaultMaxOverlaysPerChild = 1024

	// childIdleTimeout is how long a gopls child with no open overlays may sit unused before
	// the reaper closes it, reclaiming memory on the long-lived TCP daemon.
	childIdleTimeout = 10 * time.Minute

	// reaperInterval is how often the idle reaper scans for reclaimable children.
	reaperInterval = 60 * time.Second

	// spawnBackoff suppresses respawn attempts for a module root that just failed to start
	// gopls, so a crash-looping or mis-configured toolchain is not relaunched on every
	// keystroke.
	spawnBackoff = 30 * time.Second

	// managerCloseTimeout caps the aggregate shutdown wait for gopls children and the
	// reaper.
	managerCloseTimeout = 15 * time.Second
)

var (
	// ErrGoplsUnavailable is returned by Acquire when the bridge is disabled, gopls could
	// not be found, the child cap is reached, the module is in spawn backoff, or the manager
	// has been closed. Callers treat it as a signal to fall back to piko's own intelligence
	// rather than an editor-visible error.
	ErrGoplsUnavailable = errors.New("gopls bridge unavailable")
)

// Subscriber receives gopls events for one editor connection.
type Subscriber struct {
	// Diagnostics is the per-document diagnostics callback.
	Diagnostics DiagnosticsHandler

	// ChildDied, when set, fires when the gopls child for a module root exits, so the
	// connection can clear the now-stale Go diagnostics it had cached.
	ChildDied func(moduleRoot string)
}

// ManagerConfig configures a Manager. Zero-value fields fall back to safe, high defaults,
// so callers set only what they need to override.
type ManagerConfig struct {
	// Clock is the time source for backoff and the idle reaper; nil uses the real clock.
	// Injecting a clock makes the time-dependent behaviour deterministic in tests.
	Clock clock.Clock

	// GoplsPath is the configured gopls path, or empty to discover it.
	GoplsPath string

	// MaxChildren caps the number of concurrent gopls processes; <= 0 uses the default.
	// Raise it for large multi-module workspaces.
	MaxChildren int

	// MaxOverlaysPerChild caps how many virtual Go documents one gopls child holds open; <=
	// 0 uses the default. Raise it for workspaces with very many open files.
	MaxOverlaysPerChild int

	// Allow is the master switch for the bridge.
	//
	// When false the bridge is permanently off (no discovery, Acquire always degrades to
	// piko) regardless of any client request. When true, gopls is discovered lazily on first
	// use, so a disabled-by-default process still honours a per-connection opt-in.
	Allow bool
}

// Manager owns the shared gopls child processes, one per Go module root.
//
// The processes are shared across every editor connection (the TCP adapter builds a fresh
// Server per connection, so child ownership must live above the Server). gopls discovery
// is lazy: it runs on the first Enabled check, so a process started with the bridge
// disabled pays nothing until a connection opts in (e.g. VS Code via
// initializationOptions), and an operator can hard-disable the bridge with Allow=false
// regardless of any client request. Safe for concurrent use.
type Manager struct {
	// group de-duplicates concurrent spawns for the same module root.
	group singleflight.Group

	// clock is the time source for backoff and the idle reaper.
	clock clock.Clock

	// children holds the live gopls child keyed by module root.
	children map[string]*Child

	// subscribers holds the per-connection event subscribers keyed by subscription id.
	subscribers map[uint64]Subscriber

	// spawnFailures records the last failed-spawn timestamp per module root for backoff.
	spawnFailures map[string]int64

	// done is closed on Close to stop the idle reaper.
	done chan struct{}

	// reapSnapshotHook is a test-only seam invoked once in reapIdle after the candidate
	// snapshot lock is released and before the per-candidate re-check, so a test can
	// deterministically interleave a respawn and exercise the pointer-identity guard. Nil in
	// production.
	reapSnapshotHook func()

	// goplsPath is the configured gopls path, or empty to discover it.
	goplsPath string

	// resolvedPath is the discovered gopls executable path once discovery has run.
	resolvedPath string

	// mu guards the mutable manager state below.
	mu sync.Mutex

	// discoverOnce ensures gopls discovery runs at most once.
	discoverOnce sync.Once

	// reaperOnce ensures the idle reaper goroutine starts at most once.
	reaperOnce sync.Once

	// reaperWG tracks the idle reaper goroutine for shutdown.
	reaperWG sync.WaitGroup

	// nextSubID vends the next subscription identifier.
	nextSubID uint64

	// nextConnID vends the next connection identifier.
	nextConnID uint64

	// maxChildren caps the number of concurrent gopls children.
	maxChildren int

	// maxOverlays caps how many overlays one gopls child holds open.
	maxOverlays int

	// allow is the master switch from ManagerConfig.Allow.
	allow bool

	// available reports whether a usable gopls was discovered.
	available bool

	// closed reports whether the manager has been shut down.
	closed bool
}

// NewManager returns a manager configured by config.
//
// Takes config (ManagerConfig) which supplies the clock, gopls path, caps, and master
// switch.
//
// Returns *Manager which is ready for concurrent use.
func NewManager(config ManagerConfig) *Manager {
	managerClock := config.Clock
	if managerClock == nil {
		managerClock = clock.RealClock()
	}
	maxChildren := config.MaxChildren
	if maxChildren <= 0 {
		maxChildren = defaultMaxGoplsChildren
	}
	maxOverlays := config.MaxOverlaysPerChild
	if maxOverlays <= 0 {
		maxOverlays = defaultMaxOverlaysPerChild
	}
	return &Manager{
		clock:         managerClock,
		children:      make(map[string]*Child),
		subscribers:   make(map[uint64]Subscriber),
		spawnFailures: make(map[string]int64),
		done:          make(chan struct{}),
		goplsPath:     config.GoplsPath,
		maxChildren:   maxChildren,
		maxOverlays:   maxOverlays,
		allow:         config.Allow,
	}
}

// Enabled reports whether the bridge can serve requests: the master switch is on, the
// manager is open, and a usable gopls was discovered. Discovery happens here, once, so
// the cost is paid only when a connection actually wants the bridge.
//
// Returns bool which is true when the bridge is ready to serve requests.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (m *Manager) Enabled() bool {
	m.mu.Lock()
	allowed := m.allow && !m.closed
	m.mu.Unlock()
	if !allowed {
		return false
	}

	m.ensureDiscovered()

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available && !m.closed
}

// NewConnectionID vends a process-unique identifier an editor connection uses as its
// overlay-ownership token, so the shared children can reference-count which connections
// hold each overlay open.
//
// Returns uint64 which is the new connection identifier.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (m *Manager) NewConnectionID() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextConnID++
	return m.nextConnID
}

// Subscribe registers a per-connection subscriber and returns a function that removes it.
// Every child fans gopls diagnostics out to all subscribers; each subscriber filters to
// the overlays it owns, so the shared manager never misroutes one connection's
// diagnostics to another.
//
// Takes subscriber (Subscriber) which holds the per-connection event callbacks.
//
// Returns func() which deregisters the subscriber when called.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (m *Manager) Subscribe(subscriber Subscriber) func() {
	m.mu.Lock()
	id := m.nextSubID
	m.nextSubID++
	m.subscribers[id] = subscriber
	m.mu.Unlock()

	return func() {
		m.mu.Lock()
		delete(m.subscribers, id)
		m.mu.Unlock()
	}
}

// Acquire returns the gopls child for a module root.
//
// One is spawned and handshaked on first use. Concurrent callers for the same module
// share a single spawn via singleflight. A dead child is evicted and respawned (subject
// to backoff); a module in spawn backoff or a manager at the child cap degrades to
// piko-only.
//
// Takes moduleRoot (string) which is the Go module root to acquire a child for.
//
// Returns *Child which is the live gopls child for the module root.
// Returns error when the bridge is unavailable or the spawn fails.
//
// Concurrency: acquires mu while inspecting and updating the children map.
func (m *Manager) Acquire(ctx context.Context, moduleRoot string) (*Child, error) {
	if !m.Enabled() {
		return nil, ErrGoplsUnavailable
	}

	nowNano := m.clock.Now().UnixNano()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, ErrGoplsUnavailable
	}
	if existing, ok := m.children[moduleRoot]; ok {
		if existing.IsAlive() {
			existing.touch(nowNano)
			m.mu.Unlock()
			return existing, nil
		}
		delete(m.children, moduleRoot)
	}
	if last, ok := m.spawnFailures[moduleRoot]; ok {
		if nowNano-last < int64(spawnBackoff) {
			m.mu.Unlock()
			return nil, ErrGoplsUnavailable
		}

		delete(m.spawnFailures, moduleRoot)
	}
	if len(m.children) >= m.maxChildren {
		m.mu.Unlock()
		return nil, ErrGoplsUnavailable
	}
	m.mu.Unlock()

	created, spawnErr, _ := m.group.Do(moduleRoot, func() (any, error) {
		return m.spawnChild(ctx, moduleRoot)
	})
	if spawnErr != nil {
		if !errors.Is(spawnErr, context.Canceled) {
			m.recordSpawnFailure(moduleRoot)
		}
		return nil, spawnErr
	}
	child, ok := created.(*Child)
	if !ok {
		return nil, ErrGoplsUnavailable
	}
	child.touch(m.clock.Now().UnixNano())
	return child, nil
}

// Existing returns the live gopls child for a module root without spawning one, so
// connection teardown can release its overlays against an already-running child without
// resurrecting a reclaimed one.
//
// Takes moduleRoot (string) which is the Go module root to look up.
//
// Returns *Child which is the live child, or nil when none exists.
// Returns bool which is true when a live child was found.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (m *Manager) Existing(moduleRoot string) (*Child, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	child, ok := m.children[moduleRoot]
	if !ok || !child.IsAlive() {
		return nil, false
	}
	return child, true
}

// Close shuts down every gopls child concurrently, stops the idle reaper, and disables
// further acquisition. It is idempotent.
//
// Returns error which is always nil; the signature satisfies the closer contract.
//
// Concurrency: holds mu while flipping closed and draining the children map, then fans
// the per-child Close calls out across goroutines and joins them with a bounded wait.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	close(m.done)
	children := m.children
	m.children = make(map[string]*Child)
	m.mu.Unlock()

	var waitGroup sync.WaitGroup
	for _, child := range children {
		waitGroup.Go(func() {
			defer goroutine.RecoverPanic(ctx, "gopls_bridge.manager.close")
			if closeErr := child.Close(ctx); closeErr != nil {
				_, l := logger_domain.From(ctx, log)
				l.Warn("gopls child did not shut down cleanly",
					logger_domain.String("moduleRoot", child.moduleRoot),
					logger_domain.Error(closeErr))
			}
		})
	}

	if !waitWithTimeout(&waitGroup, managerCloseTimeout) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("gopls children did not all close within the deadline; abandoning the wait",
			logger_domain.String("timeout", managerCloseTimeout.String()))
	}
	if !waitWithTimeout(&m.reaperWG, managerCloseTimeout) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("gopls reaper did not stop within the deadline; abandoning the wait",
			logger_domain.String("timeout", managerCloseTimeout.String()))
	}
	return nil
}

// waitWithTimeout waits for waitGroup to reach zero, returning true on success or false
// when timeout elapses first. On timeout the helper goroutine outlives the call but is
// bounded: the work it waits on (a child Close, the reaper) is itself deadline-bounded,
// so it drains shortly after.
//
// Takes waitGroup (*sync.WaitGroup) which is the group to await.
// Takes timeout (time.Duration) which bounds the wait.
//
// Returns bool which is true when the group drained before the timeout.
func waitWithTimeout(waitGroup *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// resolvedGoplsPath returns the resolved gopls executable path once discovered, otherwise
// the configured value.
//
// Returns the discovered path, or the configured path when discovery has not run.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (m *Manager) resolvedGoplsPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.resolvedPath != "" {
		return m.resolvedPath
	}
	return m.goplsPath
}

// ensureDiscovered resolves gopls exactly once, on first use. A failure to find gopls is
// warned once (rather than silently degrading) because it means a connection asked for
// the bridge but it will be unavailable.
//
// Concurrency: gated by discoverOnce; acquires mu to publish the resolved path.
func (m *Manager) ensureDiscovered() {
	m.discoverOnce.Do(func() {
		discovered, found := DiscoverGoplsPath(m.goplsPath)
		if !found {
			_, l := logger_domain.From(context.Background(), log)
			l.Warn("gopls bridge requested but gopls was not found; falling back to piko-only intelligence",
				logger_domain.String("configuredPath", m.goplsPath))
			return
		}
		m.mu.Lock()
		m.resolvedPath = discovered
		m.available = true
		m.mu.Unlock()
	})
}

// recordSpawnFailure timestamps a failed spawn so Acquire backs off before retrying that
// module root.
//
// Takes moduleRoot (string) which is the module root whose spawn failed.
//
// Concurrency: acquires mu while recording the failure timestamp.
func (m *Manager) recordSpawnFailure(moduleRoot string) {
	m.mu.Lock()
	m.spawnFailures[moduleRoot] = m.clock.Now().UnixNano()
	m.mu.Unlock()
}

// fanOut delivers gopls diagnostics to every registered subscriber.
//
// A panicking subscriber is isolated so it cannot drop another's diagnostics or crash the
// process. It is installed as each child's diagnostics handler. Delivery is synchronous;
// a wedged editor cannot block it indefinitely because the editor transport enforces a
// write deadline and drops the connection on timeout.
//
// Takes params (*protocol.PublishDiagnosticsParams) which carries the diagnostics to
// deliver.
//
// Concurrency: acquires mu to snapshot the subscribers, then delivers outside the lock.
func (m *Manager) fanOut(ctx context.Context, params *protocol.PublishDiagnosticsParams) {
	m.mu.Lock()
	subscribers := make([]Subscriber, 0, len(m.subscribers))
	for _, subscriber := range m.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	m.mu.Unlock()

	for _, subscriber := range subscribers {
		if subscriber.Diagnostics == nil {
			continue
		}
		func() {
			defer goroutine.RecoverPanic(ctx, "gopls_bridge.manager.fanOut")
			subscriber.Diagnostics(ctx, params)
		}()
	}
}

// onChildDeath evicts a dead child and notifies subscribers so they can clear the stale
// Go diagnostics they cached for that module root. It runs on the child's monitor
// goroutine, whose context carries the dial-time logger and trace values.
//
// Takes moduleRoot (string) which is the module root whose child exited.
func (m *Manager) onChildDeath(ctx context.Context, moduleRoot string) {
	m.mu.Lock()
	if child, ok := m.children[moduleRoot]; ok && !child.IsAlive() {
		delete(m.children, moduleRoot)
	}
	subscribers := make([]Subscriber, 0, len(m.subscribers))
	for _, subscriber := range m.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	m.mu.Unlock()

	for _, subscriber := range subscribers {
		if subscriber.ChildDied == nil {
			continue
		}
		func() {
			defer goroutine.RecoverPanic(ctx, "gopls_bridge.manager.onChildDeath")
			subscriber.ChildDied(moduleRoot)
		}()
	}
}

// spawnChild creates and registers one gopls child for a module root, honouring a
// concurrent close and the child cap, and starting the idle reaper on first use.
//
// Takes moduleRoot (string) which is the module root to start a child for.
//
// Returns *Child which is the registered child.
// Returns error when the manager closed, the cap is reached, or the spawn or handshake
// fails.
func (m *Manager) spawnChild(ctx context.Context, moduleRoot string) (*Child, error) {
	ctx, l := logger_domain.From(ctx, log)

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, ErrGoplsUnavailable
	}
	if existing, ok := m.children[moduleRoot]; ok {
		if existing.IsAlive() {
			m.mu.Unlock()
			return existing, nil
		}
		delete(m.children, moduleRoot)
	}
	if len(m.children) >= m.maxChildren {
		m.mu.Unlock()
		return nil, ErrGoplsUnavailable
	}
	resolvedPath := m.resolvedPath
	m.mu.Unlock()

	stream, command, spawnErr := spawnGopls(resolvedPath, moduleRoot)
	if spawnErr != nil {
		l.Warn("failed to start gopls", logger_domain.String("moduleRoot", moduleRoot), logger_domain.Error(spawnErr))
		return nil, spawnErr
	}

	child, dialErr := dialChild(ctx, stream, command, moduleRoot, m.fanOut, m.onChildDeath)
	if dialErr != nil {
		l.Warn("failed to handshake with gopls", logger_domain.String("moduleRoot", moduleRoot), logger_domain.Error(dialErr))
		return nil, dialErr
	}

	child.maxOverlays = m.maxOverlays

	registered, registerErr := m.registerOrClose(ctx, child)
	if registerErr != nil {
		return nil, registerErr
	}

	l.Info("started gopls child", logger_domain.String("moduleRoot", moduleRoot))
	return registered, nil
}

// registerOrClose registers a freshly dialled child under its module root, or closes it
// and returns ErrGoplsUnavailable when the manager closed or the child cap filled during
// the lock-free dial. lastUsed is stamped inside the critical section so a registered
// child never looks idle to the reaper before its first use.
//
// Takes child (*Child) which is the freshly dialled child to register.
//
// Returns *Child which is the registered child, or nil when it was closed.
// Returns error when the manager closed or the child cap filled during the dial.
//
// Concurrency: acquires mu to register the child and start the reaper under the lock.
func (m *Manager) registerOrClose(ctx context.Context, child *Child) (*Child, error) {
	m.mu.Lock()
	if m.closed || len(m.children) >= m.maxChildren {
		m.mu.Unlock()
		_ = child.Close(ctx)
		return nil, ErrGoplsUnavailable
	}
	child.touch(m.clock.Now().UnixNano())
	m.children[child.moduleRoot] = child
	delete(m.spawnFailures, child.moduleRoot)

	m.startReaperLocked()
	m.mu.Unlock()
	return child, nil
}

// startReaperLocked launches the idle reaper goroutine exactly once, on the first
// successful registration, so a process that never uses the bridge never starts it. It
// must be called with m.mu held so the reaperWG.Add is serialised against Close's
// reaperWG.Wait.
func (m *Manager) startReaperLocked() {
	m.reaperOnce.Do(func() {
		m.reaperWG.Add(1)
		go m.runReaper()
	})
}

// runReaper periodically reclaims idle children until the manager closes.
func (m *Manager) runReaper() {
	defer m.reaperWG.Done()
	defer goroutine.RecoverPanic(context.Background(), "gopls_bridge.manager.reaper")

	ticker := m.clock.NewTicker(reaperInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C():
			m.reapIdle(m.clock.Now().UnixNano())
		}
	}
}

// reapIdle closes every child that has no open overlays and has been unused for longer
// than childIdleTimeout. It re-verifies each child under the lock before eviction so it
// never races a fresh Acquire.
//
// Takes now (int64) which is the current time in Unix nanoseconds.
//
// Concurrency: acquires mu to snapshot candidates and again per child to re-check before
// eviction, so it never races a fresh Acquire.
func (m *Manager) reapIdle(now int64) {
	m.mu.Lock()
	candidates := make([]*Child, 0, len(m.children))
	for _, child := range m.children {
		candidates = append(candidates, child)
	}
	m.mu.Unlock()

	if m.reapSnapshotHook != nil {
		m.reapSnapshotHook()
	}

	for _, child := range candidates {
		if child.overlayCount() != 0 || now-child.lastUsed.Load() <= int64(childIdleTimeout) {
			continue
		}
		m.mu.Lock()
		current, ok := m.children[child.moduleRoot]
		stillIdle := ok && current == child && child.overlayCount() == 0 && now-child.lastUsed.Load() > int64(childIdleTimeout)
		if stillIdle {
			delete(m.children, child.moduleRoot)
		}
		m.mu.Unlock()
		if stillIdle {
			_ = child.Close(context.Background())
		}
	}
}
