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
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

const (
	fakeBrowserScript = `#!/bin/sh
printf '%s\n' "$@" > "$FAKE_BROWSER_ARGUMENTS"
case "$2" in
--firefox-dom) cp "$FAKE_FIREFOX_RECEIPT" "$3" ;;
--form-workflow) cp "$FAKE_BROWSER_RECEIPT" "$3" ;;
*) cp "$FAKE_BROWSER_RECEIPT" "$3" ;;
esac
exit "${FAKE_BROWSER_EXIT:-0}"
`
	fakeHostPage = `<html><head><title>Home</title><meta name="csrf-token" content="action-token"><meta name="csrf-ephemeral" content="ephemeral-token"></head>` +
		`<body><div id="app"><h1>Home</h1><p>first</p><div id="case-root"><h2 id="case-title">Counter</h2><ul id="case-list"><li>row</li></ul>` +
		`<fuzz-counter></fuzz-counter></div></div></body></html>`
	fakeGoScript = `#!/bin/sh
case "$1" in
build)
	while [ $# -gt 0 ]; do
		if [ "$1" = "-o" ]; then
			cp "$FAKE_GO_OUTPUT" "$2"
			exit 0
		fi
		shift
	done
	exit 1 ;;
list)
	if [ "$GOWORK" = off ]; then echo /resolved/module; else echo /resolved/workspace; fi ;;
*)
	echo "unsupported go command $1" >&2
	exit 1 ;;
esac
`
)

type fakeBrowser struct {
	binary    string
	arguments string
	env       []string
}

func TestRunBrowserWorkerMarksRuntimeCoverageFailures(t *testing.T) {
	err := runBrowserWorker(t.Context(), t.TempDir(), nil, "browser", "sh", "-c", "echo 'collecting runtime coverage: profiler closed' >&2; exit "+strconv.Itoa(casefile.RuntimeCoverageExitCode))

	require.Error(t, err)
	assert.ErrorIs(t, err, errRuntimeCoverage)
	assert.ErrorContains(t, err, "profiler closed")
}

func TestRunBrowserWorkerLeavesOrdinaryFailuresUnmarked(t *testing.T) {
	err := runBrowserWorker(t.Context(), t.TempDir(), nil, "browser", "sh", "-c", "echo 'navigation failed: runtime coverage: mentioned mid-line' >&2; exit 1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, errRuntimeCoverage)
	assert.NoError(t, runBrowserWorker(t.Context(), t.TempDir(), nil, "browser", "true"))
}

func TestAwaitReadinessClassifiesTakenPort(t *testing.T) {
	testCases := []struct {
		name      string
		log       string
		portTaken bool
	}{
		{name: "bind failure", log: "listen tcp 127.0.0.1:4242: bind: " + syscall.EADDRINUSE.Error(), portTaken: true},
		{name: "other failure", log: "panic: configuration missing", portTaken: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			running := &hostProcess{done: make(chan struct{}), log: testCase.log, err: errors.New("exit status 1"), base: "http://127.0.0.1:1"}
			close(running.done)

			err := awaitReadiness(t.Context(), &http.Client{}, running)

			require.Error(t, err)
			assert.Equal(t, testCase.portTaken, errors.Is(err, errHostPortTaken))
			assert.True(t, running.reported)
		})
	}
}

func TestAwaitReadinessReportsSilentEarlyExit(t *testing.T) {
	running := &hostProcess{done: make(chan struct{}), base: "http://127.0.0.1:1"}
	close(running.done)

	assert.ErrorIs(t, awaitReadiness(t.Context(), &http.Client{}, running), errWorkerExitedEarly)
}

func TestStopAndWaitIgnoresRunnerStop(t *testing.T) {
	stopped := &hostProcess{done: make(chan struct{}), stop: func(error) {}, err: errHostStopped}
	close(stopped.done)
	assert.NoError(t, stopped.stopAndWait())

	crashed := &hostProcess{done: make(chan struct{}), stop: func(error) {}, err: errors.New("exit status 2")}
	close(crashed.done)
	assert.ErrorContains(t, crashed.stopAndWait(), "exit status 2")
}

func TestRedactedLogTailRemovesSecretsAndBoundsOutput(t *testing.T) {
	log := "Set-Cookie: session=abc123\ncsrf_token=deadbeef\nAuthorization: Bearer s3cr3t\nready\n"

	redacted := redactedLogTail(log)

	for _, secret := range []string{"abc123", "deadbeef", "s3cr3t"} {
		assert.NotContains(t, redacted, secret)
	}
	assert.Contains(t, redacted, "ready")

	long := strings.Repeat("é", redactedWorkerLogMax)
	tail := redactedLogTail(long)
	assert.True(t, strings.HasPrefix(tail, "[... earlier worker output truncated ...]\n"))
	assert.True(t, utf8.ValidString(tail), "tail split a rune")
	assert.LessOrEqual(t, len(tail), redactedWorkerLogMax+len("[... earlier worker output truncated ...]\n"))
}

func TestBrowserSocketRootFallsBackWhenTooDeep(t *testing.T) {
	t.Setenv("TMPDIR", "/short")
	assert.Equal(t, "/short", browserSocketRoot())

	t.Setenv("TMPDIR", "/"+strings.Repeat("d", unixSocketPathMax))
	assert.Equal(t, fallbackSocketRoot, browserSocketRoot())
}

func TestBuildCachedBinaryReusesInstalledBinary(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "cached-tool")
	require.NoError(t, os.WriteFile(binary, []byte("cached"), 0700))

	got, err := buildCachedBinary(t.Context(), t.TempDir(), nil, "cached-build", binary, "./does-not-exist")

	require.NoError(t, err)
	assert.Equal(t, binary, got)
}

func TestRemoveStagedBinaryToleratesAbsence(t *testing.T) {
	assert.NoError(t, removeStagedBinary(filepath.Join(t.TempDir(), "absent")))
}

func TestBrowserRunLeadsWithTheRecoveredModelFailure(t *testing.T) {
	workflow, receipt := formCheckFixture(t)
	receipt.FailedStep, receipt.DriverError = 2, "browser disconnected"
	receipt.Steps = receipt.Steps[:2]
	data, err := json.Marshal(receipt)
	require.NoError(t, err)
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, browserReceiptName), data, 0600))
	worker := filepath.Join(t.TempDir(), "browser-worker")
	require.NoError(t, os.WriteFile(worker, []byte("#!/bin/sh\nexit "+strconv.Itoa(casefile.RuntimeCoverageExitCode)+"\n"), 0700))
	run := &browserRun{
		c:           &casefile.Case{Expected: casefile.Expected{FormWorkflow: workflow}},
		observation: &Observation{},
		app:         t.TempDir(),
		work:        work,
		base:        "http://127.0.0.1:1",
		browserBin:  worker,
	}

	err = run.run(t.Context())

	require.Error(t, err)
	assert.ErrorIs(t, err, errRuntimeCoverage)
	assert.False(t, strings.HasPrefix(err.Error(), "infrastructure:"), "%v", err)
	_, ok := errors.AsType[formMismatch](err)
	assert.True(t, ok, "%v", err)
}

func TestHostStartupTimeoutIsADeadline(t *testing.T) {
	assert.ErrorIs(t, errHostStartupTimeout, context.DeadlineExceeded)
}

func TestSnapshotCoverageSendsTheCoverageSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Piko-Fuzz-Coverage") != "secret" {
			http.Error(writer, strings.Repeat("é", snapshotBodyMax), http.StatusForbidden)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	require.NoError(t, snapshotCoverage(t.Context(), server.Client(), server.URL, "secret"))

	err := snapshotCoverage(t.Context(), server.Client(), server.URL, "readiness-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected snapshot status 403")
	assert.Contains(t, err.Error(), "[truncated]")
	assert.True(t, utf8.ValidString(err.Error()))
}

func TestSnapshotDetailKeepsShortBodies(t *testing.T) {
	assert.Equal(t, "denied", snapshotDetail([]byte(" denied\n")))
	assert.Equal(t, strings.Repeat("a", snapshotBodyMax), snapshotDetail([]byte(strings.Repeat("a", snapshotBodyMax))))
}

func TestApplicationEnvironmentEncodesClientRecipes(t *testing.T) {
	c := generate.New("checkout-workflow", 7, 0)
	work := t.TempDir()

	env, err := applicationEnvironment(work, "/work/go.work", &c, true)

	require.NoError(t, err)
	assert.Contains(t, env, "GOWORK=/work/go.work")
	assert.Contains(t, env, "PIKO_FUZZ_CATALOG_WATCH=true")
	assert.Contains(t, env, "PIKO_FUZZ_CHECKOUT_OUTBOX="+filepath.Join(work, checkoutOutboxName))
	assert.DirExists(t, filepath.Join(work, checkoutOutboxName))
	found := false
	for _, entry := range env {
		found = found || strings.HasPrefix(entry, "PIKO_FUZZ_CHECKOUT_INPUT=")
	}
	assert.True(t, found, "checkout recipe missing")
}

func TestClientRecipesListsEveryJourney(t *testing.T) {
	client := &casefile.ClientTransition{
		Lifecycle:         casefile.ExpectedLifecycle(casefile.LifecycleInput{Draft: "x"}),
		Checkout:          casefile.ExpectedCheckout(casefile.CheckoutInput{Customer: "a", SKU: "b", Quantity: 1, Unit: 2}),
		PartialComponents: casefile.ExpectedPartialComponents(casefile.PartialComponentsInput{Edited: "e"}),
		Events:            casefile.ExpectedEvents(casefile.EventsInput{Label: "l", Delta: 1}),
	}

	recipes := clientRecipes(client)

	names := make([]string, 0, len(recipes))
	for _, recipe := range recipes {
		names = append(names, recipe.name)
	}
	assert.Equal(t, []string{"lifecycle", "checkout", "partial-components", "events"}, names)
}

func TestHostEnvironmentAddsCoverageAndWatchSettings(t *testing.T) {
	plain := hostEnvironment(&applicationRun{}, []string{"A=1"})
	assert.Equal(t, []string{"A=1", "PIKO_FUZZ_GO_COVERAGE="}, plain)

	covered := hostEnvironment(&applicationRun{coverageDirectory: "/coverage", follow: &catalogueWatch{}}, nil)
	assert.Equal(t, []string{"PIKO_FUZZ_GO_COVERAGE=/coverage/raw", "GOCOVERDIR=/coverage/raw", "PIKO_LOG_LEVEL=internal"}, covered)
}

func TestDOMRecipeOmitsExpectedValues(t *testing.T) {
	c := generate.New("markup", 7, 0)

	recipe, err := domRecipe(c.Expected.DOM)

	require.NoError(t, err)
	assert.NotContains(t, recipe, `"Want"`)
	_, err = domRecipe(&casefile.DOMPlan{})
	assert.Error(t, err)
}

func TestWriteShimOptionsLinksOnlyShimTargets(t *testing.T) {
	directory := t.TempDir()

	require.NoError(t, writeShimOptions("pages", directory))
	assert.NoFileExists(t, filepath.Join(directory, "shim_options.go"))

	require.NoError(t, writeShimOptions("auth-contracts", directory))
	source, err := os.ReadFile(filepath.Join(directory, "shim_options.go"))
	require.NoError(t, err)
	assert.Contains(t, string(source), casefile.Module+"/internal/authfixture")
}

func newFakeBrowser(t *testing.T, receipt, firefox string, exit int) fakeBrowser {
	t.Helper()
	directory := t.TempDir()
	write := func(name, content string, mode os.FileMode) string {
		path := filepath.Join(directory, name)
		require.NoError(t, os.WriteFile(path, []byte(content), mode))
		return path
	}
	browser := fakeBrowser{binary: write("browser", fakeBrowserScript, 0o700), arguments: filepath.Join(directory, "arguments")}
	browser.env = []string{
		"FAKE_BROWSER_ARGUMENTS=" + browser.arguments,
		"FAKE_BROWSER_RECEIPT=" + write("receipt.json", receipt, 0o600),
		"FAKE_FIREFOX_RECEIPT=" + write("firefox.json", firefox, 0o600),
		"FAKE_BROWSER_EXIT=" + strconv.Itoa(exit),
	}
	return browser
}

func (f fakeBrowser) recordedArguments(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(f.arguments)
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func (f fakeBrowser) run(t *testing.T, c *casefile.Case, o *Observation) (*browserRun, error) {
	t.Helper()
	work := t.TempDir()
	browser := &browserRun{observation: o, app: work, work: work, base: "http://127.0.0.1:1", browserBin: f.binary, env: f.env, c: c}
	return browser, browser.run(t.Context())
}

func TestBrowserRunReadsTheClientReceipt(t *testing.T) {
	browser := newFakeBrowser(t, `{"Before":[{"Count":"1"}],"After":[{"Count":"2"}],"Checkout":{}}`, "", 0)
	c := &casefile.Case{Request: "/main", Expected: casefile.Expected{Tag: "x-card", Client: &casefile.ClientTransition{
		Steps: []casefile.ClientStep{{Action: casefile.ClientAction{Op: casefile.SetLabel, Instance: 0, Value: "a"}}},
	}}}
	var o Observation

	_, err := browser.run(t, c, &o)

	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, o.Counts)
	require.NotNil(t, o.Client.Checkout)
	assert.NotNil(t, o.Client.Checkout.Documents)
	arguments := browser.recordedArguments(t)
	require.Len(t, arguments, 4)
	assert.Equal(t, []string{"http://127.0.0.1:1/main", "x-card"}, arguments[:2])
	assert.Contains(t, arguments[3], `"Value":"a"`)
}

func TestBrowserRunReadsTheFirefoxDOM(t *testing.T) {
	browser := newFakeBrowser(t, `{"Before":[{"Count":"1"}],"After":[{"Count":"1"}]}`, `{}`, 0)
	c := generate.New("markup", 7, 0)
	c.Target = "firefox-markup"
	var o Observation

	_, err := browser.run(t, &c, &o)

	require.NoError(t, err)
	assert.NotNil(t, o.Client.DOM)
	arguments := browser.recordedArguments(t)
	assert.Equal(t, "--firefox-dom", arguments[1])
	assert.NotContains(t, arguments[3], `"Want"`)
}

func TestBrowserRunReadsTheWorkflowReceipt(t *testing.T) {
	browser := newFakeBrowser(t, `{}`, "", 0)
	c := &casefile.Case{Expected: casefile.Expected{FormWorkflow: &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "create", Value: "one"}}}}}
	var o Observation

	_, err := browser.run(t, c, &o)

	require.NoError(t, err)
	assert.NotNil(t, o.FormWorkflow)
	arguments := browser.recordedArguments(t)
	assert.Equal(t, []string{"http://127.0.0.1:1/workflow", "--form-workflow"}, arguments[:2])
}

func TestBrowserRunRejectsInvalidReceipts(t *testing.T) {
	plain := &casefile.Case{Request: "/main"}
	for name, testCase := range map[string]struct {
		c       *casefile.Case
		receipt string
		firefox string
		want    string
	}{
		"malformed client":   {c: plain, receipt: `[`, want: "invalid shadow DOM observation"},
		"empty client":       {c: plain, receipt: `{}`, want: "invalid shadow DOM observation"},
		"malformed workflow": {c: &casefile.Case{Expected: casefile.Expected{FormWorkflow: &casefile.FormWorkflow{Steps: []casefile.FormCommand{{Op: "create", Value: "one"}}}}}, receipt: `[`, want: "invalid workflow receipt"},
		"firefox plan":       {c: &casefile.Case{Target: "firefox-markup"}, receipt: `{"Before":[{}],"After":[{}]}`, want: "missing DOM plan"},
		"firefox receipt": {c: func() *casefile.Case {
			c := generate.New("markup", 7, 0)
			c.Target = "firefox-markup"
			return &c
		}(), receipt: `{"Before":[{}],"After":[{}]}`, firefox: `[`, want: "firefox-result"},
	} {
		t.Run(name, func(t *testing.T) {
			browser := newFakeBrowser(t, testCase.receipt, testCase.firefox, 0)
			_, err := browser.run(t, testCase.c, &Observation{})
			assert.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestBrowserRunMarksRuntimeCoverageFailuresAsInfrastructure(t *testing.T) {
	browser := newFakeBrowser(t, `{}`, "", casefile.RuntimeCoverageExitCode)

	_, err := browser.run(t, &casefile.Case{Request: "/main"}, &Observation{})

	require.ErrorIs(t, err, errRuntimeCoverage)
	assert.True(t, strings.HasPrefix(err.Error(), "infrastructure:"), "%v", err)
}

func TestBrowserRunArgumentsBoundTheRecipe(t *testing.T) {
	steps := make([]casefile.ClientStep, browserActionMax+1)
	browser := &browserRun{c: &casefile.Case{Expected: casefile.Expected{Client: &casefile.ClientTransition{Steps: steps}}}}

	_, err := browser.arguments("receipt.json")

	assert.ErrorContains(t, err, "browser-recipe: exceeds")
}

func TestBrowserRunEnvironmentSelectsRecipes(t *testing.T) {
	c := generate.New("markup", 7, 0)
	c.Expected.FormShim = &casefile.FormShimContract{BrowserSubmission: true}
	browser := &browserRun{c: &c, env: []string{"BASE=1"}}

	env, err := browser.environment()

	require.NoError(t, err)
	assert.Equal(t, "BASE=1", env[0])
	assert.Contains(t, env, "PIKO_FUZZ_INTAKE_BROWSER=true")
	found := false
	for _, entry := range env {
		found = found || strings.HasPrefix(entry, "PIKO_FUZZ_DOM=")
	}
	assert.True(t, found, "DOM recipe missing")
	c.Expected.DOM = &casefile.DOMPlan{}
	_, err = browser.environment()
	assert.Error(t, err)
}

func TestBuildCachedBinaryReusesAnExistingBinary(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "worker")
	require.NoError(t, os.WriteFile(binary, []byte("cached"), 0o700))

	got, err := buildCachedBinary(t.Context(), t.TempDir(), nil, "worker", binary, "./missing")

	require.NoError(t, err)
	assert.Equal(t, binary, got)
}

func TestBuildCachedBinaryRemovesTheStagedBinaryOnFailure(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "worker")

	_, err := buildCachedBinary(t.Context(), directory, nil, "worker", binary, "./missing")

	require.ErrorContains(t, err, "worker:")
	entries, readErr := os.ReadDir(directory)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestFakeHostProcess(t *testing.T) {
	if os.Getenv("PIKO_FUZZ_FAKE_HOST") == "" {
		t.Skip("runs only as the fake application host")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__piko_fuzz_ready", func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Piko-Fuzz-Coverage") == os.Getenv("PIKO_FUZZ_COVERAGE_SECRET") {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = io.WriteString(writer, os.Getenv("PIKO_FUZZ_TOKEN"))
	})
	mux.HandleFunc("GET /", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(headerContentType, "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, fakeHostPage)
	})
	if ledger := os.Getenv("PIKO_FUZZ_FAKE_LEDGER"); ledger != "" {
		var model casefile.GoTopology
		require.NoError(t, json.Unmarshal([]byte(ledger), &model))
		mux = fakeLedgerHandler(&model, fakeHostPage)
		mux.HandleFunc("GET /__piko_fuzz_ready", func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(writer, os.Getenv("PIKO_FUZZ_TOKEN"))
		})
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:"+os.Getenv("PIKO_FUZZ_PORT"))
	require.NoError(t, err)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	assert.NoError(t, server.Serve(listener))
}

func fakeHostScript(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	return "#!/bin/sh\nPIKO_FUZZ_FAKE_HOST=1 exec " + strconv.Quote(executable) + " -test.run='^TestFakeHostProcess$'\n"
}

func installFakeGo(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	host := filepath.Join(directory, "host")
	require.NoError(t, os.WriteFile(host, []byte(fakeHostScript(t)), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "go"), []byte(fakeGoScript), 0o700))
	t.Setenv("FAKE_GO_OUTPUT", host)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func fakeWorkers(t *testing.T) string {
	t.Helper()
	workers := t.TempDir()
	host := fakeHostScript(t)
	for name, script := range map[string]string{
		"generate-worker":       "#!/bin/sh\nexit 0\n",
		"interp-worker":         host,
		"interp-worker-covered": host,
		"browser-worker":        fakeBrowserScript,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(workers, name), []byte(script), 0o700))
	}
	return workers
}

func TestRunWithWorkersObservesAnInterpretedHost(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	workers := fakeWorkers(t)
	browser := newFakeBrowser(t, `{"Before":[{"Count":"1"}],"After":[{"Count":"2"}]}`, "", 0)
	for _, entry := range browser.env {
		name, value, _ := strings.Cut(entry, "=")
		t.Setenv(name, value)
	}
	c := &casefile.Case{Request: "/main", Expected: casefile.Expected{Status: http.StatusOK, Title: "Home"}}

	o, err := RunWithWorkers(t.Context(), repo, t.TempDir(), workers, c, modeInterpreted, true)

	require.NoError(t, err)
	assert.Equal(t, modeInterpreted, o.Mode)
	assert.Equal(t, "Home", o.Heading)
	assert.Equal(t, []string{"first"}, o.Rows)
	assert.Equal(t, []string{"1", "2"}, o.Counts)
	assert.Equal(t, http.StatusOK, o.Status)
}

func TestRunMeasuredSnapshotsHostCoverage(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	output := t.TempDir()
	c := &casefile.Case{Request: "/main", Expected: casefile.Expected{Status: http.StatusOK, Title: "Home"}}

	o, err := RunMeasured(t.Context(), &MeasuredRun{Case: c, Repo: repo, Work: t.TempDir(), Workers: fakeWorkers(t), Mode: modeInterpreted, Output: output})

	require.NoError(t, err)
	assert.Equal(t, "Home", o.Heading)
	assert.DirExists(t, filepath.Join(output, "raw"))
}

func TestRunApplicationRejectsUnknownModes(t *testing.T) {
	_, err := RunWithWorkers(t.Context(), t.TempDir(), t.TempDir(), t.TempDir(), &casefile.Case{}, "other", false)
	assert.ErrorContains(t, err, `mode: unknown "other"`)

	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	_, err = RunWithWorkers(t.Context(), repo, t.TempDir(), t.TempDir(), &casefile.Case{Target: "embedded-deploy"}, modeInterpreted, false)
	assert.ErrorContains(t, err, "embedded deployment requires compiled mode")
}

func TestRunWithWorkersReportsAHostThatExitsEarly(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	workers := fakeWorkers(t)
	require.NoError(t, os.WriteFile(filepath.Join(workers, "interp-worker"), []byte("#!/bin/sh\necho bind: address already in use\nexit 1\n"), 0o700))
	c := &casefile.Case{Request: "/main", Expected: casefile.Expected{Status: http.StatusOK}}

	_, err = RunWithWorkers(t.Context(), repo, t.TempDir(), workers, c, modeInterpreted, false)

	require.ErrorIs(t, err, errHostPortTaken)
	assert.ErrorContains(t, err, "startup:")
}

func TestRunWithWorkersObservesBothTopologyHosts(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	workers := fakeWorkers(t)
	installFakeGo(t)
	c := generate.New("go-topology", 7, 0)
	model, err := json.Marshal(c.Expected.Topology)
	require.NoError(t, err)
	t.Setenv("PIKO_FUZZ_FAKE_LEDGER", string(model))

	for _, mode := range []string{modeInterpreted, modeCompiled} {
		t.Run(mode, func(t *testing.T) {
			work := t.TempDir()
			o, err := RunWithWorkers(t.Context(), repo, work, workers, &c, mode, false)

			require.NoError(t, err)
			assert.Equal(t, mode, o.Mode)
			assert.Equal(t, "Counter", o.Heading)
			require.NoError(t, checkTopology(c.Expected.Topology, o.Topology))
			if mode == modeInterpreted {
				assert.FileExists(t, filepath.Join(work, "interp-host-src", "registry.go"))
			} else {
				assert.FileExists(t, filepath.Join(work, "app", "main.go"))
			}
		})
	}
}

func TestRunWithWorkersRefusesADeploymentWithoutTheEmbedTag(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	workers := fakeWorkers(t)
	cli := "#!/bin/sh\n[ \"$1\" = build ] && [ \"$2\" = --output ] && cp " + strconv.Quote(executable) + " \"$3\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(workers, "piko-cli"), []byte(cli), 0o700))
	c := generate.New("embedded-deploy", 7, 0)
	work := t.TempDir()

	_, err = RunWithWorkers(t.Context(), repo, work, workers, &c, modeCompiled, false)

	require.ErrorContains(t, err, "deployment/tags")
	assert.FileExists(t, filepath.Join(work, "native"))
	assert.FileExists(t, filepath.Join(work, "detached-source", "cmd", "main", "main.go"))
	assert.NoDirExists(t, filepath.Join(work, "app"))
}

func TestInterpretedRegistrySelectsTheTargetSymbols(t *testing.T) {
	for name, testCase := range map[string]struct {
		c    *casefile.Case
		want string
	}{
		"plain":       {c: &casefile.Case{}, want: "package main\nimport _ \"" + casefile.Module + "/dist\"\n"},
		"collections": {c: &casefile.Case{Target: "collection-routes"}, want: casefile.CollectionModule + "/entries"},
		"workflow":    {c: &casefile.Case{Target: targetFormWorkflow}, want: workflowRegistry},
		"checkout":    {c: &casefile.Case{Target: "checkout-workflow"}, want: checkoutRegistry},
		"topology":    {c: &casefile.Case{Expected: casefile.Expected{Topology: &casefile.GoTopology{}}}, want: topologyRegistry},
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			registry, err := interpretedRegistry(directory, testCase.c)
			require.NoError(t, err)
			assert.Contains(t, registry, testCase.want)
			if testCase.c.Target == targetFormWorkflow {
				assert.FileExists(t, filepath.Join(directory, "workflow_db.go"))
			} else {
				assert.NoFileExists(t, filepath.Join(directory, "workflow_db.go"))
			}
		})
	}
	_, err := interpretedRegistry(filepath.Join(t.TempDir(), "missing"), &casefile.Case{Target: targetFormWorkflow})
	assert.Error(t, err)
}

func TestGeneratedHostSourcesUseEveryImport(t *testing.T) {
	sources := map[string]string{
		"collection registry": collectionRegistrySource,
		"collection parser":   collectionParserSource,
		"workflow registry":   workflowRegistry,
		"topology registry":   topologyRegistry,
		"checkout registry":   checkoutRegistry,
		"workflow database":   workflowDBHost,
		"native main":         nativeMain,
	}
	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			assertEveryImportUsed(t, source)
		})
	}
}

func assertEveryImportUsed(t *testing.T, source string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "host.go", source, parser.SkipObjectResolution)
	require.NoError(t, err)
	used := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if identifier, ok := selector.X.(*ast.Ident); ok {
				used[identifier.Name] = true
			}
		}
		return true
	})
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		require.NoError(t, err)
		name := importPath[strings.LastIndex(importPath, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "_" || name == "." {
			continue
		}
		assert.True(t, used[name], "%s is imported but never used", importPath)
	}
}
