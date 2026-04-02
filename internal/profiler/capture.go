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
	"fmt"
	"runtime/pprof"
	"runtime/trace"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// StartCapture begins capturing CPU and trace profiles to files in
// config.OutputDir. It returns a cleanup function that stops the active profiles
// and writes heap, block, mutex, goroutine, and allocs snapshots.
//
// Takes config (Config) which provides OutputDir for the profile files.
//
// Returns func() which must be called (typically via defer) to finalise the
// profiles and flush all data to disk.
// Returns error when the output sandbox cannot be created or the CPU profile
// cannot be started.
func StartCapture(config Config) (func(), error) {
	sandbox, err := resolveProfilerSandbox(config)
	if err != nil {
		return nil, err
	}

	cpuFile, err := sandbox.Create("cpu.pprof")
	if err != nil {
		_ = sandbox.Close()
		return nil, fmt.Errorf("creating CPU profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		_ = cpuFile.Close()
		_ = sandbox.Close()
		return nil, fmt.Errorf("starting CPU profile: %w", err)
	}

	traceFile, err := sandbox.Create("trace.out")
	if err != nil {
		pprof.StopCPUProfile()
		_ = cpuFile.Close()
		_ = sandbox.Close()
		return nil, fmt.Errorf("creating trace file: %w", err)
	}

	if err := trace.Start(traceFile); err != nil {
		pprof.StopCPUProfile()
		_ = cpuFile.Close()
		_ = traceFile.Close()
		_ = sandbox.Close()
		return nil, fmt.Errorf("starting trace: %w", err)
	}

	cleanup := func() {
		trace.Stop()
		_ = traceFile.Close()

		pprof.StopCPUProfile()
		_ = cpuFile.Close()

		writeNamedProfile(sandbox, "heap.pprof", "heap")
		writeNamedProfile(sandbox, "block.pprof", "block")
		writeNamedProfile(sandbox, "mutex.pprof", "mutex")
		writeNamedProfile(sandbox, "goroutine.pprof", "goroutine")
		writeNamedProfile(sandbox, "allocs.pprof", "allocs")

		_ = sandbox.Close()
	}

	return cleanup, nil
}

// resolveProfilerSandbox returns a sandbox for the profiler output directory.
// It prefers the injected sandbox, then the factory, then a no-op fallback.
//
// Takes config (Config) which provides the sandbox, factory, and output
// directory.
//
// Returns safedisk.Sandbox which provides write access to the output directory.
// Returns error when no sandbox can be created.
func resolveProfilerSandbox(config Config) (safedisk.Sandbox, error) {
	if config.Sandbox != nil {
		return config.Sandbox, nil
	}
	if config.SandboxFactory != nil {
		sandbox, err := config.SandboxFactory.Create("profiler", config.OutputDir, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("creating profile output sandbox via factory: %w", err)
		}
		return sandbox, nil
	}
	sandbox, err := safedisk.NewNoOpSandbox(config.OutputDir, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating profile output sandbox: %w", err)
	}
	return sandbox, nil
}

// writeNamedProfile writes a single named runtime profile to a file via the
// given sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file creation.
// Takes filename (string) which is the profile file name.
// Takes profileName (string) which is the runtime profile name.
func writeNamedProfile(sandbox safedisk.Sandbox, filename, profileName string) {
	p := pprof.Lookup(profileName)
	if p == nil {
		return
	}

	f, err := sandbox.Create(filename)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	if writeError := p.WriteTo(f, 0); writeError != nil {
		_, errorLogger := logger_domain.From(context.Background(), nil)
		errorLogger.Error("failed to write profile",
			logger_domain.String("profile", profileName),
			logger_domain.String("filename", filename),
			logger_domain.Error(writeError))
	}
}
