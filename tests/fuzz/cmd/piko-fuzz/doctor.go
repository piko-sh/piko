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

package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/tests/fuzz/internal/campaign"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// preflightMemoryCeiling caps the memory reserved for the pikopls preflight build.
	preflightMemoryCeiling = 512 << 20

	// preflightMemoryDivisor scales the memory cap for preflight admission.
	preflightMemoryDivisor = 4

	// toolPikoplsSource names the language-server source check, which also triggers the
	// preflight build.
	toolPikoplsSource = "pikopls_source"

	// browserSampleVersion is the browser-sampling scheme of newly generated cases.
	browserSampleVersion = 1

	// toolNode names the Node.js check.
	toolNode = "node"

	// goModFile is the module file whose presence proves a source checkout.
	goModFile = "go.mod"
)

var (
	// errDoctorMissingTools reports that at least one required tool or source is absent.
	errDoctorMissingTools = errors.New("doctor found missing tools or sources")

	// browserTargetNames lists targets that always use a browser under the sample policy.
	browserTargetNames = []string{
		"components", "form-workflow", "browser-events", "browser-lifecycle", "partial-components",
		"checkout-workflow", "shipped-components", "markup", "firefox-markup",
	}

	// chromeCandidates lists Chromium executable names and paths in search order.
	chromeCandidates = []string{
		"headless_shell", "headless-shell", "chromium", "chromium-browser", "google-chrome",
		"google-chrome-stable", "google-chrome-beta", "google-chrome-unstable", "/usr/bin/google-chrome",
		"/usr/local/bin/chrome", "/snap/bin/chromium", "chrome",
	}
)

// doctorReport is the doctor's JSON output.
type doctorReport struct {
	// Tools maps each required tool or source to its path, or null when missing.
	Tools map[string]*string `json:"tools"`

	// Target is the requested target selection.
	Target string `json:"target"`

	// Mode is the requested server mode.
	Mode string `json:"mode"`

	// Missing lists the required tools or sources that were not found.
	Missing []string `json:"missing"`

	// Limits is the verified kernel reading.
	Limits resource.Sample `json:"limits"`

	// MinimumHostFree is the host memory reserve in bytes.
	MinimumHostFree uint64 `json:"minimum_host_free"`

	// Verified reports that the kernel enforces the requested limits.
	Verified bool `json:"verified"`
}

// doctor checks whether the selected campaign can run.
//
// The JSON report lists required tools, missing sources and verified kernel limits.
//
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
//
// Returns error which reports unverified limits, failed preflight, missing
// prerequisites or failed output.
func doctor(ctx context.Context, repo string, options *flags) error {
	budget, err := openBudget(ctx)
	if err != nil {
		return fmt.Errorf("doctor budget: %w (use ./hack/fuzz/fuzz.sh)", err)
	}
	limits, err := budget.SampleNow(ctx)
	if err != nil {
		return err
	}
	tools := doctorTools(repo, options)
	if _, ok := tools[toolPikoplsSource]; ok && !slices.Contains(slices.Collect(maps.Values(tools)), nil) {
		path, err := preflightPikopls(ctx, repo, options, budget)
		if err != nil {
			return err
		}
		tools["pikopls_build"] = &path
	}
	report := doctorReport{
		Tools:           tools,
		Target:          options.target,
		Mode:            options.mode,
		Missing:         missingTools(tools),
		Limits:          limits,
		MinimumHostFree: budget.MinFree(),
		Verified:        true,
	}
	if err := printJSON(report); err != nil {
		return err
	}
	if len(report.Missing) > 0 {
		return errDoctorMissingTools
	}
	return nil
}

// targetSelection is the requested target list.
type targetSelection struct {
	// names lists the requested targets.
	names []string

	// all is true when every target is selected.
	all bool
}

// newTargetSelection parses a comma-separated target list.
//
// Takes target (string) which is the -target value.
//
// Returns targetSelection which holds the requested targets.
func newTargetSelection(target string) targetSelection {
	names := strings.Split(target, ",")
	return targetSelection{names: names, all: slices.Contains(names, "all")}
}

// selects reports whether any candidate target is selected.
//
// Takes candidates (...string) which lists target names.
//
// Returns bool which is true when every target or any candidate is selected.
func (s targetSelection) selects(candidates ...string) bool {
	return s.all || slices.ContainsFunc(s.names, func(name string) bool { return slices.Contains(candidates, name) })
}

// doctorTools finds the tools and sources needed by the selected targets.
//
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
//
// Returns map[string]*string which maps each check to its path, or nil when missing.
func doctorTools(repo string, options *flags) map[string]*string {
	selection := newTargetSelection(options.target)
	tools := map[string]*string{"go": lookPath("go")}
	if options.fuzzer != "" || options.engine == engineGo {
		return tools
	}
	addExecutableTools(tools, selection, options)
	addSourceTools(tools, selection, repo, options)
	return tools
}

// addExecutableTools adds the executables the selected targets run.
//
// Takes tools (map[string]*string) which receives the checks.
// Takes selection (targetSelection) which holds the requested targets.
// Takes options (*flags) which holds the server mode and browser policy.
func addExecutableTools(tools map[string]*string, selection targetSelection, options *flags) {
	if selection.selects("generated-artifacts") {
		tools[toolNode] = lookPath(toolNode)
		tools["typescript_compiler"] = lookPath("tsc")
	}
	browser := browserSelected(selection.names, selection.all, options)
	if browser && selection.selects("firefox-markup") {
		tools["firefox"] = lookPath("firefox")
	}
	if browser && selection.selects("checkout-workflow") {
		tools["pdftotext"] = lookPath("pdftotext")
	}
	apps := selection.selects(targets.ApplicationNames()...)
	examples := options.mode != modeInterp && options.browser != browserOff && selection.selects("example-contracts")
	if (apps && browser) || examples {
		tools["browser"] = firstPath(chromeCandidates)
	}
	if selection.selects("wasm", "ts", "typescript") {
		tools[toolNode] = lookPath(toolNode)
	}
}

// addSourceTools adds the module sources the selected targets build against.
//
// Takes tools (map[string]*string) which receives the checks.
// Takes selection (targetSelection) which holds the requested targets.
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the server mode.
func addSourceTools(tools map[string]*string, selection targetSelection, repo string, options *flags) {
	siblings := filepath.Dir(repo)
	apps := selection.selects(targets.ApplicationNames()...) && !slices.Equal(selection.names, []string{"embedded-deploy"})
	if (selection.all || apps || selection.selects("incremental")) && options.mode != modeCompiled {
		tools["interpreter_provider_source"] = existingFile(filepath.Join(repo, "wdk", "interp", "interp_provider_pipit", goModFile))
		if pipitpending.Local() {
			tools["pipit_source"] = existingFile(filepath.Join(siblings, "pipit", goModFile))
		}
	}
	if selection.selects("collection-routes", "content-editions", "form-workflow") {
		tools["markdown_provider_source"] = existingFile(filepath.Join(repo, "wdk", "markdown", "markdown_provider_goldmark", goModFile))
	}
	if selection.selects("form-workflow", "database") {
		tools["sqlite_engine_source"] = existingFile(filepath.Join(repo, "wdk", "db", "db_engine_sqlite", goModFile))
	}
	if selection.selects("lsp") {
		tools[toolPikoplsSource] = existingFile(filepath.Join(repo, "cmd", "pikopls", goModFile))
	}
}

// browserSelected estimates, before the campaign is planned, whether any selected case
// will launch a browser.
//
// Takes names ([]string) which lists the requested targets.
// Takes all (bool) which is true when every target is selected.
// Takes options (*flags) which holds the browser policy, seed and case count.
//
// Returns bool which is true when a browser is likely to be needed.
func browserSelected(names []string, all bool, options *flags) bool {
	switch options.browser {
	case browserEvery:
		return true
	case browserOff:
		return false
	}
	if all || slices.ContainsFunc(names, func(name string) bool { return slices.Contains(browserTargetNames, name) }) {
		return true
	}
	for _, name := range names {
		if !targets.IsApplication(name) {
			continue
		}
		for index := range options.n {
			var sampled casefile.Case
			sampled.Target, sampled.Seed, sampled.Index = name, options.seed, index
			sampled.BrowserSampleVersion = browserSampleVersion
			if campaign.BrowserCase(&sampled, options.browser) {
				return true
			}
		}
	}
	return false
}

// preflightPikopls checks that the language server builds without retaining a binary.
//
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the memory cap.
// Takes budget (*resource.Budget) which admits the build.
//
// Returns string which is "ok" when the build succeeded.
// Returns error which reports a held lock, a refused admission or a failed build.
//
// Concurrency: Holds the per-user campaign lock while waiting for admission and
// building.
func preflightPikopls(ctx context.Context, repo string, options *flags, budget *resource.Budget) (string, error) {
	release, err := resource.Lock()
	if err != nil {
		return "", err
	}
	defer release()
	memory, err := parseByteSize(options.mem)
	if err != nil {
		return "", fmt.Errorf("-mem: %w", err)
	}
	if err := budget.Admit(ctx, min(memory/preflightMemoryDivisor, preflightMemoryCeiling)); err != nil {
		return "", fmt.Errorf("cannot admit the pikopls preflight: %w", err)
	}
	environment := []string{"CGO_ENABLED=0", "GOWORK=" + process.Workspace(repo)}
	output, err := process.Run(ctx, filepath.Join(repo, "tests", "fuzz"), environment,
		"go", "build", "-o", os.DevNull, filepath.Join(repo, "cmd", "pikopls"))
	if err != nil {
		return "", fmt.Errorf("pikopls preflight build: %w\n%s", err, output)
	}
	return "ok", nil
}

// missingTools lists the checks that found nothing, in name order.
//
// Takes tools (map[string]*string) which maps each check to its path.
//
// Returns []string which lists the missing checks.
func missingTools(tools map[string]*string) []string {
	missing := []string{}
	for _, name := range slices.Sorted(maps.Keys(tools)) {
		if tools[name] == nil {
			missing = append(missing, name)
		}
	}
	return missing
}

// lookPath resolves an executable name on PATH or checks an explicit path.
//
// Takes name (string) which is the executable name or path.
//
// Returns *string which is the resolved path, or nil when not found.
func lookPath(name string) *string {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil
	}
	return &path
}

// firstPath finds the first available executable from the candidates.
//
// Takes candidates ([]string) which lists executable names or paths in preference order.
//
// Returns *string which is the first resolved path, or nil when none is found.
func firstPath(candidates []string) *string {
	for _, candidate := range candidates {
		if path := lookPath(candidate); path != nil {
			return path
		}
	}
	return nil
}

// existingFile checks whether a path points to a regular file.
//
// Takes path (string) which is the file to check.
//
// Returns *string which is path, or nil when it is absent or not a regular file.
func existingFile(path string) *string {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return &path
}
