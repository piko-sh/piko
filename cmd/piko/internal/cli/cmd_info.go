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

	"piko.sh/piko/cmd/piko/internal/inspector"
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

	// systemOverviewColumns defines the table column layout for the system overview section.
	systemOverviewColumns = []Column{
		{Header: "UPTIME"},
		{Header: "CPU"},
		{Header: "GOROUTINES"},
		{Header: "CGO CALLS"},
	}

	// buildOverviewColumns defines the table column layout for the build overview section.
	buildOverviewColumns = []Column{
		{Header: "VERSION"},
		{Header: "COMMIT"},
		{Header: "GO"},
		{Header: "OS/ARCH"},
	}

	// runtimeOverviewColumns defines the table column layout for the runtime overview section.
	runtimeOverviewColumns = []Column{
		{Header: "GOGC"},
		{Header: "GOMEMLIMIT"},
		{Header: "COMPILER"},
	}

	// memoryOverviewColumns defines the table column layout for the memory overview section.
	memoryOverviewColumns = []Column{
		{Header: "HEAP ALLOC"},
		{Header: "SYS TOTAL"},
		{Header: "LIVE OBJECTS"},
		{Header: "HEAP OBJECTS"},
	}

	// gcOverviewColumns defines the table column layout for the
	// garbage collection overview section.
	gcOverviewColumns = []Column{
		{Header: "CYCLES"},
		{Header: "LAST PAUSE"},
		{Header: "CPU FRACTION"},
		{Header: "NEXT GC"},
	}

	// processOverviewColumns defines the table column layout for the process overview section.
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
		strconv.FormatInt(safeconv.Int32ToInt64(response.GetNumGoroutines()), 10),
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

	rows := [][]string{{
		build.GetVersion(),
		inspector.ShortCommitHash(build.GetCommit()),
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
		inspector.FormatBytes(mem.GetHeapAlloc()),
		inspector.FormatBytes(mem.GetSys()),
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
		inspector.FormatNanosAsDuration(safeconv.Uint64ToInt64(gc.GetLastPauseNs())),
		inspector.FormatGCCPUFraction(gc.GetGcCpuFraction()),
		inspector.FormatBytes(gc.GetNextGc()),
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
		strconv.FormatInt(safeconv.Int32ToInt64(proc.GetPid()), 10),
		strconv.FormatInt(safeconv.Int32ToInt64(proc.GetThreadCount()), 10),
		strconv.FormatInt(safeconv.Int32ToInt64(proc.GetFdCount()), 10),
		inspector.FormatBytes(proc.GetRss()),
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
		{"Num CPUs", strconv.FormatInt(safeconv.Int32ToInt64(response.GetNumCpu()), 10)},
		{"GOMAXPROCS", strconv.FormatInt(safeconv.Int32ToInt64(response.GetGomaxprocs()), 10)},
		{"Goroutines", strconv.FormatInt(safeconv.Int32ToInt64(response.GetNumGoroutines()), 10)},
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
	p.PrintResource(infoDetailColumns, detailRowsToTable(inspector.BuildBuildDetailRows(build)))
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
	p.PrintResource(infoDetailColumns, detailRowsToTable(inspector.BuildRuntimeDetailRows(rt)))
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
	p.PrintResource(infoDetailColumns, detailRowsToTable(inspector.BuildMemoryDetailRows(mem)))
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
	p.PrintResource(infoDetailColumns, detailRowsToTable(inspector.BuildGCDetailRows(gc)))
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
	p.PrintResource(infoDetailColumns, detailRowsToTable(inspector.BuildProcessDetailRows(proc)))
}

// detailRowsToTable converts a slice of inspector.DetailRow into the
// two-column [][]string layout expected by Printer.PrintResource. The
// helper exists so each info<X> function can lift its rows from the
// shared inspector package without restating the conversion.
//
// Takes rows ([]inspector.DetailRow) which is the inspector row set.
//
// Returns [][]string ready to pass to PrintResource.
func detailRowsToTable(rows []inspector.DetailRow) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = []string{row.Label, row.Value}
	}
	return out
}
