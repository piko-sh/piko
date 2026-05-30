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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ReportEnvelope struct {
	GeneratedAtUTC string `json:"generated_at_utc"`

	Host HostInfo `json:"host"`

	Config ReportConf `json:"config"`

	Runs []Result `json:"runs"`

	Aggregates []Aggregate `json:"aggregates"`
}

type ReportConf struct {
	RunsPerCombo int `json:"runs_per_combo"`

	Warmup int `json:"warmup"`

	Modes []RunMode `json:"modes"`
}

func WriteJSONReport(suite *Suite, runs []Result, aggregates []Aggregate) error {
	envelope := ReportEnvelope{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Host:           suite.hostInfo,
		Config: ReportConf{
			RunsPerCombo: suite.Config.Runs,
			Warmup:       suite.Config.Warmup,
			Modes:        suite.Config.Modes,
		},
		Runs:       runs,
		Aggregates: aggregates,
	}
	payload, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	path := filepath.Join(suite.OutputDirectory, "latest.json")
	return os.WriteFile(path, payload, 0o644)
}

func WriteMarkdownReport(suite *Suite, aggregates []Aggregate) error {
	var builder strings.Builder
	writeMarkdownHeader(&builder, suite)
	writeMarkdownSummary(&builder, aggregates)
	writeMarkdownPerBenchmark(&builder, aggregates)
	path := filepath.Join(suite.OutputDirectory, "latest.md")
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func writeMarkdownHeader(builder *strings.Builder, suite *Suite) {
	builder.WriteString("# Piko cross-language benchmark results\n\n")
	fmt.Fprintf(builder, "_Generated %s_\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(builder, "Host: %s/%s, %d logical CPUs, %s.  ", suite.hostInfo.OS, suite.hostInfo.Arch, suite.hostInfo.CPUCountLogical, suite.hostInfo.GoVersion)
	if suite.hostInfo.CPythonImage != "" {
		fmt.Fprintf(builder, "CPython: %s.  ", suite.hostInfo.CPythonImage)
	}
	if suite.hostInfo.PyPyImage != "" {
		fmt.Fprintf(builder, "PyPy: %s.  ", suite.hostInfo.PyPyImage)
	}
	builder.WriteString("\n\n")
	fmt.Fprintf(builder, "Runs per combo: %d (warmup %d). Modes: ", suite.Config.Runs, suite.Config.Warmup)
	for index, mode := range suite.Config.Modes {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(string(mode))
	}
	builder.WriteString(".\n\n")
}

func writeMarkdownSummary(builder *strings.Builder, aggregates []Aggregate) {
	builder.WriteString("## Summary - cold-start mode (× vs piko)\n\n")
	builder.WriteString("Cold-start = median compile + median runtime per single iteration. ")
	builder.WriteString("Container exec, subprocess startup, and other methodology overhead are excluded.\n\n")
	benchmarks := uniqueBenchmarks(aggregates)
	if len(benchmarks) == 0 {
		builder.WriteString("_No successful runs to report._\n\n")
		return
	}
	builder.WriteString("| Benchmark | piko | go | cpython | pypy | yaegi | scriggo | tengo |\n")
	builder.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, benchmark := range benchmarks {
		fmt.Fprintf(builder, "| %s ", benchmark)
		pikoCold := coldStartForCell(aggregates, benchmark, RunnerPiko, ModeInnerLoop)
		for _, runner := range AllRunners() {
			ratio := ratioColdVsPiko(aggregates, benchmark, runner, ModeInnerLoop, pikoCold)
			fmt.Fprintf(builder, "| %s ", ratio)
		}
		builder.WriteString("|\n")
	}
	builder.WriteString("\n")
}

func writeMarkdownPerBenchmark(builder *strings.Builder, aggregates []Aggregate) {
	benchmarks := uniqueBenchmarks(aggregates)
	for _, benchmark := range benchmarks {
		for _, mode := range AllModes() {
			cells := filterByBenchmarkMode(aggregates, benchmark, mode)
			if len(cells) == 0 {
				continue
			}
			fmt.Fprintf(builder, "## %s - %s\n\n", benchmark, mode)
			builder.WriteString("| Runner | Compile | Runtime | Cold-start | Stddev | P95 | Peak RSS | × vs piko (cold) |\n")
			builder.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
			pikoCold := coldStartForCell(aggregates, benchmark, RunnerPiko, mode)
			for _, runner := range AllRunners() {
				aggregate, present := findAggregate(cells, runner)
				if !present {
					fmt.Fprintf(builder, "| %s | _missing_ | | | | | | |\n", runner)
					continue
				}
				ratio := ratioColdVsPiko(aggregates, benchmark, runner, mode, pikoCold)
				fmt.Fprintf(builder, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
					runner,
					formatNanos(aggregate.MedianCompileNanos),
					formatNanos(aggregate.MedianNanos),
					formatNanos(aggregate.ColdStartNanos),
					formatNanos(aggregate.StddevNanos),
					formatNanos(aggregate.P95Nanos),
					formatRSS(aggregate.PeakRSSKB),
					ratio,
				)
			}
			builder.WriteString("\n")
		}
	}
}

func uniqueBenchmarks(aggregates []Aggregate) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, aggregate := range aggregates {
		if _, ok := seen[aggregate.Benchmark]; ok {
			continue
		}
		seen[aggregate.Benchmark] = struct{}{}
		out = append(out, aggregate.Benchmark)
	}
	return out
}

func filterByBenchmarkMode(aggregates []Aggregate, benchmark string, mode RunMode) []Aggregate {
	out := []Aggregate{}
	for _, aggregate := range aggregates {
		if aggregate.Benchmark == benchmark && aggregate.Mode == mode {
			out = append(out, aggregate)
		}
	}
	return out
}

func findAggregate(cells []Aggregate, runner RunnerKind) (Aggregate, bool) {
	for _, aggregate := range cells {
		if aggregate.Runner == runner {
			return aggregate, true
		}
	}
	return Aggregate{}, false
}

func coldStartForCell(aggregates []Aggregate, benchmark string, runner RunnerKind, mode RunMode) int64 {
	for _, aggregate := range aggregates {
		if aggregate.Benchmark == benchmark && aggregate.Runner == runner && aggregate.Mode == mode {
			return aggregate.ColdStartNanos
		}
	}
	return 0
}

func ratioColdVsPiko(aggregates []Aggregate, benchmark string, runner RunnerKind, mode RunMode, pikoCold int64) string {
	if runner == RunnerPiko {
		return "1.00×"
	}
	if pikoCold == 0 {
		return "-"
	}
	for _, aggregate := range aggregates {
		if aggregate.Benchmark == benchmark && aggregate.Runner == runner && aggregate.Mode == mode {
			if aggregate.ColdStartNanos == 0 {
				return "-"
			}
			ratio := float64(aggregate.ColdStartNanos) / float64(pikoCold)
			return fmt.Sprintf("%.2f×", ratio)
		}
	}
	return "_miss_"
}

func medianForCell(aggregates []Aggregate, benchmark string, runner RunnerKind, mode RunMode) int64 {
	for _, aggregate := range aggregates {
		if aggregate.Benchmark == benchmark && aggregate.Runner == runner && aggregate.Mode == mode {
			return aggregate.MedianNanos
		}
	}
	return 0
}

func ratioVsPiko(aggregates []Aggregate, benchmark string, runner RunnerKind, mode RunMode, pikoMedian int64) string {
	for _, aggregate := range aggregates {
		if aggregate.Benchmark == benchmark && aggregate.Runner == runner && aggregate.Mode == mode {
			if pikoMedian == 0 {
				return "-"
			}
			ratio := float64(aggregate.MedianNanos) / float64(pikoMedian)
			return fmt.Sprintf("%.2f×", ratio)
		}
	}
	return "-"
}

func formatNanos(nanos int64) string {
	if nanos <= 0 {
		return "-"
	}
	switch {
	case nanos < 1_000:
		return fmt.Sprintf("%d ns", nanos)
	case nanos < 1_000_000:
		return fmt.Sprintf("%.1f µs", float64(nanos)/1_000)
	case nanos < 1_000_000_000:
		return fmt.Sprintf("%.2f ms", float64(nanos)/1_000_000)
	default:
		return fmt.Sprintf("%.2f s", float64(nanos)/1_000_000_000)
	}
}

func formatRSS(kib int64) string {
	if kib <= 0 {
		return "-"
	}
	switch {
	case kib < 1024:
		return fmt.Sprintf("%d KiB", kib)
	case kib < 1024*1024:
		return fmt.Sprintf("%.1f MiB", float64(kib)/1024)
	default:
		return fmt.Sprintf("%.2f GiB", float64(kib)/(1024*1024))
	}
}
