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
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// SubprocessRequest captures everything a runner needs to know to execute a child process
// and capture its outputs.
type SubprocessRequest struct {
	// Name is the executable path or name to run (e.g. "go", "python3").
	Name string

	// Args is the argument list (excluding Name).
	Args []string

	// Env is appended to os.Environ() before exec. Pairs are KEY=VALUE.
	Env []string

	// WorkingDir is the directory to chdir into before running. Empty means inherit the
	// parent process's cwd.
	WorkingDir string

	// Timeout bounds the wall time of the call. The child is killed with SIGKILL via the
	// platform-specific process-group kill if the timeout fires.
	Timeout time.Duration
}

// SubprocessResponse bundles the outputs and timing captured during a child process run.
type SubprocessResponse struct {
	Stdout []byte

	Stderr []byte

	WallNanos int64

	PeakRSSKB int64

	ExitCode int

	Err error
}

// RunSubprocess executes the request and returns the captured outputs. The Err field on
// the response is set on non-zero exit or timeout; the returned error is non-nil only on
// framework-level failure (cannot create the process at all).
func RunSubprocess(parent context.Context, request SubprocessRequest) (SubprocessResponse, error) {
	if request.Name == "" {
		return SubprocessResponse{}, errors.New("subprocess request: empty Name")
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeoutCause(parent, timeout, fmt.Errorf("subprocess %s timed out after %s", request.Name, timeout))
	defer cancel()

	command := exec.CommandContext(ctx, request.Name, request.Args...)
	if request.WorkingDir != "" {
		command.Dir = request.WorkingDir
	}
	command.Env = append(command.Environ(), request.Env...)
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	applyProcessAttributes(command)

	start := time.Now()
	runError := command.Run()
	elapsed := time.Since(start)

	response := SubprocessResponse{
		Stdout:    stdoutBuffer.Bytes(),
		Stderr:    stderrBuffer.Bytes(),
		WallNanos: elapsed.Nanoseconds(),
	}
	if command.ProcessState != nil {
		response.ExitCode = command.ProcessState.ExitCode()
		response.PeakRSSKB = platformPeakRSSKB(command.ProcessState.SysUsage())
	}
	if runError != nil {
		response.Err = runError
	}
	return response, nil
}

// dividerForDarwinRSS converts Darwin's bytes-reported peak RSS to KiB.
const dividerForDarwinRSS = 1024

// waitDelayAfterCancel is how long Cmd.Wait blocks after Cancel before giving up. Long
// enough for an orderly SIGKILL to flush; short enough that a stuck child does not stall
// the suite.
const waitDelayAfterCancel = 5 * time.Second

// platformOSIsDarwin reports whether the running process is on macOS. Used by the unix
// peak-RSS path to choose the byte-to-KiB conversion.
func platformOSIsDarwin() bool { return runtime.GOOS == "darwin" }
