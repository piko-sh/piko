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

// SystemStats holds system and runtime statistics for monitoring.
type SystemStats struct {
	// MonitoringListenAddr is the address the monitoring gRPC server listens on.
	MonitoringListenAddr string

	// CgroupPath is the cgroup v2 path for the process (Linux only).
	CgroupPath string

	// GC contains garbage collection statistics.
	GC GCInfo

	// Build contains version, commit, and compilation details.
	Build BuildInfo

	// Runtime holds Go runtime configuration such as GOGC and GOMEMLIMIT.
	Runtime RuntimeInfo

	// Process holds the process-level statistics such as PID, thread count,
	// resource count, and resident set size.
	Process ProcessInfo

	// Memory contains heap and system memory statistics.
	Memory MemoryInfo

	// TimestampMs is the Unix timestamp in milliseconds when the stats were
	// collected.
	TimestampMs int64

	// UptimeMs is the process uptime in milliseconds.
	UptimeMs int64

	// SystemUptimeMs is the host system uptime in milliseconds (Linux only).
	SystemUptimeMs int64

	// NumCGOCalls is the total number of CGO calls made since process start.
	NumCGOCalls int64

	// CPUMillicores is the current CPU usage in millicores.
	CPUMillicores float64

	// NumCPU is the number of CPUs available to the process.
	NumCPU int32

	// GOMAXPROCS is the maximum number of CPUs that can execute simultaneously.
	GOMAXPROCS int32

	// NumGoroutines is the current number of goroutines.
	NumGoroutines int32
}

// BuildInfo holds version and build details for the application.
type BuildInfo struct {
	// GoVersion is the Go compiler version used to build the application.
	GoVersion string

	// Version is the application version string.
	Version string

	// Commit is the git commit hash from which the binary was built.
	Commit string

	// BuildTime is when the binary was built.
	BuildTime string

	// OS is the operating system the binary was built for.
	OS string

	// Arch is the CPU architecture the binary was built for.
	Arch string

	// ModulePath is the Go module path from debug.ReadBuildInfo.
	ModulePath string

	// ModuleVersion is the module version from debug.ReadBuildInfo.
	ModuleVersion string

	// VCSTime is the VCS commit timestamp from debug.ReadBuildInfo.
	VCSTime string

	// VCSModified indicates whether the working tree had uncommitted changes.
	VCSModified bool
}

// RuntimeInfo holds Go runtime settings such as garbage collection tuning.
type RuntimeInfo struct {
	// GOGC is the current GOGC environment variable value.
	GOGC string

	// GOMEMLIMIT is the memory limit value from the Go runtime.
	GOMEMLIMIT string

	// Compiler is the name of the Go compiler toolchain (e.g. "gc").
	Compiler string
}

// GCInfo holds garbage collection statistics for the Go runtime.
type GCInfo struct {
	// RecentPauses contains recent GC pause durations in nanoseconds.
	RecentPauses []uint64

	// LastGC is the timestamp of the last garbage collection in nanoseconds.
	LastGC int64

	// PauseTotalNs is the cumulative nanoseconds spent in GC pauses.
	PauseTotalNs uint64

	// LastPauseNs is the duration of the most recent GC pause in nanoseconds.
	LastPauseNs uint64

	// GCCPUFraction is the fraction of CPU time used by the garbage collector.
	GCCPUFraction float64

	// NextGC is the heap size target for the next garbage collection cycle in
	// bytes.
	NextGC uint64

	// NumGC is the number of completed garbage collection cycles.
	NumGC uint32

	// NumForcedGC is the number of GC cycles that were forced by the application.
	NumForcedGC uint32
}

// MemoryInfo holds memory usage data for the running process.
type MemoryInfo struct {
	// Alloc is the current heap allocation in bytes.
	Alloc uint64

	// TotalAlloc is the cumulative bytes allocated over the lifetime of the
	// process.
	TotalAlloc uint64

	// Sys is the total bytes of memory obtained from the system.
	Sys uint64

	// HeapAlloc is the bytes of allocated heap objects.
	HeapAlloc uint64

	// HeapSys is the total bytes of heap memory obtained from the OS.
	HeapSys uint64

	// HeapIdle is the number of bytes in idle heap spans.
	HeapIdle uint64

	// HeapInuse is the number of bytes in in-use heap spans.
	HeapInuse uint64

	// HeapObjects is the number of allocated heap objects.
	HeapObjects uint64

	// HeapReleased is the number of bytes released to the operating system.
	HeapReleased uint64

	// StackSys is the bytes of stack memory obtained from the OS.
	StackSys uint64

	// Mallocs is the cumulative count of heap object allocations.
	Mallocs uint64

	// Frees is the cumulative count of heap objects freed.
	Frees uint64

	// LiveObjects is the number of allocated objects that are still in use.
	LiveObjects uint64

	// StackInuse is the bytes in stack spans currently in use.
	StackInuse uint64

	// MSpanInuse is the bytes of allocated mspan structures.
	MSpanInuse uint64

	// MSpanSys is the bytes of memory obtained from the OS for mspan structures.
	MSpanSys uint64

	// MCacheInuse is the bytes of allocated mcache structures.
	MCacheInuse uint64

	// MCacheSys is the bytes of memory obtained from the OS for mcache structures.
	MCacheSys uint64

	// GCSys is the bytes of memory used for garbage collection system metadata.
	GCSys uint64

	// OtherSys is the bytes of memory used for off-heap runtime allocations.
	OtherSys uint64

	// BuckHashSys is the bytes of memory used by the profiling bucket hash table.
	BuckHashSys uint64

	// Lookups is the number of pointer lookups performed by the runtime.
	Lookups uint64
}

// ProcessInfo holds details about a running process for monitoring.
type ProcessInfo struct {
	// Hostname is the system hostname where the process is running.
	Hostname string

	// Executable is the path of the running binary.
	Executable string

	// CWD is the current working directory of the process.
	CWD string

	// PID is the process identifier of the running process.
	PID int32

	// ThreadCount is the number of threads used by the process.
	ThreadCount int32

	// FDCount is the number of open resources.
	FDCount int32

	// UID is the user ID of the process owner.
	UID int32

	// GID is the group ID of the process owner.
	GID int32

	// PPID is the parent process identifier.
	PPID int32

	// RSS is the resident set size in bytes.
	RSS uint64

	// CgroupMemoryLimit is the container memory limit in bytes from the cgroup
	// filesystem. Zero indicates the limit is unknown or unlimited.
	CgroupMemoryLimit uint64

	// CgroupMemoryCurrent is the container's current memory usage in bytes
	// from the cgroup filesystem. Zero indicates the value is unavailable.
	CgroupMemoryCurrent uint64

	// IoReadBytes is the number of bytes read from storage.
	IoReadBytes uint64

	// IoWriteBytes is the number of bytes written to storage.
	IoWriteBytes uint64

	// IoRchar is the total bytes read (including page cache).
	IoRchar uint64

	// IoWchar is the total bytes written (including page cache).
	IoWchar uint64

	// MaxOpenFilesSoft is the soft limit on open resources.
	MaxOpenFilesSoft int64

	// MaxOpenFilesHard is the hard limit on open resources.
	MaxOpenFilesHard int64
}
