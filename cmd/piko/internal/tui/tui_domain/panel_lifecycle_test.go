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
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestNewPanelState(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	if state.Clock() == nil {
		t.Error("expected non-nil clock")
	}
}

func TestNewPanelState_NilClock(t *testing.T) {
	state := NewPanelState(nil)

	if state.Clock() == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestPanelState_Now(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	now := state.Now()
	if now.IsZero() {
		t.Error("expected non-zero time")
	}
}

func TestPanelState_Since(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	past := mockClock.Now().Add(-5 * time.Minute)
	since := state.Since(past)

	if since != 5*time.Minute {
		t.Errorf("expected 5m, got %v", since)
	}
}

func TestPanelState_LockUnlock(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	state.Lock()
	state.Unlock()

	state.RLock()
	state.RUnlock()
}

func TestPanelState_UpdateError(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	testErr := errors.New("test error")
	state.UpdateError(testErr)

	if !errors.Is(state.GetError(), testErr) {
		t.Error("expected error to be set")
	}
}

func TestPanelState_UpdateSuccess(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	state.UpdateError(errors.New("test error"))

	state.UpdateSuccess()

	if state.GetError() != nil {
		t.Error("expected error to be cleared")
	}
	if state.GetLastRefresh().IsZero() {
		t.Error("expected lastRefresh to be set")
	}
}

func TestPanelState_GetLastRefresh(t *testing.T) {
	mockClock := newTestClock()
	state := NewPanelState(mockClock)

	if !state.GetLastRefresh().IsZero() {
		t.Error("expected zero lastRefresh initially")
	}

	state.UpdateSuccess()
	if state.GetLastRefresh().IsZero() {
		t.Error("expected non-zero lastRefresh after success")
	}
}

func TestKeyResult_Handled(t *testing.T) {
	result := Handled()

	if !result.Handled {
		t.Error("expected Handled to be true")
	}
	if result.Cmd != nil {
		t.Error("expected nil command")
	}
}

func TestKeyResult_HandledWithCmd(t *testing.T) {
	testCmd := func() tea.Msg { return nil }
	result := HandledWithCmd(testCmd)

	if !result.Handled {
		t.Error("expected Handled to be true")
	}
	if result.Cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestKeyResult_NotHandled(t *testing.T) {
	result := NotHandled()

	if result.Handled {
		t.Error("expected Handled to be false")
	}
}

func TestViewBuilder(t *testing.T) {
	panel := NewBasePanel("test", "Test")
	panel.SetSize(80, 24)

	var mu sync.RWMutex
	vb := NewViewBuilder(&panel, nil, &mu)

	if vb.Panel() != &panel {
		t.Error("expected panel to be set")
	}

	vb.SetupView(80, 24)

	if vb.ContentWidth() != panel.ContentWidth() {
		t.Error("expected ContentWidth to match panel")
	}
	if vb.ContentHeight() != panel.ContentHeight() {
		t.Error("expected ContentHeight to match panel")
	}

	content := vb.Content()
	if content == nil {
		t.Fatal("expected non-nil content")
	}
	content.WriteString("test content")

	if vb.HeaderLines() != 0 {
		t.Error("expected 0 header lines initially")
	}
	vb.AddHeaderLines(2)
	if vb.HeaderLines() != 2 {
		t.Errorf("expected 2 header lines, got %d", vb.HeaderLines())
	}

	result := vb.Finish()
	if !strings.Contains(result, "test content") {
		t.Error("expected content in result")
	}
}

func TestViewBuilder_WithReadLock(t *testing.T) {
	panel := NewBasePanel("test", "Test")
	panel.SetSize(80, 24)

	var mu sync.RWMutex
	vb := NewViewBuilder(&panel, nil, &mu)
	vb.SetupView(80, 24)

	called := false
	vb.WithReadLock(func() {
		called = true
	})

	if !called {
		t.Error("expected render function to be called")
	}
}

func TestViewBuilder_RenderSearchHeader(t *testing.T) {
	panel := NewBasePanel("test", "Test")
	panel.SetSize(80, 24)

	search := NewSearchMixin(func() {})
	search.SetWidth(80)

	var mu sync.RWMutex
	vb := NewViewBuilder(&panel, search, &mu)
	vb.SetupView(80, 24)

	lines := vb.RenderSearchHeader(10)

	if lines < 0 {
		t.Error("expected non-negative header lines")
	}
}

func TestViewBuilder_RenderErrorState(t *testing.T) {
	panel := NewBasePanel("test", "Test")
	panel.SetSize(80, 24)

	var mu sync.RWMutex
	vb := NewViewBuilder(&panel, nil, &mu)
	vb.SetupView(80, 24)

	if vb.RenderErrorState(nil) {
		t.Error("expected false for nil error")
	}

	testErr := errors.New("test error")
	if !vb.RenderErrorState(testErr) {
		t.Error("expected true for non-nil error")
	}
}

func TestBuildRefreshCmd(t *testing.T) {
	config := RefreshConfig{
		NoProvider: func() tea.Msg { return nil },
		Fetch:      func(ctx context.Context) tea.Msg { return nil },
		Timeout:    time.Second,
	}

	command := BuildRefreshCmd(config)
	if command == nil {
		t.Error("expected non-nil command")
	}
}
