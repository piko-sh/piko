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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type RunMode string

const (
	ModeEndToEnd RunMode = "endtoend"

	ModeInnerLoop RunMode = "innerloop"
)

func AllModes() []RunMode {
	return []RunMode{ModeEndToEnd, ModeInnerLoop}
}

type RunnerKind string

const (
	RunnerPiko RunnerKind = "piko"

	RunnerGo RunnerKind = "go"

	RunnerCPython RunnerKind = "cpython"

	RunnerPyPy RunnerKind = "pypy"

	RunnerYaegi RunnerKind = "yaegi"

	RunnerScriggo RunnerKind = "scriggo"

	RunnerTengo RunnerKind = "tengo"

	RunnerMvm RunnerKind = "mvm"
)

func AllRunners() []RunnerKind {
	return []RunnerKind{RunnerPiko, RunnerGo, RunnerCPython, RunnerPyPy, RunnerYaegi, RunnerScriggo, RunnerTengo, RunnerMvm}
}

type BenchSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Args []string `json:"args"`

	ExpectedStdoutSHA string `json:"expected_stdout_sha256"`

	KInner int `json:"k_inner"`

	TimeoutSeconds int `json:"timeout_seconds"`

	Inputs []string `json:"inputs,omitempty"`

	SkipRunners []RunnerKind `json:"skip_runners,omitempty"`

	SourceSHAs map[string]string `json:"source_sha256"`
}

func LoadSpec(path string) (BenchSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BenchSpec{}, fmt.Errorf("read spec %s: %w", path, err)
	}
	var spec BenchSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return BenchSpec{}, fmt.Errorf("parse spec %s: %w", path, err)
	}
	if spec.Name == "" {
		return BenchSpec{}, fmt.Errorf("spec %s: missing name", path)
	}
	if spec.KInner <= 0 {
		return BenchSpec{}, fmt.Errorf("spec %s: k_inner must be > 0", path)
	}
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = 120
	}
	return spec, nil
}

func saveSpec(path string, spec BenchSpec) error {
	payload, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec %s: %w", path, err)
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

type Result struct {
	Benchmark string `json:"benchmark"`

	Runner RunnerKind `json:"runner"`

	Mode RunMode `json:"mode"`

	RunIndex int `json:"run_index"`

	WallNanos int64 `json:"wall_nanos"`

	InnerNanos int64 `json:"inner_nanos,omitempty"`

	CompileNanos int64 `json:"compile_nanos,omitempty"`

	PeakRSSKB int64 `json:"peak_rss_kb"`

	StdoutSHA string `json:"stdout_sha256"`

	Status string `json:"status"`

	Note string `json:"note,omitempty"`

	StderrTail string `json:"stderr_tail,omitempty"`
}

const (
	StatusOK = "ok"

	StatusSkipped = "skipped"

	StatusFailed = "failed"

	StatusMismatch = "mismatch"
)

type Aggregate struct {
	Benchmark string `json:"benchmark"`

	Runner RunnerKind `json:"runner"`

	Mode RunMode `json:"mode"`

	Runs int `json:"runs"`

	MedianNanos int64 `json:"median_nanos"`

	MeanNanos int64 `json:"mean_nanos"`

	StddevNanos int64 `json:"stddev_nanos"`

	MinNanos int64 `json:"min_nanos"`

	P95Nanos int64 `json:"p95_nanos"`

	PeakRSSKB int64 `json:"peak_rss_kb_median"`

	MedianCompileNanos int64 `json:"median_compile_nanos,omitempty"`

	ColdStartNanos int64 `json:"cold_start_nanos,omitempty"`
}

type HostInfo struct {
	OS string `json:"os"`

	Arch string `json:"arch"`

	CPUModel string `json:"cpu_model,omitempty"`

	CPUCountLogical int `json:"cpu_count_logical"`

	GoVersion string `json:"go_version,omitempty"`

	PikoCommit string `json:"piko_commit,omitempty"`

	CPythonImage string `json:"cpython_image,omitempty"`

	CPythonVersion string `json:"cpython_version,omitempty"`

	PyPyImage string `json:"pypy_image,omitempty"`

	PyPyVersion string `json:"pypy_version,omitempty"`

	ContainerRuntime string `json:"container_runtime,omitempty"`
}

type SuiteConfig struct {
	Runs int

	Warmup int

	Filter []string

	Runners []RunnerKind

	Modes []RunMode

	UseHostPython bool

	RequireRunners []RunnerKind

	Regenerate bool

	Strict bool

	CPythonImage string

	PyPyImage string

	ContainerKeep bool

	OutputDirectory string
}

func LoadSuiteConfig() SuiteConfig {
	config := SuiteConfig{
		Runs:            envIntDefault("CROSS_LANG_RUNS", 7),
		Warmup:          envIntDefault("CROSS_LANG_WARMUP", 2),
		Filter:          envSplit("CROSS_LANG_FILTER"),
		Runners:         envRunners("CROSS_LANG_RUNNERS", AllRunners()),
		Modes:           envModes("CROSS_LANG_MODE", AllModes()),
		UseHostPython:   os.Getenv("CROSS_LANG_USE_HOST_PYTHON") == "1",
		RequireRunners:  envRunners("CROSS_LANG_REQUIRE", nil),
		Regenerate:      os.Getenv("CROSS_LANG_REGEN") == "1",
		Strict:          os.Getenv("CROSS_LANG_STRICT") == "1",
		CPythonImage:    envStringDefault("CROSS_LANG_CPYTHON_IMAGE", "python:3.13-slim"),
		PyPyImage:       envStringDefault("CROSS_LANG_PYPY_IMAGE", "pypy:3.10-slim"),
		ContainerKeep:   os.Getenv("CROSS_LANG_CONTAINER_KEEP") == "1",
		OutputDirectory: envStringDefault("CROSS_LANG_OUTPUT_DIR", "results"),
	}
	if config.Runs < 1 {
		config.Runs = 1
	}
	if config.Warmup < 0 {
		config.Warmup = 0
	}
	return config
}

func (config SuiteConfig) MatchesFilter(benchmarkName string) bool {
	if len(config.Filter) == 0 {
		return true
	}
	for _, fragment := range config.Filter {
		if strings.Contains(benchmarkName, fragment) {
			return true
		}
	}
	return false
}

func (config SuiteConfig) RunnerEnabled(kind RunnerKind) bool {
	for _, enabled := range config.Runners {
		if enabled == kind {
			return true
		}
	}
	return false
}

func (config SuiteConfig) RunnerRequired(kind RunnerKind) bool {
	for _, required := range config.RequireRunners {
		if required == kind {
			return true
		}
	}
	return false
}

func SpecPath(benchmarkDir string) string {
	return filepath.Join(benchmarkDir, "spec.json")
}

type RunnerContext struct {
	Ctx context.Context

	Spec BenchSpec

	Mode RunMode

	BenchmarkDir string

	BenchmarksDir string
}

func envIntDefault(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func envStringDefault(name, fallback string) string {
	if raw := os.Getenv(name); raw != "" {
		return raw
	}
	return fallback
}

func envSplit(name string) []string {
	raw := os.Getenv(name)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func envRunners(name string, fallback []RunnerKind) []RunnerKind {
	parts := envSplit(name)
	if len(parts) == 0 {
		return fallback
	}
	out := make([]RunnerKind, 0, len(parts))
	for _, part := range parts {
		out = append(out, RunnerKind(strings.ToLower(part)))
	}
	return out
}

func envModes(name string, fallback []RunMode) []RunMode {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch raw {
	case "":
		return fallback
	case "both", "all":
		return AllModes()
	case "endtoend":
		return []RunMode{ModeEndToEnd}
	case "innerloop":
		return []RunMode{ModeInnerLoop}
	default:
		return fallback
	}
}
