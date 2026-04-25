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
	"context"
	"io"
	"time"
)

// WatchdogInspector provides read-only access to watchdog state and stored
// profiles for remote diagnostics. Implementations must be safe for concurrent
// use.
type WatchdogInspector interface {
	// ListProfiles returns metadata for all stored profile files, sorted by
	// timestamp descending (newest first).
	//
	// Returns []WatchdogProfileInfo which contains the profile metadata.
	// Returns error when the profile directory cannot be read.
	ListProfiles(ctx context.Context) ([]WatchdogProfileInfo, error)

	// DownloadProfile writes the raw compressed bytes of the named profile
	// file to the provided writer. The caller is responsible for
	// decompression.
	//
	// Takes filename (string) which identifies the profile file to download.
	// Takes w (io.Writer) which receives the compressed profile data.
	//
	// Returns error when the filename is empty or the file cannot be read.
	DownloadProfile(ctx context.Context, filename string, w io.Writer) error

	// DownloadSidecar reads the JSON sidecar paired with a profile file.
	// The sidecar carries forensic context (rule attribution, runtime/metrics
	// snapshot, system stats at capture time) that lets an operator
	// understand a capture without opening the profile in pprof first.
	//
	// Takes profileFilename (string) which is the .pb.gz profile filename;
	// the inspector resolves the matching .json sidecar.
	//
	// Returns []byte which contains the sidecar JSON, or nil when no
	// sidecar exists for the profile.
	// Returns bool which is true when a sidecar was found and read.
	// Returns error when the filename is empty or the read fails for
	// reasons other than the sidecar being absent.
	DownloadSidecar(ctx context.Context, profileFilename string) ([]byte, bool, error)

	// PruneProfiles removes stored profile files. When profileType is empty,
	// all profiles are removed; otherwise only profiles of the specified type
	// are removed.
	//
	// Takes profileType (string) which filters deletion to a specific profile
	// category. Pass empty string to delete all profiles.
	//
	// Returns int which is the number of files deleted.
	// Returns error when listing or removing files fails.
	PruneProfiles(ctx context.Context, profileType string) (int, error)

	// GetWatchdogStatus returns the current watchdog state including
	// configuration, thresholds, and runtime counters.
	//
	// Returns *WatchdogStatusInfo which contains the current watchdog state.
	GetWatchdogStatus(ctx context.Context) *WatchdogStatusInfo

	// RunContentionDiagnostic enables block + mutex profiling for the
	// configured window, captures both profiles, then disables.
	//
	// The call blocks for the diagnostic window plus capture overhead.
	// The inspector serialises concurrent invocations via the underlying
	// watchdog's TryLock-based mutex.
	//
	// Returns error when the diagnostic cannot start (already running,
	// in cooldown, watchdog stopped, no profiling controller). Errors
	// from individual capture steps are logged but not propagated;
	// partial success is preferred to all-or-nothing.
	RunContentionDiagnostic(ctx context.Context) error

	// GetStartupHistory returns the most recent startup-history entries in
	// chronological order (oldest first). Lets operators inspect crash
	// loops and previous unclean exits without shelling onto the host.
	//
	// Returns []WatchdogStartupHistoryEntry which contains the parsed
	// history.
	// Returns error when the history file cannot be read or parsed.
	GetStartupHistory(ctx context.Context) ([]WatchdogStartupHistoryEntry, error)

	// ListEvents returns recently emitted watchdog events from the
	// in-memory ring.
	//
	// Used for one-shot operator queries (such as "what fired in the
	// last hour?") without setting up a streaming subscription.
	//
	// Takes limit (int) which caps the number of returned events
	// (0 = no cap, return everything in the ring).
	// Takes since (time.Time) which filters out events emitted before
	// this instant; pass zero to disable the filter.
	// Takes eventType (string) which filters by event type when non-empty.
	//
	// Returns []WatchdogEventInfo which contains the matching events in
	// chronological order (oldest first).
	ListEvents(ctx context.Context, limit int, since time.Time, eventType string) []WatchdogEventInfo

	// SubscribeEvents registers a subscriber that receives every newly
	// emitted watchdog event. The returned channel is closed when the
	// subscription is cancelled (via the returned cancel function) or
	// when the watchdog stops.
	//
	// Takes since (time.Time) which back-fills events emitted at or after
	// the given instant before the subscription begins receiving live
	// events. Pass zero to skip back-fill and stream only new events.
	//
	// Returns <-chan WatchdogEventInfo which delivers events in emission
	// order. Subscribers that fall behind drop oldest pending events
	// rather than blocking the watchdog.
	// Returns func() which cancels the subscription and closes the
	// channel.
	SubscribeEvents(ctx context.Context, since time.Time) (<-chan WatchdogEventInfo, func())
}

// WatchdogProfileInfo describes a single stored profile file for remote
// inspection.
type WatchdogProfileInfo struct {
	// Timestamp is the capture time parsed from the file name.
	Timestamp time.Time

	// Filename is the full file name including the extension.
	Filename string

	// Type is the profile category (e.g. "heap", "goroutine").
	Type string

	// SizeBytes is the compressed file size on disk.
	SizeBytes int64

	// HasSidecar is true when a paired JSON sidecar with the same base
	// name is present alongside the profile.
	HasSidecar bool
}

// WatchdogStartupHistoryEntry is the inspector-facing view of a single
// startup-history record.
type WatchdogStartupHistoryEntry struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring
	// this process.
	StartedAt time.Time

	// StoppedAt is the wall-clock instant of clean shutdown. Zero when
	// the process exited uncleanly.
	StoppedAt time.Time

	// Hostname is the host the process ran on.
	Hostname string

	// Version is the build version of the running binary.
	Version string

	// Reason is a free-form reason recorded at stop time
	// ("clean", "unclean", "panic"). Empty when the process is still
	// running.
	Reason string

	// GomemlimitBytes is the effective Go runtime memory limit at start.
	GomemlimitBytes int64

	// PID is the operating-system process identifier.
	PID int
}

// WatchdogEventInfo is the inspector-facing view of a single watchdog
// event, including its emission timestamp. Distinct from WatchdogEvent
// (the notifier-facing struct) so the inspector surface can include
// timing without changing the notifier contract.
type WatchdogEventInfo struct {
	// EmittedAt is the wall-clock instant the watchdog emitted the
	// event.
	EmittedAt time.Time

	// Fields contains structured key-value data attached to the event.
	Fields map[string]string

	// EventType identifies the category of the event.
	EventType WatchdogEventType

	// Message is the human-readable description.
	Message string

	// Priority indicates the urgency of the event
	// (1 Normal, 2 High, 3 Critical).
	Priority WatchdogEventPriority
}

// WatchdogStatusInfo describes the current state of the runtime watchdog for
// remote inspection.
type WatchdogStatusInfo struct {
	// StartedAt records when the watchdog was started.
	StartedAt time.Time

	// ContentionDiagnosticLastRun is the most recent contention-diagnostic
	// completion time. Zero when no diagnostic has run yet.
	ContentionDiagnosticLastRun time.Time

	// ProfileDirectory is the filesystem path where profiles are stored.
	ProfileDirectory string

	// ContinuousProfilingTypes lists the profile categories the routine
	// loop captures each interval (when ContinuousProfilingEnabled is
	// true).
	ContinuousProfilingTypes []string

	// CheckInterval is the period between evaluation ticks.
	CheckInterval time.Duration

	// Cooldown is the minimum duration between consecutive captures of the
	// same profile type.
	Cooldown time.Duration

	// WarmUpDuration is the period after startup during which evaluations are
	// suppressed.
	WarmUpDuration time.Duration

	// CaptureWindow is the duration of the sliding window used for global
	// rate limiting of captures and warnings.
	CaptureWindow time.Duration

	// SchedulerLatencyP99Threshold is the p99 scheduler latency above
	// which a scheduler-latency warning is emitted.
	SchedulerLatencyP99Threshold time.Duration

	// CrashLoopWindow is the duration over which the startup history is
	// inspected for crash-loop detection.
	CrashLoopWindow time.Duration

	// ContinuousProfilingInterval is the period between routine
	// continuous-profiling captures.
	ContinuousProfilingInterval time.Duration

	// ContentionDiagnosticWindow is the period during which block + mutex
	// profiling are active during a contention diagnostic.
	ContentionDiagnosticWindow time.Duration

	// ContentionDiagnosticCooldown is the minimum interval between two
	// consecutive contention diagnostics.
	ContentionDiagnosticCooldown time.Duration

	// HeapThresholdBytes is the initial heap allocation threshold in bytes.
	HeapThresholdBytes uint64

	// HeapHighWater is the current heap allocation level that must be exceeded
	// before a new heap capture is triggered.
	HeapHighWater uint64

	// FDPressureThresholdPercent is the fraction of the soft FD limit
	// above which an FD-pressure warning is emitted.
	FDPressureThresholdPercent float64

	// GoroutineThreshold is the goroutine count above which a profile capture
	// is triggered.
	GoroutineThreshold int

	// GoroutineSafetyCeiling is the goroutine count above which captures are
	// suppressed.
	GoroutineSafetyCeiling int

	// MaxProfilesPerType is the maximum number of profile files retained per
	// profile type.
	MaxProfilesPerType int

	// MaxCapturesPerWindow is the maximum number of profile captures
	// permitted within a single CaptureWindow.
	MaxCapturesPerWindow int

	// MaxWarningsPerWindow is the maximum number of warning-only events
	// permitted within a single CaptureWindow.
	MaxWarningsPerWindow int

	// CrashLoopThreshold is the minimum number of unclean exits within
	// CrashLoopWindow that triggers a crash-loop event.
	CrashLoopThreshold int

	// ContinuousProfilingRetention is the maximum number of routine
	// profile files retained per type.
	ContinuousProfilingRetention int

	// CaptureWindowUsed is the current count of captures inside the
	// sliding window used for global rate limiting.
	CaptureWindowUsed int

	// WarningWindowUsed is the current count of warnings inside the
	// sliding warning-budget window.
	WarningWindowUsed int

	// GoroutineBaseline is the goroutine count snapshot taken at the
	// first evaluation tick, used to suppress captures during normal
	// startup growth.
	GoroutineBaseline int32

	// Enabled indicates whether the watchdog is actively monitoring.
	Enabled bool

	// Stopped indicates whether the watchdog has been shut down.
	Stopped bool

	// ContinuousProfilingEnabled indicates whether the routine
	// continuous-profiling loop is active.
	ContinuousProfilingEnabled bool

	// ContentionDiagnosticAutoFire indicates whether the contention
	// diagnostic auto-fires on repeated scheduler-latency events.
	ContentionDiagnosticAutoFire bool
}

// WatchdogNotifier delivers watchdog event notifications to external systems
// such as Slack, PagerDuty, or webhooks. Implementations must be safe for
// concurrent use.
type WatchdogNotifier interface {
	// Notify sends a watchdog event notification. Implementations should be
	// non-blocking or use internal queuing to avoid delaying the watchdog
	// evaluation loop.
	Notify(ctx context.Context, event WatchdogEvent) error
}

// WatchdogProfileUploader uploads captured diagnostic profiles to a remote
// storage backend for preservation across pod restarts. Implementations must
// be safe for concurrent use.
type WatchdogProfileUploader interface {
	// Upload stores a compressed profile with the given metadata. The data
	// parameter contains the gzip-compressed pprof profile bytes.
	Upload(ctx context.Context, profileType string, data []byte, metadata map[string]string) error
}
