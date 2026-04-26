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
	"maps"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// convertWatchdogStatus converts the gRPC status response into the
// TUI-friendly WatchdogStatus, computing derived values such as the
// utilisation gauges and the warm-up remaining duration.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the proto.
//
// Returns *tui_domain.WatchdogStatus which is ready for panel rendering.
func convertWatchdogStatus(response *pb.GetWatchdogStatusResponse) *tui_domain.WatchdogStatus {
	if response == nil {
		return nil
	}

	startedAt := timeFromMillis(response.GetStartedAtMs())
	warmUpDur := durationFromMillis(response.GetWarmUpDurationMs())
	now := time.Now()

	warmUpRemaining := time.Duration(0)
	if !startedAt.IsZero() && warmUpDur > 0 {
		elapsed := now.Sub(startedAt)
		if elapsed < warmUpDur {
			warmUpRemaining = warmUpDur - elapsed
		}
	}

	captureBudget := makeGauge(float64(response.GetCaptureWindowUsed()), float64(response.GetMaxCapturesPerWindow()))
	warningBudget := makeGauge(float64(response.GetWarningWindowUsed()), float64(response.GetMaxWarningsPerWindow()))
	heapBudget := makeGauge(float64(response.GetHeapHighWater()), float64(response.GetHeapThresholdBytes()))
	goroutines := makeGauge(0, float64(response.GetGoroutineThreshold()))

	return &tui_domain.WatchdogStatus{
		StartedAt:                    startedAt,
		LastUpdated:                  now,
		ContentionDiagnosticLastRun:  timeFromMillis(response.GetContentionDiagnosticLastRunMs()),
		ProfileDirectory:             response.GetProfileDirectory(),
		ContinuousProfilingTypes:     append([]string{}, response.GetContinuousProfilingTypes()...),
		CaptureBudget:                captureBudget,
		WarningBudget:                warningBudget,
		HeapBudget:                   heapBudget,
		Goroutines:                   goroutines,
		GoroutineSafetyCeiling:       safeconv.Int32ToInt(response.GetGoroutineSafetyCeiling()),
		GoroutineBaseline:            response.GetGoroutineBaseline(),
		FDPressureThresholdPercent:   response.GetFdPressureThresholdPercent(),
		SchedulerLatencyP99Threshold: durationFromNanos(response.GetSchedulerLatencyP99ThresholdNs()),
		CheckInterval:                durationFromMillis(response.GetCheckIntervalMs()),
		Cooldown:                     durationFromMillis(response.GetCooldownMs()),
		WarmUpRemaining:              warmUpRemaining,
		CaptureWindow:                durationFromMillis(response.GetCaptureWindowMs()),
		CrashLoopWindow:              durationFromMillis(response.GetCrashLoopWindowMs()),
		ContinuousProfilingInterval:  durationFromMillis(response.GetContinuousProfilingIntervalMs()),
		ContentionDiagnosticWindow:   durationFromMillis(response.GetContentionDiagnosticWindowMs()),
		ContentionDiagnosticCooldown: durationFromMillis(response.GetContentionDiagnosticCooldownMs()),
		MaxProfilesPerType:           safeconv.Int32ToInt(response.GetMaxProfilesPerType()),
		CrashLoopThreshold:           safeconv.Int32ToInt(response.GetCrashLoopThreshold()),
		ContinuousProfilingRetention: safeconv.Int32ToInt(response.GetContinuousProfilingRetention()),
		Enabled:                      response.GetEnabled(),
		Stopped:                      response.GetStopped(),
		ContinuousProfilingEnabled:   response.GetContinuousProfilingEnabled(),
		ContentionDiagnosticAutoFire: response.GetContentionDiagnosticAutoFire(),
	}
}

// convertWatchdogProfile converts a single profile entry.
//
// Takes entry (*pb.WatchdogProfileEntry) which is the wire entry.
//
// Returns tui_domain.WatchdogProfile which is the TUI-side type
// (zero value when entry is nil).
func convertWatchdogProfile(entry *pb.WatchdogProfileEntry) tui_domain.WatchdogProfile {
	if entry == nil {
		return tui_domain.WatchdogProfile{}
	}
	return tui_domain.WatchdogProfile{
		Timestamp:  timeFromMillis(entry.GetTimestampMs()),
		Filename:   entry.GetFilename(),
		Type:       entry.GetType(),
		SizeBytes:  entry.GetSizeBytes(),
		HasSidecar: entry.GetHasSidecar(),
	}
}

// convertStartupEntry converts a single startup-history entry.
//
// Takes entry (*pb.StartupHistoryEntry) which is the wire entry.
//
// Returns tui_domain.WatchdogStartupEntry which is the TUI-side type
// (zero value when entry is nil).
func convertStartupEntry(entry *pb.StartupHistoryEntry) tui_domain.WatchdogStartupEntry {
	if entry == nil {
		return tui_domain.WatchdogStartupEntry{}
	}
	return tui_domain.WatchdogStartupEntry{
		StartedAt:       timeFromMillis(entry.GetStartedAtMs()),
		StoppedAt:       timeFromMillis(entry.GetStoppedAtMs()),
		Hostname:        entry.GetHostname(),
		Version:         entry.GetVersion(),
		Reason:          entry.GetStopReason(),
		GomemlimitBytes: entry.GetGomemlimitBytes(),
		PID:             safeconv.Int32ToInt(entry.GetPid()),
	}
}

// convertWatchdogEvent converts a wire event into the TUI-side type.
//
// Takes msg (*pb.WatchdogEventMessage) which is the wire event.
//
// Returns tui_domain.WatchdogEvent which is the TUI-side event
// (zero value when msg is nil).
func convertWatchdogEvent(msg *pb.WatchdogEventMessage) tui_domain.WatchdogEvent {
	if msg == nil {
		return tui_domain.WatchdogEvent{}
	}
	fields := make(map[string]string, len(msg.GetFields()))
	maps.Copy(fields, msg.GetFields())
	return tui_domain.WatchdogEvent{
		EmittedAt: timeFromMillis(msg.GetEmittedAtMs()),
		Fields:    fields,
		Message:   msg.GetMessage(),
		EventType: tui_domain.WatchdogEventType(msg.GetEventType()),
		Priority:  tui_domain.WatchdogEventPriority(msg.GetPriority()),
	}
}

// makeGauge constructs a UtilisationGauge from used/limit values, with
// the percentage derived such that limit==0 produces zero.
//
// Takes used (float64) which is the consumed quantity.
// Takes limit (float64) which is the maximum allowed; zero disables
// the percentage calculation.
//
// Returns tui_domain.UtilisationGauge which is the populated gauge.
func makeGauge(used, limit float64) tui_domain.UtilisationGauge {
	gauge := tui_domain.UtilisationGauge{Used: used, Max: limit}
	if limit > 0 {
		gauge.Percent = used / limit
	}
	return gauge
}

// timeFromMillis converts a unix-millis timestamp to time.Time. Zero
// values map to a zero time.Time so callers can detect "not set".
//
// Takes ms (int64) which is the unix-milliseconds timestamp.
//
// Returns time.Time which is the converted instant or the zero value
// when ms is zero.
func timeFromMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// durationFromMillis converts a millisecond integer to time.Duration.
//
// Takes ms (int64) which is the duration in milliseconds.
//
// Returns time.Duration which is the converted duration.
func durationFromMillis(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// durationFromNanos converts a nanosecond integer to time.Duration.
//
// Takes ns (int64) which is the duration in nanoseconds.
//
// Returns time.Duration which is the converted duration.
func durationFromNanos(ns int64) time.Duration {
	return time.Duration(ns) * time.Nanosecond
}
