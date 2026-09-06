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

const (
	// shortCommitHashLen is the number of leading characters kept when truncating a commit
	// hash for the overview tables.
	shortCommitHashLen = 8

	// gcPercentMultiplier converts a fraction in [0,1] to a percentage so the "1.2300%"
	// CPU-fraction label matches what /proc-style tooling emits. Defined here so CLI and TUI
	// share the constant.
	gcPercentMultiplier = 100
)

// ShortCommitHash returns the leading bytes of a commit hash for use in compact overview
// tables. Hashes shorter than the truncation length are returned unchanged.
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

// FormatGCCPUFraction renders a GC CPU fraction in [0,1] as a four-decimal percentage so
// output reads e.g. "1.2300%".
//
// Takes fraction (float64) which is the GC CPU share reported by the runtime.
//
// Returns string with a trailing "%" sign.
func FormatGCCPUFraction(fraction float64) string {
	return fmt.Sprintf("%.4f%%", fraction*gcPercentMultiplier)
}

// FormatNanosAsDuration renders a nanosecond count as a friendly Go duration. Returns a
// hyphen for non-positive values so the caller can show "Last GC" with a glyph rather
// than a stale 1970 timestamp.
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

// BuildBuildDetailRows renders the canonical row set for a BuildInfo proto message. Rows
// are ordered for descending readability: identity first (version, commit, go version),
// platform second (os, arch), build/module metadata last.
//
// Takes build (*pb.BuildInfo) which carries the build metadata; nil returns a nil slice
// so the caller can detect "no data".
//
// Returns []DetailRow which is the labelled key/value pairs for the detail-pane.
func BuildBuildDetailRows(build *pb.BuildInfo) []DetailRow {
	if build == nil {
		return nil
	}
	return []DetailRow{
		NewDetailRow("Version", build.GetVersion()),
		NewDetailRow("Commit", build.GetCommit()),
		NewDetailRow("Go Version", build.GetGoVersion()),
		NewDetailRow("OS", build.GetOs()),
		NewDetailRow("Arch", build.GetArch()),
		NewDetailRow("Build Time", build.GetBuildTime()),
		NewDetailRow("Module", build.GetModulePath()),
		NewDetailRow("Module Version", build.GetModuleVersion()),
		NewDetailRow("VCS Modified", strconv.FormatBool(build.GetVcsModified())),
		NewDetailRow("VCS Time", build.GetVcsTime()),
	}
}

// BuildRuntimeDetailRows renders the canonical row set for a RuntimeInfo proto message.
// The compiler row is included so callers can see whether the binary was built with gc or
// gccgo.
//
// Takes runtime (*pb.RuntimeInfo) which carries the runtime config; nil returns a nil
// slice.
//
// Returns []DetailRow describing GOGC, GOMEMLIMIT, and the compiler.
func BuildRuntimeDetailRows(runtime *pb.RuntimeInfo) []DetailRow {
	if runtime == nil {
		return nil
	}
	return []DetailRow{
		NewDetailRow("GOGC", runtime.GetGogc()),
		NewDetailRow("GOMEMLIMIT", runtime.GetGomemlimit()),
		NewDetailRow("Compiler", runtime.GetCompiler()),
	}
}

// BuildMemoryDetailRows renders the canonical row set for a MemoryInfo proto message.
//
// Rows are grouped by purpose: top-level allocations first, then the heap breakdown, then
// the stack and runtime metadata pools, then counters.
//
// Takes memory (*pb.MemoryInfo) which carries the memory snapshot; nil returns a nil
// slice.
//
// Returns []DetailRow describing every sampled memory metric.
func BuildMemoryDetailRows(memory *pb.MemoryInfo) []DetailRow {
	if memory == nil {
		return nil
	}
	return []DetailRow{
		NewDetailRow("Alloc", FormatBytes(memory.GetAlloc())),
		NewDetailRow("Total Alloc", FormatBytes(memory.GetTotalAlloc())),
		NewDetailRow("Sys", FormatBytes(memory.GetSys())),
		NewDetailRow("Heap Alloc", FormatBytes(memory.GetHeapAlloc())),
		NewDetailRow("Heap Sys", FormatBytes(memory.GetHeapSys())),
		NewDetailRow("Heap Idle", FormatBytes(memory.GetHeapIdle())),
		NewDetailRow("Heap In Use", FormatBytes(memory.GetHeapInuse())),
		NewDetailRow("Heap Objects", strconv.FormatUint(memory.GetHeapObjects(), 10)),
		NewDetailRow("Heap Released", FormatBytes(memory.GetHeapReleased())),
		NewDetailRow("Stack In Use", FormatBytes(memory.GetStackInuse())),
		NewDetailRow("Stack Sys", FormatBytes(memory.GetStackSys())),
		NewDetailRow("MSpan In Use", FormatBytes(memory.GetMspanInuse())),
		NewDetailRow("MSpan Sys", FormatBytes(memory.GetMspanSys())),
		NewDetailRow("MCache In Use", FormatBytes(memory.GetMcacheInuse())),
		NewDetailRow("MCache Sys", FormatBytes(memory.GetMcacheSys())),
		NewDetailRow("GC Sys", FormatBytes(memory.GetGcSys())),
		NewDetailRow("Other Sys", FormatBytes(memory.GetOtherSys())),
		NewDetailRow("BuckHash Sys", FormatBytes(memory.GetBuckhashSys())),
		NewDetailRow("Lookups", strconv.FormatUint(memory.GetLookups(), 10)),
		NewDetailRow("Mallocs", strconv.FormatUint(memory.GetMallocs(), 10)),
		NewDetailRow("Frees", strconv.FormatUint(memory.GetFrees(), 10)),
		NewDetailRow("Live Objects", strconv.FormatUint(memory.GetLiveObjects(), 10)),
	}
}

// BuildGCDetailRows renders the canonical row set for a GCInfo proto message.
//
// When the proto carries a non-empty RecentPauses list the trailing "Recent Pauses" row
// joins each pause via FormatNanosAsDuration so callers get a consistent rendering of the
// recent-pause history.
//
// Takes gc (*pb.GCInfo) which carries the GC snapshot; nil returns a nil slice.
//
// Returns []DetailRow describing cycles, pauses, CPU fraction, and the next GC heap
// target.
func BuildGCDetailRows(gc *pb.GCInfo) []DetailRow {
	if gc == nil {
		return nil
	}
	lastGC := time.Time{}
	if lastGcNs := gc.GetLastGcNs(); lastGcNs > 0 {
		lastGC = time.Unix(0, lastGcNs)
	}
	rows := []DetailRow{
		NewDetailRow("Cycles", strconv.FormatUint(uint64(gc.GetNumGc()), 10)),
		NewDetailRow("Forced Cycles", strconv.FormatUint(uint64(gc.GetNumForcedGc()), 10)),
		NewDetailRow("Last Pause", FormatNanosAsDuration(safeconv.Uint64ToInt64(gc.GetLastPauseNs()))),
		NewDetailRow("Total Pause", FormatNanosAsDuration(safeconv.Uint64ToInt64(gc.GetPauseTotalNs()))),
		NewDetailRow("CPU Fraction", FormatGCCPUFraction(gc.GetGcCpuFraction())),
		NewDetailRow("Next GC", FormatBytes(gc.GetNextGc())),
		NewDetailRow("Last GC", FormatDetailTime(lastGC)),
	}
	if pauses := gc.GetRecentPauses(); len(pauses) > 0 {
		parts := make([]string, len(pauses))
		for i, nanoseconds := range pauses {
			parts[i] = FormatNanosAsDuration(safeconv.Uint64ToInt64(nanoseconds))
		}
		rows = append(rows, NewDetailRow("Recent Pauses", strings.Join(parts, ", ")))
	}
	return rows
}

// BuildProcessDetailRows renders the canonical row set for a ProcessInfo proto message.
//
// Rows are grouped: identifiers, thread / file-descriptor counts, RSS, I/O counters, and
// the textual hostname / executable / cwd labels at the end.
//
// Takes process (*pb.ProcessInfo) which carries the process snapshot; nil returns a nil
// slice.
//
// Returns []DetailRow describing every process metric exposed by the monitoring API.
func BuildProcessDetailRows(process *pb.ProcessInfo) []DetailRow {
	if process == nil {
		return nil
	}
	return []DetailRow{
		NewDetailRow("PID", strconv.FormatInt(int64(process.GetPid()), 10)),
		NewDetailRow("PPID", strconv.FormatInt(int64(process.GetPpid()), 10)),
		NewDetailRow("UID", strconv.FormatInt(int64(process.GetUid()), 10)),
		NewDetailRow("GID", strconv.FormatInt(int64(process.GetGid()), 10)),
		NewDetailRow("Threads", strconv.FormatInt(int64(process.GetThreadCount()), 10)),
		NewDetailRow("File Descriptors", strconv.FormatInt(int64(process.GetFdCount()), 10)),
		NewDetailRow("Max Open Files (Soft)", strconv.FormatInt(process.GetMaxOpenFilesSoft(), 10)),
		NewDetailRow("Max Open Files (Hard)", strconv.FormatInt(process.GetMaxOpenFilesHard(), 10)),
		NewDetailRow("RSS", FormatBytes(process.GetRss())),
		NewDetailRow("I/O Read Bytes", FormatBytes(process.GetIoReadBytes())),
		NewDetailRow("I/O Write Bytes", FormatBytes(process.GetIoWriteBytes())),
		NewDetailRow("I/O Read Total", FormatBytes(process.GetIoRchar())),
		NewDetailRow("I/O Write Total", FormatBytes(process.GetIoWchar())),
		NewDetailRow("Hostname", process.GetHostname()),
		NewDetailRow("Executable", process.GetExecutable()),
		NewDetailRow("CWD", process.GetCwd()),
	}
}
