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

package profiler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

const (
	// maxProfilingDuration is the maximum allowed profiling session duration.
	maxProfilingDuration = 24 * time.Hour

	// maxCaptureDuration is the maximum allowed duration for a single
	// CPU or trace capture.
	maxCaptureDuration = 5 * time.Minute

	// maxCaptureDurationSeconds is maxCaptureDuration expressed in whole
	// seconds, used for integer overflow validation before multiplying by
	// time.Second.
	maxCaptureDurationSeconds = int(maxCaptureDuration / time.Second)

	// controllerBindAddress is the address the on-demand pprof server
	// binds to. Localhost-only prevents remote access without an explicit
	// port-forward.
	controllerBindAddress = "127.0.0.1"

	// maxConcurrentSnapshots limits the number of snapshot profile captures
	// that may run simultaneously.
	maxConcurrentSnapshots = 4

	// maxTCPPort is the upper bound of the valid TCP port range.
	maxTCPPort = 65535

	// defaultCPUCaptureDuration is the fallback duration in seconds for CPU
	// profile captures when no duration is specified.
	defaultCPUCaptureDuration = 30

	// defaultTraceCaptureDuration is the fallback duration in seconds for
	// execution trace captures when no duration is specified.
	defaultTraceCaptureDuration = 5

	// goroutineLeakProfileName is the pprof lookup name for the Go 1.26
	// goroutine leak profile.
	goroutineLeakProfileName = "goroutineleak"
)

var (
	// ErrDurationExceedsMaximum is returned when the requested profiling or
	// capture duration exceeds the configured maximum.
	ErrDurationExceedsMaximum = errors.New("duration exceeds maximum")

	// ErrDurationNotPositive is returned when the requested duration is zero
	// or negative.
	ErrDurationNotPositive = errors.New("duration must be positive")

	// ErrCaptureExceedsMaximum is returned when a CPU or trace capture
	// duration exceeds maxCaptureDuration.
	ErrCaptureExceedsMaximum = errors.New("capture duration exceeds maximum")

	// ErrUnknownProfileType is returned when the requested profile type is
	// not recognised.
	ErrUnknownProfileType = errors.New("unknown profile type")

	// ErrPortOutOfRange is returned when the port number is outside the
	// valid TCP range 1-65535.
	ErrPortOutOfRange = errors.New("port out of range (1-65535)")
)

// availableProfiles lists all profile types the controller can capture.
var availableProfiles = []string{
	"heap", "goroutine", "allocs", "cpu", "trace", "block", "mutex",
}

// profileLookupNames maps capture profile type names to the runtime/pprof
// lookup names used by pprof.Lookup.
var profileLookupNames = map[string]string{
	"heap":      "heap",
	"goroutine": "goroutine",
	"allocs":    "allocs",
	"block":     "block",
	"mutex":     "mutex",
}

func init() {
	if pprof.Lookup(goroutineLeakProfileName) != nil {
		availableProfiles = append(availableProfiles, goroutineLeakProfileName)
		profileLookupNames[goroutineLeakProfileName] = goroutineLeakProfileName
	}
}

var _ monitoring_domain.ProfilingController = (*Controller)(nil)

// Controller implements monitoring_domain.ProfilingController by managing
// an on-demand pprof HTTP server and Go runtime profiling rates.
type Controller struct {
	// expiresAt records when the current profiling session will auto-disable.
	expiresAt time.Time

	// server is the on-demand pprof HTTP server handle; nil when disabled.
	server *ServerHandle

	// timer fires at session expiry to auto-disable profiling.
	timer *time.Timer

	// cancelTimer cancels the context passed to the timer goroutine so it
	// does not call Disable after Close.
	cancelTimer context.CancelFunc

	// snapshotSemaphore limits concurrent snapshot captures.
	snapshotSemaphore chan struct{}

	// port is the TCP port the pprof server listens on.
	port int

	// blockProfileRate is the configured Go runtime block profile rate.
	blockProfileRate int

	// mutexProfileFraction is the configured Go runtime mutex profile fraction.
	mutexProfileFraction int

	// originalMutexFraction stores the mutex fraction before profiling was
	// enabled, so it can be restored on disable.
	originalMutexFraction int

	// mu guards the controller's mutable state (server, timer, enabled, etc.).
	mu sync.Mutex

	// captureMu serialises CPU profile and trace captures because the Go
	// runtime only permits one of each at a time.
	captureMu sync.Mutex

	// enabled is true when the pprof server is running.
	enabled bool
}

// NewController creates a new profiling controller.
//
// Returns *Controller ready for use.
func NewController() *Controller {
	return &Controller{
		snapshotSemaphore: make(chan struct{}, maxConcurrentSnapshots),
	}
}

// Enable starts the pprof HTTP server and sets Go runtime profiling
// rates. If already enabled, it extends the expiry deadline without
// restarting the server.
//
// Takes opts (ProfilingEnableOpts) which configures the profiling
// session duration, port, and sampling rates.
//
// Returns *ProfilingStatus which describes the active profiling
// session after enabling.
// Returns error when the server fails to start or the duration
// exceeds the maximum allowed.
// Returns *monitoring_domain.ProfilingStatus which contains the current state
// after enabling.
// Returns error when the server fails to start or the duration exceeds the
// maximum allowed.
//
// Safe for concurrent use.
func (c *Controller) Enable(ctx context.Context, opts monitoring_domain.ProfilingEnableOpts) (*monitoring_domain.ProfilingStatus, error) {
	ctx, _ = logger_domain.From(ctx, log)

	if opts.Duration <= 0 {
		return nil, fmt.Errorf("profiling %w", ErrDurationNotPositive)
	}
	if opts.Duration > maxProfilingDuration {
		return nil, fmt.Errorf("profiling duration %s: %w (maximum %s)", opts.Duration, ErrDurationExceedsMaximum, maxProfilingDuration)
	}

	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}
	if port < 1 || port > maxTCPPort {
		return nil, fmt.Errorf("port %d: %w", port, ErrPortOutOfRange)
	}

	blockRate := opts.BlockProfileRate
	if blockRate == 0 {
		blockRate = DefaultBlockProfileRate
	}
	if blockRate < 0 {
		return nil, fmt.Errorf("block profile rate must be non-negative, got %d", blockRate)
	}

	mutexFraction := opts.MutexProfileFraction
	if mutexFraction == 0 {
		mutexFraction = DefaultMutexProfileFraction
	}
	if mutexFraction < 0 {
		return nil, fmt.Errorf("mutex profile fraction must be non-negative, got %d", mutexFraction)
	}

	c.mu.Lock()

	if c.enabled {
		status := c.extendSession(ctx, opts.Duration)
		c.mu.Unlock()
		return status, nil
	}

	c.mu.Unlock()

	return c.startProfilingServer(ctx, opts.Duration, port, blockRate, mutexFraction)
}

// Disable stops the pprof HTTP server and resets Go runtime profiling
// rates to their pre-enable values.
//
// Returns bool which is true if profiling was previously enabled.
// Returns error when the server fails to shut down cleanly.
//
// Safe for concurrent use.
func (c *Controller) Disable(ctx context.Context) (bool, error) {
	ctx, l := logger_domain.From(ctx, log)

	c.mu.Lock()

	if !c.enabled {
		c.mu.Unlock()
		return false, nil
	}

	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	if c.cancelTimer != nil {
		c.cancelTimer()
		c.cancelTimer = nil
	}

	serverToShutdown := c.server
	c.server = nil
	originalMutexFraction := c.originalMutexFraction
	c.enabled = false
	c.expiresAt = time.Time{}

	c.mu.Unlock()

	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(originalMutexFraction)

	var shutdownErr error
	if serverToShutdown != nil {
		shutdownCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
			errors.New("pprof server shutdown exceeded 5s"))
		shutdownErr = serverToShutdown.Shutdown(shutdownCtx)
		cancel()
	}

	l.Notice("Profiling disabled")

	return true, shutdownErr
}

// Close cancels any pending auto-disable timer and disables profiling.
//
// Returns error when the shutdown fails.
func (c *Controller) Close(ctx context.Context) error {
	_, err := c.Disable(ctx)
	return err
}

// Status returns the current profiling state.
//
// Returns *monitoring_domain.ProfilingStatus which contains the current state.
//
// Safe for concurrent use.
func (c *Controller) Status(_ context.Context) *monitoring_domain.ProfilingStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statusLocked()
}

// SnapshotFlightRecorder writes the current rolling execution trace buffer to
// the provided writer. Returns an error when the flight recorder is not
// enabled or the snapshot fails.
//
// Takes w (io.Writer) which receives the trace data.
//
// Returns error when the flight recorder is disabled or the snapshot fails.
//
// Safe for concurrent use.
func (c *Controller) SnapshotFlightRecorder(_ context.Context, w io.Writer) error {
	c.mu.Lock()
	server := c.server
	c.mu.Unlock()

	if server == nil || server.rollingTrace == nil || !server.rollingTrace.Enabled() {
		return ErrRollingTraceDisabled
	}

	_, err := server.rollingTrace.WriteTo(w)
	return err
}

// CaptureProfile captures a Go runtime profile and writes the raw data
// to the provided writer. For duration-based profiles (cpu, trace), this
// blocks for the requested duration.
//
// Takes profileType (string) which identifies the profile to capture
// (heap, goroutine, allocs, cpu, trace, block, mutex).
// Takes durationSeconds (int) which sets the capture window for
// duration-based profiles; ignored for snapshot profiles.
// Takes w (io.Writer) which receives the raw profile data.
//
// Returns string which contains any warning message.
// Returns error when the profile type is unknown or capture fails.
func (c *Controller) CaptureProfile(ctx context.Context, profileType string, durationSeconds int, w io.Writer) (string, error) {
	switch profileType {
	case "cpu":
		return c.captureCPU(ctx, durationSeconds, w)
	case "trace":
		return c.captureTrace(ctx, durationSeconds, w)
	case "heap", "goroutine", "allocs", "block", "mutex", goroutineLeakProfileName:
		return c.captureSnapshot(ctx, profileType, w)
	default:
		return "", fmt.Errorf("%w: %s (available: heap, goroutine, allocs, cpu, trace, block, mutex)", ErrUnknownProfileType, profileType)
	}
}

// extendSession extends a running profiling session's deadline. The caller
// must hold c.mu and must unlock it after this method returns.
//
// Takes duration (time.Duration) which is the new session duration.
//
// Returns *monitoring_domain.ProfilingStatus which contains the updated state.
func (c *Controller) extendSession(ctx context.Context, duration time.Duration) *monitoring_domain.ProfilingStatus {
	_, l := logger_domain.From(ctx, log)

	c.expiresAt = time.Now().Add(duration)
	if c.timer != nil {
		c.timer.Reset(duration)
	}

	l.Notice("Profiling session extended",
		logger_domain.String("expires_at", c.expiresAt.Format(time.RFC3339)),
		logger_domain.Int("port", c.port))

	status := c.statusLocked()
	status.AlreadyEnabled = true
	return status
}

// startProfilingServer starts the pprof HTTP server, sets runtime profiling
// rates, and records the new session state. The caller must NOT hold c.mu.
//
// Takes duration (time.Duration) which is the session duration.
// Takes port (int) which is the TCP port to bind.
// Takes blockRate (int) which is the block profile rate.
// Takes mutexFraction (int) which is the mutex profile fraction.
//
// Returns *monitoring_domain.ProfilingStatus which contains the session state.
// Returns error when the server fails to start.
func (c *Controller) startProfilingServer(
	ctx context.Context,
	duration time.Duration,
	port int,
	blockRate int,
	mutexFraction int,
) (*monitoring_domain.ProfilingStatus, error) {
	originalMutexFraction := runtime.SetMutexProfileFraction(0)
	runtime.SetMutexProfileFraction(originalMutexFraction)

	config := Config{
		Port:                 port,
		BindAddress:          controllerBindAddress,
		BlockProfileRate:     blockRate,
		MutexProfileFraction: mutexFraction,
	}
	SetRuntimeRates(config)

	server, err := StartServer(config)
	if err != nil {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(originalMutexFraction)
		return nil, fmt.Errorf("starting pprof server: %w", err)
	}
	server.SetErrorHandler(func(serverErr error) {
		_, errorLogger := logger_domain.From(context.Background(), log)
		errorLogger.Error("On-demand pprof server error", logger_domain.Error(serverErr))
	})

	return c.commitSession(ctx, server, config, duration, originalMutexFraction)
}

// commitSession records the new session state under the lock. If another
// goroutine enabled profiling while the server was starting, the new
// server is shut down and the existing session is returned.
//
// Takes server (*ServerHandle) which is the started pprof server.
// Takes config (Config) which holds the server configuration.
// Takes duration (time.Duration) which is the session duration.
// Takes originalMutexFraction (int) which is the pre-enable mutex fraction.
//
// Returns *ProfilingStatus which describes the newly committed
// profiling session state.
// Returns error when an enable race requires shutting down the duplicate server.
// Returns *monitoring_domain.ProfilingStatus which contains the session state.
// Returns error when an enable race requires shutting down the duplicate
// server.
func (c *Controller) commitSession(
	ctx context.Context,
	server *ServerHandle,
	config Config,
	duration time.Duration,
	originalMutexFraction int,
) (*monitoring_domain.ProfilingStatus, error) {
	_, l := logger_domain.From(ctx, log)

	c.mu.Lock()
	if c.enabled {
		return c.handleEnableRace(ctx, server, originalMutexFraction)
	}

	c.originalMutexFraction = originalMutexFraction
	c.server = server
	c.port = config.Port
	c.blockProfileRate = config.BlockProfileRate
	c.mutexProfileFraction = config.MutexProfileFraction
	c.expiresAt = time.Now().Add(duration)
	c.enabled = true
	c.startAutoDisableTimer(duration)

	l.Notice("Profiling enabled",
		logger_domain.String("expires_at", c.expiresAt.Format(time.RFC3339)),
		logger_domain.Int("port", config.Port),
		logger_domain.Int("block_profile_rate", config.BlockProfileRate),
		logger_domain.Int("mutex_profile_fraction", config.MutexProfileFraction))

	status := c.statusLocked()
	c.mu.Unlock()
	return status, nil
}

// handleEnableRace handles the case where another goroutine enabled profiling
// while the server was starting. The caller must hold c.mu.
//
// Takes server (*ServerHandle) which is the duplicate server to shut down.
// Takes originalMutexFraction (int) which is the pre-enable mutex fraction to restore.
//
// Returns *ProfilingStatus which describes the existing session that
// won the enable race.
// Returns *monitoring_domain.ProfilingStatus which contains the existing
// session state.
// Returns error when the duplicate server fails to shut down.
func (c *Controller) handleEnableRace(
	ctx context.Context,
	server *ServerHandle,
	originalMutexFraction int,
) (*monitoring_domain.ProfilingStatus, error) {
	_, l := logger_domain.From(ctx, log)

	c.mu.Unlock()
	shutdownCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
		errors.New("pprof server shutdown exceeded 5s (enable race)"))
	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		l.Warn("Failed to shut down duplicate pprof server", logger_domain.Error(shutdownErr))
	}
	cancel()
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(originalMutexFraction)

	c.mu.Lock()
	status := c.statusLocked()
	status.AlreadyEnabled = true
	c.mu.Unlock()
	return status, nil
}

// startAutoDisableTimer starts the timer that auto-disables profiling when the
// session expires. The caller must hold c.mu.
//
// Takes duration (time.Duration) which is the delay before auto-disable fires.
func (c *Controller) startAutoDisableTimer(duration time.Duration) {
	timerCtx, timerCancel := context.WithCancel(context.Background())
	c.cancelTimer = timerCancel
	c.timer = time.AfterFunc(duration, func() {
		if timerCtx.Err() != nil {
			return
		}
		_, disableLogger := logger_domain.From(timerCtx, log)
		disableLogger.Info("Profiling session expired, auto-disabling")
		_, _ = c.Disable(timerCtx)
	})
}

// captureCPU captures a CPU profile for the given duration.
//
// Takes durationSeconds (int) which is the capture duration; zero uses the default.
// Takes w (io.Writer) which receives the raw pprof data.
//
// Returns string which contains any warning message.
// Returns error when the capture fails.
func (c *Controller) captureCPU(ctx context.Context, durationSeconds int, w io.Writer) (string, error) {
	return c.captureDurationBased(ctx, durationSeconds, defaultCPUCaptureDuration,
		"CPU", pprof.StartCPUProfile, pprof.StopCPUProfile, w)
}

// captureTrace captures an execution trace for the given duration.
//
// Takes durationSeconds (int) which is the capture duration; zero uses the default.
// Takes w (io.Writer) which receives the raw trace data.
//
// Returns string which contains any warning message.
// Returns error when the capture fails.
func (c *Controller) captureTrace(ctx context.Context, durationSeconds int, w io.Writer) (string, error) {
	return c.captureDurationBased(ctx, durationSeconds, defaultTraceCaptureDuration,
		"trace", trace.Start, trace.Stop, w)
}

// captureDurationBased captures a duration-based profile (CPU or trace) by
// calling startFunc to begin recording and stopFunc to finalise. It validates
// the requested duration, serialises access via captureMu, and respects
// context cancellation.
//
// Takes durationSeconds (int) which is the requested capture duration.
// Takes defaultSeconds (int) which is the fallback when durationSeconds is zero or negative.
// Takes label (string) which identifies the profile type for error messages.
// Takes startFunc (func(io.Writer) error) which begins the capture.
// Takes stopFunc (func()) which finalises the capture.
// Takes w (io.Writer) which receives the raw profile data.
//
// Returns string which is always empty for duration-based captures.
// Returns error when the duration exceeds the maximum or the capture fails.
//
// Concurrency: serialises access via captureMu because the Go runtime only
// permits one CPU profile or trace at a time.
func (c *Controller) captureDurationBased(
	ctx context.Context,
	durationSeconds int,
	defaultSeconds int,
	label string,
	startFunc func(io.Writer) error,
	stopFunc func(),
	w io.Writer,
) (string, error) {
	if durationSeconds <= 0 {
		durationSeconds = defaultSeconds
	}
	if durationSeconds > maxCaptureDurationSeconds {
		return "", fmt.Errorf("%s capture duration %ds: %w (maximum %s)",
			label, durationSeconds, ErrCaptureExceedsMaximum, maxCaptureDuration)
	}
	duration := time.Duration(durationSeconds) * time.Second

	c.captureMu.Lock()
	defer c.captureMu.Unlock()

	if err := startFunc(w); err != nil {
		return "", fmt.Errorf("starting %s capture: %w", label, err)
	}

	captureTimer := time.NewTimer(duration)
	defer captureTimer.Stop()

	select {
	case <-captureTimer.C:
	case <-ctx.Done():
		stopFunc()
		return "", fmt.Errorf("%s capture cancelled: %w", label, ctx.Err())
	}

	stopFunc()
	return "", nil
}

// captureSnapshot captures a point-in-time profile snapshot.
//
// Takes profileType (string) which identifies the profile to capture.
// Takes w (io.Writer) which receives the raw pprof data.
//
// Returns string which contains a warning when runtime rates are not configured.
// Returns error when the profile type is unknown or capture fails.
//
// Concurrency: limits concurrent snapshots via snapshotSemaphore.
func (c *Controller) captureSnapshot(ctx context.Context, profileType string, w io.Writer) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("snapshot capture cancelled before start: %w", err)
	}

	lookupName, ok := profileLookupNames[profileType]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownProfileType, profileType)
	}

	select {
	case c.snapshotSemaphore <- struct{}{}:
		defer func() { <-c.snapshotSemaphore }()
	case <-ctx.Done():
		return "", fmt.Errorf("snapshot capture cancelled waiting for semaphore: %w", ctx.Err())
	}

	var warning string

	c.mu.Lock()
	enabled := c.enabled
	c.mu.Unlock()

	if !enabled && (profileType == "block" || profileType == "mutex") {
		warning = fmt.Sprintf(
			"%s profiling requires runtime rates to be configured; results may be empty. "+
				"Run 'piko profiling enable 10m' first.", profileType)
	}

	profile := pprof.Lookup(lookupName)
	if profile == nil {
		return "", fmt.Errorf("profile %q not found in runtime", lookupName)
	}

	if err := profile.WriteTo(w, 0); err != nil {
		return warning, fmt.Errorf("writing %s profile: %w", profileType, err)
	}

	return warning, nil
}

// statusLocked builds the current profiling status. The caller must hold c.mu.
//
// Returns *monitoring_domain.ProfilingStatus which contains the current state.
func (c *Controller) statusLocked() *monitoring_domain.ProfilingStatus {
	status := &monitoring_domain.ProfilingStatus{
		Enabled:           c.enabled,
		AvailableProfiles: availableProfiles,
		MemProfileRate:    runtime.MemProfileRate,
	}

	if c.enabled {
		status.Port = c.port
		status.ExpiresAt = c.expiresAt
		status.PprofBaseURL = "http://" + net.JoinHostPort(controllerBindAddress, strconv.Itoa(c.port)) + BasePath + "/debug/pprof/"
		status.BlockProfileRate = c.blockProfileRate
		status.MutexProfileFraction = c.mutexProfileFraction
	}

	return status
}
