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

package monitoring_domain

import (
	"fmt"
	"strconv"
	"time"
)

// WatchdogEventType identifies the category of a watchdog event.
type WatchdogEventType string

const (
	// WatchdogEventHeapThresholdExceeded is emitted when heap allocation
	// exceeds the configured high-water mark and a profile is captured.
	WatchdogEventHeapThresholdExceeded WatchdogEventType = "heap_threshold_exceeded"

	// WatchdogEventGoroutineThresholdExceeded is emitted when the goroutine
	// count exceeds the configured threshold and a profile is captured.
	WatchdogEventGoroutineThresholdExceeded WatchdogEventType = "goroutine_threshold_exceeded"

	// WatchdogEventGoroutineSafetyCeiling is emitted when the goroutine count
	// exceeds the safety ceiling and profile capture is suppressed.
	WatchdogEventGoroutineSafetyCeiling WatchdogEventType = "goroutine_safety_ceiling"

	// WatchdogEventGCPressureWarning is emitted when GC CPU fraction exceeds
	// the configured threshold.
	WatchdogEventGCPressureWarning WatchdogEventType = "gc_pressure_warning"

	// WatchdogEventCaptureError is emitted when a profile capture or storage
	// operation fails.
	WatchdogEventCaptureError WatchdogEventType = "capture_error"

	// WatchdogEventGomemlimitNotConfigured is emitted at startup when
	// GOMEMLIMIT is not set.
	WatchdogEventGomemlimitNotConfigured WatchdogEventType = "gomemlimit_not_configured"

	// WatchdogEventRSSThresholdExceeded is emitted when RSS approaches the
	// cgroup memory limit.
	WatchdogEventRSSThresholdExceeded WatchdogEventType = "rss_threshold_exceeded"

	// WatchdogEventHeapTrendWarning is emitted when heap growth rate projects
	// a breach within the configured warning horizon.
	WatchdogEventHeapTrendWarning WatchdogEventType = "heap_trend_warning"

	// WatchdogEventGoroutineLeakDetected is emitted when the Go 1.26
	// goroutine leak profile finds unreachable blocked goroutines.
	WatchdogEventGoroutineLeakDetected WatchdogEventType = "goroutine_leak_detected"

	// WatchdogEventPreDeathSnapshot is emitted when a pre-shutdown diagnostic
	// snapshot is captured.
	WatchdogEventPreDeathSnapshot WatchdogEventType = "pre_death_snapshot"

	// WatchdogEventLoopPanicked is emitted when the watchdog evaluation loop
	// panics. The loop does not auto-restart by design; this event makes the
	// failure externally visible.
	WatchdogEventLoopPanicked WatchdogEventType = "loop_panicked"

	// WatchdogEventFDPressureExceeded is emitted when the open file
	// descriptor count approaches the soft RLIMIT_NOFILE.
	WatchdogEventFDPressureExceeded WatchdogEventType = "fd_pressure_exceeded"

	// WatchdogEventSchedulerLatencyHigh is emitted when the runtime
	// /sched/latencies:seconds p99 exceeds the configured threshold,
	// indicating goroutine starvation or scheduler contention.
	WatchdogEventSchedulerLatencyHigh WatchdogEventType = "scheduler_latency_high"

	// WatchdogEventCrashLoopDetected is emitted at startup when the recent
	// startup-history shows multiple unclean exits within a short window,
	// indicating the process is in a crash loop.
	WatchdogEventCrashLoopDetected WatchdogEventType = "crash_loop_detected"

	// WatchdogEventPreviousCrashClassified is emitted at startup when the
	// most recent startup-history entry has no clean stop marker, indicating
	// the previous process exited uncleanly.
	WatchdogEventPreviousCrashClassified WatchdogEventType = "previous_crash_classified"

	// WatchdogEventRoutineProfileCaptured is emitted (when notification is
	// enabled) for each continuous-profiling routine capture. Informational
	// only -- do not page on this event.
	WatchdogEventRoutineProfileCaptured WatchdogEventType = "routine_profile_captured"

	// WatchdogEventContentionDiagnostic is emitted at the start and end of
	// a contention diagnostic so external observability sees the change in
	// runtime overhead.
	WatchdogEventContentionDiagnostic WatchdogEventType = "contention_diagnostic"
)

// WatchdogEventPriority indicates the urgency of a watchdog event.
type WatchdogEventPriority int

const (
	// WatchdogPriorityNormal is for informational events that do not require
	// immediate attention.
	WatchdogPriorityNormal WatchdogEventPriority = iota + 1

	// WatchdogPriorityHigh is for events that warrant prompt investigation.
	WatchdogPriorityHigh

	// WatchdogPriorityCritical is for events indicating imminent system
	// instability.
	WatchdogPriorityCritical
)

// WatchdogEvent describes a notable runtime event detected by the watchdog.
type WatchdogEvent struct {
	// Message is a human-readable description of the event.
	Message string

	// Fields contains structured key-value data about the event.
	Fields map[string]string

	// EventType identifies the category of this event.
	EventType WatchdogEventType

	// Priority indicates the urgency of this event.
	Priority WatchdogEventPriority
}

// NewHeapThresholdEvent creates an event for when heap allocation exceeds the
// high-water mark.
//
// Takes heapAlloc (uint64) which is the current heap allocation in bytes.
// Takes highWater (uint64) which is the high-water mark that was exceeded.
//
// Returns WatchdogEvent which describes the heap threshold breach.
func NewHeapThresholdEvent(heapAlloc, highWater uint64) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventHeapThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
		Message:   "Heap allocation exceeded the high-water mark; a diagnostic heap profile has been captured",
		Fields: map[string]string{
			"heap_alloc_bytes": strconv.FormatUint(heapAlloc, 10),
			"high_water_bytes": strconv.FormatUint(highWater, 10),
		},
	}
}

// NewGoroutineThresholdEvent creates an event for when the goroutine count
// exceeds the configured threshold.
//
// Takes count (int) which is the current goroutine count.
// Takes threshold (int) which is the configured goroutine threshold.
//
// Returns WatchdogEvent which describes the goroutine threshold breach.
func NewGoroutineThresholdEvent(count, threshold int) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventGoroutineThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
		Message:   "Goroutine count exceeded the configured threshold; a diagnostic goroutine profile has been captured",
		Fields: map[string]string{
			"goroutine_count": strconv.Itoa(count),
			"threshold":       strconv.Itoa(threshold),
		},
	}
}

// NewGoroutineSafetyCeilingEvent creates an event for when the goroutine count
// exceeds the safety ceiling and captures are suppressed.
//
// Takes count (int) which is the current goroutine count.
// Takes ceiling (int) which is the configured safety ceiling.
//
// Returns WatchdogEvent which describes the safety ceiling breach.
func NewGoroutineSafetyCeilingEvent(count, ceiling int) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventGoroutineSafetyCeiling,
		Priority:  WatchdogPriorityCritical,
		Message:   "Goroutine count exceeds the safety ceiling; profile capture is suppressed to avoid further destabilising the runtime",
		Fields: map[string]string{
			"goroutine_count": strconv.Itoa(count),
			"safety_ceiling":  strconv.Itoa(ceiling),
		},
	}
}

// NewGCPressureEvent creates an event for when GC CPU fraction exceeds the
// configured threshold.
//
// Takes fraction (float64) which is the current GC CPU fraction.
// Takes threshold (float64) which is the configured GC pressure threshold.
//
// Returns WatchdogEvent which describes the GC pressure warning.
func NewGCPressureEvent(fraction, threshold float64) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventGCPressureWarning,
		Priority:  WatchdogPriorityNormal,
		Message:   "GC CPU fraction exceeded the configured threshold",
		Fields: map[string]string{
			"gc_cpu_fraction": strconv.FormatFloat(fraction, 'f', 4, 64),
			"threshold":       strconv.FormatFloat(threshold, 'f', 4, 64),
		},
	}
}

// NewCaptureErrorEvent creates an event for when a profile capture or storage
// operation fails.
//
// Takes profileType (string) which identifies the profile that failed.
// Takes err (error) which is the error that occurred during capture or
// storage.
//
// Returns WatchdogEvent which describes the capture failure.
func NewCaptureErrorEvent(profileType string, err error) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventCaptureError,
		Priority:  WatchdogPriorityHigh,
		Message:   fmt.Sprintf("Failed to capture or store %s profile", profileType),
		Fields: map[string]string{
			"profile_type": profileType,
			"error":        err.Error(),
		},
	}
}

// NewRSSThresholdEvent creates an event for when RSS approaches the cgroup
// memory limit.
//
// Takes rss (uint64) which is the current RSS in bytes.
// Takes cgroupLimit (uint64) which is the cgroup memory limit in bytes.
// Takes threshold (uint64) which is the computed threshold in bytes.
//
// Returns WatchdogEvent which describes the RSS threshold breach.
func NewRSSThresholdEvent(rss, cgroupLimit, threshold uint64) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventRSSThresholdExceeded,
		Priority:  WatchdogPriorityCritical,
		Message:   "RSS is approaching the cgroup memory limit; a heap profile has been captured",
		Fields: map[string]string{
			"rss_bytes":          strconv.FormatUint(rss, 10),
			"cgroup_limit_bytes": strconv.FormatUint(cgroupLimit, 10),
			"threshold_bytes":    strconv.FormatUint(threshold, 10),
		},
	}
}

// NewFDPressureEvent creates an event for when the open FD count approaches
// the configured fraction of the soft RLIMIT_NOFILE. FD exhaustion is
// unrecoverable for accept loops, hence Critical priority.
//
// Takes fdCount (int32) which is the observed FD count.
// Takes fdLimitSoft (int64) which is the soft FD limit.
// Takes thresholdPercent (float64) which is the configured threshold fraction.
//
// Returns WatchdogEvent which describes the FD pressure condition.
func NewFDPressureEvent(fdCount int32, fdLimitSoft int64, thresholdPercent float64) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventFDPressureExceeded,
		Priority:  WatchdogPriorityCritical,
		Message:   "Open file descriptor count is approaching the process soft limit; investigate before accept loops fail",
		Fields: map[string]string{
			"fd_count":          strconv.FormatInt(int64(fdCount), 10),
			"fd_limit_soft":     strconv.FormatInt(fdLimitSoft, 10),
			"threshold_percent": strconv.FormatFloat(thresholdPercent, 'f', 2, 64),
		},
	}
}

// NewSchedulerLatencyEvent creates an event for when the runtime/metrics
// scheduler-latency p99 exceeds the configured threshold.
//
// Takes latencyP99 (time.Duration) which is the observed p99 latency.
// Takes threshold (time.Duration) which is the configured threshold.
// Takes consecutiveCount (int) which is the count of consecutive triggers
// inside the rule's tracking window -- used by the contention diagnostic to
// decide whether to escalate.
//
// Returns WatchdogEvent which describes the scheduler latency anomaly.
func NewSchedulerLatencyEvent(latencyP99, threshold time.Duration, consecutiveCount int) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventSchedulerLatencyHigh,
		Priority:  WatchdogPriorityHigh,
		Message:   "Scheduler p99 latency exceeded the configured threshold; goroutines are waiting for CPU",
		Fields: map[string]string{
			"latency_p99":     latencyP99.String(),
			"threshold":       threshold.String(),
			"consecutive_15m": strconv.Itoa(consecutiveCount),
		},
	}
}

// NewCrashLoopDetectedEvent creates an event for when the recent startup
// history indicates the process is in a crash loop (multiple unclean exits
// within a short window).
//
// Takes uncleanInWindow (int) which is the number of unclean entries in the
// inspection window.
// Takes windowSeconds (int) which is the window duration in seconds.
//
// Returns WatchdogEvent which describes the detected crash loop.
func NewCrashLoopDetectedEvent(uncleanInWindow int, windowSeconds int) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventCrashLoopDetected,
		Priority:  WatchdogPriorityCritical,
		Message:   "Multiple recent unclean process exits detected; the service appears to be in a crash loop",
		Fields: map[string]string{
			"unclean_in_window": strconv.Itoa(uncleanInWindow),
			"window_seconds":    strconv.Itoa(windowSeconds),
		},
	}
}

// NewPreviousCrashClassifiedEvent creates an event for when the previous
// startup-history entry was missing a clean stop marker, indicating the
// previous process exited uncleanly.
//
// Takes prev (startupHistoryEntry) which is the patched previous entry.
//
// Returns WatchdogEvent which describes the unclean prior exit.
func NewPreviousCrashClassifiedEvent(prev startupHistoryEntry) WatchdogEvent {
	startedAt := ""
	if !prev.StartedAt.IsZero() {
		startedAt = prev.StartedAt.Format(time.RFC3339)
	}
	return WatchdogEvent{
		EventType: WatchdogEventPreviousCrashClassified,
		Priority:  WatchdogPriorityHigh,
		Message:   "The previous run of this process did not exit cleanly; check logs and coredumps for the prior PID",
		Fields: map[string]string{
			"prev_pid":        strconv.Itoa(prev.PID),
			"prev_started_at": startedAt,
			"prev_version":    prev.Version,
			"prev_hostname":   prev.Hostname,
		},
	}
}

// NewContentionDiagnosticEvent creates an event marking the boundary of a
// contention diagnostic. Phase indicates "started" or "completed" so the
// operator can correlate with profile artefacts.
//
// Takes phase (string) which is the diagnostic phase ("started" or
// "completed").
// Takes window (time.Duration) which is the configured diagnostic window.
//
// Returns WatchdogEvent which describes the diagnostic boundary.
func NewContentionDiagnosticEvent(phase string, window time.Duration) WatchdogEvent {
	message := "Contention diagnostic " + phase
	return WatchdogEvent{
		EventType: WatchdogEventContentionDiagnostic,
		Priority:  WatchdogPriorityNormal,
		Message:   message,
		Fields: map[string]string{
			"phase":  phase,
			"window": window.String(),
		},
	}
}

// NewRoutineProfileCapturedEvent creates an informational event for a
// continuous-profiling routine capture.
//
// Takes profileType (string) which identifies the captured profile type.
//
// Returns WatchdogEvent which describes the routine capture.
func NewRoutineProfileCapturedEvent(profileType string) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventRoutineProfileCaptured,
		Priority:  WatchdogPriorityNormal,
		Message:   "Routine profile captured",
		Fields: map[string]string{
			"profile_type": profileType,
		},
	}
}

// NewLoopPanickedEvent creates an event emitted when the watchdog evaluation
// loop panics. The loop does not auto-restart, so this event combined with a
// stale heartbeat signals "watchdog stopped".
//
// Takes panicValue (string) which is the recovered panic value formatted as a
// string.
//
// Returns WatchdogEvent which describes the loop panic.
func NewLoopPanickedEvent(panicValue string) WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventLoopPanicked,
		Priority:  WatchdogPriorityCritical,
		Message:   "Watchdog evaluation loop panicked and stopped; runtime monitoring is no longer active",
		Fields: map[string]string{
			"panic": panicValue,
		},
	}
}

// NewGomemlimitNotConfiguredEvent creates an event emitted at startup when
// GOMEMLIMIT is not set.
//
// Returns WatchdogEvent which describes the missing GOMEMLIMIT configuration.
func NewGomemlimitNotConfiguredEvent() WatchdogEvent {
	return WatchdogEvent{
		EventType: WatchdogEventGomemlimitNotConfigured,
		Priority:  WatchdogPriorityHigh,
		Message: "GOMEMLIMIT is not configured; the watchdog will use the absolute heap " +
			"threshold. In containerised environments, use piko.WithAutoMemoryLimit for " +
			"accurate OOM-aware monitoring",
	}
}
