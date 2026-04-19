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
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
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
)

var _ WatchdogInspector = (*Watchdog)(nil)

// WatchdogConfig holds the configuration for the runtime watchdog service.
type WatchdogConfig struct {
	// ProfileDirectory is the filesystem path where captured profile files are
	// stored. Must be set before the watchdog is started.
	ProfileDirectory string

	// CheckInterval is the period between watchdog evaluation ticks. Shorter
	// intervals detect anomalies faster at negligible CPU cost.
	CheckInterval time.Duration

	// Cooldown is the minimum duration between consecutive captures of the same
	// profile type.
	Cooldown time.Duration

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

	// MaxProfilesPerType is the maximum number of profile files retained per
	// profile type before the oldest are deleted.
	MaxProfilesPerType int

	// MaxProfileSizeBytes is the maximum number of bytes a single profile
	// capture may produce.
	//
	// Captures exceeding this limit are discarded to prevent the watchdog
	// from worsening memory pressure. Default: 50 MiB.
	MaxProfileSizeBytes int64

	// DeltaProfilingEnabled enables storing a baseline heap profile alongside
	// each capture so the user can compute a diff. Default: false.
	DeltaProfilingEnabled bool

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
	// startedAt records when the watchdog was started, used to enforce the
	// warm-up period. Written once in Start before the loop goroutine begins.
	startedAt time.Time

	// heapHighWaterSetAt is the timestamp when heapHighWater was last updated,
	// used to determine when the high-water mark may be reset.
	heapHighWaterSetAt time.Time

	// lastTrendEvaluation is the timestamp of the most recent heap trend
	// regression computation. Confined to the loop goroutine.
	lastTrendEvaluation time.Time

	// lastGoroutineLeakCheck is the timestamp of the most recent goroutine
	// leak profile evaluation. Confined to the loop goroutine.
	lastGoroutineLeakCheck time.Time

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

	// systemCollector provides the current system statistics for evaluation.
	systemCollector *SystemCollector

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

	// hostname is the machine hostname, cached at construction time for use
	// in notifications and upload metadata.
	hostname string

	// previousHeapProfile stores the compressed bytes of the last captured
	// heap profile, used to provide a diff baseline for the next capture.
	//
	// Nil until the first heap capture. Protected by mu.
	previousHeapProfile []byte

	// captureTimestamps is a sliding window of all capture timestamps, used
	// for global rate limiting.
	captureTimestamps []time.Time

	// config holds the watchdog configuration including thresholds and timing.
	config WatchdogConfig

	// captureWG tracks in-flight profile capture goroutines so that Stop can
	// wait for them to finish before closing the profile store.
	captureWG sync.WaitGroup

	// backgroundWG tracks in-flight notification and upload goroutines so
	// that Stop can wait for all background work to complete.
	backgroundWG sync.WaitGroup

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

	// mu guards mutable state including heapHighWater, lastCaptureTime,
	// captureTimestamps, profilingController, previousHeapProfile, and
	// stopped.
	mu sync.Mutex

	// goroutineBaseline is the goroutine count at startup, used to avoid
	// spurious captures during normal operation. Accessed atomically.
	goroutineBaseline atomic.Int32

	// stopped indicates whether Stop has been called.
	stopped bool

	// goroutineLeakAvailable indicates whether the Go 1.26 goroutine leak
	// profile experiment is enabled. Set once in Start before the loop
	// goroutine begins.
	goroutineLeakAvailable bool
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
		CheckInterval:              defaultCheckInterval,
		HeapThresholdPercent:       defaultHeapThresholdPercent,
		HeapThresholdBytes:         defaultHeapThresholdBytes,
		GoroutineThreshold:         defaultGoroutineThreshold,
		GoroutineSafetyCeiling:     defaultGoroutineSafetyCeiling,
		GCPressureThreshold:        defaultGCPressureThreshold,
		RSSThresholdPercent:        defaultRSSThresholdPercent,
		TrendWindowSize:            defaultTrendWindowSize,
		TrendEvaluationInterval:    defaultTrendEvaluationInterval,
		TrendWarningHorizon:        defaultTrendWarningHorizon,
		GoroutineLeakCheckInterval: defaultGoroutineLeakCheckInterval,
		Cooldown:                   defaultCooldown,
		MaxCapturesPerWindow:       defaultMaxCapturesPerWindow,
		CaptureWindow:              defaultCaptureWindow,
		HighWaterResetCooldown:     defaultHighWaterResetCooldown,
		WarmUpDuration:             defaultWarmUpDuration,
		MaxProfilesPerType:         defaultMaxProfilesPerType,
		MaxProfileSizeBytes:        defaultMaxProfileSizeBytes,
		Enabled:                    true,
	}
}

// validateWatchdogConfig checks the configuration for invalid or dangerous
// values and returns an error describing the first problem found.
//
// Takes config (WatchdogConfig) which provides the configuration to validate.
//
// Returns error when the configuration contains invalid or dangerous values.
func validateWatchdogConfig(config WatchdogConfig) error {
	if err := validateWatchdogThresholds(config); err != nil {
		return err
	}

	return validateWatchdogTimings(config)
}

// validateWatchdogThresholds validates threshold and limit fields in the
// watchdog configuration.
//
// Takes config (WatchdogConfig) which provides the configuration to validate.
//
// Returns error when a threshold or limit field contains an invalid value.
func validateWatchdogThresholds(config WatchdogConfig) error {
	if config.HeapThresholdPercent <= 0 || config.HeapThresholdPercent > 1.0 {
		return fmt.Errorf("%w: HeapThresholdPercent must be in (0.0, 1.0], got %v", ErrInvalidWatchdogConfig, config.HeapThresholdPercent)
	}

	if config.RSSThresholdPercent < 0 || config.RSSThresholdPercent > 1.0 {
		return fmt.Errorf("%w: RSSThresholdPercent must be in [0.0, 1.0], got %v", ErrInvalidWatchdogConfig, config.RSSThresholdPercent)
	}

	if config.MaxProfilesPerType < 1 {
		return fmt.Errorf("%w: MaxProfilesPerType must be at least 1, got %d", ErrInvalidWatchdogConfig, config.MaxProfilesPerType)
	}

	if config.MaxCapturesPerWindow < 1 {
		return fmt.Errorf("%w: MaxCapturesPerWindow must be at least 1, got %d", ErrInvalidWatchdogConfig, config.MaxCapturesPerWindow)
	}

	if config.MaxProfileSizeBytes <= 0 {
		return fmt.Errorf("%w: MaxProfileSizeBytes must be positive, got %d", ErrInvalidWatchdogConfig, config.MaxProfileSizeBytes)
	}

	if config.GoroutineThreshold < 1 {
		return fmt.Errorf("%w: GoroutineThreshold must be at least 1, got %d", ErrInvalidWatchdogConfig, config.GoroutineThreshold)
	}

	if config.GoroutineSafetyCeiling <= config.GoroutineThreshold {
		return fmt.Errorf("%w: GoroutineSafetyCeiling (%d) must be greater than GoroutineThreshold (%d)",
			ErrInvalidWatchdogConfig, config.GoroutineSafetyCeiling, config.GoroutineThreshold)
	}

	if config.TrendWindowSize < 0 {
		return fmt.Errorf("%w: TrendWindowSize must be non-negative, got %d", ErrInvalidWatchdogConfig, config.TrendWindowSize)
	}

	return nil
}

// validateWatchdogTimings validates interval and duration fields in the
// watchdog configuration.
//
// Takes config (WatchdogConfig) which provides the configuration to validate.
//
// Returns error when an interval or duration field contains an invalid value.
func validateWatchdogTimings(config WatchdogConfig) error {
	if config.CheckInterval <= 0 {
		return fmt.Errorf("%w: CheckInterval must be positive, got %v", ErrInvalidWatchdogConfig, config.CheckInterval)
	}

	if config.Cooldown <= 0 {
		return fmt.Errorf("%w: Cooldown must be positive, got %v", ErrInvalidWatchdogConfig, config.Cooldown)
	}

	if config.CaptureWindow <= 0 {
		return fmt.Errorf("%w: CaptureWindow must be positive, got %v", ErrInvalidWatchdogConfig, config.CaptureWindow)
	}

	if config.WarmUpDuration < 0 {
		return fmt.Errorf("%w: WarmUpDuration must be non-negative, got %v", ErrInvalidWatchdogConfig, config.WarmUpDuration)
	}

	if config.HighWaterResetCooldown <= 0 {
		return fmt.Errorf("%w: HighWaterResetCooldown must be positive, got %v", ErrInvalidWatchdogConfig, config.HighWaterResetCooldown)
	}

	if config.GoroutineLeakCheckInterval <= 0 {
		return fmt.Errorf("%w: GoroutineLeakCheckInterval must be positive, got %v", ErrInvalidWatchdogConfig, config.GoroutineLeakCheckInterval)
	}

	if config.TrendWindowSize > 0 {
		if config.TrendEvaluationInterval < 0 {
			return fmt.Errorf("%w: TrendEvaluationInterval must be non-negative when trend detection is enabled, got %v", ErrInvalidWatchdogConfig, config.TrendEvaluationInterval)
		}

		if config.TrendWarningHorizon <= 0 {
			return fmt.Errorf("%w: TrendWarningHorizon must be positive when trend detection is enabled, got %v", ErrInvalidWatchdogConfig, config.TrendWarningHorizon)
		}
	}

	return nil
}

// NewWatchdog creates a new runtime watchdog that monitors system metrics and
// captures diagnostic profiles when thresholds are exceeded.
//
// Takes config (WatchdogConfig) which provides the threshold and timing
// configuration.
// Takes systemCollector (*SystemCollector) which provides the system
// statistics for periodic evaluation.
// Takes opts (...WatchdogOption) which provides optional configuration
// functions to customise clock and sandbox dependencies.
//
// Returns *Watchdog which is ready to be started.
// Returns error when the profile store cannot be initialised.
func NewWatchdog(config WatchdogConfig, systemCollector *SystemCollector, opts ...WatchdogOption) (*Watchdog, error) {
	if err := validateWatchdogConfig(config); err != nil {
		return nil, err
	}

	w := &Watchdog{
		config:          config,
		systemCollector: systemCollector,
		lastCaptureTime: make(map[string]time.Time),
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
// config, this method is a no-op.
//
// The background goroutine runs until Stop is called or the context is
// cancelled.
func (w *Watchdog) Start(ctx context.Context) {
	if !w.config.Enabled {
		return
	}

	w.resolveHeapThreshold(ctx)
	w.startedAt = w.clock.Now()

	if w.config.TrendWindowSize > 0 {
		w.heapTrendBuffer = newHeapTrendBuffer(w.config.TrendWindowSize)
	}

	w.goroutineLeakAvailable = pprof.Lookup(profileTypeGoroutineLeak) != nil

	go w.loop(ctx)
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

	w.captureAndStoreProfile(ctx, profileTypeHeap)
	w.captureAndStoreProfile(ctx, profileTypeGoroutine)
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

	if w.profileStore != nil {
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

// DownloadProfile writes the raw compressed bytes of the named profile file to
// the provided writer. The caller is responsible for decompression.
//
// Takes filename (string) which identifies the profile file to download.
// Takes writer (io.Writer) which receives the compressed profile data.
//
// Returns error when the filename is empty or the file cannot be read.
func (w *Watchdog) DownloadProfile(_ context.Context, filename string, writer io.Writer) error {
	data, err := w.profileStore.read(filename)
	if err != nil {
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
	w.mu.Unlock()

	return &WatchdogStatusInfo{
		ProfileDirectory:       w.config.ProfileDirectory,
		CheckInterval:          w.config.CheckInterval,
		Cooldown:               w.config.Cooldown,
		WarmUpDuration:         w.config.WarmUpDuration,
		StartedAt:              w.startedAt,
		HeapThresholdBytes:     w.initialHeapThreshold,
		HeapHighWater:          heapHighWater,
		GoroutineThreshold:     w.config.GoroutineThreshold,
		GoroutineSafetyCeiling: w.config.GoroutineSafetyCeiling,
		MaxProfilesPerType:     w.config.MaxProfilesPerType,
		Enabled:                w.config.Enabled,
		Stopped:                stopped,
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

// loop runs the periodic evaluation loop until stopped or the context is
// cancelled. Each tick reads system statistics and evaluates them against the
// configured thresholds.
func (w *Watchdog) loop(ctx context.Context) {
	ticker := w.clock.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(ctx, "monitoring.watchdogLoop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C():
			stats := w.systemCollector.GetStats()
			w.evaluate(ctx, &stats)
		}
	}
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
}

// triggerCapture spawns a background goroutine that captures a profile of the
// given type and stores it to disk. Errors are logged and counted but do not
// propagate to the caller.
//
// Takes profileType (string) which identifies the profile to capture (e.g.
// "heap", "goroutine").
func (w *Watchdog) triggerCapture(ctx context.Context, profileType string) {
	w.captureWG.Go(func() {
		defer goroutine.RecoverPanic(ctx, "monitoring.watchdogCapture."+profileType)
		w.captureAndStoreProfile(ctx, profileType)
	})
}

// captureAndStoreProfile captures a pprof profile via the profiling controller
// and writes the compressed data to the profile store.
//
// Takes profileType (string) which identifies the profile to capture.
//
// Safe for concurrent use; acquires the watchdog's mutex to check the stopped
// flag and profiling controller.
func (w *Watchdog) captureAndStoreProfile(ctx context.Context, profileType string) {
	if ctx.Err() != nil {
		return
	}
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	controller := w.profilingController
	w.mu.Unlock()

	ctx, l := logger_domain.From(ctx, log)

	if controller == nil {
		l.Warn("Profiling controller not available, skipping capture",
			String(logFieldProfileType, profileType),
		)
		return
	}

	var buffer bytes.Buffer

	_, err := controller.CaptureProfile(ctx, profileType, 0, &buffer)
	if err != nil {
		l.Error("Failed to capture profile",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		w.sendNotification(ctx, NewCaptureErrorEvent(profileType, err))
		return
	}

	profileData := buffer.Bytes()

	if int64(len(profileData)) > w.config.MaxProfileSizeBytes {
		l.Warn("Profile exceeds maximum size, discarding to avoid memory pressure",
			String(logFieldProfileType, profileType),
			logger_domain.Int64("profile_size_bytes", int64(len(profileData))),
			logger_domain.Int64("max_profile_size_bytes", w.config.MaxProfileSizeBytes),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		return
	}

	if err := w.profileStore.write(profileType, profileData); err != nil {
		l.Error("Failed to store profile",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		w.sendNotification(ctx, NewCaptureErrorEvent(profileType, err))
		return
	}

	w.processStoredProfile(ctx, profileType, profileData)

	l.Notice("Watchdog captured and stored diagnostic profile",
		String(logFieldProfileType, profileType),
	)
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

	if err := w.profileStore.write(profileTypeTrace, traceData); err != nil {
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

	if err := w.profileStore.write(profileTypeHeapBaseline, previousProfile); err != nil {
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
	if w.notifier == nil {
		return
	}

	detachedCtx := context.WithoutCancel(ctx)
	detachedCtx, notificationCancel := context.WithTimeoutCause(detachedCtx, notificationTimeout,
		errors.New("watchdog notification exceeded 30s budget"))

	w.backgroundWG.Go(func() {
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

	w.backgroundWG.Go(func() {
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
