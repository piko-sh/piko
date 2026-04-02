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

package daemon_adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevEventBroadcaster_ClientLifecycle(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	assert.Equal(t, 0, b.ClientCount())

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	assert.Equal(t, 1, b.ClientCount())

	ch2 := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch2)
	assert.Equal(t, 2, b.ClientCount())

	b.removeClient(ch)
	assert.Equal(t, 1, b.ClientCount())

	b.removeClient(ch2)
	assert.Equal(t, 0, b.ClientCount())
}

func TestDevEventBroadcaster_BroadcastFanout(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	ch1 := make(chan []byte, devSSEClientBuffer)
	ch2 := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch1)
	b.addClient(ch2)
	defer b.removeClient(ch1)
	defer b.removeClient(ch2)

	assert.Equal(t, 2, b.ClientCount())

	b.Broadcast(DevBuildEvent{
		Type:           "rebuild-complete",
		AffectedRoutes: []string{"/login"},
		TimestampMs:    1234567890,
	})

	msg1 := <-ch1
	msg2 := <-ch2

	assert.Contains(t, string(msg1), "event: rebuild-complete")
	assert.Contains(t, string(msg1), `"/login"`)
	assert.Equal(t, msg1, msg2, "both clients should receive identical messages")
}

func TestDevEventBroadcaster_SlowClientDrop(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	ch := make(chan []byte, 1)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.Broadcast(DevBuildEvent{Type: "rebuild-complete", AffectedRoutes: []string{"/a"}})

	done := make(chan struct{})
	go func() {
		b.Broadcast(DevBuildEvent{Type: "rebuild-complete", AffectedRoutes: []string{"/b"}})
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked on a slow client")
	}
}

func TestDevEventBroadcaster_Close(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)

	assert.Equal(t, 1, b.ClientCount())

	b.Close()
	assert.Equal(t, 0, b.ClientCount())

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after Close()")

	b.Close()
}

func TestDevEventBroadcaster_NotifyRebuildComplete(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.NotifyRebuildComplete(context.Background(), []string{
		"pages/login.pk",
		"pages/dashboard.pk",
		"partials/header.pk",
	})

	msg := <-ch
	s := string(msg)
	assert.Contains(t, s, `"/login"`)
	assert.Contains(t, s, `"/dashboard"`)
	assert.NotContains(t, s, "header")
}

func TestDevEventBroadcaster_NotifyRebuildComplete_OnlyPartials(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	defer b.removeClient(ch)

	b.NotifyRebuildComplete(context.Background(), []string{
		"partials/sidebar.pk",
	})

	msg := <-ch
	assert.Contains(t, string(msg), `"*"`)
}

func TestDevEventBroadcaster_ServeHTTP_InitialHeartbeat(t *testing.T) {
	t.Parallel()

	b := NewDevEventBroadcaster()
	defer b.Close()

	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, errors.New("test timed out waiting for initial heartbeat"))
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/_piko/dev/events", nil)
	w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	go b.ServeHTTP(w, req)

	require.Eventually(t, func() bool {
		return strings.Contains(w.Body(), "event: connected")
	}, time.Second, 10*time.Millisecond)
}

func TestPathToRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"pages/login.pk", "/login", true},
		{"pages/dashboard.pk", "/dashboard", true},
		{"pages/environments/environment_id/blueprints.pk", "/environments/environment_id/blueprints", true},
		{"pages/index.pk", "/", true},
		{"pages/blog/index.pk", "/blog", true},
		{"partials/header.pk", "", false},
		{"emails/welcome.pk", "", false},
		{"random/file.go", "", false},
	}

	for _, tt := range tests {
		route, ok := pathToRoute(tt.input)
		assert.Equal(t, tt.ok, ok, "pathToRoute(%q) ok", tt.input)
		if ok {
			assert.Equal(t, tt.expected, route, "pathToRoute(%q) route", tt.input)
		}
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	mu sync.RWMutex
}

func (f *flushRecorder) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ResponseRecorder.Write(b)
}

func (f *flushRecorder) WriteHeader(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResponseRecorder.WriteHeader(code)
}

func (f *flushRecorder) Flush() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ResponseRecorder.Flush()
}

func (f *flushRecorder) Body() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ResponseRecorder.Body.String()
}
