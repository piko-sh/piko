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

//go:build crosslang

package harness

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PythonRunner executes a benchmark by spawning the host's `python3` or
// `pypy3` interpreter on the corresponding `<benchmark>/py/<file>.py`. This
// is the host-Python fallback path; the testcontainers-backed runner
// (ContainerPythonRunner below) is the default.
type PythonRunner struct {
	kind RunnerKind

	pythonBinary string

	scriptFilename string

	availableMessage string

	availableProbed bool

	available bool
}

// NewHostCPythonRunner returns a Runner that invokes `python3` on the host.
func NewHostCPythonRunner() *PythonRunner {
	return &PythonRunner{
		kind:           RunnerCPython,
		pythonBinary:   "python3",
		scriptFilename: "cpython.py",
	}
}

// NewHostPyPyRunner returns a Runner that invokes `pypy3` on the host.
func NewHostPyPyRunner() *PythonRunner {
	return &PythonRunner{
		kind:           RunnerPyPy,
		pythonBinary:   "pypy3",
		scriptFilename: "pypy.py",
	}
}

// Kind reports the runner identity used in results.
func (runner *PythonRunner) Kind() RunnerKind { return runner.kind }

// Available probes that the interpreter binary is on PATH.
func (runner *PythonRunner) Available(ctx context.Context) (bool, string) {
	_ = ctx
	if runner.availableProbed {
		return runner.available, runner.availableMessage
	}
	runner.availableProbed = true
	if _, err := exec.LookPath(runner.pythonBinary); err != nil {
		runner.available = false
		runner.availableMessage = runner.pythonBinary + " not found on PATH"
		return false, runner.availableMessage
	}
	runner.available = true
	return true, ""
}

// Close is a no-op for the host-binary path.
func (runner *PythonRunner) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

// Run invokes the interpreter via the shared `_python_driver.py` which
// times Python's `compile()` step separately from the workload's run()
// invocations. Stdout is the canonical result hashed against the spec;
// stderr carries the INNER_ELAPSED_NS and COMPILE_NANOS markers.
func (runner *PythonRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	benchmarksRoot := filepath.Dir(benchmarkDir)
	driverPath := filepath.Join(benchmarksRoot, "_python_driver.py")
	scriptPath := filepath.Join(benchmarkDir, "py", runner.scriptFilename)
	args := []string{driverPath, scriptPath}
	args = append(args, buildRunnerArgs(mode, spec)...)

	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	response, err := RunSubprocess(parent, SubprocessRequest{
		Name:       runner.pythonBinary,
		Args:       args,
		WorkingDir: benchmarkDir,
		Timeout:    timeout,
		Env:        []string{"PYTHONHASHSEED=0", "PYTHONDONTWRITEBYTECODE=1"},
	})
	if err != nil {
		return failedResult(spec.Name, runner.kind, mode, "framework: "+err.Error()), nil
	}
	if response.Err != nil {
		return failedResult(spec.Name, runner.kind, mode, fmt.Sprintf("%s exited %d: %s; stderr=%s",
			runner.pythonBinary, response.ExitCode, response.Err.Error(), tailString(string(response.Stderr), 1024))), nil
	}

	innerNanos, compileNanos, stderrRemainder := ParseTimingMarkers(response.Stderr)
	normalised := NormaliseStdout(response.Stdout)
	stdoutSHA := SHA256Hex(normalised)

	status := StatusOK
	note := ""
	if stdoutSHA != spec.ExpectedStdoutSHA {
		status = StatusMismatch
		note = fmt.Sprintf("stdout SHA %s does not match expected %s", stdoutSHA, spec.ExpectedStdoutSHA)
	}

	return Result{
		Benchmark:    spec.Name,
		Runner:       runner.kind,
		Mode:         mode,
		WallNanos:    response.WallNanos,
		InnerNanos:   innerNanos,
		CompileNanos: compileNanos,
		PeakRSSKB:    response.PeakRSSKB,
		StdoutSHA:    stdoutSHA,
		Status:       status,
		Note:         note,
		StderrTail:   tailString(stderrRemainder, 1024),
	}, nil
}

// PythonVersionString probes the interpreter for a one-line version
// identification, used to populate the HostInfo block in the report.
func PythonVersionString(parent context.Context, binary string) string {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ContainerPythonRunner runs a benchmark inside a long-lived Docker
// container managed by testcontainers-go. One container per Python flavour
// is started lazily on first Run and reused for every subsequent
// invocation. The benchmarks directory is bind-mounted read-only at
// /benchmarks inside the container.
type ContainerPythonRunner struct {
	kind RunnerKind

	image string

	pythonBinary string

	scriptFilename string

	hostBenchDir string

	startMu sync.Mutex

	container testcontainers.Container

	startError error

	availableProbed bool

	available bool

	availableMessage string
}

// NewContainerCPythonRunner returns a containerised CPython runner using
// the given image (e.g. "python:3.13-slim"). Container is not started
// until first Run / Available call.
func NewContainerCPythonRunner(image, hostBenchDir string) *ContainerPythonRunner {
	return &ContainerPythonRunner{
		kind:           RunnerCPython,
		image:          image,
		pythonBinary:   "python3",
		scriptFilename: "cpython.py",
		hostBenchDir:   hostBenchDir,
	}
}

// NewContainerPyPyRunner returns a containerised PyPy runner using the
// given image (e.g. "pypy:3.10-slim").
func NewContainerPyPyRunner(image, hostBenchDir string) *ContainerPythonRunner {
	return &ContainerPythonRunner{
		kind:           RunnerPyPy,
		image:          image,
		pythonBinary:   "pypy3",
		scriptFilename: "pypy.py",
		hostBenchDir:   hostBenchDir,
	}
}

// Kind reports the runner identity used in results.
func (runner *ContainerPythonRunner) Kind() RunnerKind { return runner.kind }

// Available probes Docker availability by attempting to start the
// container. The result is cached for the suite lifetime.
func (runner *ContainerPythonRunner) Available(ctx context.Context) (bool, string) {
	if runner.availableProbed {
		return runner.available, runner.availableMessage
	}
	runner.availableProbed = true
	if err := runner.ensureStarted(ctx); err != nil {
		runner.available = false
		runner.availableMessage = "docker / " + runner.image + " unavailable: " + err.Error()
		return false, runner.availableMessage
	}
	runner.available = true
	return true, ""
}

// Close terminates the underlying container.
func (runner *ContainerPythonRunner) Close(ctx context.Context) error {
	runner.startMu.Lock()
	defer runner.startMu.Unlock()
	if runner.container == nil {
		return nil
	}
	terminateError := runner.container.Terminate(ctx)
	runner.container = nil
	return terminateError
}

// ensureStarted lazily creates and starts the container the first time it
// is needed. Concurrent callers are serialised; only the first one starts
// the container.
func (runner *ContainerPythonRunner) ensureStarted(parent context.Context) error {
	runner.startMu.Lock()
	defer runner.startMu.Unlock()
	if runner.container != nil {
		return nil
	}
	if runner.startError != nil {
		return runner.startError
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	hostPath := runner.hostBenchDir

	request := testcontainers.ContainerRequest{
		Image: runner.image,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Binds = append(hostConfig.Binds, hostPath+":/benchmarks:ro")
			hostConfig.Ulimits = append(hostConfig.Ulimits, &container.Ulimit{
				Name: "stack",
				Soft: 128 * 1024 * 1024,
				Hard: 128 * 1024 * 1024,
			})
		},
		WaitingFor: wait.ForExec([]string{runner.pythonBinary, "--version"}).
			WithStartupTimeout(60 * time.Second),
	}

	startedContainer, startError := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if startError != nil {
		runner.startError = startError
		return startError
	}
	runner.container = startedContainer
	return nil
}

// Run dispatches `python3 /benchmarks/<spec.Name>/py/cpython.py <flags>`
// inside the long-lived container.
func (runner *ContainerPythonRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	_ = benchmarkDir
	if err := runner.ensureStarted(parent); err != nil {
		return failedResult(spec.Name, runner.kind, mode, "container start: "+err.Error()), nil
	}

	command := []string{
		runner.pythonBinary,
		"/benchmarks/_python_driver.py",
		fmt.Sprintf("/benchmarks/%s/py/%s", spec.Name, runner.scriptFilename),
	}
	command = append(command, buildRunnerArgs(mode, spec)...)

	ctx, cancel := context.WithTimeoutCause(parent,
		time.Duration(spec.TimeoutSeconds)*time.Second,
		fmt.Errorf("container runner %s/%s timed out", spec.Name, mode))
	defer cancel()

	start := time.Now()
	exitCode, streamReader, execError := runner.container.Exec(
		ctx,
		command,
		tcexec.WithEnv([]string{"PYTHONHASHSEED=0", "PYTHONDONTWRITEBYTECODE=1"}),
	)
	wall := time.Since(start)
	if execError != nil {
		return failedResult(spec.Name, runner.kind, mode, "container exec: "+execError.Error()), nil
	}

	stdoutBytes, stderrBytes, demuxError := demultiplexDockerStream(streamReader)
	if demuxError != nil {
		return failedResult(spec.Name, runner.kind, mode, "stream demux: "+demuxError.Error()), nil
	}
	if exitCode != 0 {
		return failedResult(spec.Name, runner.kind, mode, fmt.Sprintf("exit %d; stderr=%s", exitCode, tailString(string(stderrBytes), 1024))), nil
	}

	innerNanos, compileNanos, stderrRemainder := ParseTimingMarkers(stderrBytes)
	normalised := NormaliseStdout(stdoutBytes)
	stdoutSHA := SHA256Hex(normalised)

	status := StatusOK
	note := ""
	if stdoutSHA != spec.ExpectedStdoutSHA {
		status = StatusMismatch
		note = fmt.Sprintf("stdout SHA %s does not match expected %s", stdoutSHA, spec.ExpectedStdoutSHA)
	}

	return Result{
		Benchmark:    spec.Name,
		Runner:       runner.kind,
		Mode:         mode,
		WallNanos:    wall.Nanoseconds(),
		InnerNanos:   innerNanos,
		CompileNanos: compileNanos,
		PeakRSSKB:    -1,
		StdoutSHA:    stdoutSHA,
		Status:       status,
		Note:         note,
		StderrTail:   tailString(stderrRemainder, 1024),
	}, nil
}

// demultiplexDockerStream parses Docker's frame-tagged exec output stream
// and splits it into stdout and stderr buffers. Each frame begins with an
// 8-byte header: byte 0 is the stream identifier (1=stdout, 2=stderr),
// bytes 1-3 are reserved, bytes 4-7 are the payload size big-endian, then
// the payload follows.
func demultiplexDockerStream(stream io.Reader) ([]byte, []byte, error) {
	var stdoutBuffer, stderrBuffer bytes.Buffer
	header := make([]byte, 8)
	for {
		_, readError := io.ReadFull(stream, header)
		if readError == io.EOF {
			return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
		}
		if readError == io.ErrUnexpectedEOF {
			return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
		}
		if readError != nil {
			return nil, nil, fmt.Errorf("read header: %w", readError)
		}
		streamType := header[0]
		payloadSize := binary.BigEndian.Uint32(header[4:8])
		if payloadSize == 0 {
			continue
		}
		payload := make([]byte, payloadSize)
		if _, err := io.ReadFull(stream, payload); err != nil {
			return nil, nil, fmt.Errorf("read payload %d bytes: %w", payloadSize, err)
		}
		switch streamType {
		case 1:
			stdoutBuffer.Write(payload)
		case 2:
			stderrBuffer.Write(payload)
		default:

			stderrBuffer.Write(payload)
		}
	}
}
