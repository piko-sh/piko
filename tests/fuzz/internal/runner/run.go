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

package runner

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
)

const (
	// browserActionMax bounds the saved browser actions replayed in one case.
	browserActionMax = 12

	// browserReceiptName is the browser worker's receipt, relative to the working directory.
	browserReceiptName = "counter.json"

	// browserWorkerArgumentMax is the largest browser worker command line, in arguments.
	browserWorkerArgumentMax = 4

	// chromeSocketSuffix is the longest path Chrome appends below its temporary directory
	// for its singleton Unix-domain socket.
	chromeSocketSuffix = "/piko-chrome-0123456789/.org.chromium.Chromium.XXXXXX/SingletonSocket"

	// clientTimeout bounds one HTTP request to the host.
	clientTimeout = 15 * time.Second

	// fallbackSocketRoot is the short temporary root used when os.TempDir is too deep for a
	// Unix-domain socket path.
	fallbackSocketRoot = "/tmp"

	// firefoxReceiptName is the Firefox DOM receipt, relative to the working directory.
	firefoxReceiptName = "firefox-dom.json"

	// hostStartAttempts bounds host launches when the reserved port is taken.
	hostStartAttempts = 3

	// readinessInterval separates readiness polls.
	readinessInterval = 50 * time.Millisecond

	// readinessTimeout bounds how long a host may take to become ready.
	readinessTimeout = 90 * time.Second

	// redactedWorkerLogMax bounds the redacted worker-log tail kept for session-bearing
	// cases, in bytes.
	redactedWorkerLogMax = 4 << 10

	// snapshotBodyMax bounds the diagnostic body read from a failed coverage snapshot.
	snapshotBodyMax = 512

	// unixSocketPathMax is the Linux sun_path size, including the terminating NUL byte.
	unixSocketPathMax = 108

	// privateDirectoryMode is the permission of directories the harness creates.
	privateDirectoryMode = 0o700

	// privateFileMode is the permission of files the harness writes.
	privateFileMode = 0o600

	// interpProbeFormat wraps a failure while staging an application-specific interpreted
	// host.
	interpProbeFormat = "interp-probe: %w"

	// modeInterpreted selects the interpreted server mode.
	modeInterpreted = "interp"

	// modeCompiled selects the compiled server mode.
	modeCompiled = "compiled"

	// targetFormWorkflow is the form workflow target.
	targetFormWorkflow = "form-workflow"
)

var (
	// collectionRegistrySource is the interpreted host registry for a case that uses the
	// external collection module and exports its symbols to the interpreter.
	collectionRegistrySource = `package main
import (
	"reflect"
	_ "` + casefile.Module + `/dist"
	catalog "` + casefile.CollectionModule + `/entries"
)
func init() {
	extraSymbols = map[string]map[string]reflect.Value{
		"` + casefile.CollectionModule + `/entries": {"Origin": reflect.ValueOf(catalog.Origin)},
	}
}
`

	// collectionParserSource gives the compiled host of a collections case the goldmark
	// markdown parser.
	collectionParserSource = `package main
import (
	"piko.sh/piko"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
)
func init() {
	collectionParser = func() piko.Option { return piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()) }
}
`

	// errRuntimeCoverage marks a browser worker failure while starting or collecting runtime
	// coverage.
	errRuntimeCoverage = errors.New("browser runtime coverage failed")

	// errHostPortTaken marks a host that exited because its reserved port was taken.
	errHostPortTaken = errors.New("host could not bind its reserved port")

	// errHostStopped is the cause recorded when the runner stops a host.
	errHostStopped = errors.New("host stopped by runner")

	// errHostStartupTimeout is the cause recorded when a host misses its readiness deadline.
	// It wraps context.DeadlineExceeded so the campaign files it as a resource stop rather
	// than a confirmable product timeout.
	errHostStartupTimeout = fmt.Errorf("host startup timed out: %w", context.DeadlineExceeded)

	// errWorkerExitedEarly reports a host that exited without a process error before
	// becoming ready.
	errWorkerExitedEarly = errors.New("worker exited before readiness")

	// secretPattern matches cookie, token, CSRF and session values in worker output.
	secretPattern = regexp.MustCompile(`(?i)((?:set-)?cookie|token|csrf[\w-]*|session[\w-]*|authorization|bearer)(["']?\s*[:=]?\s*["']?\s*)((?:bearer\s+|basic\s+)?[^\s"';,&]+)`)
)

// hostProcess is one running host and the outcome recorded when it exits.
type hostProcess struct {
	// err holds the host's exit error, valid after done is closed.
	err error

	// stop cancels the host with a cause.
	stop context.CancelCauseFunc

	// done is closed once the host has exited and log and err are set.
	done chan struct{}

	// log holds the bounded host output, valid after done is closed.
	log string

	// base is the host's application base URL.
	base string

	// token is the readiness identity the host echoes once it serves requests.
	token string

	// coverageSecret authorises flush requests and stays private to the harness and host.
	coverageSecret string

	// reported records that err is already part of a returned error.
	reported bool
}

// hostLaunch holds the inputs shared by every host start attempt.
type hostLaunch struct {
	// client polls readiness.
	client *http.Client

	// follow records source-watch events, when selected.
	follow *catalogueWatch

	// app is the host's working directory.
	app string

	// host is the host executable.
	host string

	// env is the host environment without its port and token.
	env []string
}

// browserRun holds the inputs of one browser worker run.
type browserRun struct {
	// c holds the saved sources, choices and expected results.
	c *casefile.Case

	// observation receives the browser evidence.
	observation *Observation

	// app is the worker's working directory.
	app string

	// work is the invocation-private working directory.
	work string

	// base is the application base URL.
	base string

	// browserBin is the browser worker executable.
	browserBin string

	// jsCoverage is the private JavaScript coverage receipt path.
	jsCoverage string

	// env is the worker environment without its per-run socket and coverage paths.
	env []string
}

// clientRecipe is one client journey recipe passed to the host through its environment.
type clientRecipe struct {
	// recipe is the journey input to encode.
	recipe any

	// name identifies the journey in failure messages.
	name string

	// variable is the environment variable carrying the encoded recipe.
	variable string
}

// MeasuredRun holds the inputs of one instrumented application run.
type MeasuredRun struct {
	// Case holds the saved sources, choices and expected results.
	Case *casefile.Case

	// Repo is the repository source directory.
	Repo string

	// Work is the invocation-private working directory.
	Work string

	// Workers is the invocation-private worker executable directory.
	Workers string

	// Mode selects the server mode.
	Mode string

	// Output is the private receipt output path.
	Output string

	// Browser selects browser execution.
	Browser bool
}

// applicationRun holds the inputs shared by every phase of one application run.
type applicationRun struct {
	// c holds the saved sources, choices and expected results.
	c *casefile.Case

	// follow records source-watch events and rebuild evidence, when selected.
	follow *catalogueWatch

	// repo is the repository source directory.
	repo string

	// work is the invocation-private working directory.
	work string

	// workers is the invocation-private worker executable directory, empty for work.
	workers string

	// app is the private generated application directory, set once staged.
	app string

	// workspaceFile is the private Go workspace file, set once staged.
	workspaceFile string

	// mode selects the server mode.
	mode string

	// coverageDirectory is the private runtime-coverage output directory, empty when
	// coverage is off.
	coverageDirectory string

	// browser selects browser execution.
	browser bool
}

// stopAndWait stops the host, waits for it to exit and returns its exit error unless the
// runner's own stop caused it.
//
// Returns error which is the host's own exit failure, or nil.
func (h *hostProcess) stopAndWait() error {
	h.stop(errHostStopped)
	<-h.done
	if h.err == nil || errors.Is(h.err, errHostStopped) {
		return nil
	}
	return h.err
}

// run drives the browser worker and records its receipt.
//
// Returns error which reports recipe, worker, receipt or evidence failure, or nil on
// success.
func (b *browserRun) run(ctx context.Context) (err error) {
	socketDirectory, err := os.MkdirTemp(browserSocketRoot(), "piko-chrome-")
	if err != nil {
		return fmt.Errorf("infrastructure: browser socket directory: %w", err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(socketDirectory)) }()
	output := filepath.Join(b.work, browserReceiptName)
	arguments, err := b.arguments(output)
	if err != nil {
		return err
	}
	b.env, err = b.environment()
	if err != nil {
		return err
	}
	env := append(slices.Clone(b.env), "PIKO_FUZZ_BROWSER_TMP="+socketDirectory, "PIKO_FUZZ_JS_COVERAGE="+b.jsCoverage)
	if err = runBrowserWorker(ctx, b.app, env, "browser", b.browserBin, arguments...); err != nil {
		if !errors.Is(err, errRuntimeCoverage) {
			return err
		}
		infrastructure := fmt.Errorf("infrastructure: %w", err)
		if recovered := b.recoverPartial(); recovered != nil {
			return errors.Join(recovered, infrastructure)
		}
		return infrastructure
	}
	return b.readReceipt(ctx)
}

// arguments builds the browser worker's command line.
//
// Takes output (string) which is the worker's receipt path.
//
// Returns []string which holds the worker arguments.
// Returns error which reports an invalid or oversized browser recipe, or nil on success.
func (b *browserRun) arguments(output string) ([]string, error) {
	c := b.c
	arguments := make([]string, 0, browserWorkerArgumentMax)
	arguments = append(arguments, b.base+c.Request, c.Expected.Tag, output)
	if workflow := c.Expected.FormWorkflow; workflow != nil {
		if _, err := workflow.Model(); err != nil {
			return nil, fmt.Errorf("workflow/recipe: %w", err)
		}
		recipe, err := json.Marshal(workflow)
		if err != nil {
			return nil, fmt.Errorf("workflow/recipe: %w", err)
		}
		return []string{b.base + "/workflow", "--form-workflow", output, string(recipe)}, nil
	}
	clientModel := c.Expected.Client
	if clientModel == nil || len(clientModel.Steps) == 0 {
		return arguments, nil
	}
	if len(clientModel.Steps) > browserActionMax {
		return nil, fmt.Errorf("browser-recipe: exceeds %d actions", browserActionMax)
	}
	actions := make([]casefile.ClientAction, 0, len(clientModel.Steps))
	for _, step := range clientModel.Steps {
		actions = append(actions, step.Action)
	}
	recipe, err := json.Marshal(actions)
	if err != nil {
		return nil, fmt.Errorf("browser-recipe: %w", err)
	}
	return append(arguments, string(recipe)), nil
}

// environment builds the browser worker's private environment.
//
// Returns []string which holds the worker environment.
// Returns error which reports an invalid browser program or DOM plan, or nil on success.
func (b *browserRun) environment() ([]string, error) {
	c := b.c
	env := slices.Clone(b.env)
	if model := c.Expected.FormShim; model != nil && model.BrowserSubmission {
		env = append(env, "PIKO_FUZZ_INTAKE_BROWSER=true")
	}
	if c.BrowserSampleVersion > 0 && c.Expected.FormWorkflow == nil {
		paths, err := c.BrowserProgramPaths()
		if err != nil {
			return nil, fmt.Errorf("browser-recipe: program paths: %w", err)
		}
		recipe, err := json.Marshal(paths)
		if err != nil {
			return nil, fmt.Errorf("browser-recipe: program paths: %w", err)
		}
		env = append(env, "PIKO_FUZZ_BROWSER_PROGRAM_PATHS="+string(recipe))
	}
	if c.Expected.DOM != nil && c.Target != "firefox-markup" {
		recipe, err := domRecipe(c.Expected.DOM)
		if err != nil {
			return nil, err
		}
		env = append(env, "PIKO_FUZZ_DOM="+recipe)
	}
	return env, nil
}

// recoverPartial keeps the verified browser evidence written before a runtime-coverage
// failure.
//
// Every journey follows one policy: an absent receipt is tolerated, and any valid
// receipt, clean or failed, is recorded and checked.
//
// Returns error which reports an invalid receipt or a model mismatch in the recovered
// evidence, or nil.
func (b *browserRun) recoverPartial() error {
	expected := b.c.Expected
	if workflow := expected.FormWorkflow; workflow != nil {
		return recoverPartialForm(b.work, workflow, b.observation)
	}
	if client := expected.Client; client != nil && client.PartialComponents != nil {
		return recoverPartialComponents(b.work, client.PartialComponents, b.observation)
	}
	if client := expected.Client; client != nil && client.Lifecycle != nil {
		return recoverPartialLifecycle(b.work, client.Lifecycle, b.observation)
	}
	return nil
}

// readReceipt decodes the browser worker's receipt and gathers dependent evidence.
//
// Returns error which reports an unreadable or invalid receipt, or failed dependent
// evidence, or nil on success.
func (b *browserRun) readReceipt(ctx context.Context) error {
	o := b.observation
	data, err := readBrowserReceipt(b.work, browserReceiptName)
	if err != nil {
		return fmt.Errorf("browser-result: %w", err)
	}
	if b.c.Expected.FormWorkflow != nil {
		o.FormWorkflow = new(casefile.FormWorkflowObservation)
		if err = json.Unmarshal(data, o.FormWorkflow); err != nil {
			return fmt.Errorf("browser-result: invalid workflow receipt: %w", err)
		}
		return nil
	}
	o.Client = new(casefile.ClientTransition)
	if err = json.Unmarshal(data, o.Client); err != nil {
		return fmt.Errorf("browser-result: invalid shadow DOM observation %q: %w", data, err)
	}
	if len(o.Client.Before) == 0 || len(o.Client.After) == 0 {
		return fmt.Errorf("browser-result: invalid shadow DOM observation %q", data)
	}
	if b.c.Target == "firefox-markup" {
		if err := b.readFirefoxDOM(ctx); err != nil {
			return err
		}
	}
	o.Counts = []string{o.Client.Before[0].Count, o.Client.After[0].Count}
	if checkout := o.Client.Checkout; checkout != nil && checkout.DriverError == "" {
		checkout.Documents, err = readCheckoutMail(ctx, b.work)
		if err != nil {
			return checkoutError{err: err, step: "invoice", field: "read", detail: ""}
		}
	}
	return nil
}

// readFirefoxDOM drives the Firefox DOM journey and records its bounded receipt.
//
// Returns error which reports a missing plan, worker failure or invalid receipt, or nil
// on success.
func (b *browserRun) readFirefoxDOM(ctx context.Context) error {
	if b.c.Expected.DOM == nil {
		return errors.New("browser-recipe: Firefox case missing DOM plan")
	}
	recipe, err := domRecipe(b.c.Expected.DOM)
	if err != nil {
		return err
	}
	path := filepath.Join(b.work, firefoxReceiptName)
	if _, err = command(ctx, b.app, b.env, "firefox", b.browserBin, b.base, "--firefox-dom", path, recipe); err != nil {
		return err
	}
	data, err := readBrowserReceipt(b.work, firefoxReceiptName)
	if err != nil {
		return fmt.Errorf("firefox-result: %w", err)
	}
	b.observation.Client.DOM = new(casefile.DOMTrace)
	if err = json.Unmarshal(data, b.observation.Client.DOM); err != nil {
		return fmt.Errorf("firefox-result: %w", err)
	}
	return nil
}

// command runs a bounded subprocess and retains its diagnostic output.
//
// Takes directory (string) which is the subprocess working directory.
// Takes env ([]string) which supplies the private process environment.
// Takes phase (string) which identifies the subprocess execution phase.
// Takes binary (string) which is the executable path.
// Takes arguments (...string) which supplies the command arguments.
//
// Returns string which is the bounded subprocess output.
// Returns error which reports a subprocess or browser-protocol failure, or nil on
// success.
func command(ctx context.Context, directory string, env []string, phase, binary string, arguments ...string) (string, error) {
	log, err := process.Run(ctx, directory, env, binary, arguments...)
	if err != nil {
		return log, fmt.Errorf("%s: %w\n%s", phase, err, log)
	}
	return log, nil
}

// runBrowserWorker runs the browser worker and classifies its failure.
//
// The worker exits with casefile.RuntimeCoverageExitCode when starting or collecting
// runtime coverage fails. Such a failure is wrapped with errRuntimeCoverage so callers
// can recover the evidence written before coverage collection.
//
// Takes directory (string) which is the worker's working directory.
// Takes env ([]string) which supplies the private process environment.
// Takes phase (string) which identifies the execution phase.
// Takes binary (string) which is the browser worker executable.
// Takes arguments (...string) which supplies the worker arguments.
//
// Returns error which reports the worker failure, wrapping errRuntimeCoverage for a
// runtime-coverage failure, or nil on success.
func runBrowserWorker(ctx context.Context, directory string, env []string, phase, binary string, arguments ...string) error {
	log, err := process.Run(ctx, directory, env, binary, arguments...)
	if err == nil {
		return nil
	}
	if exit, ok := errors.AsType[*exec.ExitError](err); ok && exit.ExitCode() == casefile.RuntimeCoverageExitCode {
		return fmt.Errorf("%s: %w: %w\n%s", phase, errRuntimeCoverage, err, log)
	}
	return fmt.Errorf("%s: %w\n%s", phase, err, log)
}

// buildCachedBinary builds a Go binary once into its cached path and installs it
// atomically.
//
// The binary is linked to a temporary name in the destination directory and renamed into
// place, so an interrupted or concurrent build never exposes a partial executable.
//
// Takes directory (string) which is the build's working directory.
// Takes env ([]string) which supplies the private process environment.
// Takes phase (string) which identifies the build phase.
// Takes binary (string) which is the cached executable path.
// Takes packages (...string) which selects the packages or files to build.
//
// Returns string which is the cached executable path.
// Returns error which reports inspection, build or installation failure, or nil on
// success.
func buildCachedBinary(ctx context.Context, directory string, env []string, phase, binary string, packages ...string) (string, error) {
	if _, err := os.Stat(binary); err == nil {
		return binary, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%s: inspect cached binary: %w", phase, err)
	}
	staged, err := os.CreateTemp(filepath.Dir(binary), filepath.Base(binary)+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("%s: stage binary: %w", phase, err)
	}
	stagedPath := staged.Name()
	if err = staged.Close(); err != nil {
		return "", errors.Join(fmt.Errorf("%s: stage binary: %w", phase, err), removeStagedBinary(stagedPath))
	}
	arguments := append([]string{"build", "-buildvcs=false", "-o", stagedPath}, packages...)
	if _, err = command(ctx, directory, env, phase, "go", arguments...); err != nil {
		return "", errors.Join(err, removeStagedBinary(stagedPath))
	}
	if err = os.Rename(stagedPath, binary); err != nil {
		return "", errors.Join(fmt.Errorf("%s: install binary: %w", phase, err), removeStagedBinary(stagedPath))
	}
	return binary, nil
}

// removeStagedBinary deletes a partially built binary.
//
// Takes path (string) which is the staged binary path.
//
// Returns error which reports a removal failure other than absence, or nil.
func removeStagedBinary(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove staged binary %s: %w", path, err)
	}
	return nil
}

// RunWithWorkers reuses immutable tool binaries in an invocation-private directory.
//
// Each application, native link, worker process and browser profile is fresh.
//
// Takes repo (string) which is the repository source directory.
// Takes work (string) which is the invocation-private working directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes mode (string) which selects the server mode.
// Takes browser (bool) which selects browser execution.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports application staging, build, host execution or
// independent-check failure, or nil on success.
func RunWithWorkers(ctx context.Context, repo, work, workers string, c *casefile.Case, mode string, browser bool) (Observation, error) {
	return runApplication(ctx, &applicationRun{
		c:                 c,
		follow:            nil,
		repo:              repo,
		work:              work,
		workers:           workers,
		app:               "",
		workspaceFile:     "",
		mode:              mode,
		coverageDirectory: "",
		browser:           browser,
	})
}

// RunMeasured instruments the linked server, and Chromium when selected.
//
// Output is private to this phase until the supervisor publishes a complete receipt.
//
// Takes run (*MeasuredRun) which holds the case, directories, mode, browser selection and
// private receipt output path.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports application staging, build, host execution or
// independent-check failure, or nil on success.
func RunMeasured(ctx context.Context, run *MeasuredRun) (Observation, error) {
	return runApplication(ctx, &applicationRun{
		c:                 run.Case,
		follow:            nil,
		repo:              run.Repo,
		work:              run.Work,
		workers:           run.Workers,
		app:               "",
		workspaceFile:     "",
		mode:              run.Mode,
		coverageDirectory: run.Output,
		browser:           run.Browser,
	})
}

// runApplication stages and runs a fresh generated application.
//
// Takes run (*applicationRun) which holds the run inputs. Its app and workspace file are
// set once the application is staged.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports application staging, build, host execution or
// independent-check failure, or nil on success.
func runApplication(ctx context.Context, run *applicationRun) (Observation, error) {
	var o Observation
	if run.mode != modeInterpreted && run.mode != modeCompiled {
		return o, fmt.Errorf("mode: unknown %q", run.mode)
	}
	if run.c.Target == "content-editions" {
		return runEditions(ctx, run)
	}
	if run.c.Target == "catalog-rebuild" {
		return runCatalogueRebuild(ctx, run)
	}
	app, workspaceFile, err := stage(run.repo, run.work, run.c)
	if err != nil {
		return o, fmt.Errorf("stage: %w", err)
	}
	run.app, run.workspaceFile = app, workspaceFile
	if run.c.Target == "embedded-deploy" {
		if run.mode != modeCompiled {
			return o, errors.New("mode: embedded deployment requires compiled mode")
		}
		return runDeployment(ctx, run)
	}
	return runStagedApplication(ctx, run)
}

// applicationEnvironment builds the private process environment of a staged application.
//
// Takes work (string) which is the invocation-private working directory.
// Takes workspaceFile (string) which is the private Go workspace file.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes follow (bool) which selects catalogue source watching.
//
// Returns []string which holds the private environment.
// Returns error which reports recipe encoding or private directory failure, or nil on
// success.
func applicationEnvironment(work, workspaceFile string, c *casefile.Case, follow bool) ([]string, error) {
	tags := strings.Join(append([]string{c.Expected.Tag}, c.Expected.AdditionalComponents...), ",")
	env := []string{
		"GOWORK=" + workspaceFile, "GOPROXY=off", "GOSUMDB=off",
		"GOFLAGS=" + os.Getenv("GOFLAGS") + " -buildvcs=false", "GOTOOLCHAIN=local",
		"PIKO_FUZZ_TAG=" + tags, "PIKO_FUZZ_COLLECTIONS=" + strconv.FormatBool(c.UsesCollections()),
	}
	if follow {
		env = append(env, "PIKO_FUZZ_CATALOG_WATCH=true")
	}
	if c.Target == "shipped-components" {
		env = append(env, "PIKO_FUZZ_SHIPPED=true")
	}
	if client := c.Expected.Client; client != nil {
		recipes, err := clientRecipeEnvironment(work, client)
		if err != nil {
			return nil, err
		}
		env = append(env, recipes...)
	}
	if c.Expected.Action != nil && c.Expected.Action.Upload {
		directory := filepath.Join(work, "upload-store")
		if err := os.Mkdir(directory, privateDirectoryMode); err != nil {
			return nil, fmt.Errorf("upload/store: %w", err)
		}
		env = append(env, "PIKO_FUZZ_UPLOAD_DIR="+directory)
	}
	if c.Target == targetFormWorkflow {
		env = append(env, "PIKO_FUZZ_WORKFLOW_DB="+filepath.Join(work, "workflow.sqlite"))
	}
	return env, nil
}

// clientRecipeEnvironment encodes the client journeys' recipes for the host.
//
// Takes work (string) which is the invocation-private working directory.
// Takes client (*casefile.ClientTransition) which holds the modelled client journeys.
//
// Returns []string which holds the recipe environment entries.
// Returns error which reports recipe encoding or outbox creation failure.
func clientRecipeEnvironment(work string, client *casefile.ClientTransition) ([]string, error) {
	var env []string
	for _, entry := range clientRecipes(client) {
		data, err := json.Marshal(entry.recipe)
		if err != nil {
			return nil, fmt.Errorf("%s/recipe: %w", entry.name, err)
		}
		env = append(env, entry.variable+"="+string(data))
	}
	if client.Checkout != nil {
		outbox := filepath.Join(work, checkoutOutboxName)
		if err := os.Mkdir(outbox, privateDirectoryMode); err != nil {
			return nil, fmt.Errorf("checkout/outbox: %w", err)
		}
		env = append(env, "PIKO_FUZZ_CHECKOUT_OUTBOX="+outbox)
	}
	return env, nil
}

// clientRecipes lists the recipes of the client journeys a case models.
//
// Takes client (*casefile.ClientTransition) which holds the modelled client journeys.
//
// Returns []clientRecipe which lists the present journeys' recipes.
func clientRecipes(client *casefile.ClientTransition) []clientRecipe {
	var recipes []clientRecipe
	if client.Lifecycle != nil {
		recipes = append(recipes, clientRecipe{recipe: client.Lifecycle.Input, name: "lifecycle", variable: "PIKO_FUZZ_LIFECYCLE"})
	}
	if client.Checkout != nil {
		recipes = append(recipes, clientRecipe{recipe: client.Checkout.Input, name: "checkout", variable: "PIKO_FUZZ_CHECKOUT_INPUT"})
	}
	if client.PartialComponents != nil {
		recipes = append(recipes, clientRecipe{recipe: client.PartialComponents.Input, name: "partial-components", variable: "PIKO_FUZZ_PARTIAL_COMPONENTS"})
	}
	if client.Events != nil {
		recipes = append(recipes, clientRecipe{recipe: client.Events.Input, name: "events", variable: "PIKO_FUZZ_EVENTS"})
	}
	return recipes
}

// runStagedApplication builds and runs the already staged application.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports application staging, build, host execution or
// independent-check failure, or nil on success.
func runStagedApplication(ctx context.Context, run *applicationRun) (Observation, error) {
	staged := *run
	staged.workers = cmp.Or(run.workers, run.work)
	var o Observation
	env, err := applicationEnvironment(staged.work, staged.workspaceFile, staged.c, staged.follow != nil)
	if err != nil {
		return o, err
	}
	o.GeneratedArtifacts, err = generateApplication(ctx, &staged, env)
	if err != nil {
		return o, err
	}
	o.ComponentCompiled = staged.c.Expected.Tag != ""
	host, err := buildHost(ctx, &staged, env)
	if err != nil {
		return o, err
	}
	browserBin := ""
	if staged.browser {
		browserBin, err = process.BuildWorker(ctx, staged.repo, staged.workers, "browser-worker", "")
		if err != nil {
			return o, err
		}
	}
	rendered, err := runHost(ctx, &staged, env, host, browserBin)
	rendered.GeneratedArtifacts = o.GeneratedArtifacts
	if err == nil && staged.c.Expected.Topology != nil {
		err = restartTopologyHost(ctx, &staged, env, host, &rendered)
	}
	if err == nil && rendered.Action != nil && rendered.Action.Upload != nil && !staged.c.Expected.Action.UploadRemoval {
		rendered.Action.Upload.DiskDigest, err = readStoredUpload(staged.work)
	}
	rendered.Mode = staged.mode
	rendered.ComponentCompiled = o.ComponentCompiled
	return rendered, err
}

// generateApplication runs the project's generator and any generated-artefact checks.
//
// Both Piko CLI paths run the project's generator before starting the server. In
// particular dev-i components import the actions module emitted by "all".
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
// Takes env ([]string) which supplies the private process environment.
//
// Returns *GeneratedArtifactReceipt which holds generated-artefact evidence, when the
// target exercises it.
// Returns error which reports workflow, generator or artefact failure, or nil on success.
func generateApplication(ctx context.Context, run *applicationRun, env []string) (*GeneratedArtifactReceipt, error) {
	if run.c.Target == targetFormWorkflow {
		if err := generateWorkflowSQL(ctx, run.app, run.work, run.workers, env); err != nil {
			return nil, err
		}
	}
	generator, err := process.BuildWorker(ctx, run.repo, run.workers, "generate-worker", "")
	if err != nil {
		return nil, err
	}
	if _, err = command(ctx, run.app, env, "generation", generator); err != nil {
		return nil, err
	}
	if run.c.Target != "generated-artifacts" {
		return nil, nil
	}
	return exerciseGeneratedArtifacts(ctx, run.app, run.work, generator, env, run.c)
}

// buildHost links the host executable for the selected mode, instrumenting it for
// coverage when requested.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
// Takes env ([]string) which supplies the private process environment.
//
// Returns string which is the host executable path.
// Returns error which reports coverage setup or link failure, or nil on success.
func buildHost(ctx context.Context, run *applicationRun, env []string) (string, error) {
	packages := "piko.sh/piko/...,example.org/pikofuzzcase/..."
	if run.mode == modeInterpreted {
		packages += ",pipit.sh/pipit/..."
	}
	buildArguments := []string{"build"}
	if run.coverageDirectory != "" {
		if err := os.MkdirAll(filepath.Join(run.coverageDirectory, "raw"), privateDirectoryMode); err != nil {
			return "", fmt.Errorf("infrastructure: coverage directory: %w", err)
		}
		buildArguments = append(buildArguments, "-cover", "-covermode=atomic", "-coverpkg="+packages)
	}
	if run.mode == modeInterpreted {
		return buildInterpretedHost(ctx, run, env, buildArguments, packages)
	}
	return buildCompiledHost(ctx, run, env, buildArguments)
}

// restartTopologyHost restarts the exact image in the same checkout and records the fresh
// instance and module resolution.
//
// Process-owned service storage, package initialisation and retained callbacks must be
// fresh in the restarted host.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
// Takes env ([]string) which supplies the private process environment.
// Takes host (string) which is the host executable path.
// Takes rendered (*Observation) which receives the restarted instance and resolution.
//
// Returns error which reports a failed restart or module resolution, or nil on success.
func restartTopologyHost(ctx context.Context, run *applicationRun, env []string, host string, rendered *Observation) error {
	restart := *run
	restart.coverageDirectory, restart.follow = "", nil
	fresh, err := runHost(ctx, &restart, env, host, "")
	if fresh.Topology != nil {
		rendered.Topology.Instances = append(rendered.Topology.Instances, fresh.Topology.Instances...)
	}
	if err != nil {
		return err
	}
	rendered.Topology.Workspace, rendered.Topology.Module, err = topologyResolution(ctx, run.app, run.workspaceFile, env)
	return err
}

// buildInterpretedHost links the interpreted host, reusing the shared worker when the
// case needs no application-specific symbols.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
// Takes env ([]string) which supplies the private process environment.
// Takes buildArguments ([]string) which holds the go build arguments.
// Takes packages (string) which selects coverage instrumentation packages.
//
// Returns string which is the host executable path.
// Returns error which reports host staging or link failure, or nil on success.
func buildInterpretedHost(ctx context.Context, run *applicationRun, env, buildArguments []string, packages string) (string, error) {
	if !run.c.RequiresProjectInterpreterHost() {
		if run.coverageDirectory != "" {
			return process.BuildCoveredWorker(ctx, run.repo, run.workers, "interp-worker", packages)
		}
		return process.BuildWorker(ctx, run.repo, run.workers, "interp-worker", "")
	}

	hostDirectory := filepath.Join(run.app, "fuzzhost")
	if err := os.Mkdir(hostDirectory, privateDirectoryMode); err != nil {
		return "", fmt.Errorf(interpProbeFormat, err)
	}
	if err := copyRequired(run.repo, filepath.Join("tests", "fuzz", "cmd", "interp-worker", "main.go"), filepath.Join(hostDirectory, "main.go")); err != nil {
		return "", fmt.Errorf(interpProbeFormat, err)
	}
	registry, err := interpretedRegistry(hostDirectory, run.c)
	if err != nil {
		return "", err
	}
	if err := writeShimOptions(run.c.Target, hostDirectory); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(hostDirectory, "registry.go"), []byte(registry), privateFileMode); err != nil {
		return "", fmt.Errorf(interpProbeFormat, err)
	}
	host := filepath.Join(run.work, "interp-app")
	if _, err := command(ctx, run.app, env, "interp-build", "go", append(buildArguments, "-o", host, "./fuzzhost")...); err != nil {
		return "", err
	}

	if err := os.Rename(hostDirectory, filepath.Join(run.work, "interp-host-src")); err != nil {
		return "", fmt.Errorf(interpProbeFormat, err)
	}
	return host, nil
}

// interpretedRegistry selects the symbol registry source of an application-specific
// interpreted host.
//
// Takes hostDirectory (string) which is the host source directory.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns string which is the registry Go source.
// Returns error which reports a failed supporting-source write, or nil on success.
func interpretedRegistry(hostDirectory string, c *casefile.Case) (string, error) {
	registry := "package main\nimport _ \"" + casefile.Module + "/dist\"\n"
	if c.UsesCollections() {
		registry = collectionRegistrySource
	}
	if c.Target == targetFormWorkflow {
		registry = workflowRegistry
		if err := os.WriteFile(filepath.Join(hostDirectory, "workflow_db.go"), []byte(workflowDBHost), privateFileMode); err != nil {
			return "", fmt.Errorf(interpProbeFormat, err)
		}
	}
	if c.Target == "checkout-workflow" {
		registry = checkoutRegistry
	}
	if c.Expected.Topology != nil {
		registry = topologyRegistry
	}
	return registry, nil
}

// buildCompiledHost links the compiled host with the case's supporting sources.
//
// Takes run (*applicationRun) which holds the run inputs and the staged application.
// Takes env ([]string) which supplies the private process environment.
// Takes buildArguments ([]string) which holds the go build arguments.
//
// Returns string which is the host executable path.
// Returns error which reports source staging or link failure, or nil on success.
func buildCompiledHost(ctx context.Context, run *applicationRun, env, buildArguments []string) (string, error) {
	c, app := run.c, run.app
	if c.UsesCollections() {
		if err := os.WriteFile(filepath.Join(app, "collection_parser.go"), []byte(collectionParserSource), privateFileMode); err != nil {
			return "", fmt.Errorf("native-probe: %w", err)
		}
	}
	if c.Target == targetFormWorkflow {
		if err := os.WriteFile(filepath.Join(app, "workflow_db.go"), []byte(workflowDBHost), privateFileMode); err != nil {
			return "", fmt.Errorf("native-probe: %w", err)
		}
	}
	if err := writeShimOptions(c.Target, app); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(nativeMain), privateFileMode); err != nil {
		return "", fmt.Errorf("native-probe: %w", err)
	}
	host := filepath.Join(run.work, "native")
	if _, err := command(ctx, app, env, "native-build", "go", append(buildArguments, "-o", host, ".")...); err != nil {
		return "", err
	}
	return host, nil
}

// runHost starts a fresh host, collects HTTP and browser evidence and waits for the host
// to exit during cleanup.
//
// Takes run (*applicationRun) which holds the run inputs, the application directory,
// coverage directory and optional source watch.
// Takes env ([]string) which supplies the private process environment.
// Takes host (string) which is the private server executable path.
// Takes browserBin (string) which is the private browser-worker executable.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports application staging, build, host execution or
// independent-check failure, joined with the host's own exit failure, or nil on success.
func runHost(ctx context.Context, run *applicationRun, env []string, host, browserBin string) (o Observation, err error) {
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	jar, err := cookiejar.New(nil)
	if err != nil {
		return o, fmt.Errorf("cookie jar: %w", err)
	}
	client := &http.Client{
		Jar:           jar,
		Transport:     transport,
		Timeout:       clientTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	launch := hostLaunch{client: client, follow: run.follow, app: run.app, host: host, env: hostEnvironment(run, env)}
	running, err := startHost(ctx, launch)
	if running == nil {
		return o, err
	}
	defer func() {
		hostErr := running.stopAndWait()
		if err != nil && hostErr != nil && !running.reported {
			err = errors.Join(err, fmt.Errorf("host: %w", hostErr))
		}
		finishHostObservation(run, running, &o)
	}()
	if err != nil {
		return o, err
	}
	o, err = observe(ctx, client, running.base, run.c)
	if err != nil {
		return o, fmt.Errorf("observe/%w", err)
	}
	err = inspectHost(ctx, run, client, running, env, browserBin, &o)
	return o, err
}

// hostEnvironment extends the application environment with the host's coverage and log
// settings.
//
// Takes run (*applicationRun) which selects coverage and source watching.
// Takes env ([]string) which supplies the private process environment.
//
// Returns []string which holds the host environment without its port and token.
func hostEnvironment(run *applicationRun, env []string) []string {
	goCoverage := ""
	if run.coverageDirectory != "" {
		goCoverage = filepath.Join(run.coverageDirectory, "raw")
	}
	hostEnv := append(slices.Clone(env), "PIKO_FUZZ_GO_COVERAGE="+goCoverage)
	if goCoverage != "" {
		hostEnv = append(hostEnv, "GOCOVERDIR="+goCoverage)
	}
	if run.follow != nil {
		hostEnv = append(hostEnv, "PIKO_LOG_LEVEL=internal")
	}
	return hostEnv
}

// finishHostObservation records the stopped host's log and watch history, removing
// session-bearing markup.
//
// Pages can contain per-session CSRF values, so auth and form-shim receipts retain only
// the extracted outcomes and a redacted log tail.
//
// Takes run (*applicationRun) which holds the case and optional source watch.
// Takes running (*hostProcess) which is the stopped host.
// Takes o (*Observation) which receives the log and watch history.
func finishHostObservation(run *applicationRun, running *hostProcess, o *Observation) {
	if run.c.Expected.Auth != nil || run.c.Expected.FormShim != nil {
		o.RawHTML = ""
		for index := range o.Probes {
			o.Probes[index].RawHTML = ""
		}
		o.WorkerLog = redactedLogTail(running.log)
	} else {
		o.WorkerLog = running.log
	}
	if run.follow != nil {
		o.CatalogWatch = run.follow.history.snapshot()
	}
}

// inspectHost gathers the upload-removal, browser, source-watch and coverage evidence
// that follows the HTTP observation.
//
// Takes run (*applicationRun) which holds the run inputs.
// Takes client (*http.Client) which is the HTTP client bound to the host.
// Takes running (*hostProcess) which is the ready host.
// Takes env ([]string) which supplies the private process environment.
// Takes browserBin (string) which is the browser worker executable, empty when unused.
// Takes o (*Observation) which receives the evidence.
//
// Returns error which reports the first failed inspection, or nil on success.
func inspectHost(ctx context.Context, run *applicationRun, client *http.Client, running *hostProcess, env []string, browserBin string, o *Observation) error {
	base := running.base
	if action := run.c.Expected.Action; action != nil && action.UploadRemoval && o.Action != nil && o.Action.Upload != nil {
		if err := observeUploadRemoval(ctx, client, base, o.RawHTML, run.work, o.Action.Upload); err != nil {
			return err
		}
	}
	if browserBin != "" {
		jsCoverage := ""
		if run.coverageDirectory != "" {
			jsCoverage = filepath.Join(run.coverageDirectory, runtimecoverage.JSArtefact)
		}
		browser := &browserRun{observation: o, app: run.app, work: run.work, base: base, browserBin: browserBin, jsCoverage: jsCoverage, env: env, c: run.c}
		if err := browser.run(ctx); err != nil {
			return err
		}
	}
	if run.follow != nil {
		if err := run.follow.inspect(ctx, client, base, o); err != nil {
			return err
		}
	}
	if run.coverageDirectory != "" {
		if err := snapshotCoverage(ctx, client, base, running.coverageSecret); err != nil {
			return fmt.Errorf("infrastructure: Go counter snapshot: %w", err)
		}
	}
	return nil
}

// startHost launches the host, retrying with a fresh port when the reserved one was taken
// between reservation and bind.
//
// Takes launch (hostLaunch) which holds the host inputs.
//
// Returns *hostProcess which is the last launched host, or nil when none started.
// Returns error which reports a startup failure, or nil once the host is ready.
func startHost(ctx context.Context, launch hostLaunch) (*hostProcess, error) {
	var running *hostProcess
	var err error
	for attempt := range hostStartAttempts {
		running, err = launchHost(ctx, launch)
		if err == nil || !errors.Is(err, errHostPortTaken) || attempt == hostStartAttempts-1 {
			return running, err
		}
		running.stop(errHostStopped)
	}
	return running, err
}

// launchHost reserves a port, starts one host and waits for it to become ready.
//
// Takes launch (hostLaunch) which holds the host inputs.
//
// Returns *hostProcess which is the started host, or nil when it never started.
// Returns error which reports reservation or readiness failure, wrapping errHostPortTaken
// when the host could not bind its port, or nil once the host is ready.
//
// Concurrency: The host runs in a goroutine until stopped or its context ends. Wait for
// the done channel before reading its output or exit error.
func launchHost(ctx context.Context, launch hostLaunch) (*hostProcess, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	address := listener.Addr().String()
	if err = listener.Close(); err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	hostContext, stop := context.WithCancelCause(ctx)
	running := &hostProcess{
		err:            nil,
		stop:           stop,
		done:           make(chan struct{}),
		log:            "",
		base:           "http://" + address,
		token:          rand.Text(),
		coverageSecret: rand.Text(),
		reported:       false,
	}
	environment := append(slices.Clone(launch.env), "PIKO_FUZZ_PORT="+port, "PIKO_FUZZ_TOKEN="+running.token, "PIKO_FUZZ_COVERAGE_SECRET="+running.coverageSecret)
	go func() {
		defer close(running.done)
		if launch.follow != nil {
			running.log, running.err = process.RunObserved(hostContext, launch.app, environment, &launch.follow.history, launch.host)
			return
		}
		running.log, running.err = process.Run(hostContext, launch.app, environment, launch.host)
	}()
	return running, awaitReadiness(ctx, launch.client, running)
}

// awaitReadiness polls the host's readiness endpoint until it answers with its token.
//
// Takes client (*http.Client) which is the HTTP client used for polling.
// Takes running (*hostProcess) which is the started host.
//
// Returns error which reports a readiness timeout or early host exit, wrapping
// errHostPortTaken when the host could not bind its port, or nil once ready.
func awaitReadiness(ctx context.Context, client *http.Client, running *hostProcess) error {
	readyContext, stopReady := context.WithTimeoutCause(ctx, readinessTimeout, errHostStartupTimeout)
	defer stopReady()
	ticker := time.NewTicker(readinessInterval)
	defer ticker.Stop()
	for {
		select {
		case <-readyContext.Done():
			return fmt.Errorf("startup: %w", context.Cause(readyContext))
		case <-running.done:
			running.reported = true
			hostErr := running.err
			if hostErr == nil {
				hostErr = errWorkerExitedEarly
			}
			if strings.Contains(running.log, syscall.EADDRINUSE.Error()) {
				return fmt.Errorf("startup: %w: %w\n%s", errHostPortTaken, hostErr, running.log)
			}
			return fmt.Errorf("startup: %w\n%s", hostErr, running.log)
		case <-ticker.C:
			status, _, data, err := get(readyContext, client, running.base, "/__piko_fuzz_ready")
			if err == nil && status == http.StatusOK && string(data) == running.token {
				return nil
			}
		}
	}
}

// browserSocketRoot selects the temporary root for Chrome's socket directory.
//
// Chrome creates a Unix-domain socket below its temporary directory, and a socket path is
// limited to the sun_path size. os.TempDir is used when the deepest socket path fits;
// otherwise the short fallback root is used.
//
// Returns string which is the selected temporary root.
func browserSocketRoot() string {
	root := os.TempDir()
	if len(root)+len(chromeSocketSuffix) < unixSocketPathMax {
		return root
	}
	return fallbackSocketRoot
}

// redactedLogTail keeps a bounded, redacted tail of a session-bearing worker log.
//
// Cookie, token, CSRF and session values are replaced, and the tail starts on a rune
// boundary with a truncation marker when earlier output was dropped.
//
// Takes log (string) which is the bounded worker output.
//
// Returns string which is the redacted tail.
func redactedLogTail(log string) string {
	redacted := secretPattern.ReplaceAllString(log, "${1}${2}[redacted]")
	if len(redacted) <= redactedWorkerLogMax {
		return redacted
	}
	start := len(redacted) - redactedWorkerLogMax
	for start < len(redacted) && !utf8.RuneStart(redacted[start]) {
		start++
	}
	return "[... earlier worker output truncated ...]\n" + redacted[start:]
}

// domRecipe encodes a DOM plan for the browser worker.
//
// Takes plan (*casefile.DOMPlan) which holds the expected DOM journey.
//
// Returns string which is the encoded driver plan.
// Returns error which reports an invalid plan or encoding failure, or nil on success.
func domRecipe(plan *casefile.DOMPlan) (string, error) {
	driverPlan, err := plan.DriverPlan()
	if err != nil {
		return "", fmt.Errorf("dom/recipe: %w", err)
	}
	recipe, err := json.Marshal(driverPlan)
	if err != nil {
		return "", fmt.Errorf("dom/recipe: %w", err)
	}
	return string(recipe), nil
}

// snapshotCoverage asks the instrumented host to flush its Go coverage data.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes secret (string) which authorises the private host coverage flush.
//
// Returns error which reports host authentication, HTTP or coverage-flush failure, or nil
// on success.
func snapshotCoverage(ctx context.Context, client *http.Client, base, secret string) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/__piko_fuzz_ready", nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-Piko-Fuzz-Coverage", secret)
	request.Host = requestHost
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, closeResponse(response)) }()
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, snapshotBodyMax+1))
	mismatch := fmt.Errorf("unexpected snapshot status %d: %s", response.StatusCode, snapshotDetail(body))
	if readErr != nil {
		return errors.Join(mismatch, fmt.Errorf("read snapshot body: %w", readErr))
	}
	return mismatch
}

// snapshotDetail formats a bounded snapshot response body for a failure message.
//
// A body longer than snapshotBodyMax is cut on a rune boundary and marked as truncated.
//
// Takes body ([]byte) which holds at most snapshotBodyMax+1 response bytes.
//
// Returns string which is the trimmed, possibly truncated body text.
func snapshotDetail(body []byte) string {
	if len(body) <= snapshotBodyMax {
		return strings.TrimSpace(string(body))
	}
	cut := snapshotBodyMax
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return strings.TrimSpace(string(body[:cut])) + " [truncated]"
}

// writeShimOptions links the generated provider configuration into the case's server.
//
// Takes target (string) which selects the subsystem under test.
// Takes directory (string) which is the generated server module directory.
//
// Returns error which reports an unknown target or provider-configuration write failure,
// or nil on success.
func writeShimOptions(target, directory string) error {
	var fixture string
	switch target {
	case "auth-contracts":
		fixture = "authfixture"
	case "form-shim-contracts":
		fixture = "formfixture"
	default:
		return nil
	}
	source := fmt.Sprintf("package main\nimport fixture %q\nfunc init() { fixtureOptions = fixture.Options() }\n", casefile.Module+"/internal/"+fixture)
	return os.WriteFile(filepath.Join(directory, "shim_options.go"), []byte(source), privateFileMode)
}
