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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultCheckInterval is the period between watchdog evaluation ticks.
	// Short intervals improve detection speed at negligible CPU cost since
	// each tick only reads cached metrics and performs integer comparisons.
	defaultCheckInterval = 500 * time.Millisecond

	// defaultHeapThresholdPercent is the fraction of GOMEMLIMIT above which
	// heap allocation triggers a profile capture.
	defaultHeapThresholdPercent = 0.85

	// defaultHeapThresholdBytes is the absolute heap allocation threshold in
	// bytes, used when GOMEMLIMIT is not set.
	defaultHeapThresholdBytes = 512 * 1024 * 1024

	// defaultGoroutineThreshold is the goroutine count above which a goroutine
	// profile capture is triggered.
	defaultGoroutineThreshold = 10000

	// defaultGoroutineSafetyCeiling is the goroutine count above which profile
	// captures are suppressed to avoid destabilising the runtime.
	defaultGoroutineSafetyCeiling = 100000

	// defaultGCPressureThreshold is the GCCPUFraction above which a GC
	// pressure warning is emitted.
	defaultGCPressureThreshold = 0.50

	// defaultCooldown is the minimum duration between consecutive captures of
	// the same profile type.
	defaultCooldown = 2 * time.Minute

	// defaultMaxCapturesPerWindow is the maximum number of profile captures
	// permitted within a single sliding capture window.
	defaultMaxCapturesPerWindow = 5

	// defaultCaptureWindow is the duration of the sliding window used for
	// global rate limiting of profile captures.
	defaultCaptureWindow = 15 * time.Minute

	// defaultMaxWarningsPerWindow is the maximum number of warning-only
	// notifications permitted within a single CaptureWindow. Warnings (GC
	// pressure, heap trend, FD pressure, scheduler latency) do not consume
	// the capture budget so a real heap leak still gets profiled even when
	// other rules are flapping.
	defaultMaxWarningsPerWindow = 10

	// defaultHighWaterResetCooldown is the minimum duration after the high-water
	// mark was last set before it may be reset to the initial threshold.
	defaultHighWaterResetCooldown = 10 * time.Minute

	// defaultWarmUpDuration is the period after startup during which the
	// watchdog suppresses all evaluations to allow the runtime to stabilise.
	defaultWarmUpDuration = 30 * time.Second

	// defaultMaxProfilesPerType is the maximum number of profile files retained
	// per profile type before the oldest are rotated out.
	defaultMaxProfilesPerType = 5

	// defaultMaxProfileSizeBytes is the maximum number of bytes a single
	// profile capture may produce before being truncated.
	defaultMaxProfileSizeBytes = 50 * 1024 * 1024

	// defaultRSSThresholdPercent is the fraction of the cgroup memory limit
	// above which RSS triggers a profile capture.
	defaultRSSThresholdPercent = 0.85

	// defaultGoroutineLeakCheckInterval is the period between goroutine leak
	// profile evaluations.
	defaultGoroutineLeakCheckInterval = 5 * time.Minute

	// defaultFDPressureThresholdPercent is the fraction of the soft FD limit
	// above which the watchdog emits an FD pressure warning. Set high
	// because FD exhaustion is unrecoverable -- by the time we cross 80% the
	// operator wants to know.
	defaultFDPressureThresholdPercent = 0.80

	// defaultSchedulerLatencyP99Threshold is the p99 scheduler latency above
	// which the watchdog emits a scheduler-latency warning. 10 ms is a
	// commonly-used "user-visible latency" floor; sustained higher values
	// indicate goroutine starvation, GC interference, or CPU contention.
	defaultSchedulerLatencyP99Threshold = 10 * time.Millisecond

	// defaultCrashLoopWindow is the time window over which the startup
	// history is inspected for crash-loop detection. A burst of unclean
	// exits within this window triggers a CrashLoopDetected event.
	defaultCrashLoopWindow = 60 * time.Second

	// defaultCrashLoopThreshold is the minimum number of unclean exits
	// within defaultCrashLoopWindow that signal a crash loop.
	defaultCrashLoopThreshold = 3

	// defaultContinuousProfilingInterval is the period between routine
	// profile captures in continuous-profiling mode (opt-in).
	defaultContinuousProfilingInterval = 10 * time.Minute

	// defaultContinuousProfilingRetention is the number of routine profile
	// files retained per type. Older files are rotated out.
	defaultContinuousProfilingRetention = 6

	// minContinuousProfilingInterval is the lower bound on the routine
	// capture interval. Faster than this would stress the heap profiler
	// without adding much forensic value.
	minContinuousProfilingInterval = 1 * time.Minute

	// maxContinuousProfilingRetention is the upper bound on routine profile
	// retention to prevent unbounded disk usage from misconfiguration.
	maxContinuousProfilingRetention = 100

	// routineProfilePrefix is the prefix used for continuous-profiling
	// captures so their rotation is independent from threshold-triggered
	// captures.
	routineProfilePrefix = "routine-"

	// defaultContentionDiagnosticWindow is the duration mutex/block
	// profiling stays enabled during a contention diagnostic before being
	// disabled and the profiles captured.
	defaultContentionDiagnosticWindow = 60 * time.Second

	// defaultContentionDiagnosticBlockProfileRate is the runtime block
	// profile rate to set during a diagnostic. 1e6 means "sample one event
	// per 1 ms of blocking" -- low enough overhead to be safe, high enough
	// to surface real contention within a 60s window.
	defaultContentionDiagnosticBlockProfileRate = 1_000_000

	// defaultContentionDiagnosticMutexProfileFraction is the runtime mutex
	// profile fraction during a diagnostic.
	defaultContentionDiagnosticMutexProfileFraction = 100

	// defaultContentionDiagnosticConsecutiveTrigger is the number of
	// scheduler-latency events within ContentionDiagnosticTriggerWindow
	// that triggers an auto-fire diagnostic when AutoFire is enabled.
	defaultContentionDiagnosticConsecutiveTrigger = 3

	// defaultContentionDiagnosticTriggerWindow is the rolling window over
	// which scheduler-latency events are counted for auto-fire.
	defaultContentionDiagnosticTriggerWindow = 15 * time.Minute

	// defaultContentionDiagnosticCooldown is the minimum interval between
	// consecutive diagnostics so the runtime is not under continuous
	// profiling overhead.
	defaultContentionDiagnosticCooldown = 30 * time.Minute

	// minContentionDiagnosticWindow is the smallest acceptable diagnostic
	// window. Shorter than this risks empty profiles.
	minContentionDiagnosticWindow = time.Second

	// maxContentionDiagnosticWindow caps the diagnostic window to keep the
	// extra profiling overhead bounded.
	maxContentionDiagnosticWindow = 5 * time.Minute

	// contentionDiagnosticOverheadPad is the additional time budget added
	// on top of the window duration when computing the auto-fire context
	// timeout, to account for the synchronous capture step that runs after
	// the window expires.
	contentionDiagnosticOverheadPad = 30 * time.Second

	// defaultEventRingSize bounds the number of recent watchdog events
	// retained in memory for inspector queries. Sized to comfortably
	// cover one capture window (15 min) of event chatter at the
	// configured warning-budget rate.
	defaultEventRingSize = 256

	// eventSubscriberBuffer is the per-subscriber channel buffer for
	// streaming watchdog events. A subscriber that does not drain fast
	// enough drops oldest pending events rather than blocking the
	// watchdog evaluation loop.
	eventSubscriberBuffer = 64

	// maxEventSubscribers caps the number of concurrent streaming
	// subscribers attached to the watchdog. Exceeding the cap returns
	// ErrEventSubscriberCapExceeded so a runaway client cannot grow the
	// subscriber slice unbounded.
	maxEventSubscribers = 1000

	// logFieldProfileType is the structured log field key for profile type.
	logFieldProfileType = "profile_type"

	// profileTypeHeap identifies heap profile captures.
	profileTypeHeap = "heap"

	// profileTypeGoroutine identifies goroutine profile captures.
	profileTypeGoroutine = "goroutine"

	// profileTypeGoroutineLeak identifies goroutine leak profile captures.
	profileTypeGoroutineLeak = "goroutineleak"

	// profileTypeTrace identifies flight recorder trace captures.
	profileTypeTrace = "trace"

	// profileTypeHeapBaseline identifies heap baseline profiles for diffing.
	profileTypeHeapBaseline = "heap-baseline"

	// preDeathSnapshotTimeout is the maximum time allowed for the pre-death
	// diagnostic snapshot before it is abandoned.
	preDeathSnapshotTimeout = 10 * time.Second

	// notificationTimeout is the maximum time allowed for a single watchdog
	// notification delivery.
	notificationTimeout = 30 * time.Second

	// uploadTimeout is the maximum time allowed for a single watchdog profile
	// upload.
	uploadTimeout = 30 * time.Second
)

var (
	// ErrInvalidWatchdogConfig is returned when the watchdog configuration
	// contains invalid values.
	ErrInvalidWatchdogConfig = errors.New("invalid watchdog configuration")

	// ErrContentionDiagnosticInProgress is returned by RunContentionDiagnostic
	// when a diagnostic is already running.
	ErrContentionDiagnosticInProgress = errors.New("contention diagnostic already in progress")

	// ErrContentionDiagnosticCooldown is returned by RunContentionDiagnostic
	// when the cooldown since the last diagnostic has not yet elapsed.
	ErrContentionDiagnosticCooldown = errors.New("contention diagnostic cooldown active")

	// ErrProfilingControllerNil is returned by RunContentionDiagnostic when
	// no profiling controller has been configured.
	ErrProfilingControllerNil = errors.New("profiling controller not available")

	// ErrWatchdogStopped is returned by RunContentionDiagnostic when the
	// watchdog has already been stopped.
	ErrWatchdogStopped = errors.New("watchdog stopped")

	// ErrEventSubscriberCapExceeded is logged by SubscribeEvents when the
	// configured maximum number of concurrent subscribers has already been
	// reached. Existing subscribers continue to receive events; the new
	// subscription receives a pre-closed channel and a no-op cancel func.
	ErrEventSubscriberCapExceeded = errors.New("watchdog event subscriber cap exceeded")
)

var _ WatchdogInspector = (*Watchdog)(nil)

// WatchdogConfig holds the configuration for the runtime watchdog service.
type WatchdogConfig struct {
	// ProfileDirectory is the filesystem path where captured profile files are
	// stored. Must be set before the watchdog is started.
	ProfileDirectory string

	// ContinuousProfilingTypes is the list of profile types captured each
	// interval; default ["heap"], allowed values heap, goroutine, and
	// allocs. CPU, trace, block, and mutex are deliberately disallowed in
	// routine mode because they are duration-based and would block the
	// loop.
	ContinuousProfilingTypes []string

	// FDPressureThresholdPercent is the soft FD limit fraction above which
	// the watchdog emits an FD pressure warning; default 0.80, zero
	// disables. The watchdog only alerts and does not capture a profile
	// because there is nothing useful to capture for FD exhaustion.
	FDPressureThresholdPercent float64

	// CaptureWindow is the duration of the sliding window used for global rate
	// limiting.
	CaptureWindow time.Duration

	// HighWaterResetCooldown is the minimum duration after the heap high-water
	// mark was last set before it can be reset to the initial threshold.
	HighWaterResetCooldown time.Duration

	// WarmUpDuration is the period after startup during which all evaluations
	// are suppressed.
	WarmUpDuration time.Duration

	// HeapThresholdBytes is the absolute heap allocation threshold in bytes.
	// Used when GOMEMLIMIT is not configured.
	HeapThresholdBytes uint64

	// HeapThresholdPercent is the fraction of GOMEMLIMIT above which heap
	// allocation triggers a profile capture. Only used when GOMEMLIMIT is set.
	HeapThresholdPercent float64

	// GCPressureThreshold is the GCCPUFraction above which a GC pressure
	// warning is emitted.
	GCPressureThreshold float64

	// RSSThresholdPercent is the fraction of the cgroup memory limit above
	// which RSS triggers a profile capture.
	//
	// Only effective when the cgroup memory limit is available. Default: 0.85.
	RSSThresholdPercent float64

	// TrendEvaluationInterval is the period between heap trend regression
	// computations. Default: 30 seconds.
	TrendEvaluationInterval time.Duration

	// TrendWarningHorizon is the projected time-to-breach below which a heap
	// trend warning is emitted. Default: 5 minutes.
	TrendWarningHorizon time.Duration

	// TrendWindowSize is the number of heap samples to retain for linear
	// regression analysis. Default: 120 (60 seconds at 500ms interval).
	TrendWindowSize int

	// GoroutineLeakCheckInterval is the period between goroutine leak profile
	// evaluations.
	//
	// Default: 5 minutes. Only effective when the Go 1.26
	// goroutineleakprofile experiment is enabled.
	GoroutineLeakCheckInterval time.Duration

	// GoroutineThreshold is the goroutine count above which a goroutine
	// profile capture is triggered.
	GoroutineThreshold int

	// GoroutineSafetyCeiling is the goroutine count above which captures are
	// suppressed to avoid worsening an already unstable runtime.
	GoroutineSafetyCeiling int

	// MaxCapturesPerWindow is the maximum number of profile captures allowed
	// within a single CaptureWindow.
	MaxCapturesPerWindow int

	// MaxWarningsPerWindow is the maximum number of warning-only events
	// permitted within a single CaptureWindow (GC pressure, heap trend,
	// FD pressure, and scheduler-latency rules share this budget).
	//
	// Defaults to 10. Warnings live in their own budget so flapping
	// warnings cannot crowd out real heap captures.
	MaxWarningsPerWindow int

	// SchedulerLatencyP99Threshold is the p99 scheduler latency above
	// which the watchdog emits a scheduler-latency warning, sourced from
	// runtime/metrics /sched/latencies:seconds; default 10ms, zero
	// disables the rule.
	SchedulerLatencyP99Threshold time.Duration

	// Cooldown is the minimum duration between consecutive captures of the same
	// profile type.
	Cooldown time.Duration

	// MaxProfileSizeBytes is the maximum number of bytes a single profile
	// capture may produce.
	//
	// Captures exceeding this limit are discarded to prevent the watchdog
	// from worsening memory pressure. Default: 50 MiB.
	MaxProfileSizeBytes int64

	// CrashLoopThreshold is the minimum number of unclean exits within
	// CrashLoopWindow that triggers a CrashLoopDetected event. Default: 3.
	CrashLoopThreshold int

	// MaxProfilesPerType is the maximum number of profile files retained per
	// profile type before the oldest are deleted.
	MaxProfilesPerType int

	// ContinuousProfilingInterval is the period between routine captures;
	// default 10 minutes, validation enforces a minimum of 1 minute.
	ContinuousProfilingInterval time.Duration

	// CheckInterval is the period between watchdog evaluation ticks. Shorter
	// intervals detect anomalies faster at negligible CPU cost.
	CheckInterval time.Duration

	// ContinuousProfilingRetention is the maximum number of routine
	// profile files retained per type. Default: 6 (~1 hour of history at
	// 10-minute cadence).
	ContinuousProfilingRetention int

	// ContentionDiagnosticCooldown is the minimum interval between two
	// consecutive diagnostics. Default: 30 minutes.
	ContentionDiagnosticCooldown time.Duration

	// ContentionDiagnosticWindowDuration is the period during which block
	// + mutex profiling are active before the diagnostic captures the
	// resulting profiles and disables them; default 60s, allowed range
	// 1s to 5m.
	ContentionDiagnosticWindowDuration time.Duration

	// ContentionDiagnosticBlockProfileRate is the value passed to
	// runtime.SetBlockProfileRate for the duration of the diagnostic.
	// Default: 1e6 (one sample per 1ms of blocking).
	ContentionDiagnosticBlockProfileRate int

	// ContentionDiagnosticMutexProfileFraction is the value passed to
	// runtime.SetMutexProfileFraction for the duration of the diagnostic.
	// Default: 100 (one in 100 mutex contentions sampled).
	ContentionDiagnosticMutexProfileFraction int

	// CrashLoopWindow is the duration over which the startup history is
	// inspected for crash-loop detection; default 60s, zero disables
	// startup-history processing entirely (no crash-loop alerts and no
	// previous-crash classification).
	CrashLoopWindow time.Duration

	// ContentionDiagnosticConsecutiveTrigger is the number of
	// scheduler-latency events within ContentionDiagnosticTriggerWindow
	// that trigger auto-fire. Default: 3.
	ContentionDiagnosticConsecutiveTrigger int

	// ContentionDiagnosticTriggerWindow is the rolling window for
	// counting scheduler-latency events for auto-fire. Default: 15
	// minutes.
	ContentionDiagnosticTriggerWindow time.Duration

	// ContinuousProfilingNotify enables informational notifications for
	// each routine capture. Default: false (suppressed) to avoid notifier
	// flooding.
	ContinuousProfilingNotify bool

	// ContinuousProfilingEnabled enables low-frequency routine profile
	// captures so post-mortem operators have recent profiles even when no
	// threshold breach occurred. Default: false (opt-in).
	ContinuousProfilingEnabled bool

	// ContentionDiagnosticAutoFire enables automatic diagnostic firing
	// when scheduler-latency events repeat. Default: false (manual only).
	ContentionDiagnosticAutoFire bool

	// DeltaProfilingEnabled enables storing a baseline heap profile alongside
	// each capture so the user can compute a diff. Default: false.
	DeltaProfilingEnabled bool

	// IncludeGoroutineStacks toggles per-goroutine stack capture. When enabled,
	// each goroutine profile firing also writes a human-readable .txt sidecar
	// containing the full stack of every goroutine (pprof debug=2), alongside
	// the existing aggregated .pb.gz binary profile.
	//
	// Useful when investigating goroutine leaks where you need to know the
	// exact call site or closure-captured arguments (e.g. which channel a
	// publisher is blocked on). Disabled by default because the sidecar can be
	// tens of megabytes per dump for processes with many thousand goroutines.
	IncludeGoroutineStacks bool

	// Enabled controls whether the watchdog is active. When false, Start is a
	// no-op.
	Enabled bool
}

// Watchdog monitors runtime metrics and automatically captures diagnostic
// profiles when anomalies are detected. It periodically reads system
// statistics from a SystemCollector and evaluates heap usage, goroutine
// counts, and GC pressure against configurable thresholds.
//
// Safe for concurrent use; mutable state is protected by a mutex and the
// stop channel coordinates shutdown. Fields that are only accessed from the
// single loop goroutine (heapTrendBuffer, lastTrendEvaluation,
// lastGoroutineLeakCheck) are safe by goroutine confinement.
type Watchdog struct {
	// lastContentionDiagnosticAt is the most recent successful diagnostic
	// completion timestamp; used to enforce the cooldown.
	lastContentionDiagnosticAt time.Time

	// heapHighWaterSetAt is the timestamp when heapHighWater was last updated,
	// used to determine when the high-water mark may be reset.
	heapHighWaterSetAt time.Time

	// lastTrendEvaluation is the timestamp of the most recent heap trend
	// regression computation. Confined to the loop goroutine.
	lastTrendEvaluation time.Time

	// lastGoroutineLeakCheck is the timestamp of the most recent goroutine
	// leak profile evaluation. Confined to the loop goroutine.
	lastGoroutineLeakCheck time.Time

	// startedAt records when the watchdog was started, used to enforce the
	// warm-up period. Written once in Start before the loop goroutine begins.
	startedAt time.Time

	// profileUploader uploads captured profiles to a remote storage backend
	// for preservation across pod restarts.
	//
	// May be nil when remote storage is not configured. Set before Start and
	// not changed afterwards.
	profileUploader WatchdogProfileUploader

	// notifier delivers event notifications to external systems such as Slack
	// or PagerDuty.
	//
	// May be nil when notifications are not configured. Set before Start and
	// not changed afterwards.
	notifier WatchdogNotifier

	// clock provides time operations; replaceable for testing.
	clock clock.Clock

	// profilingController captures pprof profiles on demand.
	//
	// May be nil if profiling is not enabled. Set before Start via
	// SetProfilingController.
	profilingController ProfilingController

	// stopCh signals the watchdog loop to exit when closed.
	stopCh chan struct{}

	// profileStore manages writing and rotating profile files on disk.
	profileStore *profileStore

	// heapTrendBuffer holds recent heap allocation samples for linear
	// regression-based trend detection.
	//
	// Nil when trend detection is disabled. Confined to the loop goroutine.
	heapTrendBuffer *heapTrendBuffer

	// lastCaptureTime records the most recent capture timestamp per profile
	// type, used for per-type cooldown enforcement.
	lastCaptureTime map[string]time.Time

	// lastWarningTime records the most recent warning timestamp per rule
	// type, used for per-rule warning cooldown enforcement. Distinct from
	// lastCaptureTime so warning rules and capture rules cannot starve
	// each other.
	lastWarningTime map[string]time.Time

	// systemCollector provides the current system statistics for evaluation.
	systemCollector *SystemCollector

	// hostname is the machine hostname, cached at construction time for use
	// in notifications and upload metadata.
	hostname string

	// captureTimestamps is a sliding window of all capture timestamps, used
	// for global rate limiting.
	captureTimestamps []time.Time

	// schedulerLatencyEvents is a small bounded ring of recent
	// scheduler-latency event timestamps. Used by the contention diagnostic
	// auto-fire to detect "N events in M minutes".
	schedulerLatencyEvents []time.Time

	// eventRing is a bounded ring buffer of recently emitted watchdog
	// events that sendNotification appends to so the inspector surface
	// (ListEvents and SubscribeEvents) can serve recent history; older
	// entries are evicted when the ring is full.
	eventRing []WatchdogEventInfo

	// eventSubscribers tracks active streaming subscribers receiving
	// newly emitted events. Subscribers that fall behind drop pending
	// events rather than blocking the watchdog.
	eventSubscribers []*watchdogEventSubscriber

	// warningTimestamps is a sliding window of warning emission timestamps
	// used for the warning-budget global rate limit. Tracked separately so
	// chatty warning rules cannot consume the capture budget.
	warningTimestamps []time.Time

	// previousHeapProfile stores the compressed bytes of the last captured
	// heap profile, used to provide a diff baseline for the next capture.
	//
	// Nil until the first heap capture. Protected by mu.
	previousHeapProfile []byte

	// config holds the watchdog configuration including thresholds and timing.
	config WatchdogConfig

	// backgroundWG tracks in-flight notification and upload goroutines so
	// that Stop can wait for all background work to complete.
	backgroundWG sync.WaitGroup

	// captureWG tracks in-flight profile capture goroutines so that Stop can
	// wait for them to finish before closing the profile store.
	captureWG sync.WaitGroup

	// initialHeapThreshold is the heap allocation threshold computed at
	// startup from GOMEMLIMIT or the configured byte threshold.
	initialHeapThreshold uint64

	// heapHighWater is the current heap allocation level that must be exceeded
	// before a new heap capture is triggered. Starts at initialHeapThreshold
	// and escalates with each capture.
	heapHighWater uint64

	// gomemlimit is the Go runtime memory limit, or math.MaxInt64 when no
	// limit is set.
	gomemlimit int64

	// cgroupMemoryLimit is the cached cgroup memory limit in bytes captured
	// once at Start. The cgroup limit is stable for the lifetime of the
	// process; reading it on every tick would be wasted work.
	cgroupMemoryLimit uint64

	// contentionMu serialises access to the contention diagnostic; only
	// one diagnostic runs at a time so block/mutex profile rates are not
	// stamped on by concurrent runs.
	contentionMu sync.Mutex

	// mu guards mutable state including heapHighWater, lastCaptureTime,
	// captureTimestamps, profilingController, previousHeapProfile, and
	// stopped.
	mu sync.Mutex

	// goroutineBaseline is the goroutine count at startup, used to avoid
	// spurious captures during normal operation. Accessed atomically.
	goroutineBaseline atomic.Int32

	// stopped indicates whether Stop has been called.
	stopped bool

	// started indicates whether Start has already been called so a second
	// invocation does not spawn duplicate loops or rewrite startup history.
	started bool

	// goroutineLeakAvailable indicates whether the Go 1.26 goroutine leak
	// profile experiment is enabled. Set once in Start before the loop
	// goroutine begins.
	goroutineLeakAvailable bool

	// heapProfilingDisabled is set in Start when runtime.MemProfileRate is
	// zero; heap-based rules and heap/allocs continuous captures are skipped.
	heapProfilingDisabled bool
}

// WatchdogOption configures optional dependencies on a Watchdog.
type WatchdogOption func(*Watchdog)

// DefaultWatchdogConfig returns a WatchdogConfig populated with production
// defaults.
//
// Returns WatchdogConfig which contains sensible default values for all
// watchdog settings.
func DefaultWatchdogConfig() WatchdogConfig {
	return WatchdogConfig{
		CheckInterval:                            defaultCheckInterval,
		HeapThresholdPercent:                     defaultHeapThresholdPercent,
		HeapThresholdBytes:                       defaultHeapThresholdBytes,
		GoroutineThreshold:                       defaultGoroutineThreshold,
		GoroutineSafetyCeiling:                   defaultGoroutineSafetyCeiling,
		GCPressureThreshold:                      defaultGCPressureThreshold,
		RSSThresholdPercent:                      defaultRSSThresholdPercent,
		TrendWindowSize:                          defaultTrendWindowSize,
		TrendEvaluationInterval:                  defaultTrendEvaluationInterval,
		TrendWarningHorizon:                      defaultTrendWarningHorizon,
		GoroutineLeakCheckInterval:               defaultGoroutineLeakCheckInterval,
		Cooldown:                                 defaultCooldown,
		MaxCapturesPerWindow:                     defaultMaxCapturesPerWindow,
		MaxWarningsPerWindow:                     defaultMaxWarningsPerWindow,
		CaptureWindow:                            defaultCaptureWindow,
		HighWaterResetCooldown:                   defaultHighWaterResetCooldown,
		WarmUpDuration:                           defaultWarmUpDuration,
		MaxProfilesPerType:                       defaultMaxProfilesPerType,
		MaxProfileSizeBytes:                      defaultMaxProfileSizeBytes,
		FDPressureThresholdPercent:               defaultFDPressureThresholdPercent,
		SchedulerLatencyP99Threshold:             defaultSchedulerLatencyP99Threshold,
		CrashLoopWindow:                          defaultCrashLoopWindow,
		CrashLoopThreshold:                       defaultCrashLoopThreshold,
		ContinuousProfilingInterval:              defaultContinuousProfilingInterval,
		ContinuousProfilingRetention:             defaultContinuousProfilingRetention,
		ContinuousProfilingTypes:                 []string{profileTypeHeap},
		ContentionDiagnosticWindowDuration:       defaultContentionDiagnosticWindow,
		ContentionDiagnosticBlockProfileRate:     defaultContentionDiagnosticBlockProfileRate,
		ContentionDiagnosticMutexProfileFraction: defaultContentionDiagnosticMutexProfileFraction,
		ContentionDiagnosticConsecutiveTrigger:   defaultContentionDiagnosticConsecutiveTrigger,
		ContentionDiagnosticTriggerWindow:        defaultContentionDiagnosticTriggerWindow,
		ContentionDiagnosticCooldown:             defaultContentionDiagnosticCooldown,
		Enabled:                                  true,
	}
}

// NewWatchdog creates a new runtime watchdog that monitors system metrics and
// captures diagnostic profiles when thresholds are exceeded.
//
// Takes config (WatchdogConfig) which provides the threshold and timing
// configuration. The struct is intentionally passed by value because
// callers commonly compose it from DefaultWatchdogConfig at the call
// site; the 296-byte copy on a once-per-process constructor path is
// negligible.
// Takes systemCollector (*SystemCollector) which provides the system
// statistics for periodic evaluation.
// Takes opts (...WatchdogOption) which provides optional configuration
// functions to customise clock and sandbox dependencies.
//
// Returns *Watchdog which is ready to be started.
// Returns error when the profile store cannot be initialised.
//
//nolint:gocritic // pass-by-value is intentional for constructor ergonomics
func NewWatchdog(config WatchdogConfig, systemCollector *SystemCollector, opts ...WatchdogOption) (*Watchdog, error) {
	if err := validateWatchdogConfig(&config); err != nil {
		return nil, err
	}

	w := &Watchdog{
		config:          config,
		systemCollector: systemCollector,
		lastCaptureTime: make(map[string]time.Time),
		lastWarningTime: make(map[string]time.Time),
		stopCh:          make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	if w.clock == nil {
		w.clock = clock.RealClock()
	}

	if config.ProfileDirectory == "" {
		config.ProfileDirectory = filepath.Join(os.TempDir(), "piko-watchdog")
		w.config.ProfileDirectory = config.ProfileDirectory
	}

	if w.profileStore == nil {
		store, err := newProfileStore(config.ProfileDirectory, config.MaxProfilesPerType, w.clock)
		if err != nil {
			return nil, fmt.Errorf("initialising watchdog profile store: %w", err)
		}
		w.profileStore = store
	}

	if w.hostname == "" {
		w.hostname, _ = os.Hostname()
	}

	return w, nil
}

// WithWatchdogClock sets the clock used by the watchdog for time operations.
// This is intended for testing with a mock clock.
//
// Takes clk (clock.Clock) which provides the time source.
//
// Returns WatchdogOption which configures the clock when applied.
func WithWatchdogClock(clk clock.Clock) WatchdogOption {
	return func(w *Watchdog) {
		w.clock = clk
	}
}

// WithWatchdogSandbox sets the sandbox used by the watchdog's profile store.
// This is intended for testing with a controlled filesystem.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for
// profile storage.
//
// Returns WatchdogOption which configures the profile store sandbox when
// applied.
func WithWatchdogSandbox(sandbox safedisk.Sandbox) WatchdogOption {
	return func(w *Watchdog) {
		w.profileStore = &profileStore{
			sandbox:            sandbox,
			maxProfilesPerType: w.config.MaxProfilesPerType,
		}
	}
}

// WithWatchdogNotifier sets the notification delivery mechanism for watchdog
// events. This is intended for integration with external alerting systems.
//
// Takes notifier (WatchdogNotifier) which delivers event notifications.
//
// Returns WatchdogOption which configures the notifier when applied.
func WithWatchdogNotifier(notifier WatchdogNotifier) WatchdogOption {
	return func(w *Watchdog) {
		w.notifier = notifier
	}
}

// WithWatchdogProfileUploader sets the remote storage backend for profile
// uploads. Profiles are uploaded after being written to local disk.
//
// Takes uploader (WatchdogProfileUploader) which handles remote storage.
//
// Returns WatchdogOption which configures the uploader when applied.
func WithWatchdogProfileUploader(uploader WatchdogProfileUploader) WatchdogOption {
	return func(w *Watchdog) {
		w.profileUploader = uploader
	}
}

// SetProfilingController sets the profiling controller used for on-demand
// profile captures. This must be called before Start for captures to succeed.
//
// Takes controller (ProfilingController) which manages pprof profile capture.
//
// Safe for concurrent use; protected by the watchdog's mutex.
func (w *Watchdog) SetProfilingController(controller ProfilingController) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.profilingController = controller
}

// Start begins periodic monitoring by spawning a background goroutine that
// evaluates system metrics on each tick. If the watchdog is disabled via
// config, the call is a no-op.
//
// The background goroutine runs until Stop is called or the context is
// cancelled.
func (w *Watchdog) Start(ctx context.Context) {
	if !w.config.Enabled {
		return
	}

	w.mu.Lock()
	if w.started || w.stopped {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.mu.Unlock()

	w.resolveHeapThreshold(ctx)
	w.checkMemProfileRate(ctx)
	w.checkProfilingController(ctx)
	w.startedAt = w.clock.Now()
	w.lastTrendEvaluation = w.startedAt

	w.cacheCgroupMemoryLimit(ctx)
	w.processStartupHistory(ctx)

	if w.config.TrendWindowSize > 0 {
		w.heapTrendBuffer = newHeapTrendBuffer(w.config.TrendWindowSize)
	}

	w.goroutineLeakAvailable = pprof.Lookup(profileTypeGoroutineLeak) != nil

	go w.loop(ctx)

	if w.config.ContinuousProfilingEnabled {
		go w.continuousProfilingLoop(ctx)
	}
}

// CapturePreDeathSnapshot captures a final diagnostic snapshot before the
// process terminates.
//
// It captures heap and goroutine profiles synchronously and sends a
// notification if configured. This is intended to be called from a shutdown
// hook when SIGTERM is received. The method respects the provided context's
// deadline to stay within the shutdown timeout budget.
//
// Safe for concurrent use; protected by the watchdog's mutex.
func (w *Watchdog) CapturePreDeathSnapshot(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	ctx, cancel := context.WithTimeoutCause(ctx, preDeathSnapshotTimeout,
		errors.New("pre-death snapshot exceeded 10s budget"))
	defer cancel()

	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	_, l := logger_domain.From(ctx, log)
	l.Notice("Capturing pre-death diagnostic snapshot")

	preDeathContext := captureContext{Rule: "pre_death"}
	w.captureAndStoreProfile(ctx, profileTypeHeap, preDeathContext)
	w.captureAndStoreProfile(ctx, profileTypeGoroutine, preDeathContext)
	w.captureFlightRecorderSnapshot(ctx)

	watchdogPreDeathSnapshotCount.Add(ctx, 1)

	w.sendNotification(ctx, WatchdogEvent{
		EventType: WatchdogEventPreDeathSnapshot,
		Priority:  WatchdogPriorityCritical,
		Message:   "Pre-death diagnostic snapshot captured during shutdown",
	})
}

// Stop signals the watchdog loop to exit and closes the profile store. Safe
// to call multiple times; only the first call closes the stop channel.
//
// Before closing the profile store, Stop marks the most recent
// startup-history entry as cleanly stopped so the next process start can
// distinguish clean exits from crashes.
//
// Safe for concurrent use; protected by the watchdog's mutex.
func (w *Watchdog) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}

	close(w.stopCh)
	w.stopped = true
	w.mu.Unlock()

	w.captureWG.Wait()

	w.backgroundWG.Wait()

	w.closeAllEventSubscribers(context.Background())

	if w.profileStore != nil {
		w.markStartupHistoryStopped(context.Background())
		_ = w.profileStore.close()
	}
}

// ListProfiles returns metadata for all stored profile files, sorted by
// timestamp descending (newest first).
//
// Returns []WatchdogProfileInfo which contains the profile metadata.
// Returns error when the profile directory cannot be read.
func (w *Watchdog) ListProfiles(_ context.Context) ([]WatchdogProfileInfo, error) {
	entries, err := w.profileStore.list()
	if err != nil {
		return nil, fmt.Errorf("listing watchdog profiles: %w", err)
	}

	result := make([]WatchdogProfileInfo, len(entries))
	for index, entry := range entries {
		result[index] = WatchdogProfileInfo(entry)
	}

	return result, nil
}

// DownloadSidecar reads the JSON sidecar paired with the named profile and
// returns its bytes.
//
// Takes profileFilename (string) which is the .pb.gz profile filename; the
// sidecar is resolved by swapping the extension.
//
// Returns []byte which is the sidecar JSON, or nil when no sidecar exists.
// Returns bool which is true when a sidecar was found and read.
// Returns error when the profile filename is empty or the read fails for
// reasons other than absence.
func (w *Watchdog) DownloadSidecar(ctx context.Context, profileFilename string) ([]byte, bool, error) {
	data, present, err := w.profileStore.readSidecar(profileFilename)
	if err != nil {
		watchdogSidecarDownloadErrorCount.Add(ctx, 1)
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			watchdogProfileFileOversizeCount.Add(ctx, 1)
		}
		return nil, false, fmt.Errorf("reading watchdog sidecar: %w", err)
	}
	if present {
		watchdogSidecarDownloadCount.Add(ctx, 1)
	}
	return data, present, nil
}

// GetStartupHistory returns the most recent startup-history entries in
// chronological order (oldest first). A missing or unreadable history file
// yields a nil slice (and counts the read error in OTel).
//
// Returns []WatchdogStartupHistoryEntry which is the parsed history.
// Returns error when an existing file cannot be parsed (corruption); a
// missing file is not an error.
func (w *Watchdog) GetStartupHistory(ctx context.Context) ([]WatchdogStartupHistoryEntry, error) {
	if w.profileStore == nil {
		return nil, nil
	}
	file, _, err := w.profileStore.readHistory()
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			watchdogProfileFileOversizeCount.Add(ctx, 1)
		}
		return nil, fmt.Errorf("reading watchdog startup history: %w", err)
	}
	if len(file.Entries) == 0 {
		return nil, nil
	}
	result := make([]WatchdogStartupHistoryEntry, len(file.Entries))
	for index, entry := range file.Entries {
		converted := WatchdogStartupHistoryEntry{
			StartedAt:       entry.StartedAt,
			Hostname:        entry.Hostname,
			Version:         entry.Version,
			Reason:          entry.Reason,
			GomemlimitBytes: entry.GomemlimitBytes,
			PID:             entry.PID,
		}
		if entry.StoppedAt != nil {
			converted.StoppedAt = *entry.StoppedAt
		}
		result[index] = converted
	}
	return result, nil
}

// DownloadProfile writes the raw compressed bytes of the named profile file to
// the provided writer. The caller is responsible for decompression.
//
// Takes filename (string) which identifies the profile file to download.
// Takes writer (io.Writer) which receives the compressed profile data.
//
// Returns error when the filename is empty or the file cannot be read.
func (w *Watchdog) DownloadProfile(ctx context.Context, filename string, writer io.Writer) error {
	data, err := w.profileStore.read(filename)
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			watchdogProfileFileOversizeCount.Add(ctx, 1)
		}
		return fmt.Errorf("downloading watchdog profile: %w", err)
	}

	if _, writeErr := writer.Write(data); writeErr != nil {
		return fmt.Errorf("writing profile data to output: %w", writeErr)
	}

	return nil
}

// PruneProfiles removes stored profile files. When profileType is empty, all
// profiles are removed; otherwise only profiles of the specified type are
// removed.
//
// Takes profileType (string) which filters deletion to a specific profile
// category. Pass empty string to delete all profiles.
//
// Returns int which is the number of files deleted.
// Returns error when listing or removing files fails.
func (w *Watchdog) PruneProfiles(_ context.Context, profileType string) (int, error) {
	if profileType == "" {
		return w.profileStore.deleteAll()
	}

	return w.profileStore.deleteByType(profileType)
}

// GetWatchdogStatus returns the current watchdog state including configuration,
// thresholds, and runtime counters.
//
// Returns *WatchdogStatusInfo which contains the current watchdog state.
//
// Safe for concurrent use; acquires the watchdog's mutex to read mutable
// fields.
func (w *Watchdog) GetWatchdogStatus(_ context.Context) *WatchdogStatusInfo {
	w.mu.Lock()
	stopped := w.stopped
	heapHighWater := w.heapHighWater
	captureWindowUsed := len(w.captureTimestamps)
	warningWindowUsed := len(w.warningTimestamps)
	contentionLastRun := w.lastContentionDiagnosticAt
	w.mu.Unlock()

	continuousTypes := append([]string(nil), w.config.ContinuousProfilingTypes...)

	return &WatchdogStatusInfo{
		StartedAt:                    w.startedAt,
		ContentionDiagnosticLastRun:  contentionLastRun,
		ProfileDirectory:             w.config.ProfileDirectory,
		ContinuousProfilingTypes:     continuousTypes,
		CheckInterval:                w.config.CheckInterval,
		Cooldown:                     w.config.Cooldown,
		WarmUpDuration:               w.config.WarmUpDuration,
		CaptureWindow:                w.config.CaptureWindow,
		SchedulerLatencyP99Threshold: w.config.SchedulerLatencyP99Threshold,
		CrashLoopWindow:              w.config.CrashLoopWindow,
		ContinuousProfilingInterval:  w.config.ContinuousProfilingInterval,
		ContentionDiagnosticWindow:   w.config.ContentionDiagnosticWindowDuration,
		ContentionDiagnosticCooldown: w.config.ContentionDiagnosticCooldown,
		HeapThresholdBytes:           w.initialHeapThreshold,
		HeapHighWater:                heapHighWater,
		FDPressureThresholdPercent:   w.config.FDPressureThresholdPercent,
		GoroutineThreshold:           w.config.GoroutineThreshold,
		GoroutineSafetyCeiling:       w.config.GoroutineSafetyCeiling,
		MaxProfilesPerType:           w.config.MaxProfilesPerType,
		MaxCapturesPerWindow:         w.config.MaxCapturesPerWindow,
		MaxWarningsPerWindow:         w.config.MaxWarningsPerWindow,
		CrashLoopThreshold:           w.config.CrashLoopThreshold,
		ContinuousProfilingRetention: w.config.ContinuousProfilingRetention,
		CaptureWindowUsed:            captureWindowUsed,
		WarningWindowUsed:            warningWindowUsed,
		GoroutineBaseline:            w.goroutineBaseline.Load(),
		Enabled:                      w.config.Enabled,
		Stopped:                      stopped,
		ContinuousProfilingEnabled:   w.config.ContinuousProfilingEnabled,
		ContentionDiagnosticAutoFire: w.config.ContentionDiagnosticAutoFire,
	}
}

// resolveHeapThreshold reads the current GOMEMLIMIT from the runtime and
// computes the initial heap threshold. Called at Start time rather than
// construction time because automemlimit sets the limit during bootstrap
// after the watchdog is constructed.
func (w *Watchdog) resolveHeapThreshold(ctx context.Context) {
	w.gomemlimit = debug.SetMemoryLimit(-1)

	if w.gomemlimit > 0 && w.gomemlimit < math.MaxInt64 {
		w.initialHeapThreshold = uint64(float64(w.gomemlimit) * w.config.HeapThresholdPercent)
	} else {
		w.initialHeapThreshold = w.config.HeapThresholdBytes

		_, l := logger_domain.From(ctx, log)
		l.Warn("GOMEMLIMIT is not configured; the watchdog will use the absolute heap "+
			"threshold. In containerised environments, use piko.WithAutoMemoryLimit for "+
			"accurate OOM-aware monitoring",
			logger_domain.Uint64("heap_threshold_bytes", w.initialHeapThreshold),
		)

		w.sendNotification(ctx, NewGomemlimitNotConfiguredEvent())
	}

	w.heapHighWater = w.initialHeapThreshold
}

// checkProfilingController warns at startup when the watchdog has no
// profiling controller wired. Without one, every capture path (continuous
// profiling, threshold-triggered captures, pre-death snapshot, contention
// diagnostic) silently no-ops, leaving operators with no profile artefacts
// when problems occur.
//
// The fix is to add piko.WithMonitoringProfiling() to the WithMonitoring
// option set, which constructs and wires the controller.
//
// Concurrency: acquires w.mu briefly to read the profilingController pointer.
func (w *Watchdog) checkProfilingController(ctx context.Context) {
	w.mu.Lock()
	hasController := w.profilingController != nil
	w.mu.Unlock()

	if hasController {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Watchdog has no profiling controller; all profile captures " +
		"(continuous, threshold-triggered, pre-death, contention) will be " +
		"silently dropped. Add piko.WithMonitoringProfiling() to your " +
		"WithMonitoring options to enable captures.")
}

// checkMemProfileRate disarms heap and allocs captures when the runtime is
// not sampling, so threshold triggers do not write empty pprofs and burn the
// cooldown budget.
func (w *Watchdog) checkMemProfileRate(ctx context.Context) {
	if runtime.MemProfileRate != 0 {
		return
	}

	w.heapProfilingDisabled = true

	_, l := logger_domain.From(ctx, log)
	l.Warn("runtime.MemProfileRate is 0; heap and allocs captures are disarmed. " +
		"Set piko.WithProfilingMemProfileRate (default 524288) to re-enable " +
		"heap-based watchdog rules")

	w.sendNotification(ctx, NewMemProfileRateDisabledEvent())
}

// cacheCgroupMemoryLimit caches the cgroup memory limit at startup.
//
// The system-collector value is stable for the lifetime of the process, so
// reading it on every tick would be wasted work; the corresponding OTel
// gauge is recorded once here rather than per tick. The same call also
// records the soft FD limit gauge once because RLIMIT_NOFILE is also
// stable for the process.
func (w *Watchdog) cacheCgroupMemoryLimit(ctx context.Context) {
	stats := w.systemCollector.GetStats()
	w.cgroupMemoryLimit = stats.Process.CgroupMemoryLimit

	if w.cgroupMemoryLimit > 0 {
		watchdogCgroupMemoryLimitBytes.Record(ctx, safeconv.Uint64ToInt64(w.cgroupMemoryLimit))
	}

	if stats.Process.MaxOpenFilesSoft > 0 {
		watchdogFDLimitSoft.Record(ctx, stats.Process.MaxOpenFilesSoft)
	}
}

// processStartupHistory drives the startup-history flow at watchdog Start.
//
// It inspects the on-disk ring, classifies the previous run (clean or
// unclean), detects crash loops, then appends a new entry for the current
// process. The whole flow is best-effort: failures are logged plus
// counted but never block startup, since the watchdog must come up even
// when the profile directory is unwritable. Set CrashLoopWindow to zero
// to disable history processing entirely.
func (w *Watchdog) processStartupHistory(ctx context.Context) {
	if w.config.CrashLoopWindow == 0 {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	file, existed, err := w.profileStore.readHistory()
	if err != nil && existed {
		watchdogStartupHistoryReadErrorCount.Add(ctx, 1)
		l.Warn("Failed to read startup history; continuing with empty history",
			logger_domain.Error(err),
		)
	}

	w.classifyPreviousEntry(ctx, &file)
	w.detectCrashLoop(ctx, file)

	now := w.clock.Now()
	entry := startupHistoryEntry{
		StartedAt:       now,
		PID:             os.Getpid(),
		Hostname:        w.hostname,
		Version:         buildVersionString(),
		GomemlimitBytes: w.gomemlimit,
	}
	file.Entries = append(file.Entries, entry)
	if len(file.Entries) > maxStartupHistoryEntries {
		file.Entries = file.Entries[len(file.Entries)-maxStartupHistoryEntries:]
	}

	if writeErr := w.profileStore.writeHistory(file); writeErr != nil {
		watchdogStartupHistoryWriteErrorCount.Add(ctx, 1)
		l.Warn("Failed to write startup history", logger_domain.Error(writeErr))
	}
}

// classifyPreviousEntry checks whether the most recent history entry exited
// uncleanly (no StoppedAt). When it did, the entry is patched with
// Reason="unclean" and a PreviousCrashClassified event is emitted so the
// operator can investigate.
//
// Takes file (*startupHistoryFile) which is mutated in place when the last
// entry needs reclassification.
func (w *Watchdog) classifyPreviousEntry(ctx context.Context, file *startupHistoryFile) {
	if len(file.Entries) == 0 {
		return
	}
	last := &file.Entries[len(file.Entries)-1]
	if last.StoppedAt != nil {
		return
	}

	last.Reason = "unclean"
	watchdogUncleanShutdownCount.Add(ctx, 1)

	_, l := logger_domain.From(ctx, log)
	l.Notice("Previous run did not exit cleanly; classifying as unclean shutdown",
		logger_domain.Int("prev_pid", last.PID),
	)

	w.sendNotification(ctx, NewPreviousCrashClassifiedEvent(*last))
}

// detectCrashLoop counts unclean entries within the CrashLoopWindow.
//
// Entries whose StoppedAt is nil count as unclean. When the count reaches
// CrashLoopThreshold a CrashLoopDetected event is emitted.
//
// Takes file (startupHistoryFile) which is the history snapshot to scan.
func (w *Watchdog) detectCrashLoop(ctx context.Context, file startupHistoryFile) {
	if w.config.CrashLoopThreshold <= 0 {
		return
	}

	cutoff := w.clock.Now().Add(-w.config.CrashLoopWindow)
	unclean := 0
	for _, entry := range file.Entries {
		if !entry.StartedAt.After(cutoff) {
			continue
		}
		if entry.StoppedAt != nil {
			continue
		}
		unclean++
	}

	if unclean < w.config.CrashLoopThreshold {
		return
	}

	watchdogCrashLoopDetectionCount.Add(ctx, 1)

	_, l := logger_domain.From(ctx, log)
	l.Error("Crash loop detected from startup history",
		logger_domain.Int("unclean_in_window", unclean),
		logger_domain.Int("window_seconds", int(w.config.CrashLoopWindow.Seconds())),
	)

	w.sendNotification(ctx, NewCrashLoopDetectedEvent(unclean, int(w.config.CrashLoopWindow.Seconds())))
}

// markStartupHistoryStopped patches the most recent entry's StoppedAt
// field so the next start can classify the previous run as clean.
//
// Called from Stop. Failure is logged but never propagated.
func (w *Watchdog) markStartupHistoryStopped(ctx context.Context) {
	if w.config.CrashLoopWindow == 0 {
		return
	}

	file, _, err := w.profileStore.readHistory()
	if err != nil {
		watchdogStartupHistoryReadErrorCount.Add(ctx, 1)
		return
	}
	if len(file.Entries) == 0 {
		return
	}

	file.Entries[len(file.Entries)-1].StoppedAt = new(w.clock.Now())
	if file.Entries[len(file.Entries)-1].Reason == "" {
		file.Entries[len(file.Entries)-1].Reason = "clean"
	}

	if writeErr := w.profileStore.writeHistory(file); writeErr != nil {
		watchdogStartupHistoryWriteErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to mark startup history as cleanly stopped", logger_domain.Error(writeErr))
	}
}

// loop runs the periodic evaluation loop until stopped or the context is
// cancelled. Each tick reads system statistics and evaluates them against
// the configured thresholds, then records the heartbeat.
//
// Recovered panics in the evaluation path are reported via OTel and the
// notifier; the loop deliberately does not auto-restart, in line with the
// "let it fail visibly" philosophy. A stale heartbeat plus an incremented
// panic counter is the externally visible signal that the watchdog
// stopped.
func (w *Watchdog) loop(ctx context.Context) {
	ticker := w.clock.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()
	defer func() {
		if r := recover(); r != nil {
			w.handleLoopPanic(ctx, r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C():
			w.evaluate(ctx, new(w.systemCollector.GetStats()))
			watchdogLoopIterationsCount.Add(ctx, 1)
			watchdogLoopLastTickEpochSeconds.Record(ctx, w.clock.Now().Unix())
		}
	}
}

// handleLoopPanic records a panic that escaped the evaluation loop.
//
// Increments the loop-panic counter, logs the failure with stack trace,
// and emits a critical event so external monitoring can see that runtime
// monitoring is no longer active. It deliberately does not re-panic, but
// does not restart the loop either; by design the watchdog stays dead so
// an absent heartbeat signals the failure.
//
// Takes recovered (any) which is the value returned by recover() in the
// loop's deferred recovery handler.
func (w *Watchdog) handleLoopPanic(ctx context.Context, recovered any) {
	watchdogLoopPanicCount.Add(ctx, 1)

	_, l := logger_domain.From(ctx, log)
	l.Error("Watchdog evaluation loop panicked",
		String("panic", fmt.Sprintf("%v", recovered)),
		String("stack", string(debug.Stack())),
	)

	w.sendNotification(ctx, NewLoopPanickedEvent(fmt.Sprintf("%v", recovered)))
}

// evaluate runs all rule evaluators against the provided system statistics.
// During the warm-up period, evaluations are suppressed to allow the runtime
// to stabilise after startup.
//
// Takes stats (*SystemStats) which contains the current system metrics.
func (w *Watchdog) evaluate(ctx context.Context, stats *SystemStats) {
	now := w.clock.Now()

	if now.Sub(w.startedAt) < w.config.WarmUpDuration {
		return
	}

	if w.goroutineBaseline.Load() == 0 {
		w.goroutineBaseline.CompareAndSwap(0, stats.NumGoroutines)
	}

	w.evaluateHeap(ctx, now, stats)
	w.evaluateGoroutines(ctx, now, stats)
	w.evaluateGCPressure(ctx, now, stats)
	w.evaluateRSS(ctx, now, stats)
	w.evaluateHeapTrend(ctx, now, stats)
	w.evaluateGoroutineLeaks(ctx, now)
	w.evaluateFDPressure(ctx, now, stats)
	w.evaluateSchedulerLatency(ctx, now, stats)
}

// triggerCapture spawns a background goroutine that captures a profile of the
// given type and stores it to disk along with its sidecar metadata. Errors
// are logged and counted but do not propagate to the caller.
//
// Takes profileType (string) which identifies the profile to capture (e.g.
// "heap", "goroutine").
// Takes capCtx (captureContext) which describes the firing rule and observed
// values for the sidecar metadata.
func (w *Watchdog) triggerCapture(ctx context.Context, profileType string, capCtx captureContext) {
	w.goSafely(&w.captureWG, func() {
		defer goroutine.RecoverPanic(ctx, "monitoring.watchdogCapture."+profileType)
		w.captureAndStoreProfile(ctx, profileType, capCtx)
	})
}

// goSafely starts fn on wg unless Stop has already run.
//
// Stop acquires w.mu before calling captureWG.Wait or backgroundWG.Wait,
// so the stopped check inside the same lock is the only way to avoid the
// "WaitGroup.Go after Wait returned" panic that would otherwise occur if
// a rule fires concurrently with shutdown.
//
// Takes wg (*sync.WaitGroup) which is the wait group to dispatch onto.
// Takes fn (func()) which is the work to run on a fresh goroutine.
//
// Safe for concurrent use; serialises with Stop via the watchdog mutex.
func (w *Watchdog) goSafely(wg *sync.WaitGroup, fn func()) {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	wg.Go(fn)
	w.mu.Unlock()
}

// captureAndStoreProfile drives a single pprof capture end to end.
//
// Captures a profile via the profiling controller and writes the
// compressed data to the profile store. After a successful write it
// persists a JSON sidecar containing rule attribution and a snapshot of
// system stats so a future operator can reconstruct what triggered the
// capture without opening the profile in pprof first.
//
// Takes profileType (string) which identifies the profile to capture.
// Takes capCtx (captureContext) which describes the firing rule and
// observed values for the sidecar metadata.
func (w *Watchdog) captureAndStoreProfile(ctx context.Context, profileType string, capCtx captureContext) {
	if ctx.Err() != nil {
		return
	}
	controller, ok := w.activeProfilingController(ctx, profileType)
	if !ok {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	profileData, ok := w.collectProfileBytes(ctx, controller, profileType)
	if !ok {
		return
	}

	timestamp, err := w.profileStore.write(profileType, profileData)
	if err != nil {
		l.Error("Failed to store profile",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		w.sendNotification(ctx, NewCaptureErrorEvent(profileType, err))
		return
	}

	w.writeSidecarMetadata(ctx, profileType, timestamp, capCtx)
	w.maybeWriteGoroutineStacks(ctx, profileType, timestamp)
	w.processStoredProfile(ctx, profileType, profileData)

	l.Notice("Watchdog captured and stored diagnostic profile",
		String(logFieldProfileType, profileType),
		String("rule", capCtx.Rule),
	)
}

// activeProfilingController returns the configured profiling controller
// when the watchdog is still running.
//
// Takes profileType (string) which identifies the profile capture being
// attempted (used as a log attribute when the controller is missing).
//
// Returns ProfilingController which is the wired controller on success.
// Returns bool which is false when the watchdog has stopped or no
// controller is wired in.
//
// Safe for concurrent use; acquires the watchdog mutex briefly.
func (w *Watchdog) activeProfilingController(ctx context.Context, profileType string) (ProfilingController, bool) {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return nil, false
	}
	controller := w.profilingController
	w.mu.Unlock()

	if controller == nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Profiling controller not available, skipping capture",
			String(logFieldProfileType, profileType),
		)
		return nil, false
	}
	return controller, true
}

// collectProfileBytes drives the controller capture and bounds the
// resulting payload by MaxProfileSizeBytes so an oversized profile never
// causes allocator pressure.
//
// Takes controller (ProfilingController) which performs the capture.
// Takes profileType (string) which identifies the profile to capture and
// is used as a log attribute on failure paths.
//
// Returns []byte which is the compressed profile on success, nil on
// failure or when discarded for size.
// Returns bool which is true on success and false when the capture
// failed or the result was discarded.
func (w *Watchdog) collectProfileBytes(ctx context.Context, controller ProfilingController, profileType string) ([]byte, bool) {
	_, l := logger_domain.From(ctx, log)

	var buffer bytes.Buffer
	if _, err := controller.CaptureProfile(ctx, profileType, 0, &buffer); err != nil {
		l.Error("Failed to capture profile",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		w.sendNotification(ctx, NewCaptureErrorEvent(profileType, err))
		return nil, false
	}

	profileData := buffer.Bytes()
	if int64(len(profileData)) > w.config.MaxProfileSizeBytes {
		l.Warn("Profile exceeds maximum size, discarding to avoid memory pressure",
			String(logFieldProfileType, profileType),
			logger_domain.Int64("profile_size_bytes", int64(len(profileData))),
			logger_domain.Int64("max_profile_size_bytes", w.config.MaxProfileSizeBytes),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		return nil, false
	}
	return profileData, true
}

// writeSidecarMetadata builds and writes the JSON sidecar that pairs with a
// profile capture. Failure is logged but never aborts the capture flow -- the
// profile is the primary artefact, the sidecar is supplementary.
//
// Takes profileType (string) which identifies the profile category.
// Takes timestamp (string) which is the timestamp portion of the profile
// filename.
// Takes capCtx (captureContext) which describes the firing rule.
func (w *Watchdog) writeSidecarMetadata(ctx context.Context, profileType, timestamp string, capCtx captureContext) {
	stats := w.systemCollector.GetStats()
	snap := w.systemCollector.lastRuntimeMetricsSnapshot()

	meta := captureMetadata{
		CapturedAt:               w.clock.Now(),
		RuleFired:                capCtx.Rule,
		ProfileType:              profileType,
		ObservedValue:            capCtx.Observed,
		Threshold:                capCtx.Threshold,
		Hostname:                 w.hostname,
		Version:                  buildVersionString(),
		PID:                      os.Getpid(),
		GomemlimitBytes:          w.gomemlimit,
		NumGoroutines:            stats.NumGoroutines,
		GCCPUFraction:            stats.GC.GCCPUFraction,
		RSSBytes:                 stats.Process.RSS,
		CgroupLimitBytes:         w.cgroupMemoryLimit,
		HeapAllocBytes:           stats.Memory.HeapAlloc,
		FDCount:                  stats.Process.FDCount,
		FDLimitSoft:              stats.Process.MaxOpenFilesSoft,
		SchedulerLatencyP99Nanos: snap.SchedulerLatencyP99.Nanoseconds(),
		GCPauseP99Nanos:          snap.GCPauseP99.Nanoseconds(),
		MutexWaitTotalSeconds:    snap.MutexWaitTotalSeconds,
		RuntimeMetricsSnapshot:   w.systemCollector.runtimeMetrics.lastSnapshotMap(snap),
	}

	if err := w.profileStore.writeMetadata(profileType, timestamp, meta); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to write capture sidecar metadata",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
	}
}

// maybeWriteGoroutineStacks writes a debug=2 per-goroutine stacks sidecar
// alongside a goroutine profile capture when WatchdogConfig.IncludeGoroutineStacks
// is enabled. The .stacks.txt file pairs by base name with the .pb.gz binary
// profile so consumers can locate either by stripping the matching extension.
//
// Failure is logged but never aborts the capture flow -- the binary profile
// is the primary artefact, the stacks file is supplementary.
//
// Takes profileType (string) which identifies the profile category. Only
// "goroutine" triggers a stacks write today; other types are no-ops so this
// helper can sit unconditionally on the capture path.
// Takes timestamp (string) which is the timestamp portion of the profile
// filename, used to keep the stacks sidecar paired with its binary.
func (w *Watchdog) maybeWriteGoroutineStacks(ctx context.Context, profileType, timestamp string) {
	if !w.config.IncludeGoroutineStacks {
		return
	}
	if profileType != profileTypeGoroutine {
		return
	}

	profile := pprof.Lookup(profileTypeGoroutine)
	if profile == nil {
		return
	}

	var buf bytes.Buffer
	if err := profile.WriteTo(&buf, 2); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to capture goroutine stacks sidecar",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		return
	}

	if err := w.profileStore.writeStacks(profileType, timestamp, buf.Bytes()); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to write goroutine stacks sidecar",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
	}
}

// processStoredProfile handles post-storage actions for a captured profile
// including upload, delta baseline storage, and flight recorder snapshots.
//
// Takes profileType (string) which identifies the profile that was captured.
// Takes profileData ([]byte) which is the raw profile bytes to upload and
// use for delta baselines.
func (w *Watchdog) processStoredProfile(ctx context.Context, profileType string, profileData []byte) {
	w.uploadProfile(ctx, profileType, profileData)

	if profileType == profileTypeHeap && w.config.DeltaProfilingEnabled {
		w.storeHeapBaseline(ctx, profileData)
	}

	if profileType == profileTypeHeap || profileType == profileTypeGoroutine {
		w.captureFlightRecorderSnapshot(ctx)
	}
}

// captureFlightRecorderSnapshot writes the current rolling execution trace
// buffer to the profile store.
//
// This provides scheduling and GC event context alongside pprof profiles.
// Silently returns when the flight recorder is not enabled.
//
// Safe for concurrent use; acquires the watchdog's mutex to read the
// profiling controller.
func (w *Watchdog) captureFlightRecorderSnapshot(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	w.mu.Lock()
	controller := w.profilingController
	w.mu.Unlock()

	if controller == nil {
		return
	}

	var buffer bytes.Buffer
	if err := controller.SnapshotFlightRecorder(ctx, &buffer); err != nil {
		return
	}

	traceData := buffer.Bytes()
	if len(traceData) == 0 {
		return
	}

	if _, err := w.profileStore.write(profileTypeTrace, traceData); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to store flight recorder snapshot", logger_domain.Error(err))
		return
	}

	w.uploadProfile(ctx, profileTypeTrace, traceData)
}

// storeHeapBaseline writes the previous heap profile as a baseline file so
// the user can compute a diff between consecutive captures using
// `go tool pprof -diff_base heap-baseline-*.pb.gz heap-*.pb.gz`.
//
// Takes currentProfileData ([]byte) which is the current heap profile to
// store as the next baseline.
//
// Safe for concurrent use; acquires the watchdog's mutex to swap the
// previous heap profile.
func (w *Watchdog) storeHeapBaseline(ctx context.Context, currentProfileData []byte) {
	w.mu.Lock()
	previousProfile := w.previousHeapProfile
	w.previousHeapProfile = make([]byte, len(currentProfileData))
	copy(w.previousHeapProfile, currentProfileData)
	w.mu.Unlock()

	if previousProfile == nil {
		return
	}

	if _, err := w.profileStore.write(profileTypeHeapBaseline, previousProfile); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to store heap baseline profile", logger_domain.Error(err))
	}
}

// sendNotification delivers a watchdog event via the configured notifier.
//
// If no notifier is set, this is a no-op. The notification is sent in a
// background goroutine tracked by backgroundWG so Stop can wait for it to
// complete.
//
// Takes event (WatchdogEvent) which describes the runtime event to deliver.
func (w *Watchdog) sendNotification(ctx context.Context, event WatchdogEvent) {
	w.recordEvent(ctx, WatchdogEventInfo{
		EmittedAt: w.clock.Now(),
		EventType: event.EventType,
		Priority:  event.Priority,
		Message:   event.Message,
		Fields:    event.Fields,
	})

	if w.notifier == nil {
		return
	}

	detachedCtx := context.WithoutCancel(ctx)
	detachedCtx, notificationCancel := context.WithTimeoutCause(detachedCtx, notificationTimeout,
		errors.New("watchdog notification exceeded 30s budget"))

	w.goSafely(&w.backgroundWG, func() {
		defer notificationCancel()
		defer goroutine.RecoverPanic(detachedCtx, "monitoring.watchdogNotification")

		if err := w.notifier.Notify(detachedCtx, event); err != nil {
			_, l := logger_domain.From(detachedCtx, log)
			l.Warn("Failed to send watchdog notification",
				String("event_type", string(event.EventType)),
				logger_domain.Error(err),
			)
			watchdogNotificationErrorCount.Add(detachedCtx, 1)
			return
		}

		watchdogNotificationSentCount.Add(detachedCtx, 1)
	})
}

// uploadProfile sends the captured profile data to the configured remote
// storage backend.
//
// If no uploader is configured, this is a no-op. The upload is best-effort
// and tracked by backgroundWG so Stop can wait for completion.
//
// Takes profileType (string) which identifies the profile category.
// Takes data ([]byte) which is the raw profile bytes to upload.
func (w *Watchdog) uploadProfile(ctx context.Context, profileType string, data []byte) {
	if w.profileUploader == nil {
		return
	}

	detachedCtx := context.WithoutCancel(ctx)
	detachedCtx, uploadCancel := context.WithTimeoutCause(detachedCtx, uploadTimeout,
		errors.New("watchdog profile upload exceeded 30s budget"))

	w.goSafely(&w.backgroundWG, func() {
		defer uploadCancel()
		defer goroutine.RecoverPanic(detachedCtx, "monitoring.watchdogUpload."+profileType)

		metadata := map[string]string{
			"hostname":     w.hostname,
			"profile_type": profileType,
			"gomemlimit":   fmt.Sprintf("%d", w.gomemlimit),
			"timestamp":    w.clock.Now().Format("2006-01-02T15:04:05Z"),
		}

		if err := w.profileUploader.Upload(detachedCtx, profileType, data, metadata); err != nil {
			_, l := logger_domain.From(detachedCtx, log)
			l.Warn("Failed to upload watchdog profile",
				String(logFieldProfileType, profileType),
				logger_domain.Error(err),
			)
			watchdogProfileUploadErrorCount.Add(detachedCtx, 1)
			return
		}

		watchdogProfileUploadCount.Add(detachedCtx, 1)
	})
}

// buildVersionString returns the application version derived from the build
// info, falling back to the package-level Version constant when unavailable.
//
// Returns string which is the resolved version identifier.
func buildVersionString() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return Version
}
