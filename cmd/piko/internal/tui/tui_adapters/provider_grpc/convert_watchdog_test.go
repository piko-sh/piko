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

package provider_grpc

import (
	"testing"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestConvertWatchdogStatusZeroValueSafe(t *testing.T) {
	if got := convertWatchdogStatus(nil); got != nil {
		t.Errorf("convertWatchdogStatus(nil) = %v, want nil", got)
	}
}

func TestConvertWatchdogStatusPopulatedFields(t *testing.T) {
	startedMs := time.Now().UnixMilli()
	resp := &pb.GetWatchdogStatusResponse{
		Enabled:                        true,
		Stopped:                        false,
		ProfileDirectory:               "/tmp/profiles",
		CheckIntervalMs:                500,
		CooldownMs:                     120000,
		WarmUpDurationMs:               30000,
		StartedAtMs:                    startedMs,
		HeapThresholdBytes:             1024 * 1024 * 512,
		HeapHighWater:                  1024 * 1024 * 256,
		GoroutineThreshold:             5000,
		GoroutineSafetyCeiling:         10000,
		MaxProfilesPerType:             5,
		CaptureWindowMs:                900000,
		MaxCapturesPerWindow:           10,
		MaxWarningsPerWindow:           20,
		FdPressureThresholdPercent:     0.85,
		SchedulerLatencyP99ThresholdNs: int64((10 * time.Millisecond).Nanoseconds()),
		CrashLoopWindowMs:              60000,
		CrashLoopThreshold:             3,
		ContinuousProfilingEnabled:     true,
		ContinuousProfilingIntervalMs:  600000,
		ContinuousProfilingTypes:       []string{"heap", "goroutine"},
		ContinuousProfilingRetention:   8,
		ContentionDiagnosticWindowMs:   30000,
		ContentionDiagnosticCooldownMs: 1800000,
		ContentionDiagnosticAutoFire:   true,
		ContentionDiagnosticLastRunMs:  startedMs - 10000,
		GoroutineBaseline:              42,
		CaptureWindowUsed:              3,
		WarningWindowUsed:              5,
	}

	got := convertWatchdogStatus(resp)
	if got == nil {
		t.Fatalf("convertWatchdogStatus returned nil for populated input")
	}
	if !got.Enabled || got.Stopped {
		t.Errorf("Enabled/Stopped wrong: %+v", got)
	}
	if got.ProfileDirectory != "/tmp/profiles" {
		t.Errorf("ProfileDirectory = %q", got.ProfileDirectory)
	}
	if got.CheckInterval != 500*time.Millisecond {
		t.Errorf("CheckInterval = %v", got.CheckInterval)
	}
	if got.SchedulerLatencyP99Threshold != 10*time.Millisecond {
		t.Errorf("SchedulerLatencyP99Threshold = %v", got.SchedulerLatencyP99Threshold)
	}
	if len(got.ContinuousProfilingTypes) != 2 {
		t.Errorf("ContinuousProfilingTypes = %v", got.ContinuousProfilingTypes)
	}
	if got.GoroutineSafetyCeiling != 10000 {
		t.Errorf("GoroutineSafetyCeiling = %d", got.GoroutineSafetyCeiling)
	}
	if got.CaptureBudget.Used != 3 || got.CaptureBudget.Max != 10 {
		t.Errorf("CaptureBudget = %+v", got.CaptureBudget)
	}
	if got.HeapBudget.Max != float64(resp.HeapThresholdBytes) {
		t.Errorf("HeapBudget.Max = %v, want %d", got.HeapBudget.Max, resp.HeapThresholdBytes)
	}
}

func TestConvertWatchdogStatusZeroLastRun(t *testing.T) {
	got := convertWatchdogStatus(&pb.GetWatchdogStatusResponse{})
	if got == nil {
		t.Fatalf("expected non-nil status")
	}
	if !got.ContentionDiagnosticLastRun.IsZero() {
		t.Errorf("zero ms should yield zero time, got %v", got.ContentionDiagnosticLastRun)
	}
	if !got.StartedAt.IsZero() {
		t.Errorf("zero ms should yield zero StartedAt, got %v", got.StartedAt)
	}
}

func TestConvertWatchdogProfileFields(t *testing.T) {
	captured := time.Now()
	entry := &pb.WatchdogProfileEntry{
		Filename:    "heap-2026-04-25.pb.gz",
		Type:        "heap",
		TimestampMs: captured.UnixMilli(),
		SizeBytes:   1024 * 1024 * 4,
		HasSidecar:  true,
	}

	got := convertWatchdogProfile(entry)
	if got.Filename != entry.Filename {
		t.Errorf("Filename mismatch")
	}
	if got.Type != "heap" {
		t.Errorf("Type = %q", got.Type)
	}
	if got.SizeBytes != entry.SizeBytes {
		t.Errorf("SizeBytes mismatch")
	}
	if !got.HasSidecar {
		t.Errorf("HasSidecar lost in conversion")
	}
	if got.Timestamp.UnixMilli() != entry.TimestampMs {
		t.Errorf("Timestamp lost precision: got %v, want %v", got.Timestamp.UnixMilli(), entry.TimestampMs)
	}
}

func TestConvertWatchdogProfileNil(t *testing.T) {
	got := convertWatchdogProfile(nil)
	if got != (tui_domain.WatchdogProfile{}) {
		t.Errorf("nil entry should yield zero value, got %+v", got)
	}
}

func TestConvertStartupEntryFields(t *testing.T) {
	now := time.Now()
	entry := &pb.StartupHistoryEntry{
		StartedAtMs:     now.Add(-time.Hour).UnixMilli(),
		StoppedAtMs:     now.Add(-time.Minute).UnixMilli(),
		Pid:             1234,
		Hostname:        "test-host",
		Version:         "v1.2.3",
		GomemlimitBytes: 1024 * 1024 * 1024,
		StopReason:      "clean",
	}

	got := convertStartupEntry(entry)
	if got.PID != 1234 {
		t.Errorf("PID = %d", got.PID)
	}
	if got.Hostname != "test-host" {
		t.Errorf("Hostname = %q", got.Hostname)
	}
	if got.Reason != "clean" {
		t.Errorf("Reason = %q", got.Reason)
	}
	if got.GomemlimitBytes != entry.GomemlimitBytes {
		t.Errorf("Gomemlimit lost")
	}
	if got.StartedAt.IsZero() || got.StoppedAt.IsZero() {
		t.Errorf("timestamps not populated: %+v", got)
	}
}

func TestConvertStartupEntryRunning(t *testing.T) {
	now := time.Now()
	entry := &pb.StartupHistoryEntry{
		StartedAtMs: now.UnixMilli(),
	}
	got := convertStartupEntry(entry)
	if !got.StoppedAt.IsZero() {
		t.Errorf("expected zero StoppedAt for running entry")
	}
	if got.IsUnclean() {
		t.Errorf("running entry without reason should not be unclean")
	}
}

func TestConvertStartupEntryNil(t *testing.T) {
	got := convertStartupEntry(nil)
	if got != (tui_domain.WatchdogStartupEntry{}) {
		t.Errorf("nil entry should yield zero value, got %+v", got)
	}
}

func TestConvertWatchdogEventFields(t *testing.T) {
	now := time.Now()
	msg := &pb.WatchdogEventMessage{
		EventType:   "heap_threshold_exceeded",
		Priority:    2,
		Message:     "heap exceeded threshold",
		Fields:      map[string]string{"heap_alloc": "1024", "threshold": "512"},
		EmittedAtMs: now.UnixMilli(),
	}

	got := convertWatchdogEvent(msg)
	if got.EventType != tui_domain.WatchdogEventHeapThresholdExceeded {
		t.Errorf("EventType = %q", got.EventType)
	}
	if got.Priority != tui_domain.WatchdogPriorityHigh {
		t.Errorf("Priority = %d", got.Priority)
	}
	if got.Message != "heap exceeded threshold" {
		t.Errorf("Message lost")
	}
	if len(got.Fields) != 2 {
		t.Errorf("Fields length = %d", len(got.Fields))
	}
	if got.Fields["threshold"] != "512" {
		t.Errorf("Field value lost")
	}
	if got.EmittedAt.UnixMilli() != msg.EmittedAtMs {
		t.Errorf("EmittedAt lost precision")
	}
}

func TestConvertWatchdogEventNil(t *testing.T) {
	got := convertWatchdogEvent(nil)
	if got.Message != "" || got.EventType != "" {
		t.Errorf("nil event should yield zero value, got %+v", got)
	}
}

func TestMakeGaugeDerivesPercent(t *testing.T) {
	cases := []struct {
		name        string
		used, max   float64
		wantPercent float64
	}{
		{name: "half", used: 5, max: 10, wantPercent: 0.5},
		{name: "zero max", used: 5, max: 0, wantPercent: 0},
		{name: "negative max", used: 5, max: -1, wantPercent: 0},
		{name: "saturated", used: 12, max: 10, wantPercent: 1.2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gauge := makeGauge(c.used, c.max)
			if gauge.Percent != c.wantPercent {
				t.Errorf("Percent = %v, want %v", gauge.Percent, c.wantPercent)
			}
		})
	}
}

func TestTimeFromMillisZero(t *testing.T) {
	if !timeFromMillis(0).IsZero() {
		t.Errorf("zero ms should yield zero time")
	}
	got := timeFromMillis(1700000000000)
	if got.IsZero() {
		t.Errorf("non-zero ms should yield non-zero time")
	}
}

func TestDurationFromMillisAndNanos(t *testing.T) {
	if got := durationFromMillis(1500); got != 1500*time.Millisecond {
		t.Errorf("durationFromMillis = %v", got)
	}
	if got := durationFromNanos(2_500_000); got != 2500*time.Microsecond {
		t.Errorf("durationFromNanos = %v", got)
	}
}
