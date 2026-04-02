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

package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// infoCategoryList is the formatted list of available category names.
const infoCategoryList = "system, build, runtime, memory, gc, process"

var (
	// infoCategoryOrder defines the display order for the overview.
	infoCategoryOrder = []struct {
		key   string
		label string
	}{
		{key: "system", label: "System"},
		{key: "build", label: "Build"},
		{key: "runtime", label: "Runtime"},
		{key: "memory", label: "Memory"},
		{key: "gc", label: "GC"},
		{key: "process", label: "Process"},
	}

	// infoCategories maps category names to their detail handler functions.
	infoCategories = map[string]func(*pb.GetSystemStatsResponse, *Printer){
		"system":  infoSystem,
		"build":   infoBuild,
		"runtime": infoRuntime,
		"memory":  infoMemory,
		"gc":      infoGC,
		"process": infoProcess,
	}

	// infoOverviewHandlers maps category names to their overview handler functions.
	infoOverviewHandlers = map[string]func(*pb.GetSystemStatsResponse, *Printer){
		"system":  infoSystemOverview,
		"build":   infoBuildOverview,
		"runtime": infoRuntimeOverview,
		"memory":  infoMemoryOverview,
		"gc":      infoGCOverview,
		"process": infoProcessOverview,
	}

	// infoFormats lists the output formats supported by the info command.
	infoFormats = []string{"table", "wide", "json"}

	// infoDetailColumns defines the two-column layout for detail views.
	infoDetailColumns = []Column{
		{Header: "FIELD"},
		{Header: "VALUE"},
	}

	systemOverviewColumns = []Column{
		{Header: "UPTIME"},
		{Header: "CPU"},
		{Header: "GOROUTINES"},
		{Header: "CGO CALLS"},
	}

	buildOverviewColumns = []Column{
		{Header: "VERSION"},
		{Header: "COMMIT"},
		{Header: "GO"},
		{Header: "OS/ARCH"},
	}

	runtimeOverviewColumns = []Column{
		{Header: "GOGC"},
		{Header: "GOMEMLIMIT"},
		{Header: "COMPILER"},
	}

	memoryOverviewColumns = []Column{
		{Header: "HEAP ALLOC"},
		{Header: "SYS TOTAL"},
		{Header: "LIVE OBJECTS"},
		{Header: "HEAP OBJECTS"},
	}

	gcOverviewColumns = []Column{
		{Header: "CYCLES"},
		{Header: "LAST PAUSE"},
		{Header: "CPU FRACTION"},
		{Header: "NEXT GC"},
	}

	processOverviewColumns = []Column{
		{Header: "PID"},
		{Header: "THREADS"},
		{Header: "FDS"},
		{Header: "RSS"},
		{Header: "HOSTNAME"},
	}
)

// runInfo displays system information, optionally filtered to a single
// category.
//
// Takes cc (*CommandContext) which provides the connection and output options.
// Takes arguments ([]string) which specifies the optional category to filter by.
//
// Returns error when the output format is invalid, the gRPC call fails, or the
// category is unknown.
func runInfo(ctx context.Context, cc *CommandContext, arguments []string) error {
	if err := validateOutputFormat(cc.Opts.Output, "info", infoFormats); err != nil {
		return err
	}

	response, err := cc.Conn.MetricsClient().GetSystemStats(ctx, &pb.GetSystemStatsRequest{})
	if err != nil {
		return grpcError("fetching system stats", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	if len(arguments) == 0 {
		infoOverview(response, p, cc.Stdout)
		return nil
	}

	category := arguments[0]
	handler, ok := infoCategories[category]
	if !ok {
		return fmt.Errorf("unknown category: %s\n\nAvailable categories: %s", category, infoCategoryList)
	}

	handler(response, p)
	return nil
}

// infoOverview prints a compact summary table for each category.
//
// Takes response (*pb.GetSystemStatsResponse) which provides the
// system stats data.
// Takes p (*Printer) which formats and outputs the table rows.
// Takes w (io.Writer) which receives the category headers and
// footer text.
func infoOverview(response *pb.GetSystemStatsResponse, p *Printer, w io.Writer) {
	for i, cat := range infoCategoryOrder {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintf(w, "=== %s ===\n", cat.label)
		infoOverviewHandlers[cat.key](response, p)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Available categories: %s\n", infoCategoryList)
	_, _ = fmt.Fprint(w, "Use \"piko info <category>\" for full details.\n")
}

// infoSystemOverview prints system overview data including uptime, CPU usage,
// goroutine count, and CGO calls.
//
// Takes response (*pb.GetSystemStatsResponse) which provides the
// system statistics.
// Takes p (*Printer) which outputs the formatted data.
func infoSystemOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	cpuString := fmt.Sprintf("%.1f / %d CPUs", response.GetCpuMillicores(), response.GetNumCpu())
	rows := [][]string{{
		formatDuration(response.GetUptimeMs()),
		cpuString,
		strconv.FormatInt(int64(response.GetNumGoroutines()), 10),
		strconv.FormatInt(response.GetNumCgoCalls(), 10),
	}}
	p.PrintResource(systemOverviewColumns, rows)
}

// infoBuildOverview prints the build information from a system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the build details.
// Takes p (*Printer) which handles the formatted output.
func infoBuildOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	build := response.GetBuild()
	if build == nil {
		return
	}

	commit := build.GetCommit()
	const commitHashLen = 8
	if len(commit) > commitHashLen {
		commit = commit[:commitHashLen]
	}

	rows := [][]string{{
		build.GetVersion(),
		commit,
		build.GetGoVersion(),
		build.GetOs() + "/" + build.GetArch(),
	}}
	p.PrintResource(buildOverviewColumns, rows)
}

// infoRuntimeOverview prints a summary table of Go runtime settings.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the runtime data.
// Takes p (*Printer) which formats and outputs the table.
func infoRuntimeOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	rt := response.GetRuntime()
	if rt == nil {
		return
	}

	rows := [][]string{{
		rt.GetGogc(),
		rt.GetGomemlimit(),
		rt.GetCompiler(),
	}}
	p.PrintResource(runtimeOverviewColumns, rows)
}

// infoMemoryOverview prints a summary of memory statistics.
//
// Takes response (*pb.GetSystemStatsResponse) which provides the memory stats.
// Takes p (*Printer) which handles the formatted output.
func infoMemoryOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	mem := response.GetMemory()
	if mem == nil {
		return
	}

	rows := [][]string{{
		formatBytes(mem.GetHeapAlloc()),
		formatBytes(mem.GetSys()),
		strconv.FormatUint(mem.GetLiveObjects(), 10),
		strconv.FormatUint(mem.GetHeapObjects(), 10),
	}}
	p.PrintResource(memoryOverviewColumns, rows)
}

// infoGCOverview prints an overview of garbage collection statistics.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the system stats.
// Takes p (*Printer) which formats and displays the output.
func infoGCOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	gc := response.GetGc()
	if gc == nil {
		return
	}

	rows := [][]string{{
		strconv.FormatUint(uint64(gc.GetNumGc()), 10),
		formatNanos(safeconv.Uint64ToInt64(gc.GetLastPauseNs())),
		fmt.Sprintf("%.4f%%", gc.GetGcCpuFraction()*100),
		formatBytes(gc.GetNextGc()),
	}}
	p.PrintResource(gcOverviewColumns, rows)
}

// infoProcessOverview prints a summary of process information from the stats.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the
// system statistics.
// Takes p (*Printer) which handles the formatted output.
func infoProcessOverview(response *pb.GetSystemStatsResponse, p *Printer) {
	proc := response.GetProcess()
	if proc == nil {
		return
	}

	rows := [][]string{{
		strconv.FormatInt(int64(proc.GetPid()), 10),
		strconv.FormatInt(int64(proc.GetThreadCount()), 10),
		strconv.FormatInt(int64(proc.GetFdCount()), 10),
		formatBytes(proc.GetRss()),
		proc.GetHostname(),
	}}
	p.PrintResource(processOverviewColumns, rows)
}

// infoSystem prints system statistics from the response to the printer.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the
// system stats data.
// Takes p (*Printer) which outputs the formatted information.
func infoSystem(response *pb.GetSystemStatsResponse, p *Printer) {
	rows := [][]string{
		{"Uptime", formatDuration(response.GetUptimeMs())},
		{"System Uptime", formatDuration(response.GetSystemUptimeMs())},
		{"CPU Millicores", fmt.Sprintf("%.1f", response.GetCpuMillicores())},
		{"Num CPUs", strconv.FormatInt(int64(response.GetNumCpu()), 10)},
		{"GOMAXPROCS", strconv.FormatInt(int64(response.GetGomaxprocs()), 10)},
		{"Goroutines", strconv.FormatInt(int64(response.GetNumGoroutines()), 10)},
		{"CGO Calls", strconv.FormatInt(response.GetNumCgoCalls(), 10)},
		{"Cgroup Path", response.GetCgroupPath()},
		{"Monitoring Address", response.GetMonitoringListenAddr()},
		{"Timestamp", formatTimestamp(response.GetTimestampMs() / 1000)},
	}
	p.PrintResource(infoDetailColumns, rows)
}

// infoBuild prints the build information from a system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the build details.
// Takes p (*Printer) which handles the formatted output.
func infoBuild(response *pb.GetSystemStatsResponse, p *Printer) {
	build := response.GetBuild()
	if build == nil {
		return
	}

	rows := [][]string{
		{"Version", build.GetVersion()},
		{"Commit", build.GetCommit()},
		{"Go Version", build.GetGoVersion()},
		{"OS", build.GetOs()},
		{"Arch", build.GetArch()},
		{"Build Time", build.GetBuildTime()},
		{"Module", build.GetModulePath()},
		{"Module Version", build.GetModuleVersion()},
		{"VCS Modified", strconv.FormatBool(build.GetVcsModified())},
		{"VCS Time", build.GetVcsTime()},
	}
	p.PrintResource(infoDetailColumns, rows)
}

// infoRuntime prints runtime information from the system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the runtime data.
// Takes p (*Printer) which outputs the formatted result.
func infoRuntime(response *pb.GetSystemStatsResponse, p *Printer) {
	rt := response.GetRuntime()
	if rt == nil {
		return
	}

	rows := [][]string{
		{"GOGC", rt.GetGogc()},
		{"GOMEMLIMIT", rt.GetGomemlimit()},
		{"Compiler", rt.GetCompiler()},
	}
	p.PrintResource(infoDetailColumns, rows)
}

// infoMemory prints memory statistics from a system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the memory data.
// Takes p (*Printer) which outputs the formatted table.
func infoMemory(response *pb.GetSystemStatsResponse, p *Printer) {
	mem := response.GetMemory()
	if mem == nil {
		return
	}

	rows := [][]string{
		{"Alloc", formatBytes(mem.GetAlloc())},
		{"Total Alloc", formatBytes(mem.GetTotalAlloc())},
		{"Sys", formatBytes(mem.GetSys())},
		{"Heap Alloc", formatBytes(mem.GetHeapAlloc())},
		{"Heap Sys", formatBytes(mem.GetHeapSys())},
		{"Heap Idle", formatBytes(mem.GetHeapIdle())},
		{"Heap In Use", formatBytes(mem.GetHeapInuse())},
		{"Heap Objects", strconv.FormatUint(mem.GetHeapObjects(), 10)},
		{"Heap Released", formatBytes(mem.GetHeapReleased())},
		{"Stack In Use", formatBytes(mem.GetStackInuse())},
		{"Stack Sys", formatBytes(mem.GetStackSys())},
		{"MSpan In Use", formatBytes(mem.GetMspanInuse())},
		{"MSpan Sys", formatBytes(mem.GetMspanSys())},
		{"MCache In Use", formatBytes(mem.GetMcacheInuse())},
		{"MCache Sys", formatBytes(mem.GetMcacheSys())},
		{"GC Sys", formatBytes(mem.GetGcSys())},
		{"Other Sys", formatBytes(mem.GetOtherSys())},
		{"BuckHash Sys", formatBytes(mem.GetBuckhashSys())},
		{"Lookups", strconv.FormatUint(mem.GetLookups(), 10)},
		{"Mallocs", strconv.FormatUint(mem.GetMallocs(), 10)},
		{"Frees", strconv.FormatUint(mem.GetFrees(), 10)},
		{"Live Objects", strconv.FormatUint(mem.GetLiveObjects(), 10)},
	}
	p.PrintResource(infoDetailColumns, rows)
}

// infoGC prints garbage collection statistics from the system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the
// GC stats to display.
// Takes p (*Printer) which handles the formatted output.
func infoGC(response *pb.GetSystemStatsResponse, p *Printer) {
	gc := response.GetGc()
	if gc == nil {
		return
	}

	rows := [][]string{
		{"Cycles", strconv.FormatUint(uint64(gc.GetNumGc()), 10)},
		{"Forced Cycles", strconv.FormatUint(uint64(gc.GetNumForcedGc()), 10)},
		{"Last Pause", formatNanos(safeconv.Uint64ToInt64(gc.GetLastPauseNs()))},
		{"Total Pause", formatNanos(safeconv.Uint64ToInt64(gc.GetPauseTotalNs()))},
		{"CPU Fraction", fmt.Sprintf("%.4f%%", gc.GetGcCpuFraction()*100)},
		{"Next GC", formatBytes(gc.GetNextGc())},
		{"Last GC", formatTimestamp(gc.GetLastGcNs() / 1_000_000_000)},
	}

	if pauses := gc.GetRecentPauses(); len(pauses) > 0 {
		parts := make([]string, len(pauses))
		for i, nanoseconds := range pauses {
			parts[i] = formatNanos(safeconv.Uint64ToInt64(nanoseconds))
		}
		rows = append(rows, []string{"Recent Pauses", strings.Join(parts, ", ")})
	}

	p.PrintResource(infoDetailColumns, rows)
}

// infoProcess prints process information from the system stats response.
//
// Takes response (*pb.GetSystemStatsResponse) which contains the process data.
// Takes p (*Printer) which handles the formatted output.
func infoProcess(response *pb.GetSystemStatsResponse, p *Printer) {
	proc := response.GetProcess()
	if proc == nil {
		return
	}

	rows := [][]string{
		{"PID", strconv.FormatInt(int64(proc.GetPid()), 10)},
		{"PPID", strconv.FormatInt(int64(proc.GetPpid()), 10)},
		{"UID", strconv.FormatInt(int64(proc.GetUid()), 10)},
		{"GID", strconv.FormatInt(int64(proc.GetGid()), 10)},
		{"Threads", strconv.FormatInt(int64(proc.GetThreadCount()), 10)},
		{"File Descriptors", strconv.FormatInt(int64(proc.GetFdCount()), 10)},
		{"Max Open Files (Soft)", strconv.FormatInt(proc.GetMaxOpenFilesSoft(), 10)},
		{"Max Open Files (Hard)", strconv.FormatInt(proc.GetMaxOpenFilesHard(), 10)},
		{"RSS", formatBytes(proc.GetRss())},
		{"I/O Read Bytes", formatBytes(proc.GetIoReadBytes())},
		{"I/O Write Bytes", formatBytes(proc.GetIoWriteBytes())},
		{"I/O Read Total", formatBytes(proc.GetIoRchar())},
		{"I/O Write Total", formatBytes(proc.GetIoWchar())},
		{"Hostname", proc.GetHostname()},
		{"Executable", proc.GetExecutable()},
		{"CWD", proc.GetCwd()},
	}
	p.PrintResource(infoDetailColumns, rows)
}
