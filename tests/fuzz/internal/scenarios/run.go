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

// Package scenarios runs case-specific assertions inside Piko's existing test harnesses.
package scenarios

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
)

const (
	// overlayFile is the staged go build -overlay specification file name.
	overlayFile = "overlay.json"

	// drawPrefixBytes is how many leading choice bytes a scenario receives.
	drawPrefixBytes = 24
)

var (
	// suites maps target names to their isolated package scenarios.
	suites = map[string][]suite{
		"incremental":   {{source: "callbacks", pkg: "wdk/interp/interp_provider_pipit"}},
		"actions":       {{source: "actions", pkg: "internal/generator/generator_adapters"}},
		"documents":     {{source: "documents_email", pkg: "tests/integration/email"}, {source: "documents_pdf", pkg: "internal/pdfwriter/pdfwriter_test"}},
		"vector-search": {{source: "vector_contract", pkg: "internal/llm/llm_adapters/vector_cache"}},
		"cache":         {{source: "vectors", pkg: "internal/llm/llm_adapters/vector_cache"}},
		"assets":        {{source: "assets", pkg: "internal/persistence"}},
		"database":      {{source: "database", pkg: "tests/integration/db_seagull_sqlite"}},
		"release":       {{source: "release", pkg: "tests/integration/release"}},
		"persistence":   {{source: "persistence", pkg: "internal/persistence"}},
		"collections":   {{source: "collections", pkg: "tests/integration/markdown_collection"}},
		"orchestrator":  {{source: "orchestrator", pkg: "tests/integration/orchestrator_pipeline"}},
	}

	// assertion recognises a scenario's stable assertion marker.
	assertion = regexp.MustCompile(`PIKO_CHECK ([a-z][a-z0-9-]{0,79}):`)

	// location recognises a generated scenario's source-line marker.
	location = regexp.MustCompile(`piko_fuzz_scenario_test\.go:(\d+)`)
)

// suite selects source assertions to overlay onto a test package.
type suite struct {
	// source names the scenario fixture under testdata.
	source string

	// pkg is the repository-relative test package receiving the overlay.
	pkg string
}

// SuiteTest names the test package and fixture a scenario overlay runs.
type SuiteTest struct {
	// Package is the repository-relative test package receiving the overlay.
	Package string

	// Source is the repository-relative ordinary Go fixture path.
	Source string

	// TestName names the required positive test result.
	TestName string

	// Files lists the minimal package source files to include, or is empty to include the
	// whole package.
	Files []string
}

// overlaySpec is the go build -overlay file format.
type overlaySpec struct {
	// Replace maps a virtual source path to its physical replacement.
	Replace map[string]string
}

// Run executes the stateful checks belonging to a package target.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports package-scenario setup, execution or assertion failure.
func Run(ctx context.Context, repository, work string, fuzzCase *casefile.Case) error {
	for _, selected := range suites[fuzzCase.Target] {
		test := SuiteTest{
			Package:  selected.pkg,
			Source:   filepath.Join("tests", "fuzz", "internal", "scenarios", "testdata", selected.source+".go"),
			TestName: "TestGeneratedPikoScenario",
			Files:    scenarioFiles(selected.source),
		}
		if err := RunSuite(ctx, repository, work, fuzzCase, test); err != nil {
			return err
		}
	}
	return nil
}

// scenarioFiles selects the minimal files required by a local package scenario.
//
// Takes source (string) which names the package scenario.
//
// Returns []string which lists minimal package source paths, or is nil to include the
// whole package.
func scenarioFiles(source string) []string {
	switch source {
	case "database", "release":
		return []string{scenarioTestFile}
	case "orchestrator":
		return []string{"harness_test.go", scenarioTestFile}
	default:
		return nil
	}
}

// RunSuite adds a case-dependent test to an existing package without modifying its
// sources.
//
// It requires a positive pass record because go test also exits successfully when the
// named test doesn't exist or skips for a missing tool.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes test (SuiteTest) which names the package, fixture, test and files to run.
//
// Returns error which reports contract setup, execution or a missing positive assertion
// receipt, or nil on success.
func RunSuite(ctx context.Context, repository, work string, fuzzCase *casefile.Case, test SuiteTest) error {
	root, err := filepath.Abs(repository)
	if err != nil {
		return fmt.Errorf("infrastructure: resolve repository: %w", err)
	}
	rendered, err := renderSuite(root, test.Source, fuzzCase.Data)
	if err != nil {
		return fmt.Errorf("infrastructure: render scenario: %w", err)
	}
	return runRenderedSuite(ctx, root, work, fuzzCase, test, rendered)
}

// runRenderedSuite stages rendered scenario bytes as an overlay and runs the named test.
//
// Takes root (string) which is the absolute repository directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes test (SuiteTest) which names the package, fixture, test and files to run.
// Takes rendered ([]byte) which holds the rendered scenario source.
//
// Returns error which reports staging, execution or a missing positive assertion receipt,
// or nil on success.
func runRenderedSuite(ctx context.Context, root, work string, fuzzCase *casefile.Case, test SuiteTest, rendered []byte) error {
	id := strings.TrimSuffix(filepath.Base(test.Source), ".go")
	suiteDirectory := filepath.Join(root, test.Package)
	spec, nested, err := stageSuite(root, work, id, suiteDirectory, fuzzCase, rendered)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(fuzzCase.Data)
	environment := slices.Concat([]string{
		fmt.Sprintf("PIKO_FUZZ_CASE_TOKEN=%x", digest[:8]),
		"PIKO_FUZZ_DRAWS=" + hex.EncodeToString(fuzzCase.Data[:min(drawPrefixBytes, len(fuzzCase.Data))]),
		"PIKO_FUZZ_CASE_DIR=" + filepath.Join(work, "case"),
		"GOFLAGS=" + strings.TrimSpace(os.Getenv("GOFLAGS")+" -buildvcs=false"),
	}, nested)
	files := test.Files
	if len(files) == 0 {
		files = []string{"."}
	}
	arguments := []string{"test", "-overlay", spec, "-tags=integration", "-count=1", "-parallel=1", "-run=^" + regexp.QuoteMeta(test.TestName) + "$", "-v"}
	output, err := process.Run(ctx, suiteDirectory, environment, "go", append(arguments, files...)...)
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if err != nil {
		if strings.Contains(output, "--- FAIL: "+test.TestName+" (") {
			return fmt.Errorf("%s/%s/%s: %w\n%s", fuzzCase.Target, id, assertionID(output), err, output)
		}
		return fmt.Errorf("infrastructure: %s suite: %w\n%s", id, err, output)
	}
	if !strings.Contains(output, "--- PASS: "+test.TestName+" (") {
		return fmt.Errorf("infrastructure: missing %s scenario result\n%s", id, output)
	}
	return nil
}

// stageSuite writes the rendered scenario, its companions and the overlay specification
// into a fresh staging directory.
//
// Takes root (string) which is the absolute repository directory.
// Takes work (string) which is the invocation-private working directory.
// Takes id (string) which names the scenario.
// Takes suiteDirectory (string) which is the absolute test package directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes rendered ([]byte) which holds the rendered scenario source.
//
// Returns string which is the overlay specification path.
// Returns []string which lists environment entries naming the staged companions.
// Returns error which reports a staging failure.
func stageSuite(root, work, id, suiteDirectory string, fuzzCase *casefile.Case, rendered []byte) (string, []string, error) {
	staged, err := os.MkdirTemp(work, "go-suite-")
	if err != nil {
		return "", nil, fmt.Errorf("infrastructure: create staging directory: %w", err)
	}
	staged, err = filepath.Abs(staged)
	if err != nil {
		return "", nil, fmt.Errorf("infrastructure: resolve staging directory: %w", err)
	}
	if err := writeFileIn(staged, scenarioTestFile, rendered); err != nil {
		return "", nil, fmt.Errorf("infrastructure: write scenario: %w", err)
	}
	nested, err := stageCompanions(root, staged, id, fuzzCase)
	if err != nil {
		return "", nil, fmt.Errorf("infrastructure: stage %s: %w", id, err)
	}
	overlay := overlaySpec{Replace: map[string]string{
		filepath.Join(suiteDirectory, scenarioTestFile): filepath.Join(staged, scenarioTestFile),
	}}
	data, err := json.Marshal(overlay)
	if err != nil {
		return "", nil, fmt.Errorf("infrastructure: encode overlay: %w", err)
	}
	if err = writeFileIn(staged, overlayFile, data); err != nil {
		return "", nil, fmt.Errorf("infrastructure: write overlay: %w", err)
	}
	return filepath.Join(staged, overlayFile), nested, nil
}

// assertionID extracts the stable assertion identity from scenario output.
//
// Takes output (string) which holds captured go test standard output and diagnostics.
//
// Returns string which is the stable marker, source-line identity or fallback assertion
// name.
func assertionID(output string) string {
	if match := assertion.FindStringSubmatch(output); match != nil {
		return match[1]
	}
	if match := location.FindStringSubmatch(output); match != nil {
		return "assertion-" + match[1]
	}
	return "assertion"
}
