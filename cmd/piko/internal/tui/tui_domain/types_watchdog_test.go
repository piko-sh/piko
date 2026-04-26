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
	"testing"
	"time"

	"piko.sh/piko/wdk/clock"
)

func TestUtilisationGaugeSeverity(t *testing.T) {
	cases := []struct {
		name    string
		percent float64
		want    Severity
	}{
		{"healthy low", 0.1, SeverityHealthy},
		{"healthy high", 0.59, SeverityHealthy},
		{"warning low", 0.6, SeverityWarning},
		{"warning high", 0.79, SeverityWarning},
		{"critical low", 0.8, SeverityCritical},
		{"critical high", 0.99, SeverityCritical},
		{"saturated", 1.0, SeveritySaturated},
		{"overflowed", 1.5, SeveritySaturated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := UtilisationGauge{Percent: tc.percent}
			if got := g.Severity(); got != tc.want {
				t.Errorf("Severity(%.2f) = %d, want %d", tc.percent, got, tc.want)
			}
		})
	}
}

func TestWatchdogEventCategory(t *testing.T) {
	cases := []struct {
		eventType WatchdogEventType
		want      WatchdogEventCategory
	}{
		{WatchdogEventHeapThresholdExceeded, WatchdogEventCategoryHeap},
		{WatchdogEventHeapTrendWarning, WatchdogEventCategoryHeap},
		{WatchdogEventGoroutineThresholdExceeded, WatchdogEventCategoryGoroutine},
		{WatchdogEventGoroutineLeakDetected, WatchdogEventCategoryGoroutine},
		{WatchdogEventGCPressureWarning, WatchdogEventCategoryGC},
		{WatchdogEventCrashLoopDetected, WatchdogEventCategoryProcess},
		{WatchdogEventLoopPanicked, WatchdogEventCategoryProcess},
		{WatchdogEventContentionDiagnostic, WatchdogEventCategoryDiagnostic},
		{WatchdogEventCaptureError, WatchdogEventCategoryDiagnostic},
		{WatchdogEventType("custom"), WatchdogEventCategoryOther},
	}
	for _, tc := range cases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			e := WatchdogEvent{EventType: tc.eventType}
			if got := e.Category(); got != tc.want {
				t.Errorf("Category(%q) = %d, want %d", tc.eventType, got, tc.want)
			}
		})
	}
}

func TestWatchdogEventPriority(t *testing.T) {
	normal := WatchdogEvent{Priority: WatchdogPriorityNormal}
	high := WatchdogEvent{Priority: WatchdogPriorityHigh}
	critical := WatchdogEvent{Priority: WatchdogPriorityCritical}

	if normal.IsCritical() || normal.IsHighOrAbove() {
		t.Errorf("normal event should not be critical or high")
	}
	if high.IsCritical() {
		t.Errorf("high event should not be critical")
	}
	if !high.IsHighOrAbove() {
		t.Errorf("high event should be high-or-above")
	}
	if !critical.IsCritical() || !critical.IsHighOrAbove() {
		t.Errorf("critical event should be critical and high-or-above")
	}
}

func TestWatchdogProfileDisplaySize(t *testing.T) {
	cases := []struct {
		want  string
		bytes int64
	}{
		{bytes: 0, want: "0 B"},
		{bytes: 512, want: "512 B"},
		{bytes: 1024, want: "1.0 KiB"},
		{bytes: 2 * 1024 * 1024, want: "2.0 MiB"},
		{bytes: 3 * 1024 * 1024 * 1024, want: "3.0 GiB"},
	}
	for _, tc := range cases {
		p := WatchdogProfile{SizeBytes: tc.bytes}
		if got := p.DisplaySize(); got != tc.want {
			t.Errorf("DisplaySize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestWatchdogProfileAgeFromNow(t *testing.T) {
	captured := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	now := captured.Add(5 * time.Minute)
	mock := clock.NewMockClock(now)

	p := WatchdogProfile{Timestamp: captured}
	if got := p.AgeFromNow(mock); got != 5*time.Minute {
		t.Errorf("AgeFromNow = %v, want %v", got, 5*time.Minute)
	}

	if got := (WatchdogProfile{}).AgeFromNow(mock); got != 0 {
		t.Errorf("AgeFromNow on zero timestamp = %v, want 0", got)
	}
}

func TestWatchdogStartupEntryHelpers(t *testing.T) {
	start := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	stop := start.Add(time.Hour)
	mock := clock.NewMockClock(stop.Add(time.Hour))

	clean := WatchdogStartupEntry{StartedAt: start, StoppedAt: stop, Reason: "clean"}
	if clean.IsRunning() {
		t.Error("clean entry should not report running")
	}
	if clean.IsUnclean() {
		t.Error("clean entry should not report unclean")
	}
	if got := clean.Duration(mock); got != time.Hour {
		t.Errorf("clean duration = %v, want %v", got, time.Hour)
	}

	running := WatchdogStartupEntry{StartedAt: start}
	if !running.IsRunning() {
		t.Error("running entry should report running")
	}
	if got := running.Duration(mock); got != 2*time.Hour {
		t.Errorf("running duration = %v, want %v", got, 2*time.Hour)
	}

	unclean := WatchdogStartupEntry{StartedAt: start, StoppedAt: stop, Reason: "panic"}
	if !unclean.IsUnclean() {
		t.Error("panic entry should report unclean")
	}
}

func TestMockWatchdogProviderRoundTrip(t *testing.T) {
	ctx := context.Background()
	m := NewMockWatchdogProvider()

	status := &WatchdogStatus{Enabled: true, ProfileDirectory: "/tmp"}
	m.SetStatus(status)
	got, err := m.GetStatus(ctx)
	if err != nil || got == nil || got.ProfileDirectory != "/tmp" {
		t.Errorf("GetStatus mismatch: %+v err=%v", got, err)
	}

	profiles := []WatchdogProfile{{Filename: "heap.pb.gz", Type: "heap", SizeBytes: 1024}}
	m.SetProfiles(profiles)
	gotProfiles, err := m.ListProfiles(ctx)
	if err != nil || len(gotProfiles) != 1 {
		t.Errorf("ListProfiles mismatch: %+v err=%v", gotProfiles, err)
	}

	if err := m.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestMockWatchdogProviderPruneByType(t *testing.T) {
	ctx := context.Background()
	m := NewMockWatchdogProvider()
	m.SetProfiles([]WatchdogProfile{
		{Filename: "heap-1.pb.gz", Type: "heap"},
		{Filename: "heap-2.pb.gz", Type: "heap"},
		{Filename: "goroutine-1.pb.gz", Type: "goroutine"},
	})

	removed, err := m.PruneProfiles(ctx, "heap")
	if err != nil {
		t.Fatalf("Prune error: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	remaining, _ := m.ListProfiles(ctx)
	if len(remaining) != 1 || remaining[0].Type != "goroutine" {
		t.Errorf("unexpected remaining profiles: %+v", remaining)
	}
}

func TestMockWatchdogProviderEmitEvent(t *testing.T) {
	ctx := t.Context()

	m := NewMockWatchdogProvider()
	ch, sub, err := m.SubscribeEvents(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}
	defer sub()

	go m.EmitEvent(WatchdogEvent{
		EmittedAt: time.Now(),
		Message:   "ping",
		EventType: WatchdogEventHeapThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
	})

	select {
	case ev := <-ch:
		if ev.Message != "ping" {
			t.Errorf("event message = %q", ev.Message)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for event")
	}
}
