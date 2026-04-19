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

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"piko.sh/piko/wdk/json"

	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// profileFilePerms is the file permission used when writing profile output files.
	profileFilePerms = 0o640

	// profileGoroutineReadBufSize is the buffer size for reading
	// goroutine snapshot headers.
	profileGoroutineReadBufSize = 256

	// profileBaselineName identifies the baseline load test phase.
	profileBaselineName = "baseline"

	// profileDefaultPprofPort is the default port for pprof endpoints.
	profileDefaultPprofPort = 6060

	// profileDefaultConcurrency is the default number of concurrent HTTP workers.
	profileDefaultConcurrency = 100

	// profileDefaultDuration is the default phase duration in seconds.
	profileDefaultDuration = 30

	// profileDefaultTopN is the default number of top entries per report section.
	profileDefaultTopN = 60

	// profileErrorChBuffer is the buffer size for the error channel.
	profileErrorChBuffer = 10000

	// profileMetricsIntervalMs is the metrics emission interval
	// in milliseconds for TUI mode.
	profileMetricsIntervalMs = 200

	// profileMaxBodyBytes caps the bytes accepted from a pprof or
	// profiler-status HTTP body.
	profileMaxBodyBytes = 256 * 1024 * 1024
)

// readAndDrainBody reads up to profileMaxBodyBytes from response.Body
// using io.LimitReader, then drains and closes the body so the
// underlying connection can be reused. Returns ErrProfileBodyTooLarge
// when the cap is exactly hit (an indication the response was likely
// truncated and should not be trusted).
//
// Takes body (io.ReadCloser) which is the HTTP response body; the caller still
// owns Close (only drains here, the deferred Close in the caller still runs).
//
// Returns []byte with the body bytes (possibly truncated to the cap).
// Returns error wrapping ErrProfileBodyTooLarge when the body is at
// least profileMaxBodyBytes.
func readAndDrainBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, profileMaxBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > profileMaxBodyBytes {
		return nil, fmt.Errorf("%w: %d bytes received", ErrProfileBodyTooLarge, len(data))
	}
	return data, nil
}

// drainAndClose drains any remaining bytes on body and closes it so
// the HTTP transport can reuse the underlying connection. Errors are
// silently ignored because callers are typically already on an error
// path.
//
// Takes body (io.ReadCloser) which is the HTTP response body to drain.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

// ErrProfileBodyTooLarge is returned when an HTTP response from a
// pprof or profiler-status endpoint exceeds profileMaxBodyBytes.
var ErrProfileBodyTooLarge = errors.New("pprof response exceeded byte budget")

// headerFlag implements flag.Value for repeatable --header flags.
type headerFlag struct {
	// headers maps header names to their values.
	headers map[string]string
}

// String returns a string representation for flag defaults.
//
// Returns string which is the comma-separated header list.
func (h *headerFlag) String() string {
	if len(h.headers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(h.headers))
	for k, v := range h.headers {
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, ", ")
}

// Set parses a "Name: Value" header and stores it.
//
// Takes value (string) which is the header in "Name: Value" format.
//
// Returns error when the format is invalid.
func (h *headerFlag) Set(value string) error {
	name, headerValue, ok := strings.Cut(value, ":")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("invalid header format %q, expected \"Name: Value\"", value)
	}
	if h.headers == nil {
		h.headers = make(map[string]string)
	}
	h.headers[strings.TrimSpace(name)] = strings.TrimSpace(headerValue)
	return nil
}

// profileFlags holds the parsed flags for the profile subcommand.
type profileFlags struct {
	// headers contains HTTP headers to send with load requests.
	headers headerFlag

	// output is the directory for .pprof files and the report.
	output string

	// cookie is an optional Cookie header value (convenience shorthand for
	// --header "Cookie: ...").
	cookie string

	// focus is an optional regex filter for function names.
	focus string

	// pprofPort is the port where the pprof debug endpoints are exposed.
	pprofPort int

	// concurrency is the number of concurrent HTTP workers.
	concurrency int

	// duration is the phase duration in seconds (baseline and each profile).
	duration int

	// topN is the number of top entries per report section.
	topN int

	// tui enables the BubbleTea live dashboard during profiling.
	tui bool
}

// mergedHeaders returns a map combining --header flags and the --cookie
// convenience flag.
//
// Returns map[string]string which contains all merged headers.
func (f *profileFlags) mergedHeaders() map[string]string {
	h := make(map[string]string, len(f.headers.headers)+1)
	maps.Copy(h, f.headers.headers)
	if f.cookie != "" {
		h["Cookie"] = f.cookie
	}
	return h
}

// pipelineConfig holds the shared configuration for a profiling pipeline,
// used by both TUI and non-TUI modes.
type pipelineConfig struct {
	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// interrupt is checked between phases; if non-nil and cancelled, the
	// pipeline stops early. Used for signal handling in non-TUI mode.
	interrupt context.Context

	// stdout receives normal output messages.
	stdout io.Writer

	// stderr receives error and diagnostic messages.
	stderr io.Writer

	// headers contains HTTP headers to send with load requests.
	headers map[string]string

	// flags holds the parsed command-line flags.
	flags *profileFlags

	// TUI-only channels. When nil, the pipeline operates in non-TUI mode.
	metricsCh chan<- metricsMessage

	// phaseCh receives phase transition messages for the TUI.
	phaseCh chan<- phaseMessage

	// url is the target URL to load test.
	url string

	// pprofBase is the base URL for pprof endpoints.
	pprofBase string

	// profilerRoot is the server root URL used for profiler capability requests.
	profilerRoot string

	// specs lists the profiles to capture during the pipeline.
	specs []profileSpec
}

// profileSpec describes a single pprof profile to capture.
type profileSpec struct {
	// name is the human-readable profile name used for file naming
	// and report headings.
	name string

	// endpoint is the pprof HTTP endpoint path (e.g. "profile", "heap").
	endpoint string

	// reports defines the report sections to generate from this profile.
	reports []profileReportConfig

	// durationBased is true when the endpoint accepts a ?seconds= parameter.
	durationBased bool

	// delta is true when the profile should be captured as a before/after
	// delta rather than a single snapshot. This isolates only the allocations
	// (or other sample values) that occurred during the load window.
	delta bool
}

// emitPhase sends a phase message to the TUI channel if present.
//
// Takes name (string) which identifies the pipeline phase.
// Takes status (phaseStatus) which is the new phase status.
func (pipeline pipelineConfig) emitPhase(name string, status phaseStatus) {
	if pipeline.phaseCh != nil {
		pipeline.phaseCh <- phaseMessage{name: name, status: status}
	}
}

// buildLoadConfig creates a loadConfig for a given phase, including TUI
// metrics channels when configured.
//
// Takes phase (string) which names the pipeline phase.
// Takes errorCh (chan<- loadErrorRecord) which receives error records.
//
// Returns loadConfig which is the configured load generator settings.
func (pipeline pipelineConfig) buildLoadConfig(phase string, errorCh chan<- loadErrorRecord) loadConfig {
	lc := loadConfig{
		url:         pipeline.url,
		concurrency: pipeline.flags.concurrency,
		maxRequests: 0,
		headers:     pipeline.headers,
		errorCh:     errorCh,
		phase:       phase,
	}
	if pipeline.metricsCh != nil {
		lc.metricsInterval = profileMetricsIntervalMs * time.Millisecond
		lc.metricsCh = pipeline.metricsCh
	}
	return lc
}

// RunProfile runs the profile subcommand, writing to os.Stdout and os.Stderr.
//
// Takes arguments ([]string) which contains the command-line arguments.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunProfile(arguments []string) int {
	return RunProfileWithIO(arguments, os.Stdout, os.Stderr)
}

// RunProfileWithIO runs the profile subcommand with explicit output writers.
//
// Takes arguments ([]string) which contains the command-line arguments.
// Takes stdout (io.Writer) which receives normal output messages.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func RunProfileWithIO(arguments []string, stdout, stderr io.Writer) int {
	flags, url, ok := parseProfileFlags(arguments, stderr)
	if !ok {
		return 1
	}

	focusRegex, err := compileFocusRegex(flags.focus)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: invalid --focus pattern %q: %v\n", flags.focus, err)
		return 1
	}

	warnShortDuration(flags.duration, stderr)

	factory, err := safedisk.NewCLIFactory(".")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error creating sandbox factory: %v\n", err)
		return 1
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx := context.WithoutCancel(sigCtx)

	profilerRoot := fmt.Sprintf("http://localhost:%d", flags.pprofPort)
	pprofBase := profilerRoot + profiler.BasePath + "/debug/pprof"
	headers := flags.mergedHeaders()

	if flags.tui {
		return runProfileTUI(ctx, profileTUIParams{
			factory:      factory,
			flags:        flags,
			url:          url,
			focusRegex:   focusRegex,
			pprofBase:    pprofBase,
			profilerRoot: profilerRoot,
			headers:      headers,
			stdout:       stdout,
			stderr:       stderr,
		})
	}

	return runProfileCLI(ctx, profileCLIParams{
		interrupt:    sigCtx,
		factory:      factory,
		flags:        flags,
		url:          url,
		focusRegex:   focusRegex,
		pprofBase:    pprofBase,
		profilerRoot: profilerRoot,
		headers:      headers,
		stdout:       stdout,
		stderr:       stderr,
	})
}

// profileCLIParams groups the parameters for runProfileCLI.
type profileCLIParams struct {
	// interrupt is the signal-aware context used for pipeline cancellation.
	interrupt context.Context

	// factory creates sandboxes for filesystem access.
	factory safedisk.Factory

	// flags holds the parsed CLI flags.
	flags *profileFlags

	// focusRegex optionally filters function names.
	focusRegex *regexp.Regexp

	// stdout receives normal output.
	stdout io.Writer

	// stderr receives error output.
	stderr io.Writer

	// headers holds HTTP request headers.
	headers map[string]string

	// url is the target URL to profile.
	url string

	// pprofBase is the pprof endpoint base URL.
	pprofBase string

	// profilerRoot is the server root URL used for profiler capability requests.
	profilerRoot string
}

// runProfileCLI executes the non-TUI profile pipeline, collecting goroutine
// counts before and after, then printing the summary.
//
// Takes params (profileCLIParams) which groups all pipeline configuration.
//
// Returns int which is the exit code: 0 on success, 1 on error.
func runProfileCLI(ctx context.Context, params profileCLIParams) int {
	startGoroutines := fetchGoroutineCount(ctx, params.pprofBase)
	printProfileHeader(params.stdout, params.url, params.pprofBase, params.flags, params.headers)

	if err := runPipeline(ctx, pipelineConfig{
		factory:      params.factory,
		flags:        params.flags,
		url:          params.url,
		pprofBase:    params.pprofBase,
		profilerRoot: params.profilerRoot,
		headers:      params.headers,
		specs:        buildProfileSpecs(params.flags, params.focusRegex),
		stdout:       params.stdout,
		stderr:       params.stderr,
		interrupt:    params.interrupt,
	}); err != nil {
		_, _ = fmt.Fprintf(params.stderr, "Error: %v\n", err)
		return 1
	}

	endGoroutines := fetchGoroutineCount(ctx, params.pprofBase)
	printProfileSummary(params.stdout, params.flags, startGoroutines, endGoroutines)

	return 0
}

// compileFocusRegex compiles the focus pattern if non-empty.
//
// Takes pattern (string) which is the regex to compile.
//
// Returns *regexp.Regexp which is the compiled regex, or nil
// when pattern is empty.
// Returns error when the pattern is invalid.
func compileFocusRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

// warnShortDuration prints a warning when the profile duration
// is below the recommended minimum.
//
// Takes duration (int) which is the phase duration in seconds.
// Takes stderr (io.Writer) which receives the warning message.
func warnShortDuration(duration int, stderr io.Writer) {
	const minRecommendedDuration = 30
	if duration < minRecommendedDuration {
		_, _ = fmt.Fprintf(stderr,
			"Warning: --duration %d is below the recommended minimum of %d seconds. "+
				"CPU profile statistics may be unreliable with short durations.\n\n",
			duration, minRecommendedDuration)
	}
}

// printProfileHeader writes the profiling session header to
// stdout.
//
// Takes stdout (io.Writer) which receives the header output.
// Takes url (string) which is the target URL being profiled.
// Takes pprofBase (string) which is the pprof base URL.
// Takes flags (*profileFlags) which holds parsed CLI flags.
// Takes headers (map[string]string) which contains HTTP
// headers.
func printProfileHeader(stdout io.Writer, url, pprofBase string, flags *profileFlags, headers map[string]string) {
	_, _ = fmt.Fprint(stdout, "Starting live server profiling session...\n")
	_, _ = fmt.Fprint(stdout, "--------------------------------------------------\n")
	_, _ = fmt.Fprintf(stdout, "App Target:       %s\n", url)
	_, _ = fmt.Fprintf(stdout, "Pprof Endpoint:   %s\n", pprofBase)
	_, _ = fmt.Fprintf(stdout, "Concurrency:      %d\n", flags.concurrency)
	_, _ = fmt.Fprintf(stdout, "Phase Duration:   %ds\n", flags.duration)
	_, _ = fmt.Fprintf(stdout, "Output:           %s/\n", flags.output)
	for k, v := range headers {
		_, _ = fmt.Fprintf(stdout, "Header:           %s: %s\n", k, v)
	}
	if flags.focus != "" {
		_, _ = fmt.Fprintf(stdout, "Focus Filter:     %s\n", flags.focus)
	}
	_, _ = fmt.Fprint(stdout, "--------------------------------------------------\n\n")
}

// printProfileSummary writes the profiling completion summary
// to stdout.
//
// Takes stdout (io.Writer) which receives the summary output.
// Takes flags (*profileFlags) which holds parsed CLI flags.
// Takes startGoroutines (int) which is the count at session
// start.
// Takes endGoroutines (int) which is the count at session end.
func printProfileSummary(stdout io.Writer, flags *profileFlags, startGoroutines, endGoroutines int) {
	_, _ = fmt.Fprint(stdout, "========================================================================\n")
	_, _ = fmt.Fprint(stdout, "Profiling complete!\n\n")

	if startGoroutines > 0 && endGoroutines > 0 {
		_, _ = fmt.Fprintf(stdout, "Goroutines: %d (start) -> %d (end)\n", startGoroutines, endGoroutines)
		if endGoroutines > startGoroutines*2 {
			_, _ = fmt.Fprint(stdout, "  WARNING: Goroutine count more than doubled, possible goroutine leak\n")
		}
		_, _ = fmt.Fprint(stdout, "\n")
	}
	reportPath := filepath.Join(flags.output, "live_performance_report.txt")
	_, _ = fmt.Fprintf(stdout, "Summary report: %s\n\n", reportPath)
	_, _ = fmt.Fprintf(stdout, "Raw profile files saved to %s/:\n", flags.output)
	_, _ = fmt.Fprint(stdout, "  cpu.pprof     - CPU usage (where time is being spent)\n")
	_, _ = fmt.Fprint(stdout, "  allocs.pprof  - Memory allocation churn (what creates garbage)\n")
	_, _ = fmt.Fprint(stdout, "  heap.pprof    - Memory in use (what is taking up RAM)\n")
	_, _ = fmt.Fprint(stdout, "  mutex.pprof   - Lock contention (where goroutines wait for locks)\n")
	_, _ = fmt.Fprint(stdout, "  block.pprof   - Goroutine blocking (where goroutines wait on I/O, channels)\n\n")
	_, _ = fmt.Fprint(stdout, "Deep-dive analysis:\n")
	_, _ = fmt.Fprintf(stdout, "  go tool pprof %s/cpu.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof -alloc_space %s/allocs.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof -alloc_objects %s/allocs.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof -inuse_space %s/heap.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof %s/mutex.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof %s/block.pprof\n", flags.output)
	_, _ = fmt.Fprint(stdout, "\nLine-level allocation analysis:\n")
	_, _ = fmt.Fprintf(stdout, "  go tool pprof -lines -top -flat -sample_index=alloc_space %s/allocs.pprof\n", flags.output)
	_, _ = fmt.Fprintf(stdout, "  go tool pprof -lines -top -flat -sample_index=alloc_objects %s/allocs.pprof\n", flags.output)
	_, _ = fmt.Fprint(stdout, "========================================================================\n")
}

// runPipeline executes the full profiling pipeline.
//
// Takes pipeline (pipelineConfig) which holds all pipeline settings.
//
// Returns error when sandbox or report file creation fails.
//
// Spawns a goroutine that writes error log entries to disk.
// The goroutine runs until the error channel is closed.
func runPipeline(ctx context.Context, pipeline pipelineConfig) error {
	outputSandbox, err := pipeline.factory.Create("profile-output", pipeline.flags.output, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("cannot create output sandbox: %w", err)
	}
	defer func() { _ = outputSandbox.Close() }()

	reportFile, err := outputSandbox.Create("live_performance_report.txt")
	if err != nil {
		return fmt.Errorf("cannot create report file: %w", err)
	}
	defer func() { _ = reportFile.Close() }()

	errorCh := make(chan loadErrorRecord, profileErrorChBuffer)
	errorLogDone := make(chan struct{})

	profilerStatus := captureProfilerMetadata(ctx, pipeline, outputSandbox)
	go func() {
		_ = writeErrorLog(errorCh, outputSandbox)
		close(errorLogDone)
	}()

	runBaseline(ctx, pipeline, reportFile, outputSandbox, errorCh)

	for _, spec := range pipeline.specs {
		if pipeline.interrupt != nil && pipeline.interrupt.Err() != nil {
			_, _ = fmt.Fprint(pipeline.stdout, "\nInterrupted, skipping remaining phases.\n")
			break
		}

		pipeline.emitPhase(spec.name, phaseActive)

		if err := capturePhase(ctx, pipeline, reportFile, spec, outputSandbox, errorCh); err != nil {
			_, _ = fmt.Fprintf(pipeline.stderr, "Warning: %s profile failed: %v\n", spec.name, err)
		}

		pipeline.emitPhase(spec.name, phaseDone)
	}

	close(errorCh)
	<-errorLogDone

	captureRollingTrace(ctx, pipeline, outputSandbox, profilerStatus)

	return nil
}

// runBaseline executes the baseline load test phase, writes its
// report and statistics to the output sandbox, and snapshots
// goroutines.
//
// Takes pipeline (pipelineConfig) which holds pipeline settings.
// Takes reportFile (io.Writer) which receives the report output.
// Takes outputSandbox (safedisk.Sandbox) which writes output files.
// Takes errorCh (chan<- loadErrorRecord) which receives error records.
func runBaseline(
	ctx context.Context,
	pipeline pipelineConfig,
	reportFile io.Writer,
	outputSandbox safedisk.Sandbox,
	errorCh chan<- loadErrorRecord,
) {
	pipeline.emitPhase(profileBaselineName, phaseActive)
	_, _ = fmt.Fprintf(pipeline.stdout, "Running baseline load test (%ds, %d concurrency)...\n",
		pipeline.flags.duration, pipeline.flags.concurrency)

	baselineCtx, baselineCancel := context.WithTimeoutCause(
		ctx,
		time.Duration(pipeline.flags.duration)*time.Second,
		errors.New("baseline load test completed"),
	)

	baselineResult := runLoad(baselineCtx, pipeline.buildLoadConfig(profileBaselineName, errorCh))
	baselineCancel()

	_, _ = fmt.Fprintf(pipeline.stdout, "  Baseline complete: %.2f request/s, %s mean latency\n\n",
		baselineResult.requestsPerSecond(), baselineResult.meanLatency().Round(time.Microsecond))

	writeLoadTestReport(reportFile, baselineResult)
	_ = writeProfileStats(outputSandbox, "baseline.stats", baselineResult, pipeline.flags.concurrency)
	snapshotGoroutines(ctx, pipeline.stdout, pipeline.stderr, pipeline.pprofBase, profileBaselineName, outputSandbox)
	pipeline.emitPhase(profileBaselineName, phaseDone)
}

// captureProfilerMetadata fetches the profiler server status from
// the remote endpoint and writes it as JSON to the output sandbox.
//
// Takes pipeline (pipelineConfig) which holds pipeline settings.
// Takes outputSandbox (safedisk.Sandbox) which writes output files.
//
// Returns *profiler.ServerStatus which is the fetched status, or nil
// if the profiler is unavailable.
func captureProfilerMetadata(
	ctx context.Context,
	pipeline pipelineConfig,
	outputSandbox safedisk.Sandbox,
) *profiler.ServerStatus {
	if pipeline.profilerRoot == "" {
		return nil
	}

	status, err := fetchProfilerStatus(ctx, pipeline.profilerRoot)
	if err != nil {
		_, _ = fmt.Fprintf(pipeline.stderr, "Warning: could not fetch profiler status: %v\n", err)
		return nil
	}
	if status == nil {
		return nil
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(pipeline.stderr, "Warning: could not encode profiler status: %v\n", err)
		return nil
	}
	if err := outputSandbox.WriteFile("profiler_status.json", data, profileFilePerms); err != nil {
		_, _ = fmt.Fprintf(pipeline.stderr, "Warning: could not write profiler status: %v\n", err)
		return nil
	}

	_, _ = fmt.Fprint(pipeline.stdout, "Profiler status: profiler_status.json\n")
	if status.RollingTrace.Enabled {
		_, _ = fmt.Fprintf(
			pipeline.stdout,
			"  Rolling trace enabled: min_age=%s max_bytes=%s\n",
			status.RollingTrace.MinAge,
			profileFormatBytes(safeconv.Uint64ToInt64(status.RollingTrace.MaxBytes)),
		)
	}
	_, _ = fmt.Fprint(pipeline.stdout, "\n")

	return status
}

// captureRollingTrace downloads the rolling trace snapshot from the
// profiler server and saves it to the output sandbox.
//
// Takes pipeline (pipelineConfig) which holds pipeline settings.
// Takes outputSandbox (safedisk.Sandbox) which writes output files.
// Takes status (*profiler.ServerStatus) which provides the download path; if
// nil or tracing is disabled, returns immediately.
func captureRollingTrace(
	ctx context.Context,
	pipeline pipelineConfig,
	outputSandbox safedisk.Sandbox,
	status *profiler.ServerStatus,
) {
	if status == nil || !status.RollingTrace.Enabled || status.RollingTrace.DownloadPath == "" {
		return
	}

	const rollingTraceDownloadTimeout = 30 * time.Second

	data, err := fetchProfilerBinary(ctx, pipeline.profilerRoot+status.RollingTrace.DownloadPath, rollingTraceDownloadTimeout)
	if err != nil {
		_, _ = fmt.Fprintf(pipeline.stderr, "Warning: rolling trace download failed: %v\n", err)
		return
	}
	if err := outputSandbox.WriteFile("rolling_trace.out", data, profileFilePerms); err != nil {
		_, _ = fmt.Fprintf(pipeline.stderr, "Warning: could not write rolling trace: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(
		pipeline.stdout,
		"Rolling trace snapshot: %s\n\n",
		filepath.Join(pipeline.flags.output, "rolling_trace.out"),
	)
}

// fetchProfilerStatus sends an HTTP GET to the profiler status
// endpoint and deserialises the JSON response.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes profilerRoot (string) which is the base URL of the profiler server.
//
// Returns *profiler.ServerStatus which is the decoded status, or
// nil when the endpoint returns 404.
// Returns error which wraps network or deserialisation failures.
func fetchProfilerStatus(ctx context.Context, profilerRoot string) (*profiler.ServerStatus, error) {
	reqCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
		errors.New("profiler status fetch exceeded 5s timeout"))
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, profilerRoot+profiler.ProfilerStatusPath, nil)
	if err != nil {
		return nil, fmt.Errorf("creating profiler status request: %w", err)
	}

	response, err := http.DefaultClient.Do(request) //nolint:gosec // local dev server
	if err != nil {
		return nil, fmt.Errorf("GET profiler status: %w", err)
	}
	defer drainAndClose(response.Body)

	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET profiler status returned status %d", response.StatusCode)
	}

	body, err := readAndDrainBody(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading profiler status body: %w", err)
	}

	var status profiler.ServerStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("decoding profiler status: %w", err)
	}

	return &status, nil
}

// fetchProfilerBinary performs an HTTP GET for binary profiler data
// such as trace snapshots.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes url (string) which is the full URL to fetch.
// Takes timeout (time.Duration) which bounds the request duration.
//
// Returns []byte which is the raw response body.
// Returns error which wraps network or read failures.
func fetchProfilerBinary(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	reqCtx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("profiler fetch exceeded %s timeout", timeout))
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", url, err)
	}

	response, err := http.DefaultClient.Do(request) //nolint:gosec // local dev server
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer drainAndClose(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned status %d", url, response.StatusCode)
	}

	data, err := readAndDrainBody(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	return data, nil
}

// capturePhase runs continuous load, fetches a pprof profile, saves
// it to disk, generates report sections, and snapshots goroutines.
//
// Takes pipeline (pipelineConfig) which holds pipeline settings.
// Takes reportFile (io.Writer) which receives the report output.
// Takes spec (profileSpec) which describes the profile to capture.
// Takes outputSandbox (safedisk.Sandbox) which writes output files.
// Takes errorCh (chan<- loadErrorRecord) which receives error records.
//
// Returns error when fetching or writing the profile fails.
//
// Spawns a goroutine that runs the load generator in the
// background while the profile is being captured.
func capturePhase(
	ctx context.Context,
	pipeline pipelineConfig,
	reportFile io.Writer,
	spec profileSpec,
	outputSandbox safedisk.Sandbox,
	errorCh chan<- loadErrorRecord,
) error {
	profilePath := filepath.Join(pipeline.flags.output, spec.name+".pprof")

	_, _ = fmt.Fprintf(pipeline.stdout, "Capturing %s profile (%ds under load)...\n", spec.name, pipeline.flags.duration)

	loadCtx, cancel := context.WithCancelCause(ctx)
	loadDone := make(chan *loadResult, 1)
	go func() {
		loadDone <- runLoad(loadCtx, pipeline.buildLoadConfig(spec.name, errorCh))
	}()

	time.Sleep(500 * time.Millisecond)

	data, fetchErr := fetchProfileData(ctx, spec, pipeline.pprofBase, pipeline.flags.duration)

	cancel(fmt.Errorf("profile capture for %s completed", spec.name))
	result := <-loadDone

	if fetchErr != nil {
		return fmt.Errorf("fetching %s: %w", spec.name, fetchErr)
	}

	if err := outputSandbox.WriteFile(spec.name+".pprof", data, profileFilePerms); err != nil {
		return fmt.Errorf("writing %s: %w", profilePath, err)
	}

	_ = writeProfileStats(outputSandbox, spec.name+".pprof.stats", result, pipeline.flags.concurrency)

	_, _ = fmt.Fprintf(pipeline.stdout, "  Saved %s (%s)\n", profilePath, profileFormatBytes(int64(len(data))))

	for _, reportConfig := range spec.reports {
		if err := generateProfileReport(reportFile, data, reportConfig, result.totalRequests); err != nil {
			_, _ = fmt.Fprintf(pipeline.stderr, "  Warning: report for %q failed: %v\n", reportConfig.sectionTitle, err)
		}
	}

	if spec.delta {
		if err := writeAllocChurnSummary(reportFile, data, result.totalRequests); err != nil {
			_, _ = fmt.Fprintf(pipeline.stderr, "  Warning: alloc churn summary failed: %v\n", err)
		}
	}

	snapshotGoroutines(ctx, pipeline.stdout, pipeline.stderr, pipeline.pprofBase, spec.name, outputSandbox)

	_, _ = fmt.Fprint(pipeline.stdout, "  Done.\n\n")
	return nil
}

// fetchProfile downloads a pprof profile from the given endpoint.
//
// When durationSecs > 0, it appends ?seconds=N to the URL.
//
// Takes pprofBase (string) which is the pprof base URL.
// Takes endpoint (string) which is the pprof endpoint path.
// Takes durationSecs (int) which is the capture duration in seconds.
//
// Returns []byte which is the raw profile data.
// Returns error when the HTTP request or response reading fails.
func fetchProfile(ctx context.Context, pprofBase, endpoint string, durationSecs int) ([]byte, error) {
	fetchURL := fmt.Sprintf("%s/%s", pprofBase, endpoint)
	if durationSecs > 0 {
		fetchURL = fmt.Sprintf("%s?seconds=%d", fetchURL, durationSecs)
	}

	reqCtx, cancel := context.WithTimeoutCause(ctx, time.Duration(durationSecs+30)*time.Second,
		fmt.Errorf("pprof %s fetch exceeded %ds timeout", endpoint, durationSecs+30))
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", fetchURL, err)
	}

	response, err := http.DefaultClient.Do(request) //nolint:gosec // local dev server
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", fetchURL, err)
	}
	defer drainAndClose(response.Body)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned status %d", fetchURL, response.StatusCode)
	}

	data, err := readAndDrainBody(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", fetchURL, err)
	}

	return data, nil
}

// fetchProfileData fetches profile data according to the spec's
// capture mode.
//
// When the spec is duration-based, it passes the seconds parameter.
// When the spec is delta-based, it takes before/after snapshots.
//
// Takes spec (profileSpec) which describes the capture mode.
// Takes pprofBase (string) which is the pprof base URL.
// Takes durationSecs (int) which is the capture window in seconds.
//
// Returns []byte which is the raw or delta profile data.
// Returns error when any fetch or delta computation fails.
func fetchProfileData(ctx context.Context, spec profileSpec, pprofBase string, durationSecs int) ([]byte, error) {
	if spec.durationBased {
		return fetchProfile(ctx, pprofBase, spec.endpoint, durationSecs)
	}

	if spec.delta {
		beforeData, err := fetchProfile(ctx, pprofBase, spec.endpoint, 0)
		if err != nil {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(durationSecs) * time.Second):
		}

		afterData, err := fetchProfile(ctx, pprofBase, spec.endpoint, 0)
		if err != nil {
			return nil, err
		}
		return computeDeltaProfile(beforeData, afterData)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(durationSecs) * time.Second):
	}
	return fetchProfile(ctx, pprofBase, spec.endpoint, 0)
}

// buildProfileSpecs returns the list of profiles to capture and their
// associated report configurations.
//
// Takes flags (*profileFlags) which provides topN and focus settings.
// Takes focusRegex (*regexp.Regexp) which optionally filters functions.
//
// Returns []profileSpec which contains the ordered profile specs.
func buildProfileSpecs(flags *profileFlags, focusRegex *regexp.Regexp) []profileSpec {
	return []profileSpec{
		buildCPUSpec(flags.topN, focusRegex),
		buildAllocsSpec(flags.topN, focusRegex),
		buildHeapSpec(flags.topN, focusRegex),
		buildContSpec("mutex", flags.topN, focusRegex),
		buildContSpec("block", flags.topN, focusRegex),
	}
}

// buildCPUSpec returns the CPU profile spec.
//
// Takes topN (int) which is the number of top entries to
// report.
// Takes focusRegex (*regexp.Regexp) which optionally filters
// function names.
//
// Returns profileSpec which describes CPU profile capture.
func buildCPUSpec(topN int, focusRegex *regexp.Regexp) profileSpec {
	return profileSpec{
		name:          "cpu",
		endpoint:      "profile",
		durationBased: true,
		reports: []profileReportConfig{
			{
				sectionTitle: "cpu",
				sampleIndex:  1,
				focusRegex:   focusRegex,
				topN:         topN,
			},
		},
	}
}

// buildAllocsSpec returns the allocation churn (delta)
// profile spec.
//
// Takes topN (int) which is the number of top entries to
// report.
// Takes focusRegex (*regexp.Regexp) which optionally filters
// function names.
//
// Returns profileSpec which describes allocation delta
// capture.
func buildAllocsSpec(topN int, focusRegex *regexp.Regexp) profileSpec {
	return profileSpec{
		name:     "allocs",
		endpoint: "heap",
		delta:    true,
		reports: []profileReportConfig{
			{
				sectionTitle: "allocs (alloc_space by line)",
				sampleIndex:  1,
				byLine:       true,
				focusRegex:   focusRegex,
				topN:         topN,
			},
			{
				sectionTitle: "allocs (alloc_objects by line)",
				sampleIndex:  0,
				byLine:       true,
				focusRegex:   focusRegex,
				topN:         topN,
			},
		},
	}
}

// buildHeapSpec returns the heap (in-use space) profile spec.
//
// Takes topN (int) which is the number of top entries to
// report.
// Takes focusRegex (*regexp.Regexp) which optionally filters
// function names.
//
// Returns profileSpec which describes heap snapshot capture.
func buildHeapSpec(topN int, focusRegex *regexp.Regexp) profileSpec {
	const heapInuseSpaceSampleIndex = 3
	return profileSpec{
		name:     "heap",
		endpoint: "heap",
		reports: []profileReportConfig{
			{
				sectionTitle: "heap (inuse_space)",
				sampleIndex:  heapInuseSpaceSampleIndex,
				focusRegex:   focusRegex,
				topN:         topN,
			},
		},
	}
}

// buildContSpec returns a duration-based contention profile
// spec (mutex or block).
//
// Takes name (string) which identifies the contention type.
// Takes topN (int) which is the number of top entries to
// report.
// Takes focusRegex (*regexp.Regexp) which optionally filters
// function names.
//
// Returns profileSpec which describes contention capture.
func buildContSpec(name string, topN int, focusRegex *regexp.Regexp) profileSpec {
	return profileSpec{
		name:          name,
		endpoint:      name,
		durationBased: true,
		reports: []profileReportConfig{
			{
				sectionTitle: name,
				sampleIndex:  1,
				focusRegex:   focusRegex,
				topN:         topN,
			},
		},
	}
}

// parseProfileFlags parses the command-line flags for the profile
// subcommand.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
// Takes stderr (io.Writer) which receives error and usage output.
//
// Returns *profileFlags which holds the parsed flag values.
// Returns string which is the target URL.
// Returns bool which is true when parsing succeeded.
func parseProfileFlags(arguments []string, stderr io.Writer) (*profileFlags, string, bool) {
	url, flagArgs := extractProfileURL(arguments)

	flags := &profileFlags{}

	fs := flag.NewFlagSet("profile", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.IntVar(&flags.pprofPort, "pprof-port", profileDefaultPprofPort, "Port where pprof endpoints are exposed")
	fs.IntVar(&flags.concurrency, "concurrency", profileDefaultConcurrency, "Number of concurrent HTTP connections")
	fs.IntVar(&flags.duration, "duration", profileDefaultDuration, "Phase duration in seconds (baseline and each profile)")
	fs.StringVar(&flags.output, "output", "pprof", "Output directory for .pprof files and report")
	fs.StringVar(&flags.cookie, "cookie", "", "Cookie header value to send with load requests")
	fs.Var(&flags.headers, "header", "HTTP header (Name: Value), repeatable")
	fs.IntVar(&flags.topN, "top", profileDefaultTopN, "Number of top entries per report section")
	fs.StringVar(&flags.focus, "focus", "", "Regex filter to focus on matching function names")
	fs.BoolVar(&flags.tui, "tui", false, "Enable live BubbleTea dashboard during profiling")

	fs.Usage = func() { profileUsage(stderr) }

	if err := fs.Parse(flagArgs); err != nil {
		return nil, "", false
	}

	if url == "" && fs.NArg() == 0 {
		_, _ = fmt.Fprint(stderr, "Error: URL to test is a required argument.\n\n")
		profileUsage(stderr)
		return nil, "", false
	}
	if url == "" {
		url = fs.Arg(0)
	}

	return flags, url, true
}

// extractProfileURL scans arguments for a URL and returns it separately
// from the remaining flag arguments.
//
// Takes arguments ([]string) which contains the raw command-line arguments.
//
// Returns string which is the extracted URL, or empty if none found.
// Returns []string which contains the remaining arguments.
func extractProfileURL(arguments []string) (string, []string) {
	for i, arg := range arguments {
		if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			remaining := make([]string, 0, len(arguments)-1)
			remaining = append(remaining, arguments[:i]...)
			remaining = append(remaining, arguments[i+1:]...)
			return arg, remaining
		}
	}
	return "", arguments
}

// profileUsage prints usage information for the profile subcommand.
//
// Takes w (io.Writer) which receives the usage text.
func profileUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `Usage: piko profile <url> [flags]

Profile a live running Piko server under load. Captures CPU, memory allocation,
heap, mutex, and blocking profiles while generating continuous HTTP load against
the specified URL.

Produces:
  - A summary text report with top offenders (including line-level allocation churn)
  - Raw .pprof files for deep-dive interactive analysis with 'go tool pprof'

Arguments:
  <url>                   The URL to load test (e.g. http://localhost:8080/)

Flags:
  --pprof-port <port>     Port where pprof endpoints are exposed (default: 6060)
  --concurrency <n>       Number of concurrent HTTP connections (default: 100)
  --duration <seconds>    Phase duration in seconds (baseline and each profile, default: 30)
  --output <directory>          Output directory for .pprof files and report (default: ./pprof)
  --header <Name: Value>  HTTP header to send with load requests (repeatable)
  --cookie <string>       Cookie header value (shorthand for --header "Cookie: ...")
  --top <n>               Number of top entries per report section (default: 60)
  --focus <pattern>       Regex filter to focus on matching function names
  --tui                   Enable live BubbleTea dashboard during profiling

Examples:
  piko profile http://localhost:8080/
  piko profile http://localhost:8080/ --pprof-port 6060 --concurrency 200
  piko profile http://localhost:8080/ --duration 60 --focus "render"
  piko profile http://localhost:8080/ --cookie "session_id=abc123"
  piko profile http://localhost:8080/ --header "Authorization: Bearer token" --header "X-Custom: val"
  piko profile http://localhost:8080/ --tui --duration 10
`)
}

// fetchGoroutineCount fetches the current goroutine count from a pprof
// endpoint by requesting /goroutine?debug=1 and parsing the "total N"
// from the first line.
//
// Takes pprofBase (string) which is the pprof endpoint base URL.
//
// Returns int which is the goroutine count, or 0 if the fetch fails.
func fetchGoroutineCount(ctx context.Context, pprofBase string) int {
	reqCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second,
		errors.New("pprof goroutine count fetch exceeded 5s timeout"))
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pprofBase+"/goroutine?debug=1", nil)
	if err != nil {
		return 0
	}

	response, err := http.DefaultClient.Do(request) //nolint:gosec // local dev server
	if err != nil {
		return 0
	}
	defer drainAndClose(response.Body)

	if response.StatusCode != http.StatusOK {
		return 0
	}

	buffer := make([]byte, profileGoroutineReadBufSize)
	n, _ := response.Body.Read(buffer)
	firstLine, _, _ := strings.Cut(string(buffer[:n]), "\n")

	var count int
	_, _ = fmt.Sscanf(firstLine, "goroutine profile: total %d", &count)
	return count
}

// snapshotGoroutines fetches the full goroutine stack dump from pprof
// (debug=2 gives full stacks with state) and writes it to disk alongside
// the profile files. This lets you diff goroutines between phases to
// identify leaks.
//
// Takes stdout (io.Writer) which receives progress messages.
// Takes stderr (io.Writer) which receives warning messages.
// Takes pprofBase (string) which is the pprof endpoint base URL.
// Takes phase (string) which names the current profiling phase.
// Takes sandbox (safedisk.Sandbox) which writes the output file.
func snapshotGoroutines(ctx context.Context, stdout, stderr io.Writer, pprofBase, phase string, sandbox safedisk.Sandbox) {
	count := fetchGoroutineCount(ctx, pprofBase)

	reqCtx, cancel := context.WithTimeoutCause(ctx, 30*time.Second,
		errors.New("pprof goroutine snapshot exceeded 30s timeout"))
	defer cancel()

	request, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pprofBase+"/goroutine?debug=2", nil)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  Warning: could not snapshot goroutines: %v\n", err)
		return
	}

	response, err := http.DefaultClient.Do(request) //nolint:gosec // local dev server
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  Warning: could not snapshot goroutines: %v\n", err)
		return
	}
	defer drainAndClose(response.Body)

	if response.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(stderr, "  Warning: goroutine snapshot returned %d\n", response.StatusCode)
		return
	}

	body, err := readAndDrainBody(response.Body)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  Warning: could not read goroutine snapshot: %v\n", err)
		return
	}

	filename := phase + ".goroutines.txt"
	if err := sandbox.WriteFile(filename, body, profileFilePerms); err != nil {
		_, _ = fmt.Fprintf(stderr, "  Warning: could not write goroutine snapshot: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(stdout, "  Goroutine snapshot: %d goroutines -> %s\n", count, filename)
}
