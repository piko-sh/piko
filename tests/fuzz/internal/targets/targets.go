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

// Package targets schedules isolated Piko subsystem checks from a saved case.
package targets

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/scenarios"
)

const (
	// workerPackage is the isolated subsystem worker package path.
	workerPackage = "piko.sh/piko/tests/fuzz/cmd/target-worker"

	// binderSafeTarget is the binder target that also runs under the safe build tag.
	binderSafeTarget = "binder-safe"
)

var (
	// ErrUnknownTarget reports a target name that is not registered.
	ErrUnknownTarget = errors.New("unknown target")

	// scenarioOnly lists targets implemented by package-test overlays.
	scenarioOnly = []string{"database", "release", "vector-search"}

	// direct lists targets implemented by the subsystem worker.
	direct = []string{
		"captcha-adapters", "auth-middleware", "csrf-contracts", "rate-limit-contracts", "captcha-contracts",
		"source-diagnostics", "actions", "annotator", "assets", "ast", "binder", "cache", "collections", "documents",
		"formatter", "i18n", "inspector", "orchestrator", "persistence", "registry", "resolver", "routing", "search",
		"storage", "syntax", "utilities",
	}

	// suites maps target names to their isolated package scenarios.
	suites = map[string]string{"incremental": "interpreted_cache_invalidation", "lsp": "lsp", "wasm": "wasm_runner"}
)

// GoFuzzer identifies a registered native Go fuzz target.
type GoFuzzer struct {
	// Package is the import path holding the fuzz function.
	Package string

	// Function is the fuzz function name.
	Function string
}

// workerFailure is the machine-readable failure line a subsystem worker prints.
type workerFailure struct {
	// Property is the stable name of the failed property, and is empty for infrastructure
	// failures.
	Property string

	// Source identifies the failed property independently of generated values.
	Source string

	// Detail describes the failure.
	Detail string

	// Infrastructure reports a harness-side failure rather than a Piko finding.
	Infrastructure bool
}

// propertyFailure identifies a worker's semantic property failure.
type propertyFailure struct {
	// Target is the subsystem target that failed.
	Target string

	// Property is the stable name of the failed property.
	Property string

	// Source identifies the failed property independently of generated values.
	Source string

	// Detail describes the failure.
	Detail string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e *propertyFailure) Error() string {
	return fmt.Sprintf("%s/%s: %s", e.Target, e.Property, e.Detail)
}

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e *propertyFailure) FailureSubject() string { return e.Source }

// ApplicationNames lists targets that launch the selected Piko servers.
//
// Returns []string which lists the registered application target names.
func ApplicationNames() []string {
	return []string{
		"auth-contracts", "form-shim-contracts", "pages", "components", "http-actions", "collection-routes",
		"content-editions", "graph-rebuild", "form-workflow", "typed-catalog", "catalog-rebuild", "browser-events",
		"browser-lifecycle", "partial-components", "markup", "firefox-markup", "source-equivalence", "route-contracts",
		"checkout-workflow", "shipped-components", "go-topology", "embedded-deploy", "generated-artifacts",
		"browser-element-reuse", "browser-lifecycle-inline",
	}
}

// IsApplication reports whether a target launches a Piko application.
//
// Takes name (string) which is the target name.
//
// Returns bool which reports whether the target launches an application server.
func IsApplication(name string) bool { return slices.Contains(ApplicationNames(), name) }

// BrowserNames lists application targets that always launch a browser unless the browser
// is turned off.
//
// Returns []string which lists the browser target names.
func BrowserNames() []string {
	return []string{
		"components", "form-workflow", "browser-events", "browser-lifecycle", "partial-components",
		"checkout-workflow", "shipped-components", "markup", "firefox-markup", "browser-element-reuse",
		"browser-lifecycle-inline",
	}
}

// IsBrowser reports whether a target always launches a browser unless the browser is
// turned off.
//
// Takes name (string) which is the target name.
//
// Returns bool which reports whether the target always launches a browser.
func IsBrowser(name string) bool { return slices.Contains(BrowserNames(), name) }

// UsesPackageWorker identifies targets executed in cmd/target-worker.
//
// Takes name (string) which is the target name.
//
// Returns bool which reports whether the target executes in the isolated package worker.
func UsesPackageWorker(name string) bool {
	return name == binderSafeTarget || slices.Contains(direct, name)
}

// List returns package targets, which run independently of server mode.
//
// Returns []string which lists package targets in sorted order.
func List() []string {
	all := slices.Concat(direct, []string{binderSafeTarget, "oracle-sensitivity", "provider-contracts", "example-contracts"}, scenarioOnly, slices.Sorted(maps.Keys(suites)))
	slices.Sort(all)
	return all
}

// Fuzzers returns the registered native Go fuzz targets.
//
// Returns map[string]GoFuzzer which maps registered fuzz target names to package and
// function entry points.
func Fuzzers() map[string]GoFuzzer {
	const shims = "piko.sh/piko/tests/fuzz/internal/shimcontracts"
	const generator = "piko.sh/piko/tests/fuzz/internal/generate"
	return map[string]GoFuzzer{
		"captcha-adapters":     {Package: shims, Function: "FuzzCaptchaAdapters"},
		"auth-middleware":      {Package: shims, Function: "FuzzAuthContract"},
		"csrf-contracts":       {Package: shims, Function: "FuzzCSRFContract"},
		"rate-limit-contracts": {Package: shims, Function: "FuzzRateLimitContract"},
		"captcha-contracts":    {Package: shims, Function: "FuzzCaptchaContract"},
		"project":              {Package: generator, Function: "FuzzProject"},
		"expression-semantics": {Package: generator, Function: "FuzzExpressionSemantics"},
		"binder":               {Package: workerPackage, Function: "FuzzBinder"},
		"cache":                {Package: workerPackage, Function: "FuzzCacheHistory"},
		"search":               {Package: workerPackage, Function: "FuzzSearchIndex"},
		"formatter":            {Package: workerPackage, Function: "FuzzFormatter"},
		"ast":                  {Package: workerPackage, Function: "FuzzAST"},
		"subsystems":           {Package: workerPackage, Function: "FuzzSubsystem"},
		"raw-sfc":              {Package: "piko.sh/piko/internal/sfcparser", Function: "FuzzParse"},
		"raw-expression":       {Package: "piko.sh/piko/internal/ast/ast_domain", Function: "FuzzParseExpression"},
		"maths":                {Package: "piko.sh/piko/wdk/maths", Function: "FuzzBigIntArithmetic"},
		"encoding":             {Package: "piko.sh/piko/wdk/encoding", Function: "FuzzEncodeBytesBase58RoundTrip"},
	}
}

// Run executes a single subsystem under the caller's process and resource policy.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports subsystem-worker setup, execution or property failure.
func Run(ctx context.Context, repository, work string, fuzzCase *casefile.Case) error {
	return RunWithWorkers(ctx, repository, work, "", fuzzCase)
}

// RunWithWorkers reuses the package's executable.
//
// The process and its global type registry are private to each case and build-tag
// selection.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports worker setup, execution or property failure.
func RunWithWorkers(ctx context.Context, repository, work, workers string, fuzzCase *casefile.Case) (err error) {
	if !slices.Contains(List(), fuzzCase.Target) {
		return fmt.Errorf("%w %q", ErrUnknownTarget, fuzzCase.Target)
	}
	if fuzzCase.Target == "oracle-sensitivity" {
		_, err := runner.MeasureSensitivity(fuzzCase)
		return err
	}
	directory, err := privateDirectory(work, "target-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, removeDirectory(directory)) }()
	input := filepath.Join(directory, "case")
	if err := casefile.Write(input, fuzzCase); err != nil {
		return fmt.Errorf("infrastructure: write case: %w", err)
	}
	return runInDirectory(ctx, repository, directory, workers, input, fuzzCase)
}

// runInDirectory dispatches a case to its scenario, suite or subsystem worker.
//
// Takes repository (string) which is the repository source directory.
// Takes directory (string) which is the case-private directory.
// Takes workers (string) which is the worker executable directory, or empty to use
// directory.
// Takes input (string) which is the saved case file path.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports worker setup, execution or property failure.
func runInDirectory(ctx context.Context, repository, directory, workers, input string, fuzzCase *casefile.Case) error {
	if IsPublicContract(fuzzCase.Target) {
		_, err := scenarios.RunPublic(ctx, repository, directory, fuzzCase)
		return err
	}
	if name, ok := suites[fuzzCase.Target]; ok {
		test := scenarios.SuiteTest{
			Package:  "tests/integration/" + name,
			Source:   "tests/fuzz/testdata/targets/" + fuzzCase.Target + ".go",
			TestName: "TestGeneratedPikoCase",
			Files:    nil,
		}
		if err := scenarios.RunSuite(ctx, repository, directory, fuzzCase, test); err != nil {
			return err
		}
		return scenarios.Run(ctx, repository, directory, fuzzCase)
	}
	if slices.Contains(scenarioOnly, fuzzCase.Target) {
		return scenarios.Run(ctx, repository, directory, fuzzCase)
	}
	workers = cmp.Or(workers, directory)
	if err := runWorker(ctx, repository, directory, workers, input, "", fuzzCase.Target); err != nil {
		return err
	}
	if fuzzCase.Target == binderSafeTarget {
		if err := runWorker(ctx, repository, directory, workers, input, "safe", fuzzCase.Target); err != nil {
			return err
		}
	}
	return scenarios.Run(ctx, repository, directory, fuzzCase)
}

// runWorker builds the subsystem worker for a build tag and runs it on the saved case.
//
// Takes repository (string) which is the repository source directory.
// Takes directory (string) which is the case-private working directory.
// Takes workers (string) which is the worker executable directory.
// Takes input (string) which is the saved case file path.
// Takes tag (string) which is the worker build tag, or empty for the default build.
// Takes target (string) which is the subsystem target name.
//
// Returns error which reports worker build, execution, cancellation or property failure.
func runWorker(ctx context.Context, repository, directory, workers, input, tag, target string) error {
	binary, err := process.BuildWorker(ctx, repository, workers, "target-worker", tag)
	if err != nil {
		return err
	}
	output, err := process.Run(ctx, directory, nil, binary, input)
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	label := target
	if target == binderSafeTarget && tag == "" {
		label += "/default"
	}
	return workerResult(output, err, label)
}

// workerResult distinguishes a semantic property failure from worker infrastructure
// errors.
//
// Takes output (string) which holds captured worker standard output and diagnostics.
// Takes err (error) which is the operation failure to inspect.
// Takes target (string) which selects the subsystem under test.
//
// Returns error which reports the named semantic failure, or an infrastructure failure
// when the worker flags its own harness failure or prints no result, or nil on success.
func workerResult(output string, err error, target string) error {
	if err == nil {
		return nil
	}
	for line := range strings.SplitSeq(output, "\n") {
		payload, found := strings.CutPrefix(line, "PIKO_TARGET_RESULT ")
		if !found {
			continue
		}
		var result workerFailure
		if json.Unmarshal([]byte(payload), &result) != nil {
			continue
		}
		if result.Infrastructure {
			return fmt.Errorf("infrastructure: %s worker: %s", target, result.Detail)
		}
		if result.Property != "" {
			return &propertyFailure{Target: target, Property: result.Property, Source: result.Source, Detail: result.Detail}
		}
	}
	return fmt.Errorf("infrastructure: %s worker: %w\n%s", target, err, output)
}

// IsPublicContract identifies package checks which retain named public-consumer evidence.
//
// Takes name (string) which is the target name.
//
// Returns bool which reports whether the target retains named public-consumer assertions.
func IsPublicContract(name string) bool {
	return name == "provider-contracts" || name == "example-contracts"
}

// RunPublicContract runs a named public-consumer contract in an isolated worker.
//
// Takes repository (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes fuzzCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns publiccontracts.Receipt which records the completed public-consumer assertions.
// Returns error which reports contract setup, execution or a missing positive assertion
// receipt, or nil on success.
func RunPublicContract(ctx context.Context, repository, work string, fuzzCase *casefile.Case) (receipt publiccontracts.Receipt, err error) {
	directory, err := privateDirectory(work, "public-contract-")
	if err != nil {
		return publiccontracts.Receipt{}, err
	}
	defer func() { err = errors.Join(err, removeDirectory(directory)) }()
	if err = casefile.Write(filepath.Join(directory, "case"), fuzzCase); err != nil {
		return publiccontracts.Receipt{}, fmt.Errorf("infrastructure: write case: %w", err)
	}
	return scenarios.RunPublic(ctx, repository, directory, fuzzCase)
}

// privateDirectory creates an absolute invocation-private directory under work.
//
// Takes work (string) which is the invocation-private working directory.
// Takes prefix (string) which prefixes the directory name.
//
// Returns string which is the absolute directory path.
// Returns error which reports a creation or resolution failure.
func privateDirectory(work, prefix string) (string, error) {
	directory, err := os.MkdirTemp(work, prefix)
	if err != nil {
		return "", fmt.Errorf("infrastructure: create %s directory: %w", prefix, err)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", errors.Join(fmt.Errorf("infrastructure: resolve %s: %w", directory, err), removeDirectory(directory))
	}
	return absolute, nil
}

// removeDirectory deletes a private directory tree.
//
// Takes directory (string) which is the directory to delete.
//
// Returns error which reports a removal failure, or nil.
func removeDirectory(directory string) error {
	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf("infrastructure: remove %s: %w", directory, err)
	}
	return nil
}
