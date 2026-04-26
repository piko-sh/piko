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

package tui_domain

import (
	"fmt"
	"time"

	"piko.sh/piko/cmd/piko/internal/inspector"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// DetailView renders the detail-pane body for the section currently
// under the cursor, with a high-fidelity ntcharts trend graph below
// when the section has historic data (memory, CPU, goroutines, GC).
// When no row is selected the system overview is shown without a
// chart.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *SystemPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	series, title := p.detailChartFor(p.currentSectionKey())
	if series == nil {
		return RenderDetailBody(nil, body, width, height)
	}
	return RenderDetailBodyWithChart(nil, body, series, title, width, height)
}

// currentSectionKey returns the key of the section under the cursor,
// or empty when none.
//
// Returns string which is the section key, or empty when no row is selected.
func (p *SystemPanel) currentSectionKey() string {
	item := p.GetItemAtCursor()
	if item == nil {
		return ""
	}
	return item.key
}

// detailChartFor returns a chart series + title for the selected
// section, or (nil, "") when the section has no chart-friendly data.
//
// Takes section (string) which is the section key.
//
// Returns []ChartSeries and string label.
func (p *SystemPanel) detailChartFor(section string) ([]ChartSeries, string) {
	switch section {
	case sectionMemory:
		return historyRingSeries("heap-alloc", p.heapHistory, SeverityWarning), "heap allocation"
	case sectionCPU:
		return historyRingSeries("cpu-millicores", p.cpuHistory, SeverityHealthy), "CPU (millicores)"
	case sectionGoroutines:
		return historyRingSeries("goroutines", p.goroutineHistory, SeverityHealthy), "goroutine count"
	case sectionGC:
		return historyRingSeries("gc-pause-µs", p.gcPauseHistory, SeverityWarning), "GC pause (µs)"
	}
	return nil, ""
}

// historyRingSeries converts a HistoryRing into a ChartSeries with
// even-spaced timestamps. Returns nil when the ring is empty.
//
// Takes name (string) which becomes the series name.
// Takes ring (*HistoryRing) which holds the samples.
// Takes severity (Severity) which drives the line colour.
//
// Returns []ChartSeries with one entry, or nil when no data is
// available.
func historyRingSeries(name string, ring *HistoryRing, severity Severity) []ChartSeries {
	if ring == nil {
		return nil
	}
	values := ring.Values()
	if len(values) == 0 {
		return nil
	}
	points := make([]ChartPoint, len(values))
	step := metricHistoryStep
	start := time.Now().Add(-step * time.Duration(len(values)-1))
	for i, v := range values {
		points[i] = ChartPoint{
			Time:  start.Add(step * time.Duration(i)),
			Value: v,
		}
	}
	return []ChartSeries{{Name: name, Points: points, Severity: severity}}
}

// buildDetailBody assembles the structured detail content based on the
// current section cursor.
//
// Returns inspector.DetailBody which is the rendered detail content for the
// selected section, falling back to the system overview when no row
// is highlighted.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *SystemPanel) buildDetailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	stats := p.stats
	p.stateMutex.RUnlock()

	if stats == nil {
		return inspector.DetailBody{
			Title:    "System",
			Subtitle: "no data yet",
		}
	}

	if item := p.GetItemAtCursor(); item != nil {
		switch item.key {
		case "memory":
			return systemMemoryDetailBody(stats)
		case "gc":
			return systemGCDetailBody(stats)
		case "process":
			return systemProcessDetailBody(stats)
		case "build":
			return systemBuildDetailBody(stats)
		case "runtime":
			return systemRuntimeDetailBody(stats)
		case "uptime":
			return systemUptimeDetailBody(stats)
		}
	}
	return systemOverviewDetailBody(stats)
}

// systemOverviewDetailBody builds the system-wide overview detail body.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing CPU, goroutines, and heap headlines.
func systemOverviewDetailBody(s *SystemStats) inspector.DetailBody {
	rows := []inspector.DetailRow{
		{Label: "Uptime", Value: inspector.FormatDuration(s.Uptime)},
		{Label: "CPU (mC)", Value: fmt.Sprintf("%.0f", s.CPUMillicores)},
		{Label: "CPUs", Value: fmt.Sprintf(FormatPercentInt, s.NumCPU)},
		{Label: "GOMAXPROCS", Value: fmt.Sprintf(FormatPercentInt, s.GOMAXPROCS)},
		{Label: "Goroutines", Value: fmt.Sprintf(FormatPercentInt, s.NumGoroutines)},
		{Label: "CGO calls", Value: fmt.Sprintf(FormatPercentInt, s.NumCGOCalls)},
		{Label: "Heap alloc", Value: inspector.FormatBytes(s.Memory.HeapAlloc)},
		{Label: "Sys", Value: inspector.FormatBytes(s.Memory.Sys)},
	}
	return inspector.DetailBody{
		Title:    "System overview",
		Subtitle: fmt.Sprintf("up %s", inspector.FormatDuration(s.Uptime)),
		Sections: []inspector.DetailSection{{Heading: "Snapshot", Rows: rows}},
	}
}

// systemMemoryDetailBody builds the memory section detail body.
//
// Lifts row construction into inspector.BuildMemoryDetailRows so the CLI
// `piko info memory` and the TUI memory detail show the same fields in
// the same order. The TUI's lossy SystemMemoryStats does not carry the
// secondary "Sys" pools (mspan / mcache / etc.) so those rows render as
// "0 B" rather than being hidden.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing heap, stack, and allocation counters.
func systemMemoryDetailBody(s *SystemStats) inspector.DetailBody {
	return inspector.DetailBody{
		Title:    "Memory",
		Subtitle: inspector.FormatBytes(s.Memory.HeapAlloc) + " heap",
		Sections: []inspector.DetailSection{{
			Heading: "Memory stats",
			Rows:    inspector.BuildMemoryDetailRows(memoryStatsToProto(s.Memory)),
		}},
	}
}

// systemGCDetailBody builds the garbage collector section detail body.
//
// Lifts row construction into inspector.BuildGCDetailRows so the CLI
// `piko info gc` and the TUI GC detail share the same labels and value
// formats.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing GC cycles, pauses, and CPU fraction.
func systemGCDetailBody(s *SystemStats) inspector.DetailBody {
	return inspector.DetailBody{
		Title:    "Garbage collector",
		Subtitle: fmt.Sprintf("%d cycles", s.GC.NumGC),
		Sections: []inspector.DetailSection{{
			Heading: "GC stats",
			Rows:    inspector.BuildGCDetailRows(gcStatsToProto(s.GC)),
		}},
	}
}

// systemProcessDetailBody builds the process section detail body.
//
// Lifts row construction into inspector.BuildProcessDetailRows so the
// CLI `piko info process` and the TUI process detail share the same
// fields. Process fields not carried by the TUI domain (UID / GID /
// PPID / executable / hostname / I/O counters) render as zero or empty
// rather than being hidden.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing PID, thread count, FD count, and RSS.
func systemProcessDetailBody(s *SystemStats) inspector.DetailBody {
	return inspector.DetailBody{
		Title:    "Process",
		Subtitle: fmt.Sprintf("PID %d", s.Process.PID),
		Sections: []inspector.DetailSection{{
			Heading: "Process info",
			Rows:    inspector.BuildProcessDetailRows(processInfoToProto(s.Process)),
		}},
	}
}

// systemBuildDetailBody builds the build-info section detail body.
//
// Lifts row construction into inspector.BuildBuildDetailRows so the CLI
// `piko info build` and the TUI build detail share the same field set.
// The TUI's lossy SystemBuildInfo does not carry the module path / VCS
// metadata so those rows render empty.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing version, commit, build time, and
// platform.
func systemBuildDetailBody(s *SystemStats) inspector.DetailBody {
	return inspector.DetailBody{
		Title:    "Build",
		Subtitle: s.Build.Version,
		Sections: []inspector.DetailSection{{
			Heading: "Build info",
			Rows:    inspector.BuildBuildDetailRows(buildInfoToProto(s.Build)),
		}},
	}
}

// systemRuntimeDetailBody builds the runtime configuration section
// detail body.
//
// Reuses inspector.BuildRuntimeDetailRows for GOGC / GOMEMLIMIT /
// Compiler so CLI and TUI agree, and appends GOMAXPROCS + Go version
// rows that the runtime detail pane shows but the CLI's `piko info
// runtime` does not.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing GOGC, GOMEMLIMIT, GOMAXPROCS, and Go
// version.
func systemRuntimeDetailBody(s *SystemStats) inspector.DetailBody {
	rows := inspector.BuildRuntimeDetailRows(runtimeConfigToProto(s.Runtime))
	rows = append(rows,
		inspector.DetailRow{Label: "GOMAXPROCS", Value: fmt.Sprintf(FormatPercentInt, s.GOMAXPROCS)},
		inspector.DetailRow{Label: "Go version", Value: s.Build.GoVersion},
	)
	return inspector.DetailBody{
		Title:    "Runtime",
		Sections: []inspector.DetailSection{{Heading: "Runtime config", Rows: rows}},
	}
}

// systemUptimeDetailBody builds the uptime section detail body.
//
// Takes s (*SystemStats) which is the current snapshot.
//
// Returns inspector.DetailBody describing uptime and process start time.
func systemUptimeDetailBody(s *SystemStats) inspector.DetailBody {
	started := s.Timestamp.Add(-s.Uptime)
	rows := []inspector.DetailRow{
		{Label: "Uptime", Value: inspector.FormatDuration(s.Uptime)},
		{Label: "Started", Value: inspector.FormatDetailTime(started)},
	}
	return inspector.DetailBody{
		Title:    "Uptime",
		Subtitle: inspector.FormatDuration(s.Uptime),
		Sections: []inspector.DetailSection{{Heading: "Process lifetime", Rows: rows}},
	}
}

// buildInfoToProto adapts the lossy SystemBuildInfo domain type into
// the proto BuildInfo expected by inspector.BuildBuildDetailRows.
// Fields the domain does not carry stay at their zero value.
//
// Takes b (SystemBuildInfo) which is the domain build snapshot.
//
// Returns *pb.BuildInfo with the carried fields populated.
func buildInfoToProto(b SystemBuildInfo) *pb.BuildInfo {
	return &pb.BuildInfo{
		Version:   b.Version,
		Commit:    b.Commit,
		GoVersion: b.GoVersion,
		Os:        b.OS,
		Arch:      b.Arch,
		BuildTime: b.BuildTime,
	}
}

// runtimeConfigToProto adapts SystemRuntimeConfig into the proto
// RuntimeInfo expected by inspector.BuildRuntimeDetailRows.
//
// Takes r (SystemRuntimeConfig) which is the domain runtime snapshot.
//
// Returns *pb.RuntimeInfo with GOGC / GOMEMLIMIT populated; the
// compiler row stays empty because the domain does not carry it.
func runtimeConfigToProto(r SystemRuntimeConfig) *pb.RuntimeInfo {
	return &pb.RuntimeInfo{
		Gogc:       r.GOGC,
		Gomemlimit: r.GOMEMLIMIT,
	}
}

// memoryStatsToProto adapts SystemMemoryStats into the proto MemoryInfo
// expected by inspector.BuildMemoryDetailRows. The TUI domain only
// carries the high-level allocation counters, so the secondary "Sys"
// pools (mspan / mcache / etc.) render as zero-byte rows.
//
// Takes m (SystemMemoryStats) which is the domain memory snapshot.
//
// Returns *pb.MemoryInfo with the carried fields populated.
func memoryStatsToProto(m SystemMemoryStats) *pb.MemoryInfo {
	return &pb.MemoryInfo{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapObjects:  m.HeapObjects,
		HeapReleased: m.HeapReleased,
		StackSys:     m.StackSys,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		LiveObjects:  m.LiveObjects,
	}
}

// gcStatsToProto adapts SystemGCStats into the proto GCInfo expected by
// inspector.BuildGCDetailRows.
//
// Takes g (SystemGCStats) which is the domain GC snapshot.
//
// Returns *pb.GCInfo with the carried fields populated.
func gcStatsToProto(g SystemGCStats) *pb.GCInfo {
	return &pb.GCInfo{
		NumGc:         g.NumGC,
		LastGcNs:      g.LastGC,
		PauseTotalNs:  g.PauseTotalNs,
		LastPauseNs:   g.LastPauseNs,
		GcCpuFraction: g.GCCPUFraction,
		NextGc:        g.NextGC,
		RecentPauses:  g.RecentPauses,
	}
}

// processInfoToProto adapts SystemProcessInfo into the proto ProcessInfo
// expected by inspector.BuildProcessDetailRows. The TUI domain only
// carries the four headline counters so most rows (UID / GID / hostname
// etc.) render as zero or empty in the detail pane.
//
// Takes p (SystemProcessInfo) which is the domain process snapshot.
//
// Returns *pb.ProcessInfo with PID / thread count / FD count / RSS
// populated.
func processInfoToProto(p SystemProcessInfo) *pb.ProcessInfo {
	return &pb.ProcessInfo{
		Pid:         safeconv.IntToInt32(p.PID),
		ThreadCount: safeconv.IntToInt32(p.ThreadCount),
		FdCount:     safeconv.IntToInt32(p.FDCount),
		Rss:         p.RSS,
	}
}
