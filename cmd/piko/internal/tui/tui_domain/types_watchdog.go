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
	"fmt"
	"time"

	"piko.sh/piko/wdk/clock"
)

// WatchdogEventType identifies the category of an emitted watchdog event.
// Mirrors the wire-level constants in the monitoring domain so panels can
// switch on them without depending on internal packages.
type WatchdogEventType string

const (
	// WatchdogEventHeapThresholdExceeded fires when heap allocation
	// surpasses the configured threshold.
	WatchdogEventHeapThresholdExceeded WatchdogEventType = "heap_threshold_exceeded"

	// WatchdogEventGoroutineThresholdExceeded fires when goroutine count
	// exceeds the configured threshold.
	WatchdogEventGoroutineThresholdExceeded WatchdogEventType = "goroutine_threshold_exceeded"

	// WatchdogEventGoroutineSafetyCeiling fires when goroutine count is
	// at or above the safety ceiling and capture is suppressed to avoid
	// destabilising the runtime.
	WatchdogEventGoroutineSafetyCeiling WatchdogEventType = "goroutine_safety_ceiling"

	// WatchdogEventGCPressureWarning fires when the GC CPU fraction
	// exceeds the configured threshold.
	WatchdogEventGCPressureWarning WatchdogEventType = "gc_pressure_warning"

	// WatchdogEventCaptureError fires when an asynchronous capture
	// failed and produced no profile.
	WatchdogEventCaptureError WatchdogEventType = "capture_error"

	// WatchdogEventGomemlimitNotConfigured warns once that GOMEMLIMIT
	// is unset and the heap threshold is using a fallback.
	WatchdogEventGomemlimitNotConfigured WatchdogEventType = "gomemlimit_not_configured"

	// WatchdogEventRSSThresholdExceeded fires when resident-set size
	// exceeds the configured percentage of the cgroup limit.
	WatchdogEventRSSThresholdExceeded WatchdogEventType = "rss_threshold_exceeded"

	// WatchdogEventHeapTrendWarning fires when linear regression on the
	// heap projects a threshold breach within the warning horizon.
	WatchdogEventHeapTrendWarning WatchdogEventType = "heap_trend_warning"

	// WatchdogEventGoroutineLeakDetected fires when the runtime's
	// goroutine-leak detector reports an unbounded growth signature.
	WatchdogEventGoroutineLeakDetected WatchdogEventType = "goroutine_leak_detected"

	// WatchdogEventPreDeathSnapshot is emitted just before the watchdog
	// captures a final pre-death set of profiles on terminating signals.
	WatchdogEventPreDeathSnapshot WatchdogEventType = "pre_death_snapshot"

	// WatchdogEventLoopPanicked fires when the watchdog evaluation loop
	// recovered from a panic.
	WatchdogEventLoopPanicked WatchdogEventType = "loop_panicked"

	// WatchdogEventFDPressureExceeded fires when open file descriptors
	// exceed the configured percentage of the soft limit.
	WatchdogEventFDPressureExceeded WatchdogEventType = "fd_pressure_exceeded"

	// WatchdogEventSchedulerLatencyHigh fires when p99 scheduler
	// latency exceeds the configured threshold.
	WatchdogEventSchedulerLatencyHigh WatchdogEventType = "scheduler_latency_high"

	// WatchdogEventCrashLoopDetected fires on start-up when the recent
	// startup history contains enough unclean exits to constitute a
	// crash loop.
	WatchdogEventCrashLoopDetected WatchdogEventType = "crash_loop_detected"

	// WatchdogEventPreviousCrashClassified records a classification of
	// the most recent prior unclean exit (panic, OOM, signal).
	WatchdogEventPreviousCrashClassified WatchdogEventType = "previous_crash_classified"

	// WatchdogEventRoutineProfileCaptured fires when continuous
	// profiling captures a routine profile on its periodic schedule.
	WatchdogEventRoutineProfileCaptured WatchdogEventType = "routine_profile_captured"

	// WatchdogEventContentionDiagnostic fires while a contention
	// diagnostic is running and again when it completes.
	WatchdogEventContentionDiagnostic WatchdogEventType = "contention_diagnostic"
)

// WatchdogEventPriority categorises the urgency of a watchdog event.
type WatchdogEventPriority int

const (
	// WatchdogPriorityNormal marks routine, low-urgency events.
	WatchdogPriorityNormal WatchdogEventPriority = iota + 1

	// WatchdogPriorityHigh marks events worth surfacing in the UI.
	WatchdogPriorityHigh

	// WatchdogPriorityCritical marks events demanding immediate attention.
	WatchdogPriorityCritical
)

// WatchdogEventCategory groups events into broad subject areas. Panels use
// this to colour and filter events without enumerating every event type.
type WatchdogEventCategory int

const (
	// WatchdogEventCategoryOther groups events that do not fit any other
	// category and acts as the default bucket.
	WatchdogEventCategoryOther WatchdogEventCategory = iota

	// WatchdogEventCategoryHeap groups heap, RSS, and memory-limit events.
	WatchdogEventCategoryHeap

	// WatchdogEventCategoryGoroutine groups goroutine-count and
	// goroutine-leak events.
	WatchdogEventCategoryGoroutine

	// WatchdogEventCategoryGC groups garbage-collector pressure events.
	WatchdogEventCategoryGC

	// WatchdogEventCategoryProcess groups crash-loop, pre-death, and
	// process-lifetime events.
	WatchdogEventCategoryProcess

	// WatchdogEventCategoryDiagnostic groups continuous profiling, capture
	// errors, scheduler latency, and FD-pressure events.
	WatchdogEventCategoryDiagnostic
)

// UtilisationGauge captures a "X of Y" measurement plus a derived
// percentage. Used in the Overview panel for budget and threshold meters.
type UtilisationGauge struct {
	// Used is the consumed amount.
	Used float64

	// Max is the limit.
	Max float64

	// Percent is the derived ratio (0.0-1.0+; values >1.0 are saturated).
	Percent float64
}

// Severity returns the severity band corresponding to the gauge's
// utilisation. The thresholds are 60% (warning), 80% (critical), and 100%
// (saturated).
//
// Returns Severity which is the band the gauge currently falls into.
func (u UtilisationGauge) Severity() Severity {
	switch {
	case u.Percent >= 1.0:
		return SeveritySaturated
	case u.Percent >= SeverityCriticalThreshold:
		return SeverityCritical
	case u.Percent >= SeverityWarningThreshold:
		return SeverityWarning
	default:
		return SeverityHealthy
	}
}

// WatchdogStatus is the TUI-friendly view of the watchdog's current state.
// Sub-categories are flattened into named gauges and durations so panels
// can render specific aspects without parsing nested structures.
type WatchdogStatus struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring.
	StartedAt time.Time

	// LastUpdated is the wall-clock instant of the most recent status refresh.
	LastUpdated time.Time

	// ContentionDiagnosticLastRun is the wall-clock instant the last
	// contention diagnostic completed.
	ContentionDiagnosticLastRun time.Time

	// ProfileDirectory is the on-disk path where captured profiles are stored.
	ProfileDirectory string

	// ContinuousProfilingTypes lists the profile types captured on the
	// continuous schedule.
	ContinuousProfilingTypes []string

	// CaptureBudget is the gauge tracking captures used against the budget.
	CaptureBudget UtilisationGauge

	// WarningBudget is the gauge tracking warnings used against the budget.
	WarningBudget UtilisationGauge

	// HeapBudget is the gauge of heap usage against the configured threshold.
	HeapBudget UtilisationGauge

	// Goroutines is the gauge of live goroutines against the safety ceiling.
	Goroutines UtilisationGauge

	// Cooldown is the minimum gap between two consecutive captures.
	Cooldown time.Duration

	// ContinuousProfilingInterval is the period between continuous profiles.
	ContinuousProfilingInterval time.Duration

	// FDPressureThresholdPercent is the fraction of the FD soft limit that
	// triggers an FD pressure event.
	FDPressureThresholdPercent float64

	// SchedulerLatencyP99Threshold is the p99 scheduler-latency limit.
	SchedulerLatencyP99Threshold time.Duration

	// CheckInterval is the period between watchdog evaluation ticks.
	CheckInterval time.Duration

	// GoroutineSafetyCeiling is the goroutine count above which captures are
	// suppressed to avoid destabilising the runtime.
	GoroutineSafetyCeiling int

	// WarmUpRemaining is how much of the post-startup warm-up window is
	// still pending.
	WarmUpRemaining time.Duration

	// CaptureWindow is the rolling window over which CaptureBudget is counted.
	CaptureWindow time.Duration

	// CrashLoopWindow is the rolling window evaluated for crash-loop detection.
	CrashLoopWindow time.Duration

	// ContinuousProfilingRetention is the count of continuous profiles
	// retained per type.
	ContinuousProfilingRetention int

	// ContentionDiagnosticWindow is the duration of a single contention
	// diagnostic capture.
	ContentionDiagnosticWindow time.Duration

	// ContentionDiagnosticCooldown is the minimum gap between contention
	// diagnostic runs.
	ContentionDiagnosticCooldown time.Duration

	// MaxProfilesPerType caps how many profiles of any single type are kept.
	MaxProfilesPerType int

	// CrashLoopThreshold is the unclean-exit count that constitutes a crash
	// loop within CrashLoopWindow.
	CrashLoopThreshold int

	// GoroutineBaseline is the recorded baseline goroutine count at start-up.
	GoroutineBaseline int32

	// Enabled is true when the watchdog is configured to run.
	Enabled bool

	// Stopped is true when the watchdog has been told to stop.
	Stopped bool

	// ContinuousProfilingEnabled is true when continuous profiling is active.
	ContinuousProfilingEnabled bool

	// ContentionDiagnosticAutoFire is true when contention diagnostics fire
	// automatically on threshold breaches.
	ContentionDiagnosticAutoFire bool
}

// WatchdogProfile describes a stored profile artefact on the server side
// of the watchdog.
type WatchdogProfile struct {
	// Timestamp is the capture time parsed from the file name.
	Timestamp time.Time

	// Filename is the full file name including extension.
	Filename string

	// Type is the profile category (heap, goroutine, trace, etc.).
	Type string

	// SizeBytes is the compressed file size on disk.
	SizeBytes int64

	// HasSidecar indicates whether a paired JSON sidecar exists.
	HasSidecar bool
}

// AgeFromNow returns the time since the profile was captured according to
// the supplied clock.
//
// Takes c (clock.Clock) which yields the current time.
//
// Returns time.Duration which is the elapsed time since Timestamp.
func (p WatchdogProfile) AgeFromNow(c clock.Clock) time.Duration {
	if c == nil {
		c = clock.RealClock()
	}
	if p.Timestamp.IsZero() {
		return 0
	}
	return c.Now().Sub(p.Timestamp)
}

// DisplaySize returns a human-friendly representation of the profile's
// compressed size.
//
// Returns string in units of bytes / KiB / MiB / GiB.
func (p WatchdogProfile) DisplaySize() string {
	const (
		kib = 1024
		mib = 1024 * 1024
		gib = 1024 * 1024 * 1024
	)
	switch {
	case p.SizeBytes >= gib:
		return fmt.Sprintf("%.1f GiB", float64(p.SizeBytes)/float64(gib))
	case p.SizeBytes >= mib:
		return fmt.Sprintf("%.1f MiB", float64(p.SizeBytes)/float64(mib))
	case p.SizeBytes >= kib:
		return fmt.Sprintf("%.1f KiB", float64(p.SizeBytes)/float64(kib))
	default:
		return fmt.Sprintf("%d B", p.SizeBytes)
	}
}

// WatchdogStartupEntry describes a single process start/stop record from
// the startup-history ring on the server.
type WatchdogStartupEntry struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring.
	StartedAt time.Time

	// StoppedAt is the wall-clock instant of clean shutdown. Zero when
	// the process did not exit cleanly or is still running.
	StoppedAt time.Time

	// Hostname is the host the process ran on.
	Hostname string

	// Version is the build version recorded for the entry.
	Version string

	// Reason is the free-form reason recorded at stop time
	// ("clean", "unclean", "panic"). Empty when running.
	Reason string

	// GomemlimitBytes is the effective Go memory limit at start.
	GomemlimitBytes int64

	// PID is the operating-system process identifier.
	PID int
}

// IsRunning reports whether the entry describes the live process (no
// stopped timestamp recorded).
//
// Returns bool which is true when StoppedAt is zero.
func (e WatchdogStartupEntry) IsRunning() bool {
	return e.StoppedAt.IsZero()
}

// IsUnclean reports whether the entry describes an exit that was not
// flagged clean.
//
// Returns bool which is true for unclean or panic exits.
func (e WatchdogStartupEntry) IsUnclean() bool {
	return e.Reason == "unclean" || e.Reason == "panic"
}

// Duration returns how long the process ran. For still-running processes
// this measures the elapsed time at the supplied clock.
//
// Takes c (clock.Clock) which yields the current time.
//
// Returns time.Duration which is the run duration.
func (e WatchdogStartupEntry) Duration(c clock.Clock) time.Duration {
	if c == nil {
		c = clock.RealClock()
	}
	if e.StartedAt.IsZero() {
		return 0
	}
	stop := e.StoppedAt
	if stop.IsZero() {
		stop = c.Now()
	}
	return stop.Sub(e.StartedAt)
}

// WatchdogEvent is the TUI-friendly view of a single watchdog event with
// its emission timestamp.
type WatchdogEvent struct {
	// EmittedAt is when the watchdog fired the event.
	EmittedAt time.Time

	// Fields are structured key-value attachments.
	Fields map[string]string

	// Message is the human-readable description.
	Message string

	// EventType identifies the category of the event.
	EventType WatchdogEventType

	// Priority indicates the urgency.
	Priority WatchdogEventPriority
}

// IsCritical reports whether the event is at critical priority.
//
// Returns bool which is true when Priority == WatchdogPriorityCritical.
func (e WatchdogEvent) IsCritical() bool {
	return e.Priority == WatchdogPriorityCritical
}

// IsHighOrAbove reports whether the event is at high or critical priority.
//
// Returns bool which is true when Priority is high or critical.
func (e WatchdogEvent) IsHighOrAbove() bool {
	return e.Priority >= WatchdogPriorityHigh
}

// Category returns the broad subject area of the event for filtering and
// colouring.
//
// Returns WatchdogEventCategory which is the matched category.
func (e WatchdogEvent) Category() WatchdogEventCategory {
	switch e.EventType {
	case WatchdogEventHeapThresholdExceeded,
		WatchdogEventHeapTrendWarning,
		WatchdogEventRSSThresholdExceeded,
		WatchdogEventGomemlimitNotConfigured:
		return WatchdogEventCategoryHeap
	case WatchdogEventGoroutineThresholdExceeded,
		WatchdogEventGoroutineSafetyCeiling,
		WatchdogEventGoroutineLeakDetected:
		return WatchdogEventCategoryGoroutine
	case WatchdogEventGCPressureWarning:
		return WatchdogEventCategoryGC
	case WatchdogEventCrashLoopDetected,
		WatchdogEventPreviousCrashClassified,
		WatchdogEventPreDeathSnapshot,
		WatchdogEventLoopPanicked:
		return WatchdogEventCategoryProcess
	case WatchdogEventContentionDiagnostic,
		WatchdogEventRoutineProfileCaptured,
		WatchdogEventCaptureError,
		WatchdogEventSchedulerLatencyHigh,
		WatchdogEventFDPressureExceeded:
		return WatchdogEventCategoryDiagnostic
	default:
		return WatchdogEventCategoryOther
	}
}

// WatchdogEventQuery is the parameter set for ListEvents calls.
type WatchdogEventQuery struct {
	// Since filters out events emitted before this instant; zero disables
	// the filter.
	Since time.Time

	// EventType filters by event type when non-empty.
	EventType string

	// Limit caps the result count; zero means no cap.
	Limit int
}
