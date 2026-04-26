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

package tui_domain

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// MockWatchdogProvider is an in-memory WatchdogProvider used by tests and
// for stand-alone TUI exercises. It exposes setters for every snapshot
// type and a channel for streaming events; failures can be injected per
// method via the Errors fields.
type MockWatchdogProvider struct {
	// Errors holds the optional error injection points used by tests.
	Errors WatchdogProviderErrors

	// streamChan delivers events to the active subscriber, when one exists.
	streamChan chan WatchdogEvent

	// status is the cached status snapshot returned by GetStatus.
	status *WatchdogStatus

	// streamCancel cancels the active subscription context, when one exists.
	streamCancel func()

	// profiles is the cached profile inventory returned by ListProfiles.
	profiles []WatchdogProfile

	// history is the cached startup history returned by GetStartupHistory.
	history []WatchdogStartupEntry

	// events is the cached one-shot event list returned by ListEvents.
	events []WatchdogEvent

	// dropped counts events discarded because the subscriber was full.
	dropped atomic.Uint64

	// pruneCalled counts invocations of PruneProfiles.
	pruneCalled atomic.Int64

	// contentionRuns counts invocations of RunContentionDiagnostic.
	contentionRuns atomic.Int64

	// mu guards status, profiles, history, and events.
	mu sync.RWMutex

	// streamMu guards streamChan and streamCancel.
	streamMu sync.Mutex

	// closed is true when Close has been invoked.
	closed bool
}

// WatchdogProviderErrors holds optional error injection points for the
// mock provider. Tests assign individual fields to simulate transport
// failures.
type WatchdogProviderErrors struct {
	// Refresh is returned from the Refresh method when non-nil.
	Refresh error

	// GetStatus is returned from the GetStatus method when non-nil.
	GetStatus error

	// ListProfiles is returned from the ListProfiles method when non-nil.
	ListProfiles error

	// StartupHistory is returned from the GetStartupHistory method when
	// non-nil.
	StartupHistory error

	// ListEvents is returned from the ListEvents method when non-nil.
	ListEvents error

	// Subscribe is returned from the SubscribeEvents method when non-nil.
	Subscribe error

	// Prune is returned from the PruneProfiles method when non-nil.
	Prune error

	// DownloadProfile is returned from the DownloadProfile method when
	// non-nil.
	DownloadProfile error

	// DownloadSidecar is returned from the DownloadSidecar method when
	// non-nil.
	DownloadSidecar error

	// ContentionDiagRun is returned from the RunContentionDiagnostic method
	// when non-nil.
	ContentionDiagRun error
}

// NewMockWatchdogProvider creates a mock with empty caches.
//
// Returns *MockWatchdogProvider ready for use.
func NewMockWatchdogProvider() *MockWatchdogProvider {
	return &MockWatchdogProvider{}
}

// Name implements Provider.
//
// Returns string which is the provider's identifier.
func (*MockWatchdogProvider) Name() string { return "mock-watchdog" }

// Health implements Provider.
//
// Returns error which is always nil for the mock.
func (*MockWatchdogProvider) Health(_ context.Context) error { return nil }

// Close implements Provider.
//
// Returns error which is always nil for the mock.
//
// Concurrency: Safe for concurrent use; guarded by streamMu.
func (m *MockWatchdogProvider) Close() error {
	m.streamMu.Lock()
	defer m.streamMu.Unlock()
	if m.streamCancel != nil {
		m.streamCancel()
		m.streamCancel = nil
	}
	if m.streamChan != nil {
		close(m.streamChan)
		m.streamChan = nil
	}
	m.closed = true
	return nil
}

// Refresh implements RefreshableProvider. It is a no-op in the mock.
//
// Returns error which is the value injected via Errors.Refresh.
func (m *MockWatchdogProvider) Refresh(_ context.Context) error {
	return m.Errors.Refresh
}

// RefreshInterval implements RefreshableProvider.
//
// Returns time.Duration which is the desired refresh cadence.
func (*MockWatchdogProvider) RefreshInterval() time.Duration {
	return 2 * time.Second
}

// SetStatus replaces the cached status snapshot.
//
// Takes status (*WatchdogStatus) which becomes the new snapshot.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) SetStatus(status *WatchdogStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}

// SetProfiles replaces the cached profile inventory.
//
// Takes profiles ([]WatchdogProfile) which is the new inventory.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) SetProfiles(profiles []WatchdogProfile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles = profiles
}

// SetStartupHistory replaces the cached startup history.
//
// Takes history ([]WatchdogStartupEntry) which is the new history.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) SetStartupHistory(history []WatchdogStartupEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = history
}

// SetEvents replaces the cached one-shot event list.
//
// Takes events ([]WatchdogEvent) which is the new list.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) SetEvents(events []WatchdogEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = events
}

// EmitEvent pushes an event to any active subscriber.
//
// Takes event (WatchdogEvent) which is the event to deliver.
//
// Concurrency: Safe for concurrent use; guarded by streamMu.
func (m *MockWatchdogProvider) EmitEvent(event WatchdogEvent) {
	m.streamMu.Lock()
	ch := m.streamChan
	m.streamMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- event:
	default:
		m.dropped.Add(1)
	}
}

// PruneCallCount returns the number of times PruneProfiles has been
// invoked, useful in tests asserting an action triggered the call.
//
// Returns int which is the call count.
func (m *MockWatchdogProvider) PruneCallCount() int {
	return int(m.pruneCalled.Load())
}

// ContentionDiagnosticRunCount returns the number of times
// RunContentionDiagnostic has been invoked.
//
// Returns int which is the call count.
func (m *MockWatchdogProvider) ContentionDiagnosticRunCount() int {
	return int(m.contentionRuns.Load())
}

// GetStatus implements WatchdogProvider.
//
// Returns *WatchdogStatus which is the cached snapshot, or nil when no
// status has been set.
// Returns error which is the value injected via Errors.GetStatus.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) GetStatus(_ context.Context) (*WatchdogStatus, error) {
	if m.Errors.GetStatus != nil {
		return nil, m.Errors.GetStatus
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status, nil
}

// ListProfiles implements WatchdogProvider.
//
// Returns []WatchdogProfile which is a copy of the cached inventory.
// Returns error which is the value injected via Errors.ListProfiles.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) ListProfiles(_ context.Context) ([]WatchdogProfile, error) {
	if m.Errors.ListProfiles != nil {
		return nil, m.Errors.ListProfiles
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]WatchdogProfile, len(m.profiles))
	copy(out, m.profiles)
	return out, nil
}

// GetStartupHistory implements WatchdogProvider.
//
// Returns []WatchdogStartupEntry which is a copy of the cached history.
// Returns error which is the value injected via Errors.StartupHistory.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) GetStartupHistory(_ context.Context) ([]WatchdogStartupEntry, error) {
	if m.Errors.StartupHistory != nil {
		return nil, m.Errors.StartupHistory
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]WatchdogStartupEntry, len(m.history))
	copy(out, m.history)
	return out, nil
}

// ListEvents implements WatchdogProvider, applying the supplied filter to
// the cached events list.
//
// Takes query (WatchdogEventQuery) which constrains the returned events.
//
// Returns []WatchdogEvent which is the filtered slice.
// Returns error which is the value injected via Errors.ListEvents.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) ListEvents(_ context.Context, query WatchdogEventQuery) ([]WatchdogEvent, error) {
	if m.Errors.ListEvents != nil {
		return nil, m.Errors.ListEvents
	}
	m.mu.RLock()
	events := append([]WatchdogEvent{}, m.events...)
	m.mu.RUnlock()

	out := make([]WatchdogEvent, 0, len(events))
	for _, e := range events {
		if !query.Since.IsZero() && e.EmittedAt.Before(query.Since) {
			continue
		}
		if query.EventType != "" && string(e.EventType) != query.EventType {
			continue
		}
		out = append(out, e)
	}
	if query.Limit > 0 && len(out) > query.Limit {
		out = out[len(out)-query.Limit:]
	}
	return out, nil
}

// SubscribeEvents implements WatchdogProvider, returning a channel the
// test can drive via EmitEvent.
//
// Returns <-chan WatchdogEvent which delivers events emitted via EmitEvent.
// Returns func() which cancels the subscription.
// Returns error which is the value injected via Errors.Subscribe.
//
// Concurrency: Safe for concurrent use; guarded by streamMu.
func (m *MockWatchdogProvider) SubscribeEvents(ctx context.Context, _ time.Time) (<-chan WatchdogEvent, func(), error) {
	if m.Errors.Subscribe != nil {
		return nil, nil, m.Errors.Subscribe
	}
	m.streamMu.Lock()
	defer m.streamMu.Unlock()

	ch := make(chan WatchdogEvent, EventStreamBufferSize)
	m.streamChan = ch

	subCtx, cancel := context.WithCancelCause(ctx)
	cancelFn := func() { cancel(errors.New("mock watchdog event subscription cancelled")) }
	m.streamCancel = cancelFn

	go func() {
		<-subCtx.Done()
		m.streamMu.Lock()
		if m.streamChan == ch {
			close(ch)
			m.streamChan = nil
		}
		m.streamMu.Unlock()
	}()

	return ch, cancelFn, nil
}

// DroppedEvents implements WatchdogProvider.
//
// Returns uint64 which is the count of events discarded due to a full
// subscriber.
func (m *MockWatchdogProvider) DroppedEvents() uint64 {
	return m.dropped.Load()
}

// PruneProfiles implements WatchdogProvider.
//
// Takes profileType (string) which is the profile type to prune; an empty
// string clears every cached profile.
//
// Returns int which is the number of profiles removed.
// Returns error which is the value injected via Errors.Prune.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (m *MockWatchdogProvider) PruneProfiles(_ context.Context, profileType string) (int, error) {
	m.pruneCalled.Add(1)
	if m.Errors.Prune != nil {
		return 0, m.Errors.Prune
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if profileType == "" {
		count := len(m.profiles)
		m.profiles = nil
		return count, nil
	}
	kept := make([]WatchdogProfile, 0, len(m.profiles))
	removed := 0
	for _, p := range m.profiles {
		if p.Type == profileType {
			removed++
			continue
		}
		kept = append(kept, p)
	}
	m.profiles = kept
	return removed, nil
}

// DownloadProfile implements WatchdogProvider, writing a placeholder body
// for tests.
//
// Takes filename (string) which is the profile name to fetch.
// Takes w (io.Writer) which receives the placeholder body.
//
// Returns error which is the value injected via Errors.DownloadProfile, or
// any error from the underlying writer.
func (m *MockWatchdogProvider) DownloadProfile(_ context.Context, filename string, w io.Writer) error {
	if m.Errors.DownloadProfile != nil {
		return m.Errors.DownloadProfile
	}
	if filename == "" {
		return nil
	}
	_, err := io.WriteString(w, "mock-profile:"+filename)
	return err
}

// DownloadSidecar implements WatchdogProvider.
//
// Returns []byte which is always nil for the mock.
// Returns bool which is always false for the mock.
// Returns error which is the value injected via Errors.DownloadSidecar.
func (m *MockWatchdogProvider) DownloadSidecar(_ context.Context, _ string) ([]byte, bool, error) {
	if m.Errors.DownloadSidecar != nil {
		return nil, false, m.Errors.DownloadSidecar
	}
	return nil, false, nil
}

// RunContentionDiagnostic implements WatchdogProvider.
//
// Returns error which is the value injected via Errors.ContentionDiagRun.
func (m *MockWatchdogProvider) RunContentionDiagnostic(_ context.Context) error {
	m.contentionRuns.Add(1)
	return m.Errors.ContentionDiagRun
}
