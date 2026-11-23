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

package campaign

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// defaultPhaseTimeout bounds one mode's execution phase.
	defaultPhaseTimeout = 5 * time.Minute

	// kindHarness classifies an infrastructure failure of the harness itself.
	kindHarness = "harness"

	// kindResource classifies a stop caused by a resource limit.
	kindResource = "resource"

	// kindInterrupted classifies a stop caused by cancellation.
	kindInterrupted = "interrupted"

	// packageCheck names the single check of a non-application target.
	packageCheck = "package"

	// statusPending marks a check that has not started.
	statusPending = "pending"

	// StatusFailed marks a failed check or a native run that found a failure.
	StatusFailed = "failed"

	// compiledPhaseBytes is the admitted memory estimate of a compiled or browser phase.
	compiledPhaseBytes = 1 << 30

	// interpretedPhaseBytes is the admitted memory estimate of an interpreted phase.
	interpretedPhaseBytes = 512 << 20

	// packagePhaseBytes is the admitted memory estimate of any other phase.
	packagePhaseBytes = 256 << 20

	// maxPhaseLabelBytes bounds an error prefix accepted as a phase label.
	maxPhaseLabelBytes = 100

	// browserSampleDecade is the number of consecutive cases sharing one browser sample.
	browserSampleDecade = 10

	// browserSampleVariants is the number of index variants the browser sample rotates
	// through.
	browserSampleVariants = 4

	// browserSampleStride advances the sampled variant between consecutive decades.
	browserSampleStride = 3
)

var (
	// errPhaseTimeout marks a mode phase that exceeded its deadline.
	errPhaseTimeout = errors.New("phase timeout: mode exceeded its time limit")

	// errDuration marks the campaign's elapsed-time limit.
	errDuration = errors.New("campaign duration reached")

	// errIncompleteBrowser reports a selected browser that returned no complete observation.
	errIncompleteBrowser = errors.New("infrastructure: selected browser returned no complete interaction observation")

	// browserTargets lists application targets that always select a browser.
	browserTargets = []string{
		"components", "form-workflow", "browser-events", "browser-lifecycle", "partial-components", "checkout-workflow",
		"shipped-components",
	}

	// recordedWorkers lists the private worker binaries whose provenance is recorded.
	recordedWorkers = []string{
		"generate-worker", "interp-worker", "interp-worker-covered", "form-sql-worker", "browser-worker", "target-worker",
		"target-worker-safe",
	}
)

// Harness runs cases under the campaign's resource policy.
type Harness struct {
	// Output receives campaign summaries and native-run headers, defaulting to stdout.
	Output io.Writer

	// Progress receives case progress and diagnostics, defaulting to stderr.
	Progress io.Writer

	// Budget admits and monitors work against measured resource limits.
	Budget interface {
		Admit(context.Context, uint64) error
		Sample(context.Context) (resource.Sample, error)
		SampleNow(context.Context) (resource.Sample, error)
	}

	// coverage stores completed runtime measurements.
	coverage *runtimecoverage.Store

	// recordHostModules records modules linked into a generated application host.
	recordHostModules func(string) error

	// seenModes tracks modes already run during this invocation.
	seenModes map[string]bool

	// trial overrides case execution for controlled tests.
	trial func(context.Context, *casefile.Case, string, string) Result

	// Repo is the repository source directory.
	Repo string

	// State is the campaign state directory.
	State string

	// WorkersDir is the private worker directory.
	WorkersDir string

	// phaseTimeout overrides defaultPhaseTimeout when positive.
	phaseTimeout time.Duration
}

// subjectError is a failure that identifies its failed property.
type subjectError interface {
	error

	// FailureSubject identifies the failed property independently of generated values.
	//
	// Returns string which is the stable property identifier.
	FailureSubject() string
}

// phaseRequest describes one mode phase executed under admission and measurement.
type phaseRequest struct {
	// action performs the phase in its private directory.
	action func(context.Context, string) error

	// scratch is the trial's private scratch directory.
	scratch string

	// name identifies the mode or check.
	name string

	// estimate is the admitted incremental memory estimate in bytes.
	estimate uint64
}

// caseTrial holds one case attempt's harness, case and accumulating result.
type caseTrial struct {
	// c holds the attempted case.
	c *casefile.Case

	// r receives the attempt's checks, observations and failures.
	r *Result

	// scratch is the attempt's private scratch directory.
	scratch string

	// browser selects the browser-sampling policy.
	browser string

	// harness runs the attempt's phases.
	harness Harness
}

// campaignRun holds the state of one Execute invocation.
type campaignRun struct {
	// harness runs the campaign's cases.
	harness Harness

	// started is the invocation's start time.
	started time.Time

	// r holds the campaign checkpoint.
	r *Report

	// workersRecorded tracks worker binaries whose provenance is recorded.
	workersRecorded map[string]bool

	// dir is the campaign checkpoint and evidence directory.
	dir string

	// sample is the invocation's starting resource sample.
	sample resource.Sample

	// previousCPU holds the CPU accounting committed by earlier invocations.
	previousCPU CampaignCPU

	// previous is the elapsed time committed by earlier invocations.
	previous time.Duration

	// startCPU is the cgroup CPU use at the invocation's start in microseconds.
	startCPU uint64

	// cpuOK reports whether the starting CPU use was readable.
	cpuOK bool
}

// WithRuntimeCoverage opens a runtime-coverage store for this invocation.
//
// Takes root (string) which is the runtime-coverage store directory.
//
// Returns Harness which is a copy configured with the runtime-coverage store on success.
// Returns error which reports an invalid disk budget or inaccessible coverage store, or
// nil on success.
func (h Harness) WithRuntimeCoverage(root string) (Harness, error) {
	disk, err := strconv.ParseUint(os.Getenv(diskBudgetVariable), 10, 64)
	if err != nil {
		return h, fmt.Errorf("runtime coverage needs a verified disk budget: %w", err)
	}
	h.coverage, err = runtimecoverage.Open(root, disk)
	return h, err
}

// RunOne runs fresh workers and checks each mode before comparing them.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes mode (string) which selects the server mode.
// Takes browser (string) which selects browser execution.
//
// Returns Result which holds the classified case outcome and independent failures.
func (h Harness) RunOne(ctx context.Context, c *casefile.Case, mode, browser string) (r Result) {
	if h.trial != nil {
		return h.trial(ctx, c, mode, browser)
	}
	start := time.Now()
	defer func() { r.Elapsed = time.Since(start) }()
	r.Target = c.Target
	r.Index = c.Index
	r.Hash = c.Hash()
	r.Kind = checkStatusPass
	r.Checks = map[string]CheckMetric{}
	r.Observations = map[string]runner.Observation{}
	scratch, err := os.MkdirTemp("", "piko-fuzz-trial-")
	if err != nil {
		return failure(kindHarness, "", err, r)
	}
	defer func() {
		if removeErr := os.RemoveAll(scratch); removeErr != nil {
			h.progressf("removing trial scratch %s: %v\n", scratch, removeErr)
		}
	}()
	trial := &caseTrial{c: c, r: &r, harness: h, scratch: scratch, browser: browser}
	if !targets.IsApplication(c.Target) {
		return trial.runPackage(ctx)
	}
	return trial.runModes(ctx, selectedModes(c, mode))
}

// Execute continues the campaign from its first uncommitted case.
//
// Elapsed includes case generation, admission, both workers and confirmation. Time
// between invocations is excluded.
//
// Takes dir (string) which is the campaign checkpoint and evidence directory.
// Takes r (*Report) which holds the saved receipt or campaign report.
//
// Returns error which reports campaign setup, checkpoint, resource-limit or stopping
// failure, or nil on success.
func (h Harness) Execute(ctx context.Context, dir string, r *Report) error {
	if err := r.validateOptions(); err != nil {
		return err
	}
	if r.Next == uint64(len(r.Options.Targets))*r.Options.Cases && r.Active == "" &&
		(r.Status == "complete" || r.Status == "complete-with-findings") {
		return completionError(r)
	}
	active, err := loadActive(dir, r)
	if err != nil {
		return err
	}
	run, err := h.startRun(ctx, dir, r)
	if err != nil {
		return err
	}
	if r.Options.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, r.Options.Duration-run.previous, errDuration)
		defer cancel()
	}
	for r.Next/uint64(len(r.Options.Targets)) < r.Options.Cases {
		finished, err := run.step(ctx, active)
		active = nil
		if finished {
			return err
		}
	}
	r.Status = "complete"
	if len(r.Metrics.ConfirmedFingerprints) > 0 {
		r.Status = "complete-with-findings"
	}
	if err := run.persist(); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(run.harness.output(), r.Metrics.Summary()); err != nil {
		return fmt.Errorf("writing campaign summary: %w", err)
	}
	return completionError(r)
}

// startRun records the invocation's starting time, retained evidence and CPU use.
//
// Takes dir (string) which is the campaign checkpoint and evidence directory.
// Takes r (*Report) which holds the campaign checkpoint.
//
// Returns *campaignRun which holds the invocation state.
// Returns error which reports an invalid soak disk budget, unreadable resources or an
// inaccessible runtime-coverage store, or nil on success.
func (h Harness) startRun(ctx context.Context, dir string, r *Report) (*campaignRun, error) {
	run := &campaignRun{
		r:               r,
		workersRecorded: map[string]bool{},
		harness:         h,
		dir:             dir,
		sample:          resource.Sample{},
		started:         time.Now(),
		previousCPU:     r.Metrics.InvocationCPU,
		previous:        r.Elapsed,
		startCPU:        0,
		cpuOK:           false,
	}
	if err := r.Retained.start(dir, r.Options.Suite); err != nil {
		return nil, err
	}
	sample, err := h.Budget.SampleNow(ctx)
	if err != nil {
		return nil, err
	}
	run.sample = sample
	run.startCPU, run.cpuOK = resource.CPUUsage(sample.Cgroup)
	if r.Options.RuntimeCoverage {
		run.harness, err = run.harness.WithRuntimeCoverage(filepath.Join(dir, "runtime-coverage"))
		if err != nil {
			return nil, err
		}
	}
	run.harness.recordHostModules = r.Provenance.recordWorker
	run.harness.seenModes = map[string]bool{}
	return run, nil
}

// saveFailure cold-confirms and saves a case failure.
//
// Takes dir (string) which is the campaign directory receiving the finding bundle.
// Takes o (Options) which holds the saved campaign options.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes r (*Result) which receives the case execution result.
//
// Returns error which reports confirmation or saved-evidence publication failure, or nil
// on success.
func (h Harness) saveFailure(ctx context.Context, dir string, o Options, c *casefile.Case, r *Result) error {
	bundle := failurePath(dir, c, r)
	_, err := os.Stat(filepath.Join(bundle, "case.json"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := casefile.Write(bundle, c); err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("inspecting failure bundle %s: %w", bundle, err)
	}
	if err := saveJSON(filepath.Join(bundle, "options.json"), o); err != nil {
		return err
	}
	if err := saveJSON(filepath.Join(bundle, "original.json"), r); err != nil {
		return err
	}
	if confirmable(*r) && ctx.Err() == nil {
		again := h.RunOne(ctx, c, o.Mode, o.Browser)
		if err := saveJSON(filepath.Join(bundle, "confirmation.json"), again); err != nil {
			return err
		}
		if again.Kind == kindResource || again.Kind == kindInterrupted || again.Kind == kindHarness {
			r.Kind = again.Kind
			r.Message = "confirmation: " + again.Message
		} else if !r.matches(again) {
			r.Kind = "flaky"
		} else {
			r.Confirmed = true
		}
	}
	return saveJSON(filepath.Join(bundle, "result.json"), r)
}

// runPhase admits, measures and bounds one mode phase.
//
// The phase runs under a defaultPhaseTimeout deadline whose cause is errPhaseTimeout, so
// a hang is reported as a confirmable timeout rather than a resource stop.
//
// Takes request (phaseRequest) which describes the phase to run.
// Takes r (*Result) which receives the phase's check metrics.
//
// Returns error which reports admission, measurement, cgroup OOM, timeout or action
// failure, or nil on success.
func (h Harness) runPhase(ctx context.Context, request phaseRequest, r *Result) error {
	check := r.Checks[request.name]
	admissionStart := time.Now()
	err := h.Budget.Admit(ctx, request.estimate)
	check.Admission = AdmissionMetric{EstimatedBytes: request.estimate, Elapsed: time.Since(admissionStart), Granted: err == nil}
	r.Checks[request.name] = check
	if err != nil {
		return err
	}
	before, err := h.Budget.Sample(ctx)
	if err != nil {
		return err
	}
	directory := filepath.Join(request.scratch, request.name)
	if err := os.Mkdir(directory, directoryPermissions); err != nil {
		return fmt.Errorf("creating phase directory: %w", err)
	}
	temporary := filepath.Join(directory, "tmp")
	if err := os.Mkdir(temporary, directoryPermissions); err != nil {
		return fmt.Errorf("creating phase temporary directory: %w", err)
	}
	phase, cancel := context.WithTimeoutCause(process.WithTempDir(ctx, temporary), cmp.Or(h.phaseTimeout, defaultPhaseTimeout), errPhaseTimeout)
	defer cancel()
	check.FirstInInvocation = !h.seenModes[request.name]
	if h.seenModes != nil {
		h.seenModes[request.name] = true
	}
	check.Status = "running"
	r.Checks[request.name] = check
	phaseStart := time.Now()
	check.Memory, check.Cost, err = resource.MeasureCheck(before.Cgroup, directory, func() error { return request.action(phase, directory) })
	check.Elapsed = time.Since(phaseStart)
	check.Status = checkStatusPass
	if err != nil {
		check.Status = StatusFailed
	}
	r.Checks[request.name] = check
	if err != nil {
		after, readErr := h.Budget.SampleNow(context.WithoutCancel(ctx))
		if readErr != nil {
			return fmt.Errorf("resource measurement: %w", readErr)
		}
		if after.OOMSince(before) {
			return fmt.Errorf("%w: %s cgroup OOM", resource.ErrLimit, request.name)
		}
	}
	if phase.Err() != nil {
		return context.Cause(phase)
	}
	return err
}

// generateCase builds a case and converts a generator panic into an error.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which is the case's stream position.
// Takes size (string) which selects the structural generation bounds.
//
// Returns casefile.Case which holds the generated case.
// Returns error which names the target, seed and index of a failed or panicking
// generation, or nil on success.
func (h Harness) generateCase(target string, seed, index uint64, size string) (c casefile.Case, err error) {
	defer func() { h.recoverGenerator(recover(), &err, target, seed, index) }()
	c, err = generate.Build(target, seed, index, size)
	if err != nil {
		return c, fmt.Errorf("generating %s seed %d index %d: %w", target, seed, index, err)
	}
	return c, nil
}

// replayCase rebuilds a case from a recorded tape and converts a generator panic into an
// error.
//
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which is the case's stream position.
// Takes data ([]byte) which holds the recorded choice tape.
// Takes size (string) which selects the structural generation bounds.
//
// Returns casefile.Case which holds the regenerated case.
// Returns error which names the target, seed and index of a failed or panicking replay,
// or nil on success.
func (h Harness) replayCase(target string, seed, index uint64, data []byte, size string) (c casefile.Case, err error) {
	defer func() { h.recoverGenerator(recover(), &err, target, seed, index) }()
	c, err = generate.BuildReplay(target, seed, index, data, size)
	if err != nil {
		return c, fmt.Errorf("replaying %s seed %d index %d: %w", target, seed, index, err)
	}
	return c, nil
}

// recoverGenerator converts a recovered generator panic into an error.
//
// The stack is written once to the progress writer and never placed in the error.
//
// Takes recovered (any) which is the value recover returned in the deferred call, or nil.
// Takes err (*error) which receives the converted panic.
// Takes target (string) which selects the subsystem under test.
// Takes seed (uint64) which initialises the deterministic choice stream.
// Takes index (uint64) which is the case's stream position.
func (h Harness) recoverGenerator(recovered any, err *error, target string, seed, index uint64) {
	if recovered == nil {
		return
	}
	h.progressf("generator panic for %s seed %d index %d: %v\n%s", target, seed, index, recovered, debug.Stack())
	*err = fmt.Errorf("generating %s seed %d index %d: panic: %v", target, seed, index, recovered)
}

// output selects the writer for summaries and run headers.
//
// Returns io.Writer which is the configured output, or standard output.
func (h Harness) output() io.Writer {
	if h.Output != nil {
		return h.Output
	}
	return os.Stdout
}

// progressf writes one progress or diagnostic line.
//
// A failed progress write leaves the case outcome unchanged.
//
// Takes format (string) which is the line's format string.
// Takes arguments (...any) which supplies the format operands.
func (h Harness) progressf(format string, arguments ...any) {
	writer := h.Progress
	if writer == nil {
		writer = os.Stderr
	}
	_, _ = fmt.Fprintf(writer, format, arguments...)
}

// run admits, measures and bounds one phase of the attempt.
//
// Takes name (string) which identifies the mode or check.
// Takes action (func(context.Context, string) error) which performs the phase in its
// private directory.
//
// Returns error which reports admission, measurement, timeout or action failure, or nil
// on success.
func (t *caseTrial) run(ctx context.Context, name string, action func(context.Context, string) error) error {
	request := phaseRequest{action: action, scratch: t.scratch, name: name, estimate: phaseEstimate(t.c, name, t.browser)}
	return t.harness.runPhase(ctx, request, t.r)
}

// runPackage runs a non-application target's single package check.
//
// Returns Result which holds the classified package outcome.
func (t *caseTrial) runPackage(ctx context.Context) Result {
	c := t.c
	t.r.Checks[packageCheck] = pendingCheck(c.Target == "example-contracts", nil)
	var mutations []runner.MutationCheck
	var receipt publiccontracts.Receipt
	err := t.run(ctx, packageCheck, func(ctx context.Context, dir string) error {
		if c.Target == "oracle-sensitivity" {
			var cause error
			mutations, cause = runner.MeasureSensitivity(c)
			return cause
		}
		if targets.IsPublicContract(c.Target) {
			var err error
			receipt, err = targets.RunPublicContract(ctx, t.harness.Repo, dir, c)
			return err
		}
		return targets.RunWithWorkers(ctx, t.harness.Repo, dir, t.harness.WorkersDir, c)
	})
	if receipt.Contract != "" {
		check := t.r.Checks[packageCheck]
		check.PublicContract = &receipt
		check.BrowserObserved = receipt.BrowserVisited
		t.r.Checks[packageCheck] = check
	}
	if len(mutations) > 0 {
		check := t.r.Checks[packageCheck]
		check.Mutations = mutations
		t.r.Checks[packageCheck] = check
	}
	if err != nil {
		return classify(ctx, "target", packageCheck, err, *t.r)
	}
	return *t.r
}

// runModes runs and checks each selected server mode, then compares their observations.
//
// Takes selected ([]string) which lists the server modes in execution order.
//
// Returns Result which holds the classified case outcome and independent failures.
func (t *caseTrial) runModes(ctx context.Context, selected []string) Result {
	r := t.r
	if err := t.planModes(selected); err != nil {
		return *r
	}
	var first runner.Observation
	for position, m := range selected {
		o, verified, stopped := t.runMode(ctx, m)
		if stopped {
			return *r
		}
		if !verified {
			continue
		}
		if position == 0 {
			first = o
			continue
		}
		if r.Kind == checkStatusPass {
			if err := runner.CompareCase(t.c, &first, &o); err != nil {
				return failure("disagreement", "both", err, *r)
			}
		}
	}
	return *r
}

// planModes records a pending check with its planned browser routes for each mode.
//
// Takes selected ([]string) which lists the server modes in execution order.
//
// Returns error which reports an invalid saved route plan after recording it as a harness
// failure, or nil on success.
func (t *caseTrial) planModes(selected []string) error {
	for _, m := range selected {
		selectedBrowser := BrowserCase(t.c, t.browser)
		routes, err := plannedBrowserRoutes(t.c, selectedBrowser)
		if err != nil {
			*t.r = failure(kindHarness, m, err, *t.r)
			return err
		}
		t.r.Checks[m] = pendingCheck(selectedBrowser, routes)
	}
	return nil
}

// runMode runs one server mode and checks its observation independently.
//
// Takes m (string) which selects the server mode.
//
// Returns runner.Observation which holds the worker observation.
// Returns bool which reports whether the mode passed its independent checks.
// Returns bool which reports whether a resource, interruption or harness failure ends the
// attempt.
func (t *caseTrial) runMode(ctx context.Context, m string) (o runner.Observation, verified, stopped bool) {
	r := t.r
	err := t.run(ctx, m, func(ctx context.Context, dir string) error {
		var runErr error
		o, runErr = t.observe(ctx, dir, m)
		return runErr
	})
	r.Observations[m] = o
	check := r.Checks[m]
	err = check.recordObservation(t.c, &o, err)
	r.Checks[m] = check
	if err != nil {
		*r = classify(ctx, "worker", m, err, *r)
		return o, false, r.Kind == kindResource || r.Kind == kindInterrupted || r.Kind == kindHarness
	}
	if issues := runner.IndependentFailures(t.c, &o); len(issues) > 0 {
		check.Status = StatusFailed
		r.Checks[m] = check
		for _, issue := range issues {
			*r = failure("expectation", m, issue, *r)
		}
		return o, false, false
	}
	check.ProgramVerified = true
	r.Checks[m] = check
	return o, true, false
}

// observe runs one server mode's worker and records its host provenance.
//
// Takes dir (string) which is the phase's private directory.
// Takes m (string) which selects the server mode.
//
// Returns runner.Observation which holds the worker observation.
// Returns error which reports a worker, coverage or provenance failure, or nil on
// success.
func (t *caseTrial) observe(ctx context.Context, dir, m string) (runner.Observation, error) {
	c := t.c
	selected := BrowserCase(c, t.browser)
	var o runner.Observation
	var runErr error
	if t.harness.coverage == nil {
		o, runErr = runner.RunWithWorkers(ctx, t.harness.Repo, dir, t.harness.WorkersDir, c, m, selected)
	} else {
		if err := t.harness.coverage.Prepare(); err != nil {
			return o, err
		}
		temporary := filepath.Join(dir, "coverage")
		o, runErr = runner.RunMeasured(ctx, &runner.MeasuredRun{
			Case:    c,
			Repo:    t.harness.Repo,
			Work:    dir,
			Workers: t.harness.WorkersDir,
			Mode:    m,
			Output:  temporary,
			Browser: selected,
		})
		if runErr == nil {
			if err := t.publishCoverage(ctx, temporary, m, selected); err != nil {
				return o, err
			}
		}
	}
	if err := t.recordHost(dir, m, runErr); err != nil {
		return o, err
	}
	return o, runErr
}

// recordHost records the modules linked into the mode's generated application host.
//
// Takes dir (string) which is the phase's private directory.
// Takes m (string) which selects the server mode.
// Takes runErr (error) which is the worker failure, or nil.
//
// Returns error which reports an unreadable host binary, or nil when it is recorded, not
// required, or absent after a worker failure.
func (t *caseTrial) recordHost(dir, m string, runErr error) error {
	if t.harness.recordHostModules == nil || (m != modeCompiled && !t.c.RequiresProjectInterpreterHost()) {
		return nil
	}
	name := "native"
	if m == modeInterp {
		name = "interp-app"
	}
	receiptErr := t.harness.recordHostModules(filepath.Join(dir, name))
	if receiptErr != nil && (runErr == nil || !errors.Is(receiptErr, fs.ErrNotExist)) {
		return fmt.Errorf("infrastructure: reading %s host provenance: %w", m, receiptErr)
	}
	return nil
}

// publishCoverage publishes one mode's runtime-coverage measurement.
//
// Takes temporary (string) which is the measurement's output directory.
// Takes m (string) which selects the server mode.
// Takes selected (bool) which reports whether the case selects a browser.
//
// Returns error which reports a coverage store failure, or nil on success.
func (t *caseTrial) publishCoverage(ctx context.Context, temporary, m string, selected bool) error {
	receipt, err := t.harness.coverage.Publish(ctx, temporary, runtimecoverage.Receipt{
		GoScopes:            nil,
		JavaScriptFunctions: nil,
		Artefact:            "",
		GoBasis:             "",
		Target:              t.c.Target,
		Mode:                m,
		Hash:                t.r.Hash,
		GoStatements:        runtimecoverage.Counts{},
		Index:               t.c.Index,
		BrowserSelected:     selected,
	})
	if err != nil {
		return fmt.Errorf("infrastructure: runtime coverage: %w", err)
	}
	if t.r.RuntimeCoverage == nil {
		t.r.RuntimeCoverage = map[string]runtimecoverage.Receipt{}
	}
	t.r.RuntimeCoverage[m] = receipt
	return nil
}

// persist commits the invocation's CPU, retained evidence and elapsed time to the
// checkpoint.
//
// Returns error which reports checkpoint publication failure, or nil on success.
func (run *campaignRun) persist() error {
	r := run.r
	total := run.previousCPU
	if used, ok := resource.CPUUsage(run.sample.Cgroup); ok && run.cpuOK && used >= run.startCPU {
		total.Usec += used - run.startCPU
		total.MeasuredInvocations++
	} else {
		total.UnmeasuredInvocations++
	}
	r.Metrics.InvocationCPU = total
	if run.harness.coverage != nil {
		r.Retained.CoverageBytes = run.harness.coverage.RetainedBytes()
	}
	r.Retained.observe()
	r.Elapsed = run.previous + time.Since(run.started)
	return checkpoint(run.dir, r)
}

// stop records why the campaign paused and persists the checkpoint.
//
// Takes cause (error) which is the stopping cause.
//
// Returns error which is the stopping cause, the completion error at the time limit, or a
// checkpoint failure.
func (run *campaignRun) stop(cause error) error {
	r := run.r
	r.Status = "paused"
	r.LastStop = cause.Error()
	if errors.Is(cause, errDuration) {
		r.Status = "time-limit"
	}
	if errors.Is(cause, resource.ErrLimit) {
		r.Status = kindResource
	}
	if err := run.persist(); err != nil {
		return err
	}
	if r.Status == "time-limit" {
		return completionError(r)
	}
	return cause
}

// step attempts and commits the next stream case.
//
// Takes active (*casefile.Case) which is the saved active case to resume, or nil to
// generate the next case.
//
// Returns bool which reports whether Execute must stop and return the error.
// Returns error which is the stopping error, or nil.
func (run *campaignRun) step(ctx context.Context, active *casefile.Case) (bool, error) {
	r := run.r
	if growth := r.Retained.growthStop(); growth != nil {
		return true, run.stop(growth)
	}
	if cause := context.Cause(ctx); cause != nil {
		return true, run.stop(cause)
	}
	position := r.Next / uint64(len(r.Options.Targets))
	target := r.Options.Targets[r.Next%uint64(len(r.Options.Targets))]
	activeDir := filepath.Join(run.dir, "active")
	c, err := run.nextCase(active, target, position, activeDir)
	if err != nil {
		return true, err
	}
	r.Active = c.Hash()
	r.Status = "running"
	r.LastStop = ""
	if err := run.persist(); err != nil {
		return true, err
	}
	obs := run.harness.RunOne(ctx, &c, r.Options.Mode, r.Options.Browser)
	if err := run.recordWorkers(); err != nil {
		return true, err
	}
	if finished, err := run.settle(ctx, &c, &obs, target, position); finished {
		return true, err
	}
	if err := run.commit(ctx, &c, &obs, activeDir); err != nil {
		return true, err
	}
	if obs.Kind != checkStatusPass && (!r.Options.KeepGoing || !obs.Confirmed) {
		r.Status = obs.Kind
		if err := run.persist(); err != nil {
			return true, err
		}
		return true, fmt.Errorf("%s[%d] %s in %s: %s", target, position, obs.Kind, obs.Mode, obs.Message)
	}
	return false, nil
}

// settle retains a failing case and stops the campaign on an incomplete outcome.
//
// Takes c (*casefile.Case) which holds the attempted case.
// Takes obs (*Result) which holds the attempt's result.
// Takes target (string) which names the case's target.
// Takes position (uint64) which is the case's stream position.
//
// Returns bool which reports whether Execute must stop and return the error.
// Returns error which is the stopping error, or nil.
func (run *campaignRun) settle(ctx context.Context, c *casefile.Case, obs *Result, target string, position uint64) (bool, error) {
	if obs.Kind == checkStatusPass {
		return false, nil
	}
	if obs.Kind != kindInterrupted {
		if err := run.retainFailure(ctx, c, obs, target, position); err != nil {
			return true, err
		}
	}
	return run.stopIncomplete(ctx, c, obs, target, position)
}

// nextCase resumes the saved active case or generates and publishes the next one.
//
// Takes active (*casefile.Case) which is the saved active case, or nil.
// Takes target (string) which selects the subsystem under test.
// Takes position (uint64) which is the case's stream position for its target.
// Takes activeDir (string) which is the active case's saved directory.
//
// Returns casefile.Case which holds the case to attempt.
// Returns error which reports a generation or publication failure, or nil on success.
func (run *campaignRun) nextCase(active *casefile.Case, target string, position uint64, activeDir string) (casefile.Case, error) {
	if active != nil {
		return *active, nil
	}
	options := &run.r.Options
	seed, streamIndex := StreamPosition(options.Suite, options.Seed, position)
	c, err := run.harness.generateCase(target, seed, streamIndex, options.Size)
	if err != nil {
		return c, err
	}

	if err := os.RemoveAll(activeDir); err != nil {
		return c, err
	}
	if err := casefile.Write(activeDir, &c); err != nil {
		return c, err
	}
	return c, nil
}

// recordWorkers records the provenance of each private worker binary once.
//
// Returns error which reports an unreadable worker binary, or nil on success.
func (run *campaignRun) recordWorkers() error {
	if run.harness.WorkersDir == "" {
		return nil
	}
	for _, worker := range recordedWorkers {
		if run.workersRecorded[worker] {
			continue
		}
		workerPath := filepath.Join(run.harness.WorkersDir, worker)
		if _, err := os.Stat(workerPath); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := run.r.Provenance.recordWorker(workerPath); err != nil {
			return err
		}
		run.workersRecorded[worker] = true
	}
	return nil
}

// retainFailure saves a failing case's bundle and updates retained-byte accounting.
//
// Takes c (*casefile.Case) which holds the failing case.
// Takes obs (*Result) which holds the failing result, updated by confirmation.
// Takes target (string) which names the case's target for progress output.
// Takes position (uint64) which is the case's stream position for progress output.
//
// Returns error which reports confirmation or bundle publication failure, or nil on
// success.
func (run *campaignRun) retainFailure(ctx context.Context, c *casefile.Case, obs *Result, target string, position uint64) error {
	bundle := failurePath(run.dir, c, obs)
	previousBytes, known := evidenceBytes(bundle)
	if err := run.harness.saveFailure(ctx, run.dir, run.r.Options, c, obs); err != nil {
		return err
	}
	run.r.Retained.updateBundle(bundle, previousBytes, known)
	run.harness.progressf("%s[%d]: %s (%s)\n", target, position, obs.Kind, obs.Phase)
	return nil
}

// stopIncomplete stops the campaign after an interrupted, resource or harness outcome.
//
// Takes c (*casefile.Case) which holds the attempted case.
// Takes obs (*Result) which holds the non-passing result.
// Takes target (string) which names the case's target.
// Takes position (uint64) which is the case's stream position.
//
// Returns bool which reports whether Execute must stop and return the error.
// Returns error which is the stopping error, or nil.
func (run *campaignRun) stopIncomplete(ctx context.Context, c *casefile.Case, obs *Result, target string, position uint64) (bool, error) {
	r := run.r
	switch obs.Kind {
	case kindInterrupted, kindResource, kindHarness:
		if r.IncompleteChecks == nil {
			r.IncompleteChecks = map[string]ModeMetrics{}
		}
		for name := range obs.Checks {
			check := obs.Checks[name]
			observation := obs.Observations[name]
			metrics := r.IncompleteChecks[name]
			metrics.observe(name, &check)
			metrics.observeCaseObligations(c, name, &check, &observation)
			r.IncompleteChecks[name] = metrics
		}
		if r.IncompleteAttempts == nil {
			r.IncompleteAttempts = map[string]uint64{}
		}
		r.IncompleteAttempts[obs.Kind]++
		if cause := context.Cause(ctx); cause != nil {
			return true, run.stop(cause)
		}
		r.Status = obs.Kind
		r.LastStop = obs.Message
		if err := run.persist(); err != nil {
			return true, err
		}
		return true, fmt.Errorf("%s[%d] %s: %s", target, position, obs.Kind, obs.Message)
	}
	if cause := context.Cause(ctx); cause != nil {
		return true, run.stop(cause)
	}
	return false, nil
}

// commit records a finished case and advances the stream position.
//
// Counts and stream position are committed together, only after all observations and
// optional confirmation have been written.
//
// Takes c (*casefile.Case) which holds the finished case.
// Takes obs (*Result) which holds the case's result.
// Takes activeDir (string) which is the active case's saved directory.
//
// Returns error which reports a resource, checkpoint or cleanup failure, or nil on
// success.
func (run *campaignRun) commit(ctx context.Context, c *casefile.Case, obs *Result, activeDir string) error {
	r := run.r

	sample, err := run.harness.Budget.Sample(context.WithoutCancel(ctx))
	if err != nil {
		return err
	}
	r.Resources = sample
	r.record(c, obs)
	r.Next++
	r.Active = ""
	if err := run.persist(); err != nil {
		return err
	}
	return os.RemoveAll(activeDir)
}

// record updates detailed metrics and the checkpoint outcome counters.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes obs (*Result) which holds the case execution result.
func (r *Report) record(c *casefile.Case, obs *Result) {
	r.Metrics.Observe(c, obs, obs.Confirmed)
	if len(obs.RuntimeCoverage) > 0 && r.RuntimeCoverage == nil {
		r.RuntimeCoverage = map[string]RuntimeCoverageCount{}
	}
	for mode := range obs.RuntimeCoverage {
		r.recordCoverage(mode, obs.RuntimeCoverage[mode])
	}
	r.ensureCounters()
	r.Outcomes[obs.Kind]++
	r.ByTarget[c.Target]++
	for mode := range obs.Checks {
		stat := obs.Checks[mode]
		if stat.Status == checkStatusPass || stat.Status == StatusFailed {
			r.Modes[mode]++
		}
		if stat.BrowserObserved {
			r.Browsers++
		}
	}
	if targets.IsApplication(c.Target) {
		for _, feature := range c.Features {
			r.ByFeature[feature]++
		}
	}
}

// ensureCounters creates the checkpoint's outcome counters when absent.
func (r *Report) ensureCounters() {
	if r.Outcomes == nil {
		r.Outcomes = map[string]uint64{}
	}
	if r.ByTarget == nil {
		r.ByTarget = map[string]uint64{}
	}
	if r.ByFeature == nil {
		r.ByFeature = map[string]uint64{}
	}
	if r.Modes == nil {
		r.Modes = map[string]uint64{}
	}
}

// recordCoverage counts one mode's runtime-coverage receipt.
//
// Takes mode (string) which identifies the check mode.
// Takes receipt (runtimecoverage.Receipt) which holds the collected coverage receipt.
func (r *Report) recordCoverage(mode string, receipt runtimecoverage.Receipt) {
	counts := r.RuntimeCoverage[mode]
	counts.Runs++
	counts.LastGoStatements = receipt.GoStatements
	counts.LastJavaScriptFunctions = receipt.JavaScriptFunctions
	counts.LastArtefact = filepath.Join("runtime-coverage", receipt.Artefact)
	if receipt.JavaScriptFunctions != nil {
		counts.BrowserRuns++
	}
	r.RuntimeCoverage[mode] = counts
}

// BrowserCase applies the saved case's browser-sampling policy.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes policy (string) which selects the browser-sampling policy.
//
// Returns bool which reports whether the saved case selects a browser.
func BrowserCase(c *casefile.Case, policy string) bool {
	if !targets.IsApplication(c.Target) || policy == "off" {
		return false
	}
	if c.Expected.DOM != nil {
		return true
	}
	if policy == "every" || slices.Contains(browserTargets, c.Target) {
		return true
	}
	if c.BrowserSampleVersion == 0 {
		return c.Index%browserSampleDecade == c.Seed%browserSampleDecade
	}

	decade := c.Index / browserSampleDecade
	residue := (c.Seed%browserSampleVariants + browserSampleStride*(decade%browserSampleVariants)) % browserSampleVariants
	choices := (browserSampleDecade-1-residue)/browserSampleVariants + 1
	offset := residue + browserSampleVariants*((c.Seed/browserSampleVariants+decade/browserSampleVariants)%choices)
	return c.Index%browserSampleDecade == offset
}

// selectedModes lists the server modes a case runs under the requested mode.
//
// Takes c (*casefile.Case) which holds the case's target.
// Takes mode (string) which is the requested server mode, or both.
//
// Returns []string which lists the server modes in execution order.
func selectedModes(c *casefile.Case, mode string) []string {
	interpreted := pipitpending.Local()
	if mode != "both" {
		if mode == modeInterp && !interpreted {
			return nil
		}
		return []string{mode}
	}
	if c.Target == "embedded-deploy" || !interpreted {
		return []string{modeCompiled}
	}
	return []string{modeInterp, modeCompiled}
}

// pendingCheck creates a check that has not started.
//
// Takes browserSelected (bool) which reports whether the check selected a browser.
// Takes routes (map[string]runner.BrowserRouteEvidence) which records the planned browser
// routes, or nil.
//
// Returns CheckMetric which holds the pending check.
func pendingCheck(browserSelected bool, routes map[string]runner.BrowserRouteEvidence) CheckMetric {
	var check CheckMetric
	check.Status = statusPending
	check.BrowserSelected = browserSelected
	check.BrowserRoutes = routes
	return check
}

// classify distinguishes semantic failures from resource and infrastructure failures.
//
// Takes kind (string) which identifies the failure or provider category.
// Takes mode (string) which selects the server mode.
// Takes err (error) which is the operation failure to inspect.
// Takes r (Result) which holds the case execution result.
//
// Returns Result which holds the classified case outcome and independent failures.
func classify(ctx context.Context, kind, mode string, err error, r Result) Result {
	if cause := context.Cause(ctx); cause != nil {
		if errors.Is(cause, resource.ErrLimit) {
			return failure(kindResource, mode, cause, r)
		}
		return failure(kindInterrupted, mode, cause, r)
	}
	if errors.Is(err, resource.ErrLimit) || errors.Is(err, syscall.ENOSPC) {
		return failure(kindResource, mode, err, r)
	}
	if errors.Is(err, errPhaseTimeout) {
		return failure("timeout", mode, err, r)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return failure(kindResource, mode, err, r)
	}
	if infrastructureFailure(err) {
		kind = kindHarness
	}
	return failure(kind, mode, err, r)
}

// infrastructureFailure reports whether an error comes from the harness rather than the
// case.
//
// Takes err (error) which is the operation failure to inspect.
//
// Returns bool which reports whether the failure is an infrastructure failure.
func infrastructureFailure(err error) bool {
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return true
	}
	message := err.Error()
	for _, prefix := range []string{
		"resource measurement:", "reading kernel resource limits:", "stage:", "worker-build/", "native-probe:", "listen:",
		"infrastructure:",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	for _, fragment := range []string{
		casefile.BrowserBudgetMarker, "module lookup disabled", "executable file not found", "chrome failed to start:",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

// phaseLabel extracts the failing execution phase.
//
// Takes err (error) which is the operation failure to inspect.
//
// Returns string which is the failing execution phase.
func phaseLabel(err error) string {
	firstLine, _, _ := strings.Cut(err.Error(), "\n")
	label, _, _ := strings.Cut(firstLine, ":")
	if len(label) > maxPhaseLabelBytes {
		return "execution"
	}
	return label
}

// failure records an independent mode or parity failure.
//
// Takes kind (string) which identifies the failure or provider category.
// Takes mode (string) which selects the server mode.
// Takes err (error) which is the operation failure to inspect.
// Takes r (Result) which holds the case execution result.
//
// Returns Result which holds the classified case outcome and independent failures.
func failure(kind, mode string, err error, r Result) Result {
	stopped := kind == kindInterrupted || kind == kindHarness || kind == kindResource
	if metric, ok := r.Checks[mode]; ok && stopped && metric.Status != statusPending {
		metric.Status = "incomplete"
		r.Checks[mode] = metric
	}
	issue := ModeFailure{Kind: kind, Mode: mode, Phase: phaseLabel(err), Subject: "", Message: err.Error()}
	if located, ok := errors.AsType[subjectError](err); ok {
		issue.Subject = located.FailureSubject()
	}
	r.Failures = append(r.Failures, issue)
	if len(r.Failures) == 1 || stopped {
		r.Kind, r.Mode, r.Phase, r.Subject, r.Message = issue.Kind, issue.Mode, issue.Phase, issue.Subject, issue.Message
	}
	return r
}

// confirmable reports whether a result is eligible for cold confirmation.
//
// A phase timeout is confirmable so that a deterministic hang is confirmed, committed and
// the stream advances past it.
//
// Takes r (Result) which holds the case execution result.
//
// Returns bool which reports whether the result contains a confirmable failure.
func confirmable(r Result) bool {
	return r.Kind == "expectation" || r.Kind == "disagreement" || r.Kind == "target" || r.Kind == "worker" || r.Kind == "timeout"
}

// loadActive verifies saved source and generator identities before resuming.
//
// Older saved source directories can still be replayed independently of the generator.
//
// Takes dir (string) which is the saved campaign directory.
// Takes r (*Report) which holds the saved receipt or campaign report.
//
// Returns *casefile.Case which holds the saved active case, or is nil when no case is
// active.
// Returns error which reports unreadable saved sources or a generator or source-identity
// mismatch, or nil on success.
func loadActive(dir string, r *Report) (*casefile.Case, error) {
	var active *casefile.Case
	if r.Active != "" {
		c, err := casefile.Read(filepath.Join(dir, "active"))
		if err != nil {
			return nil, err
		}
		targetCount := uint64(len(r.Options.Targets))
		seed, index := StreamPosition(r.Options.Suite, r.Options.Seed, r.Next/targetCount)
		if c.Hash() != r.Active || c.Index != index || c.Target != r.Options.Targets[r.Next%targetCount] || c.Seed != seed {
			return nil, errors.New("saved active case differs from the campaign checkpoint")
		}
		if c.Generator != generate.Version || c.ProfileSHA256 != generate.ProfileSHA256 {
			return nil, errors.New("saved active generator or profile differs; replay the archived source individually")
		}
		active = &c
	}
	if r.Generator == "" && r.GeneratorProfile == "" && (r.Next == 0 || active != nil) {
		r.Generator = generate.Version
		r.GeneratorProfile = generate.ProfileSHA256
	}
	if r.Generator != generate.Version || r.GeneratorProfile != generate.ProfileSHA256 {
		return nil, errors.New("saved generator or profile differs; replay archived sources or start a new stream")
	}
	return active, nil
}

// completionError reports confirmed findings after the campaign stops.
//
// Takes r (*Report) which holds the saved receipt or campaign report.
//
// Returns error which reports confirmed campaign findings, or nil when no confirmed
// findings were saved.
func completionError(r *Report) error {
	if len(r.Metrics.ConfirmedFingerprints) > 0 {
		return errors.New("campaign recorded confirmed findings")
	}
	return nil
}

// failurePath selects the saved bundle path for a case failure.
//
// Takes dir (string) which is the campaign directory containing saved findings.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes r (*Result) which holds the case execution result.
//
// Returns string which is the saved finding bundle path.
func failurePath(dir string, c *casefile.Case, r *Result) string {
	return filepath.Join(dir, "failures", fmt.Sprintf("%d-%s-%s", c.Index, c.Target, r.Fingerprint()))
}

// phaseEstimate selects the admitted memory estimate for one mode.
//
// Serial admissions bound concurrent compiler and browser working sets.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes name (string) which identifies the mode or check.
// Takes browser (string) which selects the browser-sampling policy.
//
// Returns uint64 which is the estimated incremental memory in bytes.
func phaseEstimate(c *casefile.Case, name, browser string) uint64 {
	switch {
	case name == modeCompiled || BrowserCase(c, browser):
		return compiledPhaseBytes
	case name == modeInterp:
		return interpretedPhaseBytes
	default:
		return packagePhaseBytes
	}
}

// plannedBrowserRoutes lists the browser routes a check will visit before it runs.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes selected (bool) which reports whether a browser was selected.
//
// Returns map[string]runner.BrowserRouteEvidence which records each planned route.
// Returns error which reports an invalid saved route plan, or nil on success.
func plannedBrowserRoutes(c *casefile.Case, selected bool) (map[string]runner.BrowserRouteEvidence, error) {
	routes, err := runner.BrowserRouteResults(c, &runner.Observation{}, false)
	if err != nil {
		return nil, fmt.Errorf("infrastructure: planning browser routes: %w", err)
	}
	for path, route := range routes {
		route.Selected = selected
		routes[path] = route
	}
	return routes, nil
}
