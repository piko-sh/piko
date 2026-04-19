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

package inspector

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// shortCommitHashLen is the number of leading characters kept when
// truncating a commit hash for the overview tables.
const shortCommitHashLen = 8

// gcPercentMultiplier converts a fraction in [0,1] to a percentage so
// the "1.2300%" CPU-fraction label matches what /proc-style tooling
// emits. Defined here so CLI and TUI share the constant.
const gcPercentMultiplier = 100

// ShortCommitHash returns the leading bytes of a commit hash for use in
// compact overview tables. Hashes shorter than the truncation length
// are returned unchanged.
//
// Takes commit (string) which is the full commit hash.
//
// Returns string with at most shortCommitHashLen characters.
func ShortCommitHash(commit string) string {
	if len(commit) > shortCommitHashLen {
		return commit[:shortCommitHashLen]
	}
	return commit
}

// FormatGCCPUFraction renders a GC CPU fraction in [0,1] as a
// four-decimal percentage so output reads e.g. "1.2300%".
//
// Takes fraction (float64) which is the GC CPU share reported by the
// runtime.
//
// Returns string with a trailing "%" sign.
func FormatGCCPUFraction(fraction float64) string {
	return fmt.Sprintf("%.4f%%", fraction*gcPercentMultiplier)
}

// FormatNanosAsDuration renders a nanosecond count as a friendly Go
// duration. Returns a hyphen for non-positive values so the
// caller can show "Last GC" with a glyph rather than a stale 1970 timestamp.
//
// Takes nanoseconds (int64) which is the nanosecond count.
//
// Returns string with the formatted duration or EmDashGlyph.
func FormatNanosAsDuration(nanoseconds int64) string {
	if nanoseconds <= 0 {
		return hyphenGlyph
	}
	return time.Duration(nanoseconds).String()
}

// BuildBuildDetailRows renders the canonical row set for a BuildInfo
// proto message. Rows are ordered for descending readability: identity
// first (version, commit, go version), platform second (os, arch),
// build/module metadata last.
//
// Takes build (*pb.BuildInfo) which carries the build metadata; nil
// returns a nil slice so the caller can detect "no data".
//
// Returns []DetailRow which is the labelled key/value pairs for the
// detail-pane.
func BuildBuildDetailRows(build *pb.BuildInfo) []DetailRow {
	if build == nil {
		return nil
	}
	return []DetailRow{
		{Label: "Version", Value: build.GetVersion()},
		{Label: "Commit", Value: build.GetCommit()},
		{Label: "Go Version", Value: build.GetGoVersion()},
		{Label: "OS", Value: build.GetOs()},
		{Label: "Arch", Value: build.GetArch()},
		{Label: "Build Time", Value: build.GetBuildTime()},
		{Label: "Module", Value: build.GetModulePath()},
		{Label: "Module Version", Value: build.GetModuleVersion()},
		{Label: "VCS Modified", Value: strconv.FormatBool(build.GetVcsModified())},
		{Label: "VCS Time", Value: build.GetVcsTime()},
	}
}

// BuildRuntimeDetailRows renders the canonical row set for a
// RuntimeInfo proto message. The compiler row is included so callers
// can see whether the binary was built with gc or gccgo.
//
// Takes runtime (*pb.RuntimeInfo) which carries the runtime config;
// nil returns a nil slice.
//
// Returns []DetailRow describing GOGC, GOMEMLIMIT, and the compiler.
func BuildRuntimeDetailRows(runtime *pb.RuntimeInfo) []DetailRow {
	if runtime == nil {
		return nil
	}
	return []DetailRow{
		{Label: "GOGC", Value: runtime.GetGogc()},
		{Label: "GOMEMLIMIT", Value: runtime.GetGomemlimit()},
		{Label: "Compiler", Value: runtime.GetCompiler()},
	}
}

// BuildMemoryDetailRows renders the canonical row set for a MemoryInfo
// proto message.
//
// Rows are grouped by purpose: top-level allocations first, then the
// heap breakdown, then the stack and runtime metadata pools, then
// counters.
//
// Takes memory (*pb.MemoryInfo) which carries the memory snapshot; nil
// returns a nil slice.
//
// Returns []DetailRow describing every sampled memory metric.
func BuildMemoryDetailRows(memory *pb.MemoryInfo) []DetailRow {
	if memory == nil {
		return nil
	}
	return []DetailRow{
		{Label: "Alloc", Value: FormatBytes(memory.GetAlloc())},
		{Label: "Total Alloc", Value: FormatBytes(memory.GetTotalAlloc())},
		{Label: "Sys", Value: FormatBytes(memory.GetSys())},
		{Label: "Heap Alloc", Value: FormatBytes(memory.GetHeapAlloc())},
		{Label: "Heap Sys", Value: FormatBytes(memory.GetHeapSys())},
		{Label: "Heap Idle", Value: FormatBytes(memory.GetHeapIdle())},
		{Label: "Heap In Use", Value: FormatBytes(memory.GetHeapInuse())},
		{Label: "Heap Objects", Value: strconv.FormatUint(memory.GetHeapObjects(), 10)},
		{Label: "Heap Released", Value: FormatBytes(memory.GetHeapReleased())},
		{Label: "Stack In Use", Value: FormatBytes(memory.GetStackInuse())},
		{Label: "Stack Sys", Value: FormatBytes(memory.GetStackSys())},
		{Label: "MSpan In Use", Value: FormatBytes(memory.GetMspanInuse())},
		{Label: "MSpan Sys", Value: FormatBytes(memory.GetMspanSys())},
		{Label: "MCache In Use", Value: FormatBytes(memory.GetMcacheInuse())},
		{Label: "MCache Sys", Value: FormatBytes(memory.GetMcacheSys())},
		{Label: "GC Sys", Value: FormatBytes(memory.GetGcSys())},
		{Label: "Other Sys", Value: FormatBytes(memory.GetOtherSys())},
		{Label: "BuckHash Sys", Value: FormatBytes(memory.GetBuckhashSys())},
		{Label: "Lookups", Value: strconv.FormatUint(memory.GetLookups(), 10)},
		{Label: "Mallocs", Value: strconv.FormatUint(memory.GetMallocs(), 10)},
		{Label: "Frees", Value: strconv.FormatUint(memory.GetFrees(), 10)},
		{Label: "Live Objects", Value: strconv.FormatUint(memory.GetLiveObjects(), 10)},
	}
}

// BuildGCDetailRows renders the canonical row set for a GCInfo proto
// message.
//
// When the proto carries a non-empty RecentPauses list the trailing
// "Recent Pauses" row joins each pause via FormatNanosAsDuration so
// callers get a consistent rendering of the recent-pause history.
//
// Takes gc (*pb.GCInfo) which carries the GC snapshot; nil returns a
// nil slice.
//
// Returns []DetailRow describing cycles, pauses, CPU fraction, and the
// next GC heap target.
func BuildGCDetailRows(gc *pb.GCInfo) []DetailRow {
	if gc == nil {
		return nil
	}
	lastGC := time.Time{}
	if lastGcNs := gc.GetLastGcNs(); lastGcNs > 0 {
		lastGC = time.Unix(0, lastGcNs)
	}
	rows := []DetailRow{
		{Label: "Cycles", Value: strconv.FormatUint(uint64(gc.GetNumGc()), 10)},
		{Label: "Forced Cycles", Value: strconv.FormatUint(uint64(gc.GetNumForcedGc()), 10)},
		{Label: "Last Pause", Value: FormatNanosAsDuration(safeconv.Uint64ToInt64(gc.GetLastPauseNs()))},
		{Label: "Total Pause", Value: FormatNanosAsDuration(safeconv.Uint64ToInt64(gc.GetPauseTotalNs()))},
		{Label: "CPU Fraction", Value: FormatGCCPUFraction(gc.GetGcCpuFraction())},
		{Label: "Next GC", Value: FormatBytes(gc.GetNextGc())},
		{Label: "Last GC", Value: FormatDetailTime(lastGC)},
	}
	if pauses := gc.GetRecentPauses(); len(pauses) > 0 {
		parts := make([]string, len(pauses))
		for i, nanoseconds := range pauses {
			parts[i] = FormatNanosAsDuration(safeconv.Uint64ToInt64(nanoseconds))
		}
		rows = append(rows, DetailRow{Label: "Recent Pauses", Value: strings.Join(parts, ", ")})
	}
	return rows
}

// BuildProcessDetailRows renders the canonical row set for a
// ProcessInfo proto message.
//
// Rows are grouped: identifiers, thread / file-descriptor counts, RSS,
// I/O counters, and the textual hostname / executable / cwd labels at
// the end.
//
// Takes process (*pb.ProcessInfo) which carries the process snapshot;
// nil returns a nil slice.
//
// Returns []DetailRow describing every process metric exposed by the
// monitoring API.
func BuildProcessDetailRows(process *pb.ProcessInfo) []DetailRow {
	if process == nil {
		return nil
	}
	return []DetailRow{
		{Label: "PID", Value: strconv.FormatInt(int64(process.GetPid()), 10)},
		{Label: "PPID", Value: strconv.FormatInt(int64(process.GetPpid()), 10)},
		{Label: "UID", Value: strconv.FormatInt(int64(process.GetUid()), 10)},
		{Label: "GID", Value: strconv.FormatInt(int64(process.GetGid()), 10)},
		{Label: "Threads", Value: strconv.FormatInt(int64(process.GetThreadCount()), 10)},
		{Label: "File Descriptors", Value: strconv.FormatInt(int64(process.GetFdCount()), 10)},
		{Label: "Max Open Files (Soft)", Value: strconv.FormatInt(process.GetMaxOpenFilesSoft(), 10)},
		{Label: "Max Open Files (Hard)", Value: strconv.FormatInt(process.GetMaxOpenFilesHard(), 10)},
		{Label: "RSS", Value: FormatBytes(process.GetRss())},
		{Label: "I/O Read Bytes", Value: FormatBytes(process.GetIoReadBytes())},
		{Label: "I/O Write Bytes", Value: FormatBytes(process.GetIoWriteBytes())},
		{Label: "I/O Read Total", Value: FormatBytes(process.GetIoRchar())},
		{Label: "I/O Write Total", Value: FormatBytes(process.GetIoWchar())},
		{Label: "Hostname", Value: process.GetHostname()},
		{Label: "Executable", Value: process.GetExecutable()},
		{Label: "CWD", Value: process.GetCwd()},
	}
}
