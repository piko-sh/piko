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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// nativeBuildTimeout bounds the native worker compilation.
	nativeBuildTimeout = 5 * time.Minute

	// nativeReportingAllowance bounds native worker startup and reporting beyond the fuzz
	// duration.
	nativeReportingAllowance = 2 * time.Minute

	// nativeAdmissionBytes is the memory estimate admitted before a native invocation.
	nativeAdmissionBytes = 512 << 20
)

var (
	// errNativeBuildTimeout marks a native worker build that exceeded its deadline.
	errNativeBuildTimeout = errors.New("native worker build timeout")

	// errNativeRunTimeout marks a native worker run that exceeded its deadline.
	errNativeRunTimeout = errors.New("native worker run timeout")
)

// NativeRun is sufficient to rebuild and replay a Go fuzz target by its registered name.
type NativeRun struct {
	// Provenance identifies linked modules and source checkouts.
	Provenance Provenance

	// Started is the invocation's start time.
	Started time.Time

	// RequestedLimits records requested resource limits.
	RequestedLimits map[string]string

	// Fuzzer identifies the registered native Go fuzz target.
	Fuzzer string

	// Status records the native fuzzing outcome.
	Status string

	// Output retains the bounded native worker output.
	Output string

	// Resources holds whole-run resource readings.
	Resources resource.Sample

	// Build holds the worker compilation cost.
	Build NativePhase

	// Execution holds the native-worker execution cost.
	Execution NativePhase

	// Duration is the requested run duration.
	Duration time.Duration

	// Cases bounds native fuzz executions when no duration is supplied.
	Cases uint64
}

// nativeWorker describes one compiled native worker invocation.
type nativeWorker struct {
	// binary is the compiled worker.
	binary string

	// dir is the worker's working directory.
	dir string

	// record is the receipt path.
	record string

	// args holds the worker arguments.
	args []string

	// timeout bounds the worker run.
	timeout time.Duration
}

// NativePhase captures separate go test compilation and native worker operation.
type NativePhase struct {
	// Cost holds measured CPU and working-directory costs.
	Cost resource.PhaseCost

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// Memory holds sampled phase memory.
	Memory resource.PhaseMemory
}

// Native builds and runs the selected native Go fuzz target.
//
// Takes dir (string) which is the native invocation's state directory.
// Takes requested (*NativeRun) which holds the registered native invocation and requested
// limits. Native copies these settings before writing the receipt.
// Takes replay (bool) which selects replay of the saved native input.
//
// Returns error which reports native-target setup, compilation, execution or
// receipt-write failure, or nil on success.
func (h Harness) Native(ctx context.Context, dir string, requested *NativeRun, replay bool) error {
	n := *requested
	scratch, err := os.MkdirTemp("", "piko-fuzz-native-")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.RemoveAll(scratch); removeErr != nil {
			h.progressf("removing native scratch %s: %v\n", scratch, removeErr)
		}
	}()
	ctx = process.WithTempDir(ctx, scratch)
	fuzzer, ok := targets.Fuzzers()[n.Fuzzer]
	if !ok {
		return fmt.Errorf("unknown Go fuzzer %q", n.Fuzzer)
	}
	record := filepath.Join(dir, "native.json")
	if replay {
		record = filepath.Join(dir, "replay.json")
	}
	if err := h.prepareNative(ctx, dir, &n, &fuzzer, record, replay); err != nil {
		return err
	}
	binary := filepath.Join(dir, "worker.test")
	defer func() { _ = os.Remove(binary) }()
	if err := h.buildNative(ctx, &n, &fuzzer, scratch, binary, record); err != nil {
		return err
	}
	if err := n.Provenance.recordWorker(binary); err != nil {
		return err
	}
	if !replay && fuzzer.Package == "piko.sh/piko/internal/sfcparser" {
		if err := copyParserSeeds(filepath.Join(h.Repo, "internal", "sfcparser", "testdata"), filepath.Join(dir, "testdata")); err != nil {
			return err
		}
	}
	args, timeout, err := h.nativeArguments(dir, &n, &fuzzer, replay)
	if err != nil {
		return err
	}
	n.Status = "running"
	if err := saveJSON(record, n); err != nil {
		return err
	}
	worker := nativeWorker{binary: binary, dir: dir, record: record, args: args, timeout: timeout}
	return h.executeNative(ctx, &n, &fuzzer, &worker)
}

// prepareNative admits a native invocation and saves its initial receipt.
//
// Takes dir (string) which is the native invocation's state directory.
// Takes n (*NativeRun) which receives provenance, resources and requested limits.
// Takes fuzzer (*targets.GoFuzzer) which is the registered native fuzz target.
// Takes record (string) which is the receipt path.
// Takes replay (bool) which selects replay of the saved native input.
//
// Returns error which reports admission, provenance, header or receipt failure, or nil on
// success.
func (h Harness) prepareNative(ctx context.Context, dir string, n *NativeRun, fuzzer *targets.GoFuzzer, record string, replay bool) error {
	if err := h.Budget.Admit(ctx, nativeAdmissionBytes); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, directoryPermissions); err != nil {
		return err
	}
	var err error
	n.Provenance, err = RecordProvenance(ctx, h.Repo)
	if err != nil {
		return err
	}
	n.Resources, err = h.Budget.SampleNow(ctx)
	if err != nil {
		return err
	}
	n.RequestedLimits = map[string]string{}
	for _, key := range []string{"CPUS", "MEMORY_BYTES", "MIN_FREE_BYTES", "DISK_BYTES"} {
		n.RequestedLimits[key] = os.Getenv("PIKO_FUZZ_" + key)
	}
	if err := h.writeNativeHeader(n, fuzzer.Function, record, replay); err != nil {
		return err
	}
	var idle NativePhase
	n.Build, n.Execution = idle, idle
	n.Status = "building"
	return saveJSON(record, n)
}

// buildNative compiles the native fuzz worker and records the build phase.
//
// Takes n (*NativeRun) which receives the build cost and status.
// Takes fuzzer (*targets.GoFuzzer) which is the registered native fuzz target.
// Takes scratch (string) which is the build's working directory for cost sampling.
// Takes binary (string) which is the worker binary destination.
// Takes record (string) which is the receipt path updated on failure.
//
// Returns error which reports a sampling, build or receipt failure, or nil on success.
func (h Harness) buildNative(ctx context.Context, n *NativeRun, fuzzer *targets.GoFuzzer, scratch, binary, record string) error {
	build, cancel := context.WithTimeoutCause(ctx, nativeBuildTimeout, errNativeBuildTimeout)
	defer cancel()
	beforeBuild, err := h.Budget.SampleNow(ctx)
	if err != nil {
		return err
	}
	var out string
	started := time.Now()
	n.Build.Memory, n.Build.Cost, err = resource.MeasureCheck(beforeBuild.Cgroup, scratch, func() error {
		var buildErr error
		out, buildErr = process.Run(build, h.Repo, nil, "go", "test", "-tags=fuzz", "-fuzz=^"+regexp.QuoteMeta(fuzzer.Function)+"$", "-c", "-o", binary, fuzzer.Package)
		return buildErr
	})
	n.Build.Elapsed = time.Since(started)
	cause := context.Cause(build)
	if err == nil {
		return nil
	}
	n.Status = "build-error"

	after, sampleErr := h.Budget.SampleNow(context.WithoutCancel(ctx))
	if sampleErr == nil {
		n.Resources = after
		if after.OOMSince(beforeBuild) {
			err = fmt.Errorf("%w: native build OOM", resource.ErrLimit)
		}
	}
	if errors.Is(cause, context.Canceled) {
		n.Status = "paused"
	}
	if errors.Is(cause, errNativeBuildTimeout) || errors.Is(cause, resource.ErrLimit) || errors.Is(err, resource.ErrLimit) {
		n.Status = kindResource
	}
	n.Output = out
	if saveErr := saveJSON(record, n); saveErr != nil {
		return saveErr
	}
	return fmt.Errorf("build %s: %w\n%s", fuzzer.Package, err, out)
}

// nativeArguments selects the native worker's arguments and deadline.
//
// Takes dir (string) which is the native invocation's state directory.
// Takes n (*NativeRun) which holds the requested duration and case count.
// Takes fuzzer (*targets.GoFuzzer) which is the registered native fuzz target.
// Takes replay (bool) which selects replay of the saved native input.
//
// Returns []string which holds the worker arguments.
// Returns time.Duration which bounds the worker run.
// Returns error which reports a missing retained failure corpus on replay, or nil on
// success.
func (h Harness) nativeArguments(dir string, n *NativeRun, fuzzer *targets.GoFuzzer, replay bool) ([]string, time.Duration, error) {
	timeout := n.Duration + nativeReportingAllowance
	if replay {
		files, err := os.ReadDir(filepath.Join(dir, "testdata", "fuzz", fuzzer.Function))
		if err != nil {
			return nil, timeout, err
		}
		if len(files) == 0 {
			return nil, timeout, fmt.Errorf("no retained failure corpus for %s", fuzzer.Function)
		}

		return []string{"-test.run=^" + regexp.QuoteMeta(fuzzer.Function) + "$", "-test.parallel=1", "-test.timeout=1m"}, timeout, nil
	}
	fuzzTime := n.Duration.String()
	if n.Duration <= 0 {
		fuzzTime = strconv.FormatUint(n.Cases, 10) + "x"
		timeout = nativeBuildTimeout
	}
	return []string{
		"-test.parallel=1", "-test.run=^$", "-test.timeout=0",
		"-test.fuzz=^" + regexp.QuoteMeta(fuzzer.Function) + "$", "-test.fuzztime=" + fuzzTime,
		"-test.fuzzcachedir=" + filepath.Join(h.State, ".cache", "fuzz", n.Fuzzer),
	}, timeout, nil
}

// executeNative runs the native worker and records its execution phase and status.
//
// Takes n (*NativeRun) which receives the execution cost, output and status.
// Takes fuzzer (*targets.GoFuzzer) which is the registered native fuzz target.
// Takes worker (*nativeWorker) which describes the compiled worker invocation.
//
// Returns error which reports a sampling, worker or receipt failure, or nil on success.
func (h Harness) executeNative(ctx context.Context, n *NativeRun, fuzzer *targets.GoFuzzer, worker *nativeWorker) error {
	dir, record := worker.dir, worker.record
	run, cancelRun := context.WithTimeoutCause(ctx, worker.timeout, errNativeRunTimeout)
	defer cancelRun()
	before, err := h.Budget.SampleNow(ctx)
	if err != nil {
		return err
	}
	var out string
	started := time.Now()
	n.Execution.Memory, n.Execution.Cost, err = resource.MeasureCheck(before.Cgroup, dir, func() error {
		var workerErr error
		out, workerErr = process.Run(run, dir, nil, worker.binary, worker.args...)
		return workerErr
	})
	n.Execution.Elapsed = time.Since(started)

	after, readErr := h.Budget.SampleNow(context.WithoutCancel(ctx))
	if readErr != nil {
		return readErr
	}
	n.Resources = after
	if after.OOMSince(before) {
		err = fmt.Errorf("%w: native worker OOM", resource.ErrLimit)
	}
	n.Output = strings.TrimSpace(out)
	n.Status = nativeStatus(run, err, out, fuzzer.Function)
	if saveErr := saveJSON(record, n); saveErr != nil {
		return saveErr
	}
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", fuzzer.Function, err, out)
	}
	return nil
}

// writeNativeHeader writes the native run's limits and provenance summary.
//
// Takes n (*NativeRun) which holds the native invocation and its measured limits.
// Takes function (string) which is the native fuzz function name.
// Takes record (string) which is the receipt path holding full provenance.
// Takes replay (bool) which selects replay of the saved native input.
//
// Returns error which reports a failed write, or nil on success.
func (h Harness) writeNativeHeader(n *NativeRun, function, record string, replay bool) error {
	writer := h.output()
	if _, err := fmt.Fprintf(writer, "Go fuzz target %s, CPU quota %.2f, allowed CPUs %v, memory cap %d bytes, swap cap %d bytes, one fuzz worker\n",
		function, n.Resources.QuotaCPUs, n.Resources.CPUs, n.Resources.MemoryMax, n.Resources.SwapMax); err != nil {
		return fmt.Errorf("writing native header: %w", err)
	}
	if !replay && n.Duration > 0 {
		if _, err := fmt.Fprintf(writer, "Worker build deadline 5m, followed by %s of Go fuzzing and at most 2m for startup and reporting\n", n.Duration); err != nil {
			return fmt.Errorf("writing native header: %w", err)
		}
	}
	if _, err := fmt.Fprintf(writer, "%s, workspace SHA-256 %s, %d local source receipts; full provenance in %s\n",
		n.Provenance.Go, n.Provenance.WorkspaceSHA256, len(n.Provenance.Sources), record); err != nil {
		return fmt.Errorf("writing native header: %w", err)
	}
	return nil
}

// LoadNative reads a saved native fuzz invocation.
//
// Takes dir (string) which is the saved native invocation directory.
//
// Returns NativeRun which holds the saved native target configuration and run limits.
// Returns error which reports a saved-run read or decoding failure, or nil on success.
func LoadNative(dir string) (NativeRun, error) {
	var n NativeRun
	data, err := smallRead(dir, "native.json", maxResultBytes)
	if err != nil {
		return n, err
	}
	if err := json.Unmarshal(data, &n); err != nil {
		return n, fmt.Errorf("decoding native campaign receipt in %s: %w", dir, err)
	}
	if n.Fuzzer == "" || n.Duration < 0 || n.Cases == 0 {
		return n, fmt.Errorf("invalid native campaign receipt in %s", dir)
	}
	return n, nil
}

// nativeStatus classifies a finished native worker run.
//
// Takes err (error) which is the worker or resource failure, or nil.
// Takes out (string) which holds the worker output.
// Takes function (string) which is the native fuzz function name.
//
// Returns string which is pass, failed, paused, resource or harness.
func nativeStatus(run context.Context, err error, out, function string) string {
	if err == nil {
		return "pass"
	}
	switch {
	case errors.Is(context.Cause(run), context.Canceled):
		return "paused"
	case run.Err() != nil || errors.Is(err, resource.ErrLimit):
		return kindResource
	case !strings.Contains(out, "--- FAIL: "+function) && !strings.Contains(out, "Failing input written to testdata/"):
		return "harness"
	}
	return StatusFailed
}

// copyParserSeeds copies the parser's on-disk seed corpus into the private campaign
// directory.
//
// Takes source (string) which is the parser's seed corpus directory.
// Takes destination (string) which is the campaign's testdata directory.
//
// Returns error which reports an unreadable seed or a failed copy, or nil on success.
func copyParserSeeds(source, destination string) (err error) {
	seeds, err := os.OpenRoot(source)
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(destination, directoryPermissions)
	}
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, seeds.Close()) }()
	files, err := fs.Glob(seeds.FS(), "*.pk")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, directoryPermissions); err != nil {
		return err
	}
	target, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, target.Close()) }()
	for _, file := range files {
		data, err := seeds.ReadFile(file)
		if err != nil {
			return err
		}
		if err := target.WriteFile(file, data, filePermissions); err != nil {
			return err
		}
	}
	return nil
}
