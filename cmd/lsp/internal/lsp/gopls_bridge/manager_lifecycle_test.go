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
	"strconv"
	"sync"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func forceEnabled(manager *Manager) {
	manager.mu.Lock()
	manager.allow = true
	manager.available = true
	manager.mu.Unlock()
	manager.discoverOnce.Do(func() {})
}

func newAliveChild(moduleRoot string) *Child {
	return &Child{
		overlays:   make(map[protocol.DocumentURI]*overlayState),
		done:       make(chan struct{}),
		moduleRoot: moduleRoot,
	}
}

func newDeadChild(moduleRoot string) *Child {
	child := &Child{
		overlays:   make(map[protocol.DocumentURI]*overlayState),
		done:       make(chan struct{}),
		moduleRoot: moduleRoot,
	}
	child.dead.Store(true)
	close(child.done)
	return child
}

func registerChild(manager *Manager, child *Child) {
	manager.mu.Lock()
	manager.children[child.moduleRoot] = child
	manager.mu.Unlock()
}

func childRegistered(manager *Manager, moduleRoot string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	_, ok := manager.children[moduleRoot]
	return ok
}

func TestReapIdleEvictsIdleChild(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	child := newDeadChild("/idle")
	child.touch(0)
	registerChild(manager, child)

	manager.reapIdle(time.Now().UnixNano())

	assert.False(t, childRegistered(manager, "/idle"), "an idle child with no overlays is reaped")
}

func TestReapIdleKeepsBusyOrRecentChild(t *testing.T) {
	t.Parallel()

	now := time.Now().UnixNano()

	t.Run("a recently used child is kept", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{})
		child := newDeadChild("/recent")
		child.touch(now)
		registerChild(manager, child)

		manager.reapIdle(now)
		assert.True(t, childRegistered(manager, "/recent"))
	})

	t.Run("a child with open overlays is kept", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{})
		child := newDeadChild("/busy")
		child.touch(0)
		child.overlays["file:///x.go"] = &overlayState{owners: map[uint64]struct{}{1: {}}}
		registerChild(manager, child)

		manager.reapIdle(now)
		assert.True(t, childRegistered(manager, "/busy"), "a child with live overlays is never reaped")
	})
}

func TestExisting(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	_, ok := manager.Existing("/m")
	assert.False(t, ok, "no child returned when none is registered")

	alive := &Child{overlays: make(map[protocol.DocumentURI]*overlayState), done: make(chan struct{}), moduleRoot: "/m"}
	registerChild(manager, alive)
	got, ok := manager.Existing("/m")
	assert.True(t, ok)
	assert.Same(t, alive, got)

	dead := newDeadChild("/d")
	registerChild(manager, dead)
	_, ok = manager.Existing("/d")
	assert.False(t, ok, "a dead child is not returned")
}

func TestNewConnectionIDIsMonotonic(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	first := manager.NewConnectionID()
	second := manager.NewConnectionID()
	assert.NotEqual(t, first, second)
	assert.Greater(t, second, first)
}

func TestChildDone(t *testing.T) {
	t.Parallel()

	child := newAliveChild("/m")
	select {
	case <-child.Done():
		t.Fatal("a live child's done channel is open")
	default:
	}
	close(child.done)
	select {
	case <-child.Done():
	default:
		t.Fatal("done should be readable once closed")
	}
}

func TestAcquireReusesAliveChild(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	forceEnabled(manager)
	existing := newAliveChild("/m")
	registerChild(manager, existing)

	got, err := manager.Acquire(context.Background(), "/m")
	assert.NoError(t, err)
	assert.Same(t, existing, got, "an alive child is reused without respawning")
}

func TestAcquireGuards(t *testing.T) {
	t.Parallel()

	t.Run("respects the child cap", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{})
		forceEnabled(manager)
		for i := range defaultMaxGoplsChildren {
			registerChild(manager, newAliveChild("/m"+strconv.Itoa(i)))
		}
		_, err := manager.Acquire(context.Background(), "/overflow")
		assert.True(t, errors.Is(err, ErrGoplsUnavailable), "a manager at the child cap degrades to piko")
	})

	t.Run("honours spawn backoff", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{})
		forceEnabled(manager)
		manager.recordSpawnFailure("/flaky")
		_, err := manager.Acquire(context.Background(), "/flaky")
		assert.True(t, errors.Is(err, ErrGoplsUnavailable), "a module in spawn backoff is not retried")
	})
}

func TestRunReaperReclaimsIdleChildOnTick(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	manager := NewManager(ManagerConfig{Clock: mockClock})
	forceEnabled(manager)

	child := newDeadChild("/idle")
	child.touch(0)
	registerChild(manager, child)

	manager.mu.Lock()
	manager.startReaperLocked()
	manager.mu.Unlock()

	require.Eventually(t, func() bool {
		mockClock.Advance(reaperInterval + time.Second)
		return !childRegistered(manager, "/idle")
	}, 2*time.Second, 10*time.Millisecond, "the idle child is reclaimed when the reaper ticks")

	require.NoError(t, manager.Close(context.Background()))
}

func TestManagerCloseDoesNotRaceReaperStart(t *testing.T) {
	t.Parallel()

	for index := range 200 {
		manager := NewManager(ManagerConfig{})
		forceEnabled(manager)

		var registrars sync.WaitGroup
		for registrarIndex := range 3 {
			child := newDeadChild("/race" + strconv.Itoa(index) + "-" + strconv.Itoa(registrarIndex))
			registrars.Go(func() {
				_, _ = manager.registerOrClose(context.Background(), child)
			})
		}

		_ = manager.Close(context.Background())
		registrars.Wait()
	}
}

func TestOnChildDeathNotifiesSubscribersAndEvicts(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	dead := newDeadChild("/m")
	registerChild(manager, dead)

	var notified string
	manager.Subscribe(Subscriber{
		ChildDied: func(moduleRoot string) { notified = moduleRoot },
	})

	manager.onChildDeath(context.Background(), "/m")

	assert.Equal(t, "/m", notified, "subscribers are told which module's child died")
	assert.False(t, childRegistered(manager, "/m"), "a dead child is evicted")
}

type recordingCloser struct{ closed chan struct{} }

func (r *recordingCloser) Close() error {
	close(r.closed)
	return nil
}

func TestRegisterOrCloseClosesSurplusChild(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{MaxChildren: 1})
	registerChild(manager, newAliveChild("/already-here"))

	closed := make(chan struct{})
	surplus := &Child{
		overlays:   make(map[protocol.DocumentURI]*overlayState),
		done:       make(chan struct{}),
		moduleRoot: "/surplus",
		stream:     &recordingCloser{closed: closed},
	}

	surplus.dead.Store(true)
	close(surplus.done)

	registered, err := manager.registerOrClose(context.Background(), surplus)

	require.ErrorIs(t, err, ErrGoplsUnavailable)
	assert.Nil(t, registered)
	assert.False(t, childRegistered(manager, "/surplus"), "the surplus child is not registered once the cap is full")
	select {
	case <-closed:
	default:
		t.Fatal("the surplus child was not closed")
	}
}

func TestReapIdleKeepsRespawnedChild(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})
	stale := newDeadChild("/root")
	stale.touch(0)
	registerChild(manager, stale)

	fresh := newAliveChild("/root")
	manager.reapSnapshotHook = func() {
		registerChild(manager, fresh)
	}

	manager.reapIdle(time.Now().UnixNano())

	got, ok := manager.Existing("/root")
	require.True(t, ok, "the freshly respawned child must survive the reap")
	assert.Same(t, fresh, got, "only the stale pointer is evicted, not the respawn")
	select {
	case <-fresh.Done():
		t.Fatal("the respawned child must not be closed")
	default:
	}
}
