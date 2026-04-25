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
	"bufio"
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// systemSampleInterval is the time between CPU usage samples.
	systemSampleInterval = time.Second

	// clockTicksPerSecond is the number of clock ticks per second.
	// On Linux this is usually 100.
	clockTicksPerSecond = 100

	// millicoresPerCore is the number of millicores in one CPU core.
	millicoresPerCore = 1000

	// gcPauseHistorySize is the size of the GC pause circular buffer in runtime.
	gcPauseHistorySize = 256

	// recentPausesCount is the number of recent GC pauses to include in the response.
	recentPausesCount = 10

	// systemStatFileParts is the expected number of parts when parsing
	// /proc/self/stat.
	systemStatFileParts = 13

	// defaultGOGC is the default garbage collection target percentage.
	defaultGOGC = "100"

	// pageSize is the memory page size in bytes (4 KB on Linux).
	pageSize = 4096

	// limitsLineFieldCount is the minimum number of whitespace-delimited fields
	// expected in a /proc/self/limits line to extract soft and hard limits.
	limitsLineFieldCount = 5
)

// SystemCollector gathers runtime statistics for the system.
// It implements SystemStatsProvider.
type SystemCollector struct {
	// startTime records when the collector was created, used to calculate uptime.
	startTime time.Time

	// lastCPUSampleAt is the timestamp of the previous CPU sample; used to
	// calculate elapsed time for CPU usage.
	lastCPUSampleAt time.Time

	// clock provides time operations; defaults to real clock if not set.
	clock clock.Clock

	// stopCh signals the collector to stop processing when closed.
	stopCh chan struct{}

	// runtimeMetrics samples the runtime/metrics view for histogram-derived
	// signals (GC pause percentiles, scheduler latency, mutex contention)
	// that runtime.MemStats does not expose.
	runtimeMetrics *runtimeMetricsCollector

	// listenAddr is the monitoring gRPC server listen address.
	listenAddr string

	// lastSnapshot is the most recent runtime/metrics snapshot, refreshed
	// once per tick alongside memStats.
	lastSnapshot runtimeMetricsSnapshot

	// memStats stores the runtime memory statistics used to populate the
	// legacy MemoryInfo and GCInfo fields. Updated each tick.
	memStats runtime.MemStats

	// lastCPUTime is the CPU time in ticks from the previous sample.
	lastCPUTime uint64

	// cpuMillicores is the current CPU usage; 1 core = 1000 millicores.
	cpuMillicores float64

	// mu guards access to collector state during concurrent operations.
	mu sync.RWMutex

	// stopped indicates whether the collector has been stopped.
	stopped bool
}

// SystemCollectorOption configures a SystemCollector.
type SystemCollectorOption func(*SystemCollector)

// NewSystemCollector creates a new system statistics collector.
//
// Takes opts (...SystemCollectorOption) which provides optional configuration
// functions to customise the collector behaviour.
//
// Returns *SystemCollector which is ready to collect system metrics.
func NewSystemCollector(opts ...SystemCollectorOption) *SystemCollector {
	c := &SystemCollector{
		clock:           nil,
		startTime:       time.Time{},
		lastCPUSampleAt: time.Time{},
		stopCh:          make(chan struct{}),
		memStats:        runtime.MemStats{},
		runtimeMetrics:  newRuntimeMetricsCollector(),
		lastCPUTime:     0,
		cpuMillicores:   0,
		mu:              sync.RWMutex{},
		stopped:         false,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.clock == nil {
		c.clock = clock.RealClock()
	}
	c.startTime = c.clock.Now()

	return c
}

// Start begins periodic sampling of system statistics.
//
// Spawns a background goroutine that runs until the context is cancelled.
func (c *SystemCollector) Start(ctx context.Context) {
	c.sample()

	go c.loop(ctx)
}

// Stop stops the collector.
//
// Safe for concurrent use. Calling Stop multiple times is safe; only the first
// call closes the stop channel.
func (c *SystemCollector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}
}

// GetStats returns the current system statistics in domain format.
// Implements SystemStatsProvider.
//
// Returns SystemStats which contains memory, CPU, GC, and process metrics.
//
// Safe for concurrent use; protected by a read lock.
func (c *SystemCollector) GetStats() SystemStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := c.clock.Now()
	uptime := now.Sub(c.startTime)

	return SystemStats{
		TimestampMs:          now.UnixMilli(),
		UptimeMs:             uptime.Milliseconds(),
		SystemUptimeMs:       readSystemUptime(),
		MonitoringListenAddr: c.listenAddr,
		CgroupPath:           readCgroupPath(),
		NumCPU:               safeconv.IntToInt32(runtime.NumCPU()),
		GOMAXPROCS:           safeconv.IntToInt32(runtime.GOMAXPROCS(0)),
		NumGoroutines:        safeconv.IntToInt32(runtime.NumGoroutine()),
		NumCGOCalls:          runtime.NumCgoCall(),
		CPUMillicores:        c.cpuMillicores,
		Memory:               c.buildMemoryInfo(),
		GC:                   c.buildGCInfo(),
		Schedule:             c.buildSchedulerInfo(),
		Sync:                 c.buildSyncInfo(),
		Build:                BuildInfo(buildBuildInfo()),
		Runtime:              RuntimeInfo(buildRuntimeConfig()),
		Process:              buildPublicProcessInfo(),
	}
}

// buildMemoryInfo converts the current runtime.MemStats and runtime/metrics
// snapshot into the public MemoryInfo struct.
//
// Returns MemoryInfo which contains heap, stack, allocator statistics, and
// runtime/metrics-derived heap class partitions.
func (c *SystemCollector) buildMemoryInfo() MemoryInfo {
	return MemoryInfo{
		Alloc:        c.memStats.Alloc,
		TotalAlloc:   c.memStats.TotalAlloc,
		Sys:          c.memStats.Sys,
		HeapAlloc:    c.memStats.HeapAlloc,
		HeapSys:      c.memStats.HeapSys,
		HeapIdle:     c.memStats.HeapIdle,
		HeapInuse:    c.memStats.HeapInuse,
		HeapObjects:  c.memStats.HeapObjects,
		HeapReleased: c.memStats.HeapReleased,
		StackSys:     c.memStats.StackSys,
		Mallocs:      c.memStats.Mallocs,
		Frees:        c.memStats.Frees,
		LiveObjects:  c.memStats.Mallocs - c.memStats.Frees,
		StackInuse:   c.memStats.StackInuse,
		MSpanInuse:   c.memStats.MSpanInuse,
		MSpanSys:     c.memStats.MSpanSys,
		MCacheInuse:  c.memStats.MCacheInuse,
		MCacheSys:    c.memStats.MCacheSys,
		GCSys:        c.memStats.GCSys,
		OtherSys:     c.memStats.OtherSys,
		BuckHashSys:  c.memStats.BuckHashSys,
		Lookups:      c.memStats.Lookups,

		HeapObjectsBytes:  c.lastSnapshot.HeapObjectsBytes,
		HeapFreeBytes:     c.lastSnapshot.HeapFreeBytes,
		HeapReleasedBytes: c.lastSnapshot.HeapReleasedBytes,
		HeapStacksBytes:   c.lastSnapshot.HeapStacksBytes,
		HeapUnusedBytes:   c.lastSnapshot.HeapUnusedBytes,
		TotalBytes:        c.lastSnapshot.TotalMemoryBytes,
	}
}

// buildGCInfo combines legacy MemStats GC data with runtime/metrics
// histogram-derived percentiles for the public GCInfo struct.
//
// Returns GCInfo which contains pause times, recent pauses ring, and pause
// histogram percentiles.
func (c *SystemCollector) buildGCInfo() GCInfo {
	internal := c.buildGCStats()
	return GCInfo{
		RecentPauses:  internal.RecentPauses,
		LastGC:        internal.LastGC,
		PauseTotalNs:  internal.PauseTotalNs,
		LastPauseNs:   internal.LastPauseNs,
		GCCPUFraction: internal.GCCPUFraction,
		NextGC:        internal.NextGC,
		NumGC:         internal.NumGC,
		NumForcedGC:   internal.NumForcedGC,
		PauseP50:      c.lastSnapshot.GCPauseP50,
		PauseP95:      c.lastSnapshot.GCPauseP95,
		PauseP99:      c.lastSnapshot.GCPauseP99,
	}
}

// buildSchedulerInfo populates the SchedulerInfo struct from the latest
// runtime/metrics snapshot.
//
// Returns SchedulerInfo which contains scheduler latency percentiles,
// goroutine count, and GOMAXPROCS as observed by runtime/metrics.
func (c *SystemCollector) buildSchedulerInfo() SchedulerInfo {
	return SchedulerInfo{
		LatencyP50:     c.lastSnapshot.SchedulerLatencyP50,
		LatencyP99:     c.lastSnapshot.SchedulerLatencyP99,
		GoroutineCount: safeconv.Int64ToInt32(c.lastSnapshot.Goroutines),
		GoMaxProcs:     safeconv.Int64ToInt32(c.lastSnapshot.GoMaxProcs),
	}
}

// buildSyncInfo populates the SyncInfo struct from the latest runtime/metrics
// snapshot. The MutexWaitTotalSeconds counter is only populated when mutex
// profiling is enabled via runtime.SetMutexProfileFraction; otherwise it
// stays at zero.
//
// Returns SyncInfo which contains the cumulative mutex wait time.
func (c *SystemCollector) buildSyncInfo() SyncInfo {
	return SyncInfo{
		MutexWaitTotalSeconds: c.lastSnapshot.MutexWaitTotalSeconds,
	}
}

// lastRuntimeMetricsSnapshot returns the most recent runtime/metrics
// snapshot for in-package callers that need direct access to
// histogram-derived signals (such as the watchdog sidecar metadata
// writer).
//
// The returned snapshot is a value copy and safe to retain across goroutines.
//
// Returns runtimeMetricsSnapshot which contains the latest sampled values.
//
// Safe for concurrent use; protected by the collector's read lock.
func (c *SystemCollector) lastRuntimeMetricsSnapshot() runtimeMetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSnapshot
}

// loop runs the periodic sampling loop until stopped.
func (c *SystemCollector) loop(ctx context.Context) {
	ticker := c.clock.NewTicker(systemSampleInterval)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(ctx, "monitoring.systemCollectorLoop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C():
			c.sample()
		}
	}
}

// sample collects current runtime statistics.
//
// Safe for concurrent use; protected by the collector's mutex.
func (c *SystemCollector) sample() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()

	runtime.ReadMemStats(&c.memStats)
	if c.runtimeMetrics != nil {
		c.lastSnapshot = c.runtimeMetrics.sample(now)
	}

	cpuTime := readProcessCPUTime()
	if cpuTime > 0 && c.lastCPUTime > 0 && !c.lastCPUSampleAt.IsZero() {
		elapsed := now.Sub(c.lastCPUSampleAt).Seconds()
		if elapsed > 0 {
			ticksDelta := cpuTime - c.lastCPUTime
			cpuSeconds := float64(ticksDelta) / clockTicksPerSecond
			coresUsed := cpuSeconds / elapsed
			c.cpuMillicores = coresUsed * millicoresPerCore
		}
	}
	c.lastCPUTime = cpuTime
	c.lastCPUSampleAt = now
}

// internalGCStats holds GC statistics in internal format.
type internalGCStats struct {
	// RecentPauses holds the most recent GC pause durations in nanoseconds.
	RecentPauses []uint64

	// LastGC is the Unix timestamp of the last garbage collection.
	LastGC int64

	// PauseTotalNs is the total time spent in GC pauses in nanoseconds.
	PauseTotalNs uint64

	// LastPauseNs is the duration of the most recent GC pause in nanoseconds.
	LastPauseNs uint64

	// GCCPUFraction is the fraction of CPU time used by the garbage collector.
	GCCPUFraction float64

	// NextGC is the heap size target for the next garbage collection cycle.
	NextGC uint64

	// NumGC is the number of completed garbage collection cycles.
	NumGC uint32

	// NumForcedGC is the number of GC cycles forced by the application.
	NumForcedGC uint32
}

// buildGCStats builds GC statistics from the current memStats.
//
// Returns internalGCStats which contains the collected garbage collection
// metrics including recent pause times and CPU fraction.
func (c *SystemCollector) buildGCStats() internalGCStats {
	var lastPauseNs uint64
	if c.memStats.NumGC > 0 {
		lastPauseNs = c.memStats.PauseNs[(c.memStats.NumGC-1)%gcPauseHistorySize]
	}

	numPauses := min(recentPausesCount, int(c.memStats.NumGC))
	recentPauses := make([]uint64, numPauses)
	for i := range numPauses {
		index := (int(c.memStats.NumGC) - 1 - i) % gcPauseHistorySize
		recentPauses[i] = c.memStats.PauseNs[index]
	}

	return internalGCStats{
		RecentPauses:  recentPauses,
		LastGC:        safeconv.Uint64ToInt64(c.memStats.LastGC),
		PauseTotalNs:  c.memStats.PauseTotalNs,
		LastPauseNs:   lastPauseNs,
		GCCPUFraction: c.memStats.GCCPUFraction,
		NextGC:        c.memStats.NextGC,
		NumGC:         c.memStats.NumGC,
		NumForcedGC:   c.memStats.NumForcedGC,
	}
}

var (
	// Version is the application version, set at build time using -ldflags.
	Version = "dev"

	// Commit is the git commit hash, set via -ldflags at build time.
	Commit = "unknown"

	// BuildTime is the build timestamp (set via -ldflags).
	BuildTime = "unknown"
)

// internalBuildInfo holds build-time information about the application.
type internalBuildInfo struct {
	// GoVersion is the Go version used to build the binary.
	GoVersion string

	// Version is the application version string.
	Version string

	// Commit is the git commit hash of the build.
	Commit string

	// BuildTime is when the binary was built.
	BuildTime string

	// OS is the operating system for which the binary was built.
	OS string

	// Arch is the target architecture for the build.
	Arch string

	// ModulePath is the Go module path from debug.ReadBuildInfo.
	ModulePath string

	// ModuleVersion is the module version from debug.ReadBuildInfo.
	ModuleVersion string

	// VCSTime is the VCS commit timestamp.
	VCSTime string

	// VCSModified indicates uncommitted changes at build time.
	VCSModified bool
}

// internalProcessInfo holds process details from the operating system.
type internalProcessInfo struct {
	// Hostname is the system hostname where the process is running.
	Hostname string

	// Executable is the path of the running binary.
	Executable string

	// CWD is the current working directory of the process.
	CWD string

	// PID is the process identifier.
	PID int

	// ThreadCount is the number of threads used by the process.
	ThreadCount int

	// FDCount is the number of open file descriptors.
	FDCount int

	// UID is the user ID of the process owner.
	UID int

	// GID is the group ID of the process owner.
	GID int

	// PPID is the parent process identifier.
	PPID int

	// RSS is the resident set size in bytes.
	RSS uint64

	// IoReadBytes is the number of bytes read from storage.
	IoReadBytes uint64

	// IoWriteBytes is the number of bytes written to storage.
	IoWriteBytes uint64

	// IoRchar is the total bytes read (including page cache).
	IoRchar uint64

	// IoWchar is the total bytes written (including page cache).
	IoWchar uint64

	// MaxOpenFilesSoft is the soft limit on open file descriptors.
	MaxOpenFilesSoft int64

	// MaxOpenFilesHard is the hard limit on open file descriptors.
	MaxOpenFilesHard int64
}

// internalRuntimeConfig holds runtime settings for Go memory management.
type internalRuntimeConfig struct {
	// GOGC sets the garbage collection target percentage; empty uses Go default.
	GOGC string

	// GOMEMLIMIT is the Go runtime memory limit setting.
	GOMEMLIMIT string

	// Compiler is the name of the Go compiler toolchain (e.g. "gc").
	Compiler string
}

// internalIOStats holds I/O counters from /proc/self/io.
type internalIOStats struct {
	// Rchar is the total bytes read (including page cache).
	Rchar uint64

	// Wchar is the total bytes written (including page cache).
	Wchar uint64

	// ReadBytes is the number of bytes read from storage.
	ReadBytes uint64

	// WriteBytes is the number of bytes written to storage.
	WriteBytes uint64
}

var _ SystemStatsProvider = (*SystemCollector)(nil)

// WithSystemCollectorClock sets the clock for the SystemCollector.
//
// Takes clk (clock.Clock) which provides the time source for the collector.
//
// Returns SystemCollectorOption which configures the clock when applied.
func WithSystemCollectorClock(clk clock.Clock) SystemCollectorOption {
	return func(c *SystemCollector) {
		c.clock = clk
	}
}

// WithListenAddress sets the monitoring server listen address reported in stats.
//
// Takes addr (string) which is the address the monitoring server listens on.
//
// Returns SystemCollectorOption which configures the listen address.
func WithListenAddress(addr string) SystemCollectorOption {
	return func(c *SystemCollector) {
		c.listenAddr = addr
	}
}

// buildPublicProcessInfo gathers OS-level process details and converts them
// to the public ProcessInfo struct.
//
// Returns ProcessInfo which contains process ID, file descriptors, memory,
// and I/O statistics.
func buildPublicProcessInfo() ProcessInfo {
	p := buildProcessInfo()
	return ProcessInfo{
		Hostname:            p.Hostname,
		Executable:          p.Executable,
		CWD:                 p.CWD,
		PID:                 safeconv.IntToInt32(p.PID),
		ThreadCount:         safeconv.IntToInt32(p.ThreadCount),
		FDCount:             safeconv.IntToInt32(p.FDCount),
		UID:                 safeconv.IntToInt32(p.UID),
		GID:                 safeconv.IntToInt32(p.GID),
		PPID:                safeconv.IntToInt32(p.PPID),
		RSS:                 p.RSS,
		CgroupMemoryLimit:   readCgroupMemoryLimit(),
		CgroupMemoryCurrent: readCgroupMemoryCurrent(),
		IoReadBytes:         p.IoReadBytes,
		IoWriteBytes:        p.IoWriteBytes,
		IoRchar:             p.IoRchar,
		IoWchar:             p.IoWchar,
		MaxOpenFilesSoft:    p.MaxOpenFilesSoft,
		MaxOpenFilesHard:    p.MaxOpenFilesHard,
	}
}

// readProcessCPUTime reads the process CPU time from /proc/self/stat.
//
// Returns uint64 which is the total CPU ticks (utime + stime), or 0 if the
// file cannot be read or parsed.
func readProcessCPUTime() uint64 {
	file, err := os.Open("/proc/self/stat")
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0
	}
	line := scanner.Text()

	closeParenIndex := strings.LastIndex(line, ")")
	if closeParenIndex < 0 || closeParenIndex+2 >= len(line) {
		return 0
	}

	fields := strings.Fields(line[closeParenIndex+2:])
	if len(fields) < systemStatFileParts {
		return 0
	}

	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}

	return utime + stime
}

// buildBuildInfo returns build-time information.
//
// Returns internalBuildInfo which contains version, commit, build time, and
// runtime details including VCS metadata from debug.ReadBuildInfo.
func buildBuildInfo() internalBuildInfo {
	info := internalBuildInfo{
		GoVersion: runtime.Version(),
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		info.ModulePath = bi.Main.Path
		info.ModuleVersion = bi.Main.Version

		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.modified":
				info.VCSModified = s.Value == "true"
			case "vcs.time":
				info.VCSTime = s.Value
			}
		}
	}

	return info
}

// buildProcessInfo returns OS-level process information.
//
// Returns internalProcessInfo which contains the current process ID, thread
// count, file descriptor count, resident set size, and I/O statistics.
func buildProcessInfo() internalProcessInfo {
	hostname, _ := os.Hostname()
	executable, _ := os.Executable()
	cwd, _ := os.Getwd()
	softFD, hardFD := readMaxOpenFiles()
	ioStats := readIOStats()

	return internalProcessInfo{
		Hostname:         hostname,
		Executable:       executable,
		CWD:              cwd,
		PID:              os.Getpid(),
		ThreadCount:      readThreadCount(),
		FDCount:          readFDCount(),
		UID:              os.Getuid(),
		GID:              os.Getgid(),
		PPID:             os.Getppid(),
		RSS:              readRSS(),
		IoReadBytes:      ioStats.ReadBytes,
		IoWriteBytes:     ioStats.WriteBytes,
		IoRchar:          ioStats.Rchar,
		IoWchar:          ioStats.Wchar,
		MaxOpenFilesSoft: softFD,
		MaxOpenFilesHard: hardFD,
	}
}

// buildRuntimeConfig returns runtime configuration values.
//
// Returns internalRuntimeConfig which contains the GOGC and GOMEMLIMIT settings
// from environment variables or their default values.
func buildRuntimeConfig() internalRuntimeConfig {
	gogc := os.Getenv("GOGC")
	if gogc == "" {
		gogc = defaultGOGC
	}

	gomemlimit := os.Getenv("GOMEMLIMIT")
	if gomemlimit == "" {
		if limit := debug.SetMemoryLimit(-1); limit > 0 {
			gomemlimit = formatBytes(limit)
		} else {
			gomemlimit = "unlimited"
		}
	}

	return internalRuntimeConfig{
		GOGC:       gogc,
		GOMEMLIMIT: gomemlimit,
		Compiler:   runtime.Compiler,
	}
}

// readThreadCount reads the thread count from /proc/self/status.
//
// Returns int which is the number of threads, or 0 if the file cannot be read
// or the thread count line is not found.
func readThreadCount() int {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Threads:") {
			continue
		}

		return parseThreadCountLine(line)
	}
	return 0
}

// parseThreadCountLine extracts the thread count from a /proc/self/status line.
//
// Takes line (string) which is a line from /proc/self/status containing thread
// count information.
//
// Returns int which is the parsed thread count, or 0 if parsing fails.
func parseThreadCountLine(line string) int {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}

	count, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return count
}

// readFDCount reads the number of open file descriptors.
//
// Returns int which is the count of open file descriptors, or 0 if the count
// cannot be determined.
func readFDCount() int {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0
	}
	return len(entries)
}

// readRSS reads the resident set size from /proc/self/statm.
//
// Returns uint64 which is the RSS in bytes, or 0 if the file cannot be read
// or parsed.
func readRSS() uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}

	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	return pages * pageSize
}

// readMaxOpenFiles reads the soft and hard limits for open file descriptors
// from /proc/self/limits.
//
// Returns int64 which is the soft limit, or 0 on non-Linux or parse failure.
// Returns int64 which is the hard limit, or 0 on non-Linux or parse failure.
func readMaxOpenFiles() (soft, hard int64) {
	file, err := os.Open("/proc/self/limits")
	if err != nil {
		return 0, 0
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Max open files") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < limitsLineFieldCount {
			return 0, 0
		}

		s, err1 := strconv.ParseInt(fields[3], 10, 64)
		h, err2 := strconv.ParseInt(fields[4], 10, 64)
		if err1 != nil || err2 != nil {
			return 0, 0
		}

		return s, h
	}

	return 0, 0
}

// readIOStats reads I/O counters from /proc/self/io.
//
// Returns internalIOStats which contains the read/write byte counters, or a
// zero struct on non-Linux or parse failure.
func readIOStats() internalIOStats {
	file, err := os.Open("/proc/self/io")
	if err != nil {
		return internalIOStats{}
	}
	defer func() { _ = file.Close() }()

	var stats internalIOStats

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		key, valString, found := strings.Cut(line, ": ")
		if !found {
			continue
		}

		value, err := strconv.ParseUint(strings.TrimSpace(valString), 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "rchar":
			stats.Rchar = value
		case "wchar":
			stats.Wchar = value
		case "read_bytes":
			stats.ReadBytes = value
		case "write_bytes":
			stats.WriteBytes = value
		}
	}

	return stats
}

// readSystemUptime reads the host system uptime from /proc/uptime.
//
// Returns int64 which is the uptime in milliseconds, or 0 on non-Linux or
// parse failure.
func readSystemUptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	return int64(seconds * 1000)
}

// readCgroupPath reads the cgroup v2 path from /proc/self/cgroup.
//
// Returns string which is the cgroup path (from the "0::" line), or empty on
// non-Linux or parse failure.
func readCgroupPath() string {
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if suffix, ok := strings.CutPrefix(line, "0::"); ok {
			return suffix
		}
	}

	return ""
}

// formatBytes formats bytes as a human-readable string.
//
// Takes b (int64) which is the byte count to format.
//
// Returns string which is the formatted size with appropriate unit suffix.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + "B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(b)/float64(div), 'f', 1, 64) + string("KMGTPE"[exp]) + "iB"
}
