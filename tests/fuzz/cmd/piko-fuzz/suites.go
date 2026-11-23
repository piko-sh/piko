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
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/tests/fuzz/internal/campaign"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/coverage"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/inventory"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// soakDuration is the default soak suite duration.
	soakDuration = 30 * time.Minute

	// extendedCases is the default case count of the extended suite.
	extendedCases = 64

	// previewDurationCases is the preview window of a duration-bounded campaign.
	previewDurationCases = 64

	// inventoryDefaultCases is the default inventory window per target.
	inventoryDefaultCases = 16

	// previewBasis explains what a preview counts.
	previewBasis = "Generated obligations only. At most 512 positions per target are previewed. No runtime observations."

	// inventoryBasis explains how the inventory report counts features.
	inventoryBasis = "Each source feature is counted once per file and once per project. " +
		"Identical support files in different projects count once in each project. " +
		"Optional campaign evidence uses its saved targets, generator profile and completed positions."

	// suiteSmoke names the smoke suite preset.
	suiteSmoke = "smoke"

	// suiteExtended names the extended suite preset and its generator size.
	suiteExtended = "extended"

	// environmentBrowser is the matrix environment that needs browser execution.
	environmentBrowser = "browser"

	// localPipit names the local Pipit interpreter tool.
	localPipit = "local Pipit"

	// chromium names the Chromium browser tool.
	chromium = "Chromium"
)

var (
	// errPreviewOverflow reports a preview window that runs past the stream's end.
	errPreviewOverflow = errors.New("preview window overflows")
)

// previewTarget summarises one target's planned checks. Fields are in printed order.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type previewTarget struct {
	// Target identifies the previewed target.
	Target string

	// Modes lists the applicable execution modes.
	Modes []string

	// Tools lists the required tools.
	Tools []string

	// PreviewCases counts the previewed cases.
	PreviewCases int

	// BrowserSelected counts the previewed cases selected for browser execution.
	BrowserSelected int

	// HTTPRequests counts planned HTTP requests by route role.
	HTTPRequests map[string]int `json:",omitempty"`

	// BrowserRoutes counts planned browser visits by route role.
	BrowserRoutes map[string]int `json:",omitempty"`

	// Matrices holds the previewed finite matrices.
	Matrices map[string]*previewMatrix `json:",omitempty"`
}

// previewMatrix counts finite cells in a campaign preview.
type previewMatrix struct {
	// cells records the generated finite cells.
	cells map[string]bool

	// selected records the generated cells selected for execution.
	selected map[string]bool

	// Environment identifies the required host environment.
	Environment string

	// Feasible counts the matrix's feasible cells.
	Feasible int

	// Generated counts the distinct generated cells.
	Generated int

	// Selected counts the distinct cells selected for execution.
	Selected int

	// Missing counts the feasible cells not yet generated.
	Missing int
}

// previewReport is the printed campaign preview. Fields are in printed order.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type previewReport struct {
	// Suite names the previewed suite preset.
	Suite string

	// Browser is the previewed browser policy.
	Browser string

	// Basis explains what the preview counts.
	Basis string

	// Size is the generated program size.
	Size string

	// Seed is the previewed stream's base seed.
	Seed uint64

	// Start is the first previewed stream position.
	Start uint64

	// Duration is the requested campaign duration.
	Duration string

	// Targets lists the per-target previews.
	Targets []previewTarget
}

// sourceGap compares authored and generated source features. Fields are in printed order.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type sourceGap struct {
	// Shared counts authored features present in generated sources.
	Shared int

	// Missing counts authored features absent from generated sources.
	Missing int

	// MissingKeys lists authored features absent from generated sources.
	MissingKeys []string
}

// executionEvidence summarises a campaign's recorded execution. Fields are in printed
// order.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type executionEvidence struct {
	// Profile identifies the saved campaign's generator profile.
	Profile string

	// Platform identifies the saved campaign's runtime platform.
	Platform string

	// Status is the saved campaign's latest status.
	Status string

	// LastStop describes the saved campaign's latest stop.
	LastStop string

	// CompletedPositions counts committed stream positions.
	CompletedPositions uint64

	// Options holds the saved campaign choices and bounds.
	Options campaign.Options

	// Targets holds per-target outcomes.
	Targets map[string]campaign.TargetMetrics

	// Checks holds the committed checks.
	Checks map[string]campaign.ModeMetrics

	// IncompleteChecks holds the interrupted check attempts.
	IncompleteChecks map[string]campaign.ModeMetrics

	// WorkerModules records module versions linked into each worker binary.
	WorkerModules map[string]map[string]string
}

// inventoryOutput is the printed source inventory and execution evidence. Fields are in
// printed order.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type inventoryOutput struct {
	// Basis describes how the inventory was counted.
	Basis string

	// Profile identifies the generator profile.
	Profile string

	// Size is the generated program size.
	Size string

	// Targets lists the inventoried application targets.
	Targets []string

	// Seed is the generated window's base seed.
	Seed uint64

	// Start is the generated window's first position.
	Start uint64

	// CasesPerTarget counts the generated positions per target.
	CasesPerTarget uint64

	// Catalogue lists Piko-owned templates.
	Catalogue inventory.Catalogue `json:"Catalog"`

	// Corpus holds authored source features.
	Corpus inventory.Corpus

	// Generated holds generated source features.
	Generated inventory.Population

	// SourceComparison compares authored and generated source features.
	SourceComparison map[string]sourceGap

	// CompositionComparison compares authored and generated feature compositions.
	CompositionComparison map[string]sourceGap

	// PublicProviders lists the public provider contracts.
	PublicProviders []publiccontracts.Provider

	// Examples lists the authored example contracts.
	Examples []inventory.ExampleContract

	// Campaign holds optional saved campaign evidence.
	Campaign *executionEvidence `json:",omitempty"`
}

// applySuite applies a named target preset and validates explicit overrides.
//
// Takes command (string) which names the campaign CLI subcommand.
//
// Returns error which reports an unknown preset or incompatible explicit option, or nil
// on success.
func (options *flags) applySuite(command string) error {
	if options.suite == "" {
		return nil
	}
	if command != commandCampaign && command != commandPreview && command != commandDoctor {
		return errors.New("-suite applies to campaign, preview and doctor")
	}
	if options.engine != engineSeeded || options.fuzzer != "" {
		return errors.New("suites use the seeded engine")
	}
	if !options.set[flagTarget] {
		options.target = "all"
	}
	switch options.suite {
	case suiteSmoke:
		options.defaultCases(1)
	case suiteExtended:
		options.applyExtendedSuite()
	case "soak":
		if !options.set[flagDuration] && !options.set[flagCases] {
			options.duration = soakDuration
		}
	default:
		return fmt.Errorf("unknown suite %q (smoke, extended, soak)", options.suite)
	}
	return nil
}

// applyExtendedSuite applies the extended preset's size, case count and browser policy
// where they were not given explicitly.
func (options *flags) applyExtendedSuite() {
	if !options.set[flagSize] {
		options.size = suiteExtended
	}
	options.defaultCases(extendedCases)
	if !options.set[flagBrowser] {
		options.browser = browserEvery
	}
}

// defaultCases sets the case count when -n was not given explicitly.
//
// Takes cases (uint64) which is the preset case count.
func (options *flags) defaultCases(cases uint64) {
	if !options.set[flagCases] {
		options.n = cases
	}
}

// matrices accumulates finite-cell coverage for a previewed case.
//
// Takes previewed (*casefile.Case) which holds the saved sources, choices and expected
// results.
// Takes browser (bool) which selects browser execution.
func (target *previewTarget) matrices(previewed *casefile.Case, browser bool) {
	for _, matrix := range generate.Matrices(previewed) {
		if target.Matrices == nil {
			target.Matrices = map[string]*previewMatrix{}
		}
		counted := target.Matrices[matrix.Name]
		if counted == nil {
			counted = &previewMatrix{
				cells:       map[string]bool{},
				selected:    map[string]bool{},
				Environment: matrix.Environment,
				Feasible:    matrix.Feasible,
				Generated:   0,
				Selected:    0,
				Missing:     0,
			}
			target.Matrices[matrix.Name] = counted
		}
		for _, cell := range matrix.Cells {
			counted.cells[cell] = true
			if matrix.Environment != environmentBrowser || browser {
				counted.selected[cell] = true
			}
		}
		counted.Generated, counted.Selected = len(counted.cells), len(counted.selected)
		counted.Missing = counted.Feasible - counted.Generated
	}
}

// previewTools records the modes and tools that a target's kind requires.
//
// Takes options (*flags) which holds the parsed command options.
// Takes name (string) which is the target.
// Takes count (uint64) which is the number of previewed positions.
func (target *previewTarget) previewTools(options *flags, name string, count uint64) {
	switch name {
	case targetIncremental:
		target.Modes = []string{"warm interpreter fixture"}
		target.Tools = append(target.Tools, localPipit)
	case "wasm":
		target.Tools = append(target.Tools, "node")
	case targetExampleContracts:
		target.Modes = []string{"compiled authored example"}
		target.Tools = append(target.Tools, chromium)
		target.BrowserSelected = boundedCount(count)
	case "generated-artifacts":
		target.Tools = append(target.Tools, "node", "tsc")
	case "lsp":
		target.Tools = append(target.Tools, "pikopls")
	case "source-diagnostics":
		for offset := range count {
			seed, position := campaign.StreamPosition(options.suite, options.seed, options.index+offset)
			generated := generate.NewSized(name, seed, position, options.size)
			target.matrices(&generated, false)
		}
	case "database", "form-workflow":
		target.Tools = append(target.Tools, "local Seagull and SQLite")
	}
}

// previewApplication records the host modes, requests, browser routes and tools of an
// application target.
//
// Takes options (*flags) which holds the parsed command options.
// Takes name (string) which is the application target.
// Takes count (uint64) which is the number of previewed positions.
func (target *previewTarget) previewApplication(options *flags, name string, count uint64) {
	target.Modes = []string{options.mode}
	if options.mode == modeBoth {
		target.Modes = []string{modeInterp, modeCompiled}
	}
	if name == targetEmbeddedDeploy {
		target.Modes = []string{modeCompiled}
	}
	if slices.Contains(target.Modes, modeInterp) {
		target.Tools = append(target.Tools, localPipit)
	}
	target.HTTPRequests = map[string]int{}
	target.BrowserRoutes = map[string]int{}
	for offset := range count {
		seed, position := campaign.StreamPosition(options.suite, options.seed, options.index+offset)
		generated := generate.NewSized(name, seed, position, options.size)
		target.previewCase(&generated, campaign.BrowserCase(&generated, options.browser))
	}
	if target.BrowserSelected > 0 {
		target.Tools = append(target.Tools, chromium)
		if name == "checkout-workflow" {
			target.Tools = append(target.Tools, "pdftotext")
		}
		if name == "firefox-markup" {
			target.Tools = append(target.Tools, "Firefox (WebDriver BiDi)")
		}
	}
}

// previewCase counts the matrices, HTTP requests and browser routes of one generated
// application case.
//
// Takes generated (*casefile.Case) which is the generated case.
// Takes browser (bool) which reports whether the case runs in a browser.
func (target *previewTarget) previewCase(generated *casefile.Case, browser bool) {
	target.matrices(generated, browser)
	target.HTTPRequests[generated.Request]++
	for index := range generated.Probes {
		target.HTTPRequests[generated.Probes[index].Path]++
	}
	if !browser {
		return
	}
	target.BrowserSelected++
	routes, _ := runner.BrowserRouteResults(generated, &runner.Observation{}, false)
	for path := range routes {
		target.BrowserRoutes[path]++
	}
}

// preview prints the bounded target and cell preview.
//
// Takes options (*flags) which holds the parsed command options.
// Takes list ([]string) which lists the selected targets.
//
// Returns error which reports invalid generation bounds or output encoding failure, or
// nil on success.
func preview(options *flags, list []string) error {
	count := min(options.n, coverage.MaxCases)
	if options.duration > 0 && !options.set[flagCases] && options.suite != suiteExtended && options.suite != suiteSmoke {
		count = previewDurationCases
	}
	if options.index > math.MaxUint64-(count-1) {
		return errPreviewOverflow
	}
	report := previewReport{
		Targets:  nil,
		Suite:    options.suite,
		Browser:  options.browser,
		Basis:    previewBasis,
		Size:     options.size,
		Duration: options.duration.String(),
		Seed:     options.seed,
		Start:    options.index,
	}
	for _, name := range list {
		report.Targets = append(report.Targets, previewOne(options, name, count))
	}
	return printJSON(report)
}

// previewOne previews the planned checks of one target.
//
// Takes options (*flags) which holds the parsed command options.
// Takes name (string) which is the target.
// Takes count (uint64) which is the number of previewed positions.
//
// Returns previewTarget which summarises the target's planned checks.
func previewOne(options *flags, name string, count uint64) previewTarget {
	var row previewTarget
	row.Target = name
	row.Modes = []string{"package"}
	row.Tools = []string{engineGo}
	row.PreviewCases = boundedCount(count)
	row.previewTools(options, name, count)
	if targets.IsApplication(name) {
		row.previewApplication(options, name, count)
	}
	return row
}

// boundedCount converts a preview position count to an int, saturating at the largest
// int.
//
// Takes count (uint64) which is the position count.
//
// Returns int which is the count, or math.MaxInt when it does not fit.
func boundedCount(count uint64) int {
	if count > math.MaxInt {
		return math.MaxInt
	}
	return int(count)
}

// compareKeys finds authored source features missing from generated sources.
//
// Takes authored (map[string]inventory.Frequency) which holds feature frequencies in
// authored sources.
// Takes generated (map[string]inventory.Frequency) which holds feature frequencies in
// generated sources.
//
// Returns sourceGap which lists authored features missing from generated sources.
func compareKeys(authored, generated map[string]inventory.Frequency) sourceGap {
	var gap sourceGap
	for _, key := range slices.Sorted(maps.Keys(authored)) {
		if _, found := generated[key]; found {
			gap.Shared++
		} else {
			gap.Missing++
			gap.MissingKeys = append(gap.MissingKeys, key)
		}
	}
	return gap
}

// inventoryReport prints source inventory and available execution evidence.
//
// Takes repo (string) which is the repository source directory.
// Takes options (*flags) which holds the parsed command options.
// Takes directory (string) which is the optional saved campaign directory.
//
// Returns error which reports source-inventory, saved-report or output failure, or nil on
// success.
func inventoryReport(ctx context.Context, repo string, options *flags, directory string) error {
	actual, err := savedEvidence(directory)
	if err != nil {
		return err
	}
	cases := options.n
	if !options.set[flagCases] {
		cases = inventoryDefaultCases
	}
	if cases > coverage.MaxCases || options.index > math.MaxUint64-(cases-1) {
		return fmt.Errorf("inventory accepts between 1 and %d positions per target", coverage.MaxCases)
	}
	names, err := inventoryTargets(options)
	if err != nil {
		return err
	}
	output, err := authoredInventory(ctx, repo)
	if err != nil {
		return err
	}
	var generated inventory.Population
	for _, name := range names {
		for offset := range cases {
			generatedCase := generate.NewSized(name, options.seed, options.index+offset, options.size)
			if err := generated.AddCase(ctx, &generatedCase); err != nil {
				return err
			}
		}
	}
	output.SourceComparison = map[string]sourceGap{}
	output.CompositionComparison = map[string]sourceGap{}
	for name, population := range output.Corpus.Populations {
		output.SourceComparison[name] = compareKeys(population.Features, generated.Features)
		output.CompositionComparison[name] = compareKeys(population.Compositions, generated.Compositions)
	}
	output.Basis = inventoryBasis
	output.Profile = generate.ProfileSHA256
	output.Size = options.size
	output.Targets = names
	output.Seed = options.seed
	output.Start = options.index
	output.CasesPerTarget = cases
	output.Generated = generated
	output.Campaign = actual
	return printJSON(output)
}

// savedEvidence loads the execution evidence of an optional saved campaign.
//
// Takes directory (string) which is the saved campaign directory, or empty for none.
//
// Returns *executionEvidence which summarises the campaign, or nil when no directory was
// given.
// Returns error which reports an unreadable campaign report, or nil on success.
func savedEvidence(directory string) (*executionEvidence, error) {
	if directory == "" {
		return nil, nil
	}
	report, err := campaign.Load(directory)
	if err != nil {
		return nil, err
	}
	return &executionEvidence{
		Options:            report.Options,
		Targets:            report.Metrics.Targets,
		Checks:             report.Metrics.Checks,
		IncompleteChecks:   report.IncompleteChecks,
		WorkerModules:      report.Provenance.WorkerModules,
		Profile:            report.GeneratorProfile,
		Platform:           report.Runtime,
		Status:             report.Status,
		LastStop:           report.LastStop,
		CompletedPositions: report.Next,
	}, nil
}

// inventoryTargets selects the application targets an inventory generates.
//
// Takes options (*flags) which holds the parsed command options.
//
// Returns []string which lists the distinct application targets.
// Returns error which reports an unknown or repeated target, or nil on success.
func inventoryTargets(options *flags) ([]string, error) {
	if !options.set[flagTarget] || options.target == "all" {
		return targets.ApplicationNames(), nil
	}
	names := strings.Split(options.target, ",")
	for position, name := range names {
		if !targets.IsApplication(name) || slices.Contains(names[:position], name) {
			return nil, fmt.Errorf("inventory needs distinct application targets, unknown or repeated %q", name)
		}
	}
	return names, nil
}

// authoredInventory reads the Piko-owned catalogue, examples, public providers and
// authored source corpus of the repository.
//
// Takes repo (string) which is the repository source directory.
//
// Returns inventoryOutput which holds the authored inventory fields.
// Returns error which reports an unreadable repository or inventory failure, or nil on
// success.
func authoredInventory(ctx context.Context, repo string) (output inventoryOutput, err error) {
	root, err := os.OpenRoot(repo)
	if err != nil {
		return output, err
	}
	defer func() {
		err = errors.Join(err, root.Close())
	}()
	if output.Catalogue, err = inventory.Discover(root.FS()); err != nil {
		return output, err
	}
	if output.Examples, err = inventory.ExampleContracts(root.FS()); err != nil {
		return output, err
	}
	if output.PublicProviders, err = publiccontracts.Providers(root.FS()); err != nil {
		return output, err
	}
	output.Corpus, err = inventory.Repository(ctx, root.FS())
	return output, err
}
