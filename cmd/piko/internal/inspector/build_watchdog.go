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
	"fmt"
	"strings"
	"time"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

const (
	// watchdogEventPriorityNormal is the int wire value for the Normal
	// watchdog event priority, mirrored from the proto definition.
	watchdogEventPriorityNormal = 1

	// watchdogEventPriorityHigh is the int wire value for the High
	// watchdog event priority.
	watchdogEventPriorityHigh = 2

	// watchdogEventPriorityCritical is the int wire value for the
	// Critical watchdog event priority.
	watchdogEventPriorityCritical = 3

	// fmtDecimalInt is the printf verb used for decimal integer rendering
	// across watchdog status / table output.
	fmtDecimalInt = "%d"
)

// WatchdogEventPriorityLabel maps the int priority encoded over the
// wire to a human-readable label used in tail / table output.
//
// Takes priority (int32) which is the wire priority value.
//
// Returns string which is one of "normal", "high", "critical", or
// "unknown".
func WatchdogEventPriorityLabel(priority int32) string {
	switch priority {
	case watchdogEventPriorityNormal:
		return "normal"
	case watchdogEventPriorityHigh:
		return "high"
	case watchdogEventPriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// FilterWatchdogProfilesByType returns only profiles whose type matches
// the given filter (case-insensitive).
//
// Takes profiles ([]*pb.WatchdogProfileEntry) which is the full list.
// Takes typeFilter (string) which is the type to match against.
//
// Returns []*pb.WatchdogProfileEntry which contains only matching
// profiles. The slice is freshly allocated so callers may safely retain
// references.
func FilterWatchdogProfilesByType(profiles []*pb.WatchdogProfileEntry, typeFilter string) []*pb.WatchdogProfileEntry {
	filtered := make([]*pb.WatchdogProfileEntry, 0, len(profiles))
	for _, profile := range profiles {
		if strings.EqualFold(profile.GetType(), typeFilter) {
			filtered = append(filtered, profile)
		}
	}
	return filtered
}

// BuildWatchdogStatusCoreRows builds the lifecycle DetailRows shown at
// the top of `piko watchdog status`.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailRow rendered as the Lifecycle section.
func BuildWatchdogStatusCoreRows(response *pb.GetWatchdogStatusResponse) []DetailRow {
	statusValue := "disabled"
	if response.GetEnabled() {
		statusValue = "enabled"
	}
	if response.GetStopped() {
		statusValue = "stopped"
	}

	return []DetailRow{
		{Label: "Status", Value: statusValue, IsStatus: true},
		{Label: "Check Interval", Value: FormatMilliseconds(response.GetCheckIntervalMs())},
		{Label: "Cooldown", Value: FormatMilliseconds(response.GetCooldownMs())},
		{Label: "Capture Window", Value: FormatMilliseconds(response.GetCaptureWindowMs())},
		{Label: "Captures In Window", Value: fmt.Sprintf("%d / %d", response.GetCaptureWindowUsed(), response.GetMaxCapturesPerWindow())},
		{Label: "Warnings In Window", Value: fmt.Sprintf("%d / %d", response.GetWarningWindowUsed(), response.GetMaxWarningsPerWindow())},
		{Label: "Profile Directory", Value: response.GetProfileDirectory()},
		{Label: "Warm-Up Remaining", Value: WatchdogWarmUpRemaining(response)},
	}
}

// BuildWatchdogStatusThresholdRows builds the threshold DetailRows
// covering heap, goroutine, FD, and scheduler-latency configuration.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailRow rendered as the Thresholds section.
func BuildWatchdogStatusThresholdRows(response *pb.GetWatchdogStatusResponse) []DetailRow {
	return []DetailRow{
		{Label: "Heap Threshold", Value: FormatBytes(response.GetHeapThresholdBytes())},
		{Label: "Heap High-Water Mark", Value: FormatBytes(response.GetHeapHighWater())},
		{Label: "Goroutine Threshold", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineThreshold())},
		{Label: "Goroutine Safety Ceiling", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineSafetyCeiling())},
		{Label: "Goroutine Baseline", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineBaseline())},
		{Label: "FD Pressure Threshold", Value: fmt.Sprintf("%.0f%%", response.GetFdPressureThresholdPercent()*100)},
		{Label: "Scheduler Latency p99 Threshold", Value: FormatDurationNanos(response.GetSchedulerLatencyP99ThresholdNs())},
		{Label: "Max Profiles Per Type", Value: fmt.Sprintf(fmtDecimalInt, response.GetMaxProfilesPerType())},
	}
}

// BuildWatchdogStatusCrashLoopRows builds the crash-loop detection
// DetailRows.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailRow rendered as the Crash Loop Detection section.
func BuildWatchdogStatusCrashLoopRows(response *pb.GetWatchdogStatusResponse) []DetailRow {
	return []DetailRow{
		{Label: "Crash Loop Window", Value: FormatMilliseconds(response.GetCrashLoopWindowMs())},
		{Label: "Crash Loop Threshold", Value: fmt.Sprintf(fmtDecimalInt, response.GetCrashLoopThreshold())},
	}
}

// BuildWatchdogStatusContinuousRows builds the continuous-profiling
// DetailRows.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailRow rendered as the Continuous Profiling section.
func BuildWatchdogStatusContinuousRows(response *pb.GetWatchdogStatusResponse) []DetailRow {
	continuousProfilingValue := "disabled"
	if response.GetContinuousProfilingEnabled() {
		continuousProfilingValue = "enabled"
	}
	return []DetailRow{
		{Label: "Continuous Profiling", Value: continuousProfilingValue, IsStatus: true},
		{Label: "Continuous Profiling Interval", Value: FormatMilliseconds(response.GetContinuousProfilingIntervalMs())},
		{Label: "Continuous Profiling Types", Value: strings.Join(response.GetContinuousProfilingTypes(), ", ")},
		{Label: "Continuous Profiling Retention", Value: fmt.Sprintf(fmtDecimalInt, response.GetContinuousProfilingRetention())},
	}
}

// BuildWatchdogStatusContentionRows builds the contention-diagnostic
// DetailRows.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailRow rendered as the Contention Diagnostic section.
func BuildWatchdogStatusContentionRows(response *pb.GetWatchdogStatusResponse) []DetailRow {
	contentionAutoFireValue := "manual"
	if response.GetContentionDiagnosticAutoFire() {
		contentionAutoFireValue = "auto-fire"
	}
	return []DetailRow{
		{Label: "Contention Diagnostic Mode", Value: contentionAutoFireValue, IsStatus: true},
		{Label: "Contention Diagnostic Window", Value: FormatMilliseconds(response.GetContentionDiagnosticWindowMs())},
		{Label: "Contention Diagnostic Cooldown", Value: FormatMilliseconds(response.GetContentionDiagnosticCooldownMs())},
		{Label: "Contention Diagnostic Last Run", Value: FormatOptionalTime(response.GetContentionDiagnosticLastRunMs())},
	}
}

// WatchdogWarmUpRemaining computes how much warm-up time remains,
// returning "complete" if the warm-up period has elapsed.
//
// Takes response (*pb.GetWatchdogStatusResponse) which provides the
// warm-up duration and server start time.
//
// Returns string which is the formatted remaining warm-up time or
// "complete".
func WatchdogWarmUpRemaining(response *pb.GetWatchdogStatusResponse) string {
	warmUpDuration := time.Duration(response.GetWarmUpDurationMs()) * time.Millisecond
	if warmUpDuration == 0 {
		return "complete"
	}

	startedAt := time.UnixMilli(response.GetStartedAtMs())
	warmUpEnd := startedAt.Add(warmUpDuration)
	remaining := time.Until(warmUpEnd)

	if remaining <= 0 {
		return "complete"
	}

	return remaining.Truncate(time.Second).String()
}

// WatchdogHistoryEntry is the JSON-friendly view of a single
// startup-history entry served by the inspector.
type WatchdogHistoryEntry struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring
	// the process, formatted as RFC 3339.
	StartedAt string `json:"startedAt"`

	// StoppedAt is the clean-shutdown instant; empty when the process
	// exited uncleanly.
	StoppedAt string `json:"stoppedAt,omitempty"`

	// Hostname is the host the run executed on.
	Hostname string `json:"hostname"`

	// Version is the build version reported by the run.
	Version string `json:"version"`

	// StopReason is the free-form classification recorded at stop time
	// ("clean", "unclean", "panic"). Empty when the process is the
	// current run.
	StopReason string `json:"stopReason,omitempty"`

	// GomemlimitBytes is the effective Go memory limit at start.
	GomemlimitBytes int64 `json:"gomemlimitBytes,omitempty"`

	// PID is the operating-system process identifier.
	PID int32 `json:"pid"`
}

// BuildWatchdogHistoryEntries converts wire-form startup-history
// entries into the JSON-friendly result struct used by both the table
// renderer and JSON output mode.
//
// Takes entries ([]*pb.StartupHistoryEntry) which is the proto slice.
//
// Returns []WatchdogHistoryEntry ready for table or JSON rendering.
func BuildWatchdogHistoryEntries(entries []*pb.StartupHistoryEntry) []WatchdogHistoryEntry {
	results := make([]WatchdogHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		stoppedAt := ""
		if entry.GetStoppedAtMs() > 0 {
			stoppedAt = time.UnixMilli(entry.GetStoppedAtMs()).UTC().Format(time.RFC3339)
		}
		results = append(results, WatchdogHistoryEntry{
			StartedAt:       time.UnixMilli(entry.GetStartedAtMs()).UTC().Format(time.RFC3339),
			StoppedAt:       stoppedAt,
			Hostname:        entry.GetHostname(),
			Version:         entry.GetVersion(),
			StopReason:      entry.GetStopReason(),
			PID:             entry.GetPid(),
			GomemlimitBytes: entry.GetGomemlimitBytes(),
		})
	}
	return results
}

// BuildWatchdogHistoryRows formats history entries as table rows,
// applying fallbacks for empty StoppedAt and StopReason fields.
//
// Takes results ([]WatchdogHistoryEntry) which is the input slice.
//
// Returns [][]string with one row per entry, in column order
// PID, STARTED, STOPPED, REASON, HOST, VERSION.
func BuildWatchdogHistoryRows(results []WatchdogHistoryEntry) [][]string {
	rows := make([][]string, len(results))
	for index, entry := range results {
		stopped := entry.StoppedAt
		if stopped == "" {
			stopped = "(running or unclean)"
		}
		reason := entry.StopReason
		if reason == "" {
			reason = "-"
		}
		rows[index] = []string{
			fmt.Sprintf(fmtDecimalInt, entry.PID),
			entry.StartedAt,
			stopped,
			reason,
			entry.Hostname,
			entry.Version,
		}
	}
	return rows
}

// WatchdogEventResult is the JSON-friendly view of a single watchdog
// event.
type WatchdogEventResult struct {
	// Fields contains the structured key-value attachments.
	Fields map[string]string `json:"fields,omitempty"`

	// EmittedAt is the wall-clock instant the event was emitted, RFC 3339.
	EmittedAt string `json:"emittedAt"`

	// EventType is the snake_case event identifier
	// (e.g. "heap_threshold_exceeded").
	EventType string `json:"eventType"`

	// Message is the human-readable description.
	Message string `json:"message"`

	// Priority is 1=Normal, 2=High, 3=Critical.
	Priority int32 `json:"priority"`
}

// BuildWatchdogEventResult adapts a proto WatchdogEventMessage into the
// JSON-rendered WatchdogEventResult struct.
//
// Takes event (*pb.WatchdogEventMessage) which is the wire event.
//
// Returns WatchdogEventResult populated from the proto fields.
func BuildWatchdogEventResult(event *pb.WatchdogEventMessage) WatchdogEventResult {
	return WatchdogEventResult{
		EmittedAt: time.UnixMilli(event.GetEmittedAtMs()).UTC().Format(time.RFC3339),
		EventType: event.GetEventType(),
		Priority:  event.GetPriority(),
		Message:   event.GetMessage(),
		Fields:    event.GetFields(),
	}
}
