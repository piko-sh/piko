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

// Command piko-fuzz runs and replays local Piko correctness campaigns.
package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"piko.sh/json"

	"piko.sh/piko/internal/testutil/pipitpending"
	"piko.sh/piko/tests/fuzz/internal/campaign"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/coverage"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/targets"
	"piko.sh/safedisk"
)

const (
	// warmInterpreterMessage explains the warm-server requirement for incremental checks.
	warmInterpreterMessage = "incremental needs a warm interpreted server. Use graph-rebuild for native snapshots"

	// deploymentMessage explains the compiled-artefact requirement for deployment checks.
	deploymentMessage = "embedded-deploy requires a compiled artefact and supports -mode compiled or both"

	// exampleMessage explains the compiled and browser requirements for example checks.
	exampleMessage = "example-contracts runs authored production binaries with Chromium and requires -mode compiled or both and -browser sample or every"

	// toolsDirectoryName is the campaign-state subdirectory that holds per-process tool
	// binaries.
	toolsDirectoryName = "tools"

	// defaultReduceDuration bounds a reduction when no -duration is given.
	defaultReduceDuration = 10 * time.Minute

	// defaultCases is the default case count per target.
	defaultCases = 100

	// privateDirectoryMode is the permission of directories the command creates.
	privateDirectoryMode = 0o700

	// campaignTimeLayout names a new campaign directory by its UTC start time.
	campaignTimeLayout = campaign.DirectoryTimeLayout

	// commandCampaign starts a new campaign.
	commandCampaign = "campaign"

	// commandResume resumes a saved campaign.
	commandResume = "resume"

	// commandReplay replays a saved case or native campaign.
	commandReplay = "replay"

	// commandReduce reduces a saved failing case.
	commandReduce = "reduce"

	// commandStatus prints a saved campaign's status line.
	commandStatus = "status"

	// commandPreview previews a campaign's planned checks.
	commandPreview = "preview"

	// commandGenerate prints or writes one generated case.
	commandGenerate = "gen"

	// modeInterp selects the interpreted application host.
	modeInterp = "interp"

	// modeCompiled selects the compiled application host.
	modeCompiled = "compiled"

	// modeBoth selects both application hosts.
	modeBoth = "both"

	// browserOff disables browser execution.
	browserOff = "off"

	// browserSample selects browser execution for a sample of cases.
	browserSample = "sample"

	// browserEvery selects browser execution for every case.
	browserEvery = "every"

	// engineGo selects native Go coverage-guided fuzzing.
	engineGo = "go"

	// engineSeeded selects seeded project generation.
	engineSeeded = "seeded"

	// flagTarget names the -target option.
	flagTarget = "target"

	// flagMode names the -mode option.
	flagMode = "mode"

	// flagBrowser names the -browser option.
	flagBrowser = "browser"

	// flagEngine names the -engine option.
	flagEngine = "engine"

	// flagFuzzer names the -fuzzer option.
	flagFuzzer = "fuzzer"

	// flagCases names the -n option.
	flagCases = "n"

	// flagDuration names the -duration option.
	flagDuration = "duration"

	// flagSize names the -size option.
	flagSize = "size"

	// targetExampleContracts names the authored example contracts target.
	targetExampleContracts = "example-contracts"

	// targetEmbeddedDeploy names the embedded deployment target.
	targetEmbeddedDeploy = "embedded-deploy"

	// targetIncremental names the warm interpreter incremental target.
	targetIncremental = "incremental"

	// statusPass is the recorded status of a passing run.
	statusPass = "pass"

	// statusFailed is the recorded status of a native run that found a failure.
	statusFailed = campaign.StatusFailed
)

var (
	// openBudget verifies the campaign limits and returns its resource budget.
	openBudget = resource.Open

	// errInterpretedNeedsLocalPipit reports that interpreted mode requires local Pipit
	// sources.
	errInterpretedNeedsLocalPipit = errors.New("interpreted mode needs local pipit sources: set PIKO_PIPIT_LOCAL=1 and use ./hack/fuzz/fuzz.sh workspace --local-pipit")

	// errReduceDuration is the cause recorded when a reduction reaches its time limit.
	errReduceDuration = fmt.Errorf("reduction duration reached: %w", context.DeadlineExceeded)

	// errReplayDuration is the cause recorded when a replay reaches its time limit.
	errReplayDuration = fmt.Errorf("replay duration reached: %w", context.DeadlineExceeded)
)

// flags holds parsed campaign command options.
type flags struct {
	// set tracks explicitly supplied options.
	set map[string]bool

	// target selects one target, a comma-separated list or all targets.
	target string

	// mode selects the application host mode.
	mode string

	// browser selects the browser execution policy.
	browser string

	// state is the local campaign state directory.
	state string

	// out is the new destination directory.
	out string

	// engine selects seeded generation or native Go fuzzing.
	engine string

	// fuzzer names the registered native Go fuzz function.
	fuzzer string

	// suite names the selected suite preset.
	suite string

	// size selects the generated program size.
	size string

	// mem is the requested aggregate memory cap.
	mem string

	// reserve is the requested minimum available host memory.
	reserve string

	// disk is the requested campaign storage budget.
	disk string

	// seed is the deterministic stream's base seed.
	seed uint64

	// index is the first case position.
	index uint64

	// n bounds the case count per target.
	n uint64

	// duration limits the campaign's elapsed time.
	duration time.Duration

	// cpus is the aggregate CPU ceiling.
	cpus int

	// jobs is the number of concurrent case slots.
	jobs int

	// runtimeCoverage enables instrumented runtime-coverage collection.
	runtimeCoverage bool

	// keepGoing continues after a saved failure.
	keepGoing bool
}

// generatedCase is a generated case printed with its sources.
//
//nolint:govet // fieldalignment: fields are in printed JSON order
type generatedCase struct {
	// Case is the generated case.
	casefile.Case

	// Sources holds the generated source files by path.
	Sources map[string]string
}

// parse parses command options and retains positional arguments.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes arguments ([]string) which supplies the command arguments.
//
// Returns flags which holds parsed command options.
// Returns []string which holds remaining positional arguments.
// Returns error which reports an invalid command option or value, or nil on success.
func parse(command string, arguments []string) (flags, []string, error) {
	var options flags
	flagSet := flag.NewFlagSet(command, flag.ContinueOnError)
	registerFlags(flagSet, &options)

	var paths []string
	if len(arguments) > 0 && !strings.HasPrefix(arguments[0], "-") {
		paths = append(paths, arguments[0])
		arguments = arguments[1:]
	}
	if err := flagSet.Parse(arguments); err != nil {
		return options, nil, err
	}
	paths = append(paths, flagSet.Args()...)
	options.set = map[string]bool{}
	flagSet.Visit(func(supplied *flag.Flag) { options.set[supplied.Name] = true })
	if err := validateOptions(command, &options); err != nil {
		return options, nil, err
	}
	err := options.applySuite(command)
	return options, paths, err
}

// registerFlags binds every command option to its field in options.
//
// Takes flagSet (*flag.FlagSet) which receives the option definitions.
// Takes options (*flags) which receives the parsed values.
func registerFlags(flagSet *flag.FlagSet, options *flags) {
	flagSet.StringVar(&options.target, flagTarget, "pages", "one target, a comma-separated list, or all")
	flagSet.StringVar(&options.mode, flagMode, modeBoth, "interp, compiled or both")
	flagSet.StringVar(&options.browser, flagBrowser, browserSample, "off, sample or every")
	flagSet.StringVar(&options.engine, flagEngine, engineSeeded, "seeded project generation or go coverage-guided fuzzing")
	flagSet.BoolVar(&options.runtimeCoverage, "runtime-coverage", false,
		"retain instrumented Go host statements and selected Chromium JS functions per application case/mode")
	flagSet.StringVar(&options.size, flagSize, "", "generated program size: tiny, medium or extended")
	flagSet.StringVar(&options.suite, "suite", "", "smoke, extended or soak; -target can narrow the selection")
	flagSet.BoolVar(&options.keepGoing, "keep-going", false, "continue after confirmed discrepancies and retain a failing exit status")
	flagSet.StringVar(&options.fuzzer, flagFuzzer, "", "registered Go fuzz function (see help)")
	flagSet.StringVar(&options.state, "state", os.Getenv("PIKO_FUZZ_STATE"), "local campaign directory")
	flagSet.StringVar(&options.out, "out", "", "new destination directory")
	flagSet.Uint64Var(&options.seed, "seed", 1, "base seed")
	flagSet.Uint64Var(&options.index, "index", 0, "case index for gen")
	flagSet.Uint64Var(&options.n, flagCases, defaultCases, "case count per target")
	flagSet.DurationVar(&options.duration, flagDuration, 0, "wall-time budget")
	flagSet.IntVar(&options.cpus, "cpus", 2, "aggregate CPU ceiling")
	flagSet.IntVar(&options.jobs, "jobs", 1, "concurrent case slots")
	flagSet.StringVar(&options.mem, "mem", "4GiB", "aggregate memory cap")
	flagSet.StringVar(&options.reserve, "min-free", "", "minimum available host memory")
	flagSet.StringVar(&options.disk, "disk", "20GiB", "campaign storage budget")
}

// validateOptions checks parsed option values that do not depend on the target selection.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes options (*flags) which holds the parsed command options.
//
// Returns error which reports an invalid option value, or nil on success.
func validateOptions(command string, options *flags) error {
	validMode := slices.Contains([]string{modeBoth, modeInterp, modeCompiled}, options.mode)
	validBrowser := slices.Contains([]string{browserSample, browserOff, browserEvery}, options.browser)
	validEngine := slices.Contains([]string{engineGo, engineSeeded}, options.engine)
	if !validMode || !validBrowser || !validEngine {
		return errors.New("invalid mode, browser policy or engine")
	}
	if options.mode == modeInterp && !pipitpending.Local() {
		return errInterpretedNeedsLocalPipit
	}
	if options.duration < 0 || options.n == 0 || options.jobs != 1 || options.cpus < 1 {
		return errors.New("use positive CPU and case counts, one job and a non-negative duration")
	}
	if !slices.Contains([]string{"", "tiny", "medium", "extended"}, options.size) {
		return errors.New("unknown generator size")
	}
	sizedCommands := []string{commandGenerate, "coverage", "inventory", commandPreview, commandCampaign}
	if options.set[flagSize] && !slices.Contains(sizedCommands, command) {
		return fmt.Errorf("%s uses the saved generation size", command)
	}
	return nil
}

// allTargets lists application and package targets.
//
// Returns []string which lists application targets followed by package targets.
func allTargets() []string {
	return slices.Concat(targets.ApplicationNames(), targets.List())
}

// selected validates the requested target selection.
//
// Takes options (*flags) which holds the parsed command options.
//
// Returns []string which lists validated target names.
// Returns error which reports an unknown or incompatible target selection, or nil on
// success.
func selected(options *flags) ([]string, error) {
	known := allTargets()
	list := strings.Split(options.target, ",")
	if options.target == "all" {
		list = allSelectableTargets(options, known)
	}
	seen := map[string]bool{}
	hasApplication := false
	for _, target := range list {
		if !slices.Contains(known, target) || seen[target] {
			return nil, fmt.Errorf("unknown or repeated target %q", target)
		}
		seen[target] = true
		hasApplication = hasApplication || targets.IsApplication(target)
	}
	if err := validateSelection(options, seen, hasApplication); err != nil {
		return nil, err
	}
	return list, nil
}

// allSelectableTargets lists every known target the selected host mode and browser policy
// can run.
//
// Takes options (*flags) which holds the parsed command options.
// Takes known ([]string) which lists every known target.
//
// Returns []string which lists the runnable targets in known order.
func allSelectableTargets(options *flags, known []string) []string {
	list := slices.Clone(known)
	if options.mode == modeInterp || options.browser == browserOff {
		list = slices.DeleteFunc(list, func(target string) bool { return target == targetExampleContracts })
	}
	if options.mode == modeInterp {
		list = slices.DeleteFunc(list, func(target string) bool { return target == targetEmbeddedDeploy })
	}
	if options.mode == modeCompiled {
		list = slices.DeleteFunc(list, func(target string) bool { return target == targetIncremental })
	}
	return list
}

// validateSelection checks that the selected targets support the explicit host mode and
// browser options.
//
// Takes options (*flags) which holds the parsed command options.
// Takes seen (map[string]bool) which holds the selected target names.
// Takes hasApplication (bool) which reports whether an application target is selected.
//
// Returns error which reports an incompatible option, or nil on success.
func validateSelection(options *flags, seen map[string]bool, hasApplication bool) error {
	if seen[targetEmbeddedDeploy] && options.mode == modeInterp {
		return errors.New(deploymentMessage)
	}
	if seen[targetExampleContracts] && (options.mode == modeInterp || options.browser == browserOff) {
		return errors.New(exampleMessage)
	}
	warm := seen[targetIncremental]
	if warm && options.mode == modeCompiled {
		return errors.New(warmInterpreterMessage)
	}
	if options.set[flagBrowser] && !hasApplication && !seen[targetExampleContracts] {
		return errors.New("-browser requires an application target")
	}
	if options.set[flagMode] && !hasApplication && !warm && !seen[targetExampleContracts] {
		return errors.New("-mode requires an application target or incremental")
	}
	return nil
}

// help prints command usage.
func help() {
	fmt.Println("Use ./hack/fuzz/fuzz.sh {doctor,gen,promote,coverage,inventory,preview,campaign,replay,reduce,status,report,resume}")
	fmt.Println("inventory compares Piko-owned templates and generated applications. preview -suite smoke|extended|soak lists obligations and tools.")
	fmt.Println("coverage -target pages -seed 1 -n 100 -browser sample inventories generated template directives (max 512 cases).")
	fmt.Println("campaign -runtime-coverage retains per-host Go profiles and V8 coverage for selected browser cases. replay accepts the same flag.")
	fmt.Println("Targets:", strings.Join(allTargets(), ", "))
	fmt.Println("Go fuzz functions (-engine go -fuzzer NAME):", strings.Join(slices.Sorted(maps.Keys(targets.Fuzzers())), ", "))
}

// printJSON writes an indented JSON value to standard output.
//
// Takes value (any) which is the value to encode.
//
// Returns error which reports JSON encoding or standard-output write failure, or nil on
// success.
func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding output: %w", err)
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

// latest finds the last named campaign directory containing a saved report.
//
// Takes state (string) which is the campaign state directory.
//
// Returns string which is the last named campaign directory with a saved report.
// Returns error which reports an unreadable or empty campaign directory, or nil on
// success.
func latest(state string) (latestDirectory string, err error) {
	campaigns := filepath.Join(state, "campaigns")
	root, err := os.OpenRoot(campaigns)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, root.Close())
	}()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return "", err
	}
	for _, entry := range slices.Backward(entries) {
		if !entry.IsDir() {
			continue
		}
		for _, name := range []string{"report.json", "native.json"} {
			_, err := root.Stat(filepath.Join(entry.Name(), name))
			if err == nil {
				return filepath.Join(campaigns, entry.Name()), nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return "", err
			}
		}
	}
	return "", errors.New("no local campaigns")
}

// rootDirectory finds the repository containing the campaign module.
//
// Returns string which is the repository root.
// Returns error which reports that the repository root could not be found, or nil on
// success.
func rootDirectory() (string, error) {
	if root := os.Getenv("PIKO_FUZZ_ROOT"); root != "" {
		return filepath.Abs(root)
	}
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err = os.Stat(filepath.Join(current, "tests", "fuzz", "go.mod")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("run from the Piko checkout or set PIKO_FUZZ_ROOT")
		}
		current = parent
	}
}

// prepareToolsDirectory creates this process's private tool directory under the campaign
// state and removes directories left behind by processes that have exited.
//
// Takes state (string) which is the campaign state directory.
//
// Returns string which is the new tool directory.
// Returns error which reports a directory that cannot be created or a stale directory
// that cannot be removed, or nil on success.
func prepareToolsDirectory(state string) (string, error) {
	root := filepath.Join(state, toolsDirectoryName)
	if err := createPrivateDirectory(state, toolsDirectoryName); err != nil {
		return "", fmt.Errorf("creating tool directory root: %w", err)
	}
	if err := removeStaleTools(root); err != nil {
		return "", err
	}
	directory, err := os.MkdirTemp(root, strconv.Itoa(os.Getpid())+"-")
	if err != nil {
		return "", fmt.Errorf("creating tool directory: %w", err)
	}
	return directory, nil
}

// removeStaleTools deletes tool directories whose owning process is no longer running.
//
// Takes directory (string) which is the directory holding per-process tool directories.
//
// Returns error which joins every failed removal, or nil on success.
func removeStaleTools(directory string) (err error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return fmt.Errorf("opening tool directories: %w", err)
	}
	defer func() {
		err = errors.Join(err, root.Close())
	}()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return fmt.Errorf("listing tool directories: %w", err)
	}
	var failures []error
	for _, entry := range entries {
		if toolsOwnerRunning(entry.Name()) {
			continue
		}
		if err := root.RemoveAll(entry.Name()); err != nil {
			failures = append(failures, fmt.Errorf("removing stale tool directory %s: %w", entry.Name(), err))
		}
	}
	return errors.Join(failures...)
}

// toolsOwnerRunning reports whether the process named by a tool directory is still
// running.
//
// Takes name (string) which is the tool directory name, prefixed with its owner's process
// ID.
//
// Returns bool which is true when another live process owns the directory.
func toolsOwnerRunning(name string) bool {
	owner, _, found := strings.Cut(name, "-")
	if !found {
		return false
	}
	processID, err := strconv.Atoi(owner)
	if err != nil || processID <= 0 || processID == os.Getpid() {
		return false
	}
	process, err := os.FindProcess(processID)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

// run executes the requested campaign command. Campaign commands monitor resource
// pressure and cancel active work when limits are exceeded.
//
// Takes arguments ([]string) which supplies the command arguments.
//
// Returns error which reports invalid options, unavailable resources or the campaign
// command failure.
func run(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 || arguments[0] == "help" {
		help()
		return nil
	}
	command := arguments[0]
	if command == commandPlan {
		return plan(ctx, os.Stdout, arguments[1:])
	}
	options, paths, err := parse(command, arguments[1:])
	if err != nil {
		return err
	}
	repo, err := prepareState(&options)
	if err != nil {
		return err
	}
	list, nativeCampaign, err := selectForCommand(command, &options)
	if err != nil {
		return err
	}
	path, err := commandPath(paths)
	if err != nil {
		return err
	}
	handled, err := runOfflineCommand(ctx, command, repo, &options, path, list)
	if handled {
		return err
	}
	return runCampaignCommand(ctx, command, repo, &options, path, list, nativeCampaign)
}

// prepareState finds the repository and resolves the absolute campaign state directory.
//
// Takes options (*flags) which holds the parsed command options and receives the resolved
// state directory.
//
// Returns string which is the repository root.
// Returns error which reports an unknown repository or an unresolvable state directory,
// or nil on success.
func prepareState(options *flags) (string, error) {
	repo, err := rootDirectory()
	if err != nil {
		return "", err
	}
	options.state, err = filepath.Abs(cmp.Or(options.state, filepath.Join(repo, ".piko", "fuzz")))
	if err != nil {
		return "", err
	}
	return repo, nil
}

// selectForCommand selects the targets of a seeded command or validates a native
// campaign's options.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes options (*flags) which holds the parsed command options and receives the selected
// engine and fuzzer.
//
// Returns []string which lists the selected seeded targets.
// Returns bool which is true when the command starts a native Go fuzz campaign.
// Returns error which reports an invalid selection, or nil on success.
func selectForCommand(command string, options *flags) ([]string, bool, error) {
	if command == commandCampaign {
		defaultNativeEngine(options)
	}
	nativeCampaign := command == commandCampaign && (options.engine == engineGo || options.fuzzer != "")
	var list []string
	if nativeCampaign {
		if err := validateNativeOptions(options); err != nil {
			return nil, false, err
		}
	} else if command == commandCampaign || command == commandGenerate || command == commandPreview {
		var err error
		if list, err = selected(options); err != nil {
			return nil, false, err
		}
	}
	seededApplication := command == commandCampaign && !nativeCampaign && slices.ContainsFunc(list, targets.IsApplication)
	if options.set["runtime-coverage"] && command != commandReplay && !seededApplication {
		return nil, false, errors.New("-runtime-coverage applies to seeded application campaigns; resume uses its saved setting")
	}
	return list, nativeCampaign, nil
}

// defaultNativeEngine selects the Go engine for a duration-bounded campaign that names a
// registered Go fuzzer without choosing an engine or fuzzer.
//
// Takes options (*flags) which holds the parsed command options and receives the selected
// engine.
func defaultNativeEngine(options *flags) {
	if options.suite != "" || options.duration <= 0 || options.set[flagEngine] || options.set[flagFuzzer] {
		return
	}
	if _, ok := targets.Fuzzers()[options.target]; ok {
		options.engine = engineGo
	}
}

// commandPath resolves the optional positional directory argument.
//
// Takes paths ([]string) which holds the positional arguments.
//
// Returns string which is the absolute directory, or empty when none was given.
// Returns error which reports more than one directory or an unresolvable path, or nil on
// success.
func commandPath(paths []string) (string, error) {
	if len(paths) > 1 {
		return "", errors.New("at most one directory is accepted")
	}
	if len(paths) == 0 {
		return "", nil
	}
	return filepath.Abs(paths[0])
}

// runOfflineCommand dispatches commands that run outside the campaign runner.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the optional positional directory.
// Takes list ([]string) which lists the selected targets.
//
// Returns bool which is true when the command was handled here.
// Returns error which reports the handled command's failure or an unknown command, or nil
// on success.
func runOfflineCommand(ctx context.Context, command, repo string, options *flags, path string, list []string) (bool, error) {
	switch command {
	case "promote":
		return true, promote(path, options.out)
	case "inventory":
		return true, inventoryReport(ctx, repo, options, path)
	case commandPreview:
		if path != "" {
			return true, fmt.Errorf("%s accepts flags only", command)
		}
		return true, preview(options, list)
	case "coverage":
		return true, coverageReport(ctx, options, path)
	case commandGenerate:
		return true, generateCase(options, list)
	case commandStatus, "report":
		return true, statusReport(command, options.state, path)
	case commandDoctor:
		defaultNativeEngine(options)
		return true, doctor(ctx, repo, options)
	case commandCampaign, commandResume, commandReplay, commandReduce:
		return false, nil
	default:
		return true, fmt.Errorf("unknown command %q", command)
	}
}

// generateCase prints or writes the generated case of the one selected target.
//
// Takes options (*flags) which holds the parsed command options.
// Takes list ([]string) which lists the selected targets.
//
// Returns error which reports a selection other than one target or an output failure, or
// nil on success.
func generateCase(options *flags, list []string) error {
	if len(list) != 1 {
		return errors.New("gen takes one target")
	}
	generated := generate.NewSized(list[0], options.seed, options.index, options.size)
	if options.out == "" {
		return printJSON(generatedCase{Case: generated, Sources: generated.Files})
	}
	return casefile.Write(options.out, &generated)
}

// runCampaignCommand opens the resource budget and runs a campaign, resume, replay or
// reduce command.
//
// Takes command (string) which names the campaign CLI subcommand.
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the optional saved campaign or case directory.
// Takes list ([]string) which lists the selected seeded targets.
// Takes nativeCampaign (bool) which selects a native Go fuzz campaign.
//
// Returns error which reports unavailable resources or the command's failure, or nil on
// success.
//
// Concurrency: Holds the per-user campaign lock while the command runs. A resource
// monitor can cancel active work.
func runCampaignCommand(ctx context.Context, command, repo string, options *flags, path string, list []string, nativeCampaign bool) error {
	budget, err := openBudget(ctx)
	if err != nil {
		return fmt.Errorf("campaign budget: %w (use ./hack/fuzz/fuzz.sh)", err)
	}
	release, err := resource.Lock()
	if err != nil {
		return err
	}
	defer release()
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	go budget.Monitor(ctx, cancel)
	workerDirectory, err := prepareToolsDirectory(options.state)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(workerDirectory); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "removing tool directory:", err)
		}
	}()
	var harness campaign.Harness
	harness.Repo = repo
	harness.State = options.state
	harness.WorkersDir = workerDirectory
	harness.Budget = budget
	if command == commandResume || command == commandReplay || command == commandReduce {
		return resumeOrReplay(ctx, harness, command, options, path)
	}
	if path != "" {
		return errors.New("campaign accepts flags; use replay to run a case directory")
	}
	if nativeCampaign {
		return startNative(ctx, harness, options)
	}
	return startSeeded(ctx, harness, repo, options, list)
}

// startNative creates a native campaign directory and runs the selected Go fuzzer.
//
// Takes harness (campaign.Harness) which runs the native campaign.
// Takes options (*flags) which holds the parsed command options.
//
// Returns error which reports a directory or native-run failure, or nil on success.
func startNative(ctx context.Context, harness campaign.Harness, options *flags) error {
	parent := filepath.Join(options.state, "campaigns")
	if err := createPrivateDirectory(options.state, "campaigns"); err != nil {
		return err
	}
	directory, err := os.MkdirTemp(parent, time.Now().UTC().Format(campaignTimeLayout)+"-go-")
	if err != nil {
		return err
	}
	fmt.Println("Campaign", directory)
	var native campaign.NativeRun
	native.Fuzzer = options.fuzzer
	native.Started = time.Now().UTC()
	native.Duration = options.duration
	native.Cases = options.n
	return harness.Native(ctx, directory, &native, false)
}

// startSeeded creates a seeded campaign and executes it.
//
// Takes harness (campaign.Harness) which runs the campaign.
// Takes repo (string) which is the repository root.
// Takes options (*flags) which holds the parsed command options.
// Takes list ([]string) which lists the selected targets.
//
// Returns error which reports a campaign creation or execution failure, or nil on
// success.
func startSeeded(ctx context.Context, harness campaign.Harness, repo string, options *flags, list []string) error {
	cases := options.n
	if options.duration > 0 && !options.set[flagCases] && options.suite != suiteSmoke && options.suite != suiteExtended {
		cases = math.MaxUint64 / uint64(len(list))
	}
	campaignOptions := campaign.Options{
		Size:            options.size,
		Suite:           options.suite,
		RuntimeCoverage: options.runtimeCoverage,
		Targets:         list,
		Mode:            options.mode,
		Browser:         options.browser,
		Seed:            options.seed,
		Cases:           cases,
		Duration:        options.duration,
		KeepGoing:       options.keepGoing,
	}
	directory, report, err := campaign.New(ctx, repo, options.state, campaignOptions)
	if err != nil {
		return err
	}
	fmt.Println("Campaign", directory)
	return harness.Execute(ctx, directory, report)
}

// validateNativeOptions checks and completes the options of a Go fuzz campaign.
//
// Takes options (*flags) which holds the parsed command options and receives the selected
// fuzzer.
//
// Returns error which reports an option that does not apply to Go fuzzing or an
// unregistered fuzzer, or nil on success.
func validateNativeOptions(options *flags) error {
	if options.set[flagSize] {
		return errors.New("-size applies to seeded generation")
	}
	if options.set[flagMode] || options.set[flagBrowser] {
		return errors.New("native Go fuzz targets run independently of app mode and browser")
	}
	if options.set[flagEngine] && options.engine == engineSeeded {
		return errors.New("-fuzzer selects the Go engine")
	}
	if options.set[flagTarget] && options.fuzzer != "" && options.fuzzer != options.target {
		return errors.New("-target and -fuzzer select different Go fuzz targets")
	}
	options.fuzzer = cmp.Or(options.fuzzer, options.target)
	if _, ok := targets.Fuzzers()[options.fuzzer]; !ok {
		return errors.New("choose a registered -fuzzer from ./hack/fuzz/fuzz.sh help")
	}
	return nil
}

// coverageReport prints the template directive inventory of a saved case or a generated
// window.
//
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the optional saved case directory.
//
// Returns error which reports an unreadable case, an invalid window or an inventory
// failure, or nil on success.
func coverageReport(ctx context.Context, options *flags, path string) error {
	if path != "" {
		saved, err := casefile.Read(path)
		if err != nil {
			return err
		}
		if !targets.IsApplication(saved.Target) {
			return errors.New("coverage takes a saved application case")
		}
		collector := coverage.New(saved.Target, options.browser, saved.Seed, saved.Index)
		if err := collector.Add(ctx, &saved, campaign.BrowserCase(&saved, options.browser)); err != nil {
			return err
		}
		return printJSON(collector.Result())
	}
	if !targets.IsApplication(options.target) {
		return fmt.Errorf("coverage takes one application target: %s", strings.Join(targets.ApplicationNames(), ", "))
	}
	if options.n > coverage.MaxCases || options.index > math.MaxUint64-(options.n-1) {
		return fmt.Errorf("choose a coverage window of 1 to %d cases in range", coverage.MaxCases)
	}
	collector := coverage.New(options.target, options.browser, options.seed, options.index)
	for offset := range options.n {
		generated := generate.NewSized(options.target, options.seed, options.index+offset, options.size)
		if err := collector.Add(ctx, &generated, campaign.BrowserCase(&generated, options.browser)); err != nil {
			return err
		}
	}
	return printJSON(collector.Result())
}

// statusReport prints the status line or full report of a saved campaign.
//
// Takes command (string) which is "status" or "report".
// Takes state (string) which is the campaign state directory.
// Takes path (string) which is the campaign directory, or empty for the latest one.
//
// Returns error which reports an unreadable campaign or output failure, or nil on
// success.
func statusReport(command, state, path string) error {
	if path == "" {
		var err error
		if path, err = latest(state); err != nil {
			return err
		}
	}
	native, err := campaign.LoadNative(path)
	if err == nil {
		if command == commandStatus {
			fmt.Printf("%s\n%s, Go fuzzer %s, quota %.2f CPUs, cumulative kernel memory.peak %d bytes\n",
				path, native.Status, native.Fuzzer, native.Resources.QuotaCPUs, native.Resources.MemoryPeak)
			return nil
		}
		return printJSON(native)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	report, err := campaign.Load(path)
	if err != nil {
		return err
	}
	if command != commandStatus {
		return printJSON(report)
	}
	printSeededStatus(path, &report)
	return nil
}

// printSeededStatus prints the status lines of a saved seeded campaign.
//
// Takes path (string) which is the campaign directory.
// Takes report (*campaign.Report) which is the saved campaign report.
func printSeededStatus(path string, report *campaign.Report) {
	fmt.Printf("%s\n%s, next trial %d, active runtime %s\n%s\n",
		path, report.Status, report.Next, report.Elapsed.Round(time.Millisecond), report.Metrics.Summary())
	if report.Resources.Cgroup != "" {
		fmt.Printf("Cumulative kernel memory.peak %d bytes, memory.max %d bytes, CPU quota %.2f\n",
			report.Resources.MemoryPeak, report.Resources.MemoryMax, report.Resources.QuotaCPUs)
	}
	for _, mode := range []string{modeInterp, modeCompiled} {
		runtime, ok := report.RuntimeCoverage[mode]
		if !ok {
			continue
		}
		fmt.Printf("%s runtime coverage: %d Go receipts, %d Chromium receipts, latest %s (%d/%d Go statements",
			mode, runtime.Runs, runtime.BrowserRuns, runtime.LastArtefact, runtime.LastGoStatements.Covered, runtime.LastGoStatements.Total)
		if functions := runtime.LastJavaScriptFunctions; functions != nil {
			fmt.Printf(", %d/%d JS functions", functions.Covered, functions.Total)
		}
		fmt.Println(")")
	}
	if report.LastStop != "" {
		fmt.Println(report.LastStop)
	}
}

// resumeOrReplay resumes a saved campaign or replays or reduces a saved case.
//
// Takes harness (campaign.Harness) which runs cases with the opened resource budget.
// Takes command (string) which is "resume", "replay" or "reduce".
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the saved campaign or case directory.
//
// Returns error which reports an invalid option, an unreadable directory or the run's
// failure, or nil on success.
func resumeOrReplay(ctx context.Context, harness campaign.Harness, command string, options *flags, path string) error {
	if path == "" {
		if command != commandResume {
			return errors.New("a saved case directory is required")
		}
		var err error
		if path, err = latest(options.state); err != nil {
			return err
		}
	}
	native, err := campaign.LoadNative(path)
	if err == nil {
		return resumeNative(ctx, harness, command, options, path, &native)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if command == commandResume {
		return resumeSeeded(ctx, harness, options, path)
	}
	return replaySavedCase(ctx, harness, command, options, path)
}

// resumeSeeded resumes a saved seeded campaign with any explicit bound overrides.
//
// Takes harness (campaign.Harness) which runs the campaign.
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the saved campaign directory.
//
// Returns error which reports an override of a saved selection, an unreadable report or
// the run's failure, or nil on success.
func resumeSeeded(ctx context.Context, harness campaign.Harness, options *flags, path string) error {
	report, err := campaign.Load(path)
	if err != nil {
		return err
	}
	for _, key := range []string{flagTarget, flagMode, flagBrowser, "seed", flagEngine, flagFuzzer} {
		if options.set[key] {
			return fmt.Errorf("resume uses the saved -%s setting", key)
		}
	}
	if options.set[flagDuration] {
		report.Options.Duration = options.duration
	}
	if options.set[flagCases] {
		report.Options.Cases = options.n
	}
	if options.set["keep-going"] {
		report.Options.KeepGoing = options.keepGoing
	}
	fmt.Println("Campaign", path)
	return harness.Execute(ctx, path, &report)
}

// resumeNative resumes or replays a saved Go fuzz campaign.
//
// Takes harness (campaign.Harness) which runs the native campaign.
// Takes command (string) which is "resume", "replay" or "reduce".
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the saved campaign directory.
// Takes native (*campaign.NativeRun) which is the saved native campaign and receives the
// explicit bound overrides.
//
// Returns error which reports an option that does not apply to Go fuzzing or the run's
// failure, or nil on success.
func resumeNative(ctx context.Context, harness campaign.Harness, command string, options *flags, path string, native *campaign.NativeRun) error {
	if options.runtimeCoverage {
		return errors.New("runtime coverage uses application hosts")
	}
	if command == commandReduce {
		return errors.New("native Go fuzz minimises failures during the original campaign")
	}
	if command == commandResume && native.Status == statusPass {
		fmt.Println("Native campaign already completed", path)
		return nil
	}
	if command == commandResume && native.Status == statusFailed {
		return errors.New("native campaign recorded a failure; use replay to check its saved corpus")
	}
	if options.set[flagDuration] {
		native.Duration = options.duration
	}
	if options.set[flagCases] {
		native.Cases = options.n
	}
	return harness.Native(ctx, path, native, command == commandReplay)
}

// replaySavedCase replays or reduces one saved application case.
//
// Takes harness (campaign.Harness) which runs the case.
// Takes command (string) which is "replay" or "reduce".
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the saved case directory.
//
// Returns error which reports an invalid option, an unreadable case or a failing result,
// or nil on success.
func replaySavedCase(ctx context.Context, harness campaign.Harness, command string, options *flags, path string) error {
	saved, err := casefile.Read(path)
	if err != nil {
		return err
	}
	savedOptions, err := campaign.SavedOptions(path)
	if err != nil {
		return err
	}
	if err := overrideReplayOptions(&saved, options, &savedOptions); err != nil {
		return err
	}
	if command == commandReduce {
		return reduceSavedCase(ctx, harness, options, path, &saved, savedOptions)
	}
	if options.duration > 0 {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeoutCause(ctx, options.duration, errReplayDuration)
		defer stop()
	}
	if options.runtimeCoverage {
		if harness, err = replayCoverageHarness(harness, &saved); err != nil {
			return err
		}
	}
	result := harness.RunOne(ctx, &saved, savedOptions.Mode, savedOptions.Browser)
	if err := printJSON(result); err != nil {
		return err
	}
	if result.Kind != statusPass {
		return fmt.Errorf("%s: %s", result.Kind, result.Message)
	}
	return nil
}

// reduceSavedCase reduces a saved failing case within the requested time limit.
//
// Takes harness (campaign.Harness) which runs the reduction attempts.
// Takes options (*flags) which holds the parsed command options.
// Takes path (string) which is the saved case directory.
// Takes saved (*casefile.Case) which is the saved case.
// Takes savedOptions (campaign.Options) which holds the case's replay options.
//
// Returns error which reports the reduction's failure, or nil on success.
func reduceSavedCase(ctx context.Context, harness campaign.Harness, options *flags, path string, saved *casefile.Case, savedOptions campaign.Options) error {
	out := cmp.Or(options.out, filepath.Join(path, "minimal"))
	limit := cmp.Or(options.duration, defaultReduceDuration)
	bounded, stop := context.WithTimeoutCause(ctx, limit, errReduceDuration)
	defer stop()
	attempts, err := harness.Reduce(bounded, path, out, saved, savedOptions)
	fmt.Printf("%d reduction attempts, output %s\n", attempts, out)
	return err
}

// replayCoverageHarness returns a harness that retains runtime coverage of a replay in a
// new directory under the campaign state.
//
// Takes harness (campaign.Harness) which is the replay harness.
// Takes saved (*casefile.Case) which is the saved case.
//
// Returns campaign.Harness which records runtime coverage.
// Returns error which reports a non-application case or a directory failure, or nil on
// success.
func replayCoverageHarness(harness campaign.Harness, saved *casefile.Case) (campaign.Harness, error) {
	if !targets.IsApplication(saved.Target) {
		return harness, errors.New("runtime coverage uses application hosts")
	}
	base := filepath.Join(harness.State, "replays")
	if err := createPrivateDirectory(harness.State, "replays"); err != nil {
		return harness, err
	}
	out, err := os.MkdirTemp(base, "runtime-coverage-")
	if err != nil {
		return harness, err
	}
	covered, err := harness.WithRuntimeCoverage(out)
	if err != nil {
		return harness, err
	}
	_, _ = fmt.Fprintln(os.Stderr, "Runtime coverage", out)
	return covered, nil
}

// overrideReplayOptions applies explicit -mode and -browser options to a saved case's
// options.
//
// Takes saved (*casefile.Case) which is the saved case.
// Takes options (*flags) which holds the parsed command options.
// Takes savedOptions (*campaign.Options) which receives the overrides.
//
// Returns error which reports an override the case's target does not support, or nil on
// success.
func overrideReplayOptions(saved *casefile.Case, options *flags, savedOptions *campaign.Options) error {
	if options.set[flagMode] {
		if err := replayModeSupported(saved.Target, options.mode); err != nil {
			return err
		}
		savedOptions.Mode = options.mode
	}
	if options.set[flagBrowser] {
		if saved.Target == targetExampleContracts && options.browser == browserOff {
			return errors.New(exampleMessage)
		}
		if !targets.IsApplication(saved.Target) && saved.Target != targetExampleContracts {
			return errors.New("-browser applies to application targets")
		}
		savedOptions.Browser = options.browser
	}
	return nil
}

// replayModeSupported checks that a saved case's target can run in an explicit host mode.
//
// Takes target (string) which is the saved case's target.
// Takes mode (string) which is the requested host mode.
//
// Returns error which reports an unsupported mode, or nil on success.
func replayModeSupported(target, mode string) error {
	if target == targetExampleContracts && mode == modeInterp {
		return errors.New(exampleMessage)
	}
	if target == targetEmbeddedDeploy && mode == modeInterp {
		return errors.New(deploymentMessage)
	}
	if target == targetIncremental && mode == modeCompiled {
		return errors.New(warmInterpreterMessage)
	}
	if !targets.IsApplication(target) && target != targetIncremental && target != targetExampleContracts {
		return errors.New("-mode requires an application target or incremental")
	}
	return nil
}

// createPrivateDirectory creates a private directory confined beneath the campaign state
// directory, creating the state directory first when it is missing.
//
// Takes state (string) which is the operator-chosen campaign state directory.
// Takes name (string) which is the directory to create, relative to state.
//
// Returns error which reports a directory that cannot be created or that would escape
// state, or nil on success.
func createPrivateDirectory(state, name string) (err error) {
	sandbox, err := safedisk.NewSandbox(state, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("opening state directory %s: %w", state, err)
	}
	defer func() { err = errors.Join(err, sandbox.Close()) }()
	if err := sandbox.MkdirAll(name, privateDirectoryMode); err != nil {
		return fmt.Errorf("creating %s under %s: %w", name, state, err)
	}
	return nil
}

// execute runs the campaign command under a signal-cancelled context.
//
// Returns int which is the process exit status.
func execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, stop)
	if err := run(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// main runs the campaign command and reports its exit status.
func main() {
	os.Exit(execute())
}
