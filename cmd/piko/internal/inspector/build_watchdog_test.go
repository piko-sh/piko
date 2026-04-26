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

package inspector

import (
	"strings"
	"testing"
	"time"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestBuildWatchdogStatusCoreRowsDisabled(t *testing.T) {
	t.Parallel()

	response := &pb.GetWatchdogStatusResponse{}
	rows := BuildWatchdogStatusCoreRows(response)
	rowsByLabel := indexRows(rows)

	status, ok := rowsByLabel["Status"]
	if !ok {
		t.Fatalf("Status row missing")
	}
	if status.Value != "disabled" {
		t.Errorf("Status = %q, want disabled", status.Value)
	}
	if !status.IsStatus {
		t.Errorf("Status.IsStatus = false, want true")
	}
}

func TestBuildWatchdogStatusCoreRowsEnabled(t *testing.T) {
	t.Parallel()

	response := &pb.GetWatchdogStatusResponse{
		Enabled:              true,
		CheckIntervalMs:      500,
		CooldownMs:           60_000,
		CaptureWindowMs:      300_000,
		CaptureWindowUsed:    1,
		MaxCapturesPerWindow: 5,
		WarningWindowUsed:    0,
		MaxWarningsPerWindow: 10,
		ProfileDirectory:     "/tmp/profiles",
	}
	rows := BuildWatchdogStatusCoreRows(response)
	rowsByLabel := indexRows(rows)

	if rowsByLabel["Status"].Value != "enabled" {
		t.Errorf("Status = %q, want enabled", rowsByLabel["Status"].Value)
	}
	if rowsByLabel["Check Interval"].Value != "500ms" {
		t.Errorf("Check Interval = %q, want 500ms", rowsByLabel["Check Interval"].Value)
	}
	if rowsByLabel["Captures In Window"].Value != "1 / 5" {
		t.Errorf("Captures In Window = %q, want 1 / 5", rowsByLabel["Captures In Window"].Value)
	}
	if rowsByLabel["Profile Directory"].Value != "/tmp/profiles" {
		t.Errorf("Profile Directory = %q, want /tmp/profiles", rowsByLabel["Profile Directory"].Value)
	}
}

func TestBuildWatchdogStatusCoreRowsStopped(t *testing.T) {
	t.Parallel()

	response := &pb.GetWatchdogStatusResponse{Enabled: true, Stopped: true}
	rows := BuildWatchdogStatusCoreRows(response)
	rowsByLabel := indexRows(rows)
	if rowsByLabel["Status"].Value != "stopped" {
		t.Errorf("Status = %q, want stopped", rowsByLabel["Status"].Value)
	}
}

func TestBuildWatchdogStatusThresholdRows(t *testing.T) {
	t.Parallel()

	response := &pb.GetWatchdogStatusResponse{
		HeapThresholdBytes:             1024 * 1024 * 1024,
		HeapHighWater:                  512 * 1024 * 1024,
		GoroutineThreshold:             1000,
		GoroutineSafetyCeiling:         5000,
		GoroutineBaseline:              100,
		FdPressureThresholdPercent:     0.85,
		SchedulerLatencyP99ThresholdNs: int64(50 * time.Millisecond),
		MaxProfilesPerType:             20,
	}
	rows := BuildWatchdogStatusThresholdRows(response)
	rowsByLabel := indexRows(rows)

	wantLabels := []string{
		"Heap Threshold",
		"Heap High-Water Mark",
		"Goroutine Threshold",
		"Goroutine Safety Ceiling",
		"Goroutine Baseline",
		"FD Pressure Threshold",
		"Scheduler Latency p99 Threshold",
		"Max Profiles Per Type",
	}
	for _, label := range wantLabels {
		if _, ok := rowsByLabel[label]; !ok {
			t.Errorf("missing row %q", label)
		}
	}
	if rowsByLabel["Goroutine Threshold"].Value != "1000" {
		t.Errorf("Goroutine Threshold = %q, want 1000", rowsByLabel["Goroutine Threshold"].Value)
	}
	if rowsByLabel["FD Pressure Threshold"].Value != "85%" {
		t.Errorf("FD Pressure Threshold = %q, want 85%%", rowsByLabel["FD Pressure Threshold"].Value)
	}
}

func TestBuildWatchdogStatusCrashLoopRows(t *testing.T) {
	t.Parallel()

	response := &pb.GetWatchdogStatusResponse{
		CrashLoopWindowMs:  600_000,
		CrashLoopThreshold: 3,
	}
	rows := BuildWatchdogStatusCrashLoopRows(response)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	rowsByLabel := indexRows(rows)
	if rowsByLabel["Crash Loop Threshold"].Value != "3" {
		t.Errorf("Crash Loop Threshold = %q, want 3", rowsByLabel["Crash Loop Threshold"].Value)
	}
}

func TestBuildWatchdogStatusContinuousRows(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		response      *pb.GetWatchdogStatusResponse
		name          string
		wantStatus    string
		wantTypesRow  string
		wantRetention string
	}{
		{
			name:          "enabled",
			response:      &pb.GetWatchdogStatusResponse{ContinuousProfilingEnabled: true, ContinuousProfilingTypes: []string{"cpu", "heap"}, ContinuousProfilingRetention: 12},
			wantStatus:    "enabled",
			wantTypesRow:  "cpu, heap",
			wantRetention: "12",
		},
		{
			name:          "disabled",
			response:      &pb.GetWatchdogStatusResponse{},
			wantStatus:    "disabled",
			wantTypesRow:  "",
			wantRetention: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := BuildWatchdogStatusContinuousRows(tc.response)
			rowsByLabel := indexRows(rows)
			if rowsByLabel["Continuous Profiling"].Value != tc.wantStatus {
				t.Errorf("status = %q, want %q", rowsByLabel["Continuous Profiling"].Value, tc.wantStatus)
			}
			if !rowsByLabel["Continuous Profiling"].IsStatus {
				t.Errorf("IsStatus = false, want true")
			}
			if rowsByLabel["Continuous Profiling Types"].Value != tc.wantTypesRow {
				t.Errorf("types = %q, want %q", rowsByLabel["Continuous Profiling Types"].Value, tc.wantTypesRow)
			}
			if rowsByLabel["Continuous Profiling Retention"].Value != tc.wantRetention {
				t.Errorf("retention = %q, want %q", rowsByLabel["Continuous Profiling Retention"].Value, tc.wantRetention)
			}
		})
	}
}

func TestBuildWatchdogStatusContentionRows(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		response   *pb.GetWatchdogStatusResponse
		name       string
		wantStatus string
	}{
		{name: "manual mode", response: &pb.GetWatchdogStatusResponse{}, wantStatus: "manual"},
		{name: "auto-fire", response: &pb.GetWatchdogStatusResponse{ContentionDiagnosticAutoFire: true}, wantStatus: "auto-fire"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows := BuildWatchdogStatusContentionRows(tc.response)
			rowsByLabel := indexRows(rows)
			if rowsByLabel["Contention Diagnostic Mode"].Value != tc.wantStatus {
				t.Errorf("Contention Diagnostic Mode = %q, want %q", rowsByLabel["Contention Diagnostic Mode"].Value, tc.wantStatus)
			}
			if !rowsByLabel["Contention Diagnostic Mode"].IsStatus {
				t.Errorf("IsStatus = false, want true")
			}
		})
	}
}

func TestWatchdogWarmUpRemaining(t *testing.T) {
	t.Parallel()

	t.Run("zero duration returns complete", func(t *testing.T) {
		t.Parallel()
		got := WatchdogWarmUpRemaining(&pb.GetWatchdogStatusResponse{})
		if got != "complete" {
			t.Errorf("got %q, want complete", got)
		}
	})

	t.Run("warm-up already elapsed", func(t *testing.T) {
		t.Parallel()
		response := &pb.GetWatchdogStatusResponse{
			WarmUpDurationMs: 1000,
			StartedAtMs:      time.Now().Add(-time.Hour).UnixMilli(),
		}
		got := WatchdogWarmUpRemaining(response)
		if got != "complete" {
			t.Errorf("got %q, want complete", got)
		}
	})

	t.Run("warm-up still in progress", func(t *testing.T) {
		t.Parallel()
		response := &pb.GetWatchdogStatusResponse{
			WarmUpDurationMs: int64(time.Hour / time.Millisecond),
			StartedAtMs:      time.Now().UnixMilli(),
		}
		got := WatchdogWarmUpRemaining(response)
		if got == "complete" || got == "" {
			t.Errorf("got %q, want a non-empty remaining duration", got)
		}
	})
}

func TestBuildWatchdogHistoryEntries(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		got := BuildWatchdogHistoryEntries(nil)
		if len(got) != 0 {
			t.Errorf("got %d, want 0", len(got))
		}
		if got == nil {
			t.Errorf("got nil, want allocated empty slice")
		}
	})

	t.Run("populated entries", func(t *testing.T) {
		t.Parallel()
		entries := []*pb.StartupHistoryEntry{
			{
				StartedAtMs:     1700000000000,
				StoppedAtMs:     1700003600000,
				Pid:             12345,
				Hostname:        "host-1",
				Version:         "v0.1.0",
				StopReason:      "clean",
				GomemlimitBytes: 1024,
			},
			{
				StartedAtMs: 1700004000000,
				Pid:         67890,
				Hostname:    "host-2",
				Version:     "v0.2.0",
			},
		}
		got := BuildWatchdogHistoryEntries(entries)
		if len(got) != 2 {
			t.Fatalf("got %d, want 2", len(got))
		}
		if got[0].PID != 12345 {
			t.Errorf("PID = %d, want 12345", got[0].PID)
		}
		if got[0].Hostname != "host-1" {
			t.Errorf("Hostname = %q", got[0].Hostname)
		}
		if got[0].StoppedAt == "" {
			t.Errorf("StoppedAt is empty for completed entry")
		}
		if got[1].StoppedAt != "" {
			t.Errorf("StoppedAt = %q for in-progress entry, want empty", got[1].StoppedAt)
		}
	})
}

func TestBuildWatchdogHistoryRows(t *testing.T) {
	t.Parallel()

	results := []WatchdogHistoryEntry{
		{StartedAt: "2026-01-01T00:00:00Z", StoppedAt: "2026-01-01T01:00:00Z", PID: 100, Hostname: "h1", Version: "v1", StopReason: "clean"},
		{StartedAt: "2026-01-02T00:00:00Z", PID: 200, Hostname: "h2", Version: "v2"},
	}
	rows := BuildWatchdogHistoryRows(results)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	if rows[0][0] != "100" {
		t.Errorf("rows[0][PID] = %q, want 100", rows[0][0])
	}
	if rows[0][2] != "2026-01-01T01:00:00Z" {
		t.Errorf("rows[0][STOPPED] = %q", rows[0][2])
	}
	if rows[0][3] != "clean" {
		t.Errorf("rows[0][REASON] = %q", rows[0][3])
	}
	if rows[1][2] != "(running or unclean)" {
		t.Errorf("rows[1][STOPPED] = %q, want fallback string", rows[1][2])
	}
	if rows[1][3] != "-" {
		t.Errorf("rows[1][REASON] = %q, want hyphen fallback", rows[1][3])
	}
}

func TestBuildWatchdogEventResult(t *testing.T) {
	t.Parallel()

	event := &pb.WatchdogEventMessage{
		EventType:   "heap_threshold_exceeded",
		Priority:    2,
		Message:     "Heap exceeded",
		Fields:      map[string]string{"heap": "1.2GiB"},
		EmittedAtMs: 1700000000000,
	}
	got := BuildWatchdogEventResult(event)
	if got.EventType != "heap_threshold_exceeded" {
		t.Errorf("EventType = %q", got.EventType)
	}
	if got.Priority != 2 {
		t.Errorf("Priority = %d, want 2", got.Priority)
	}
	if got.Message != "Heap exceeded" {
		t.Errorf("Message = %q", got.Message)
	}
	if got.Fields["heap"] != "1.2GiB" {
		t.Errorf("Fields[heap] = %q", got.Fields["heap"])
	}
	if got.EmittedAt == "" {
		t.Errorf("EmittedAt empty")
	}

	if !strings.Contains(got.EmittedAt, "T") {
		t.Errorf("EmittedAt = %q, expected RFC 3339 form", got.EmittedAt)
	}
}

func TestWatchdogEventPriorityLabel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want string
		in   int32
	}{
		{name: "negative", in: -1, want: "unknown"},
		{name: "zero", in: 0, want: "unknown"},
		{name: "normal", in: 1, want: "normal"},
		{name: "high", in: 2, want: "high"},
		{name: "critical", in: 3, want: "critical"},
		{name: "out of range", in: 99, want: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := WatchdogEventPriorityLabel(tc.in)
			if got != tc.want {
				t.Errorf("WatchdogEventPriorityLabel(%d) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFilterWatchdogProfilesByType(t *testing.T) {
	t.Parallel()

	profiles := []*pb.WatchdogProfileEntry{
		{Type: "cpu", Filename: "cpu-1.pprof"},
		{Type: "heap", Filename: "heap-1.pprof"},
		{Type: "CPU", Filename: "cpu-2.pprof"},
		{Type: "goroutine", Filename: "g.pprof"},
	}

	testCases := []struct {
		name      string
		filter    string
		wantCount int
	}{
		{name: "exact match cpu", filter: "cpu", wantCount: 2},
		{name: "case insensitive HEAP", filter: "HEAP", wantCount: 1},
		{name: "no match", filter: "block", wantCount: 0},
		{name: "empty filter matches none", filter: "", wantCount: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FilterWatchdogProfilesByType(profiles, tc.filter)
			if len(got) != tc.wantCount {
				t.Errorf("got %d, want %d", len(got), tc.wantCount)
			}
		})
	}
}

func TestFormatDurationMillis(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want string
		in   int64
	}{
		{in: 0, want: "0ms"},
		{in: 500, want: "500ms"},
		{in: 1500, want: "1.5s"},
		{in: 90_000, want: "1m30s"},
		{in: 3_660_000, want: "1h1m"},
	}

	for _, tc := range testCases {
		if got := FormatMilliseconds(tc.in); got != tc.want {
			t.Errorf("FormatMilliseconds(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func indexRows(rows []DetailRow) map[string]DetailRow {
	out := make(map[string]DetailRow, len(rows))
	for _, row := range rows {
		out[row.Label] = row
	}
	return out
}
