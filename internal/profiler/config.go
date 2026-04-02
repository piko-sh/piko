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
	"time"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// DefaultPort is the default port for the pprof HTTP server.
	DefaultPort = 6060

	// DefaultBindAddress is the default network address for the pprof server.
	DefaultBindAddress = "localhost"

	// DefaultBlockProfileRate is the default block profiling rate.
	// A value of 1000 captures blocking events of 1 microsecond or longer,
	// which is suitable for long-running servers.
	DefaultBlockProfileRate = 1000

	// DefaultMutexProfileFraction is the default mutex profiling fraction.
	// A value of 10 captures 1/10 of mutex contention events.
	DefaultMutexProfileFraction = 10

	// DefaultOutputDir is the default directory for captured profile files.
	DefaultOutputDir = "./profiles"

	// CaptureBlockProfileRate is the block profile rate for short-lived
	// capture sessions where every event should be recorded.
	CaptureBlockProfileRate = 1

	// CaptureMutexProfileFraction is the mutex profile fraction for
	// short-lived capture sessions where every event should be recorded.
	CaptureMutexProfileFraction = 1

	// DefaultMemProfileRate is 0, meaning the Go runtime default (512KB
	// sampling) is used.
	DefaultMemProfileRate = 0

	// DefaultRollingTraceMinAge keeps roughly the most recent 15 seconds
	// of execution trace data when rolling trace capture is enabled.
	DefaultRollingTraceMinAge = 15 * time.Second

	// DefaultRollingTraceMaxBytes caps the rolling trace buffer at roughly
	// 16 MiB when rolling trace capture is enabled.
	DefaultRollingTraceMaxBytes = 16 * 1024 * 1024

	// CaptureMemProfileRate captures at 4096-byte granularity for
	// short-lived builds where allocation detail matters.
	CaptureMemProfileRate = 4096

	// BasePath is the URL prefix for all profiling endpoints.
	BasePath = "/_piko"
)

// Config holds settings for profiling. Fields are used differently depending
// on whether the config is used for server mode or capture mode.
type Config struct {
	// Sandbox provides sandboxed filesystem access for writing profile files
	// (capture mode only). When nil, a default sandbox rooted at OutputDir
	// is created.
	Sandbox safedisk.Sandbox

	// SandboxFactory creates sandboxes when Sandbox is nil (capture mode
	// only). When non-nil and Sandbox is nil, this factory is used instead
	// of safedisk.NewNoOpSandbox.
	SandboxFactory safedisk.Factory

	// BindAddress is the network address to bind the pprof server to
	// (server mode only).
	BindAddress string

	// OutputDir is the directory for writing .pprof files (capture mode only).
	OutputDir string

	// Port is the HTTP port for the pprof server (server mode only).
	Port int

	// BlockProfileRate controls the granularity of the block profile.
	// After calling runtime.SetBlockProfileRate, the profiler samples one
	// blocking event per this many nanoseconds of blocking.
	BlockProfileRate int

	// MutexProfileFraction controls the fraction of mutex contention events
	// reported. On average 1/n events are reported.
	MutexProfileFraction int

	// MemProfileRate controls the memory profiling sample rate in bytes.
	MemProfileRate int

	// EnableRollingTrace enables a bounded in-memory rolling execution trace
	// buffer for long-running profiling server mode.
	EnableRollingTrace bool

	// RollingTraceMinAge is the minimum trace history the recorder should
	// try to retain when rolling trace capture is enabled.
	RollingTraceMinAge time.Duration

	// RollingTraceMaxBytes is the maximum in-memory budget hint for the
	// rolling trace buffer when rolling trace capture is enabled.
	RollingTraceMaxBytes uint64
}
