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

// Package main drives Chromium and Firefox journeys for a fuzz case and writes their
// receipts.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// envJSCoverage names the optional JavaScript coverage output path.
	envJSCoverage = "PIKO_FUZZ_JS_COVERAGE"

	// envProgramPaths names the JSON browser program route list.
	envProgramPaths = "PIKO_FUZZ_BROWSER_PROGRAM_PATHS"

	// envIntake selects the intake form submission journey when set to "true".
	envIntake = "PIKO_FUZZ_INTAKE_BROWSER"

	// envCheckout names the JSON checkout journey recipe.
	envCheckout = "PIKO_FUZZ_CHECKOUT_INPUT"

	// envPartialComponents names the JSON partial-component journey recipe.
	envPartialComponents = "PIKO_FUZZ_PARTIAL_COMPONENTS"

	// envEvents names the JSON component event journey recipe.
	envEvents = "PIKO_FUZZ_EVENTS"

	// envLifecycle names the JSON lifecycle journey recipe.
	envLifecycle = "PIKO_FUZZ_LIFECYCLE"

	// envDOM names the JSON DOM plan recipe.
	envDOM = "PIKO_FUZZ_DOM"

	// envWorkflowDatabase names the per-mode workflow SQLite database path.
	envWorkflowDatabase = "PIKO_FUZZ_WORKFLOW_DB"

	// envBrowserTemporary names the browser socket directory.
	envBrowserTemporary = "PIKO_FUZZ_BROWSER_TMP"

	// primaryBudget bounds the primary navigation, readiness wait and first increment.
	primaryBudget = 20 * time.Second

	// actionBudget bounds one saved component interaction.
	actionBudget = 3 * time.Second

	// maxActionValueBytes is the largest accepted interaction value in bytes.
	maxActionValueBytes = 256

	// maxActionRecipeBytes is the largest accepted encoded action recipe in bytes.
	maxActionRecipeBytes = 8192

	// maxActions is the largest accepted number of saved interactions.
	maxActions = 12

	// usageExitCode is the exit status for a malformed command line.
	usageExitCode = 2

	// modeArgumentCount is the argument count of the Firefox and form-workflow modes.
	modeArgumentCount = 4

	// probeArgumentCount is the argument count of a probe without saved actions.
	probeArgumentCount = 3

	// jsonInputIndex is the position of the JSON input argument, which holds the DOM plan or
	// form recipe of a mode, or the saved actions of a probe.
	jsonInputIndex = 3

	// settleScript waits two animation frames so queued rendering can run.
	settleScript = "new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true))))"

	// probeSnapshot is the component snapshot applied to the hosts expression.
	probeSnapshot = `.map(host => {
		const root = host.shadowRoot;
		const text = selector => root.querySelector(selector)?.textContent ?? "";
		const controls = root.querySelector("#client-input");
		const rows = Array.from(root.querySelectorAll("#client-list li"));
		let extra;
		if (controls) {
			const hosts = (globalThis.__pikoFuzzRows ??= new WeakMap());
			if (!hosts.has(host)) hosts.set(host, {nodes:new WeakMap(), next:1});
			const ids = hosts.get(host);
			const keys = rows.map(node => {
				if (!ids.nodes.has(node)) ids.nodes.set(node,ids.next++);
				return ids.nodes.get(node);
			});
			extra = {Input:controls.value, HostLabel:host.getAttribute("label"), Handled:text("#client-handled"), Bubbled:text("#client-bubbled"), Keys:keys};
			const check = root.querySelector("#client-check"), select = root.querySelector("#client-select");
			if (check && select) extra.Form = {Checked:check.checked, HostChecked:host.hasAttribute("agreed"), CheckedState:text("#client-agreed"),
				Choice:select.value, ChoiceState:text("#client-choice"), HostChoice:host.getAttribute("choice")};
		}
		return {
			Advanced: ` + advancedSnapshot + `,
			Controls: extra,
			Count: text("#count"),
			Label: text("#client-label"),
			Rows: rows.map(el => el.textContent),
			Visible: !!root.querySelector("#client-visible"),
			Lifecycle: text("#client-life"),
			Slot: (root.querySelector('slot[name="caption"]')?.assignedNodes({flatten:true}) ?? []).map(el => el.textContent).join("")
		};
	})`
)

var (
	// recipeVariables lists every environment variable the worker reads.
	recipeVariables = []string{
		envJSCoverage, envProgramPaths, envIntake, envCheckout, envPartialComponents,
		envEvents, envLifecycle, envDOM, envWorkflowDatabase, envBrowserTemporary,
	}

	// errNoInstances reports a page with no registered component instances.
	errNoInstances = errors.New("no registered instances")
)

// probeRecipe holds the optional browser journeys selected by the environment.
type probeRecipe struct {
	// checkout holds the checkout journey input, or nil when not selected.
	checkout *casefile.CheckoutInput

	// partialComponents holds the partial-component journey input, or nil.
	partialComponents *casefile.PartialComponentsInput

	// events holds the component event journey input, or nil.
	events *casefile.EventsInput

	// lifecycle holds the lifecycle journey input, or nil.
	lifecycle *casefile.LifecycleInput

	// dom holds the DOM plan, or nil.
	dom *casefile.DOMPlan

	// coveragePath is the JavaScript coverage output path, or empty.
	coveragePath string

	// routePaths lists the browser program routes.
	routePaths []string

	// routed records the primary visit under its route name. The runner asks for evidence by
	// setting the program paths variable, even when it lists no secondary routes.
	routed bool

	// intake selects the intake form submission journey.
	intake bool
}

// budget sizes the browser session from the selected journeys.
//
// Takes actions (int) which counts the saved component interactions.
//
// Returns time.Duration which is the sum of every selected journey's step budgets plus
// the start-up margin.
func (recipe probeRecipe) budget(actions int) time.Duration {
	budget := sessionMargin + primaryBudget + time.Duration(actions)*actionBudget + programRoutesBudget(recipe.routePaths)
	if recipe.intake {
		budget += intakeBudget()
	}
	if recipe.checkout != nil {
		budget += checkoutBudget()
	}
	if recipe.partialComponents != nil {
		budget += partialComponentsBudget(*recipe.partialComponents)
	}
	if recipe.events != nil {
		budget += eventsBudget()
	}
	if recipe.lifecycle != nil {
		budget += lifecycleBudget(*recipe.lifecycle)
	}
	if recipe.dom != nil {
		budget += domBudget(*recipe.dom)
	}
	if recipe.coveragePath != "" {
		budget += coverageBudget
	}
	return budget
}

// probeScripts holds the scripts that select, wait for and snapshot the probed component
// instances.
type probeScripts struct {
	// settle waits for queued custom-element callbacks and DOM updates to render.
	settle browserAction

	// tag is the quoted custom element name.
	tag string

	// hosts selects every probed component instance.
	hosts string

	// ready reports that every instance rendered its counter and increment control.
	ready string

	// snapshot reads every instance's public state.
	snapshot string
}

// probe runs component interactions and writes their observed state.
//
// Takes address (string) which is the application URL to visit.
// Takes tag (string) which identifies the custom element.
// Takes result (string) which is the receipt output path.
// Takes actions ([]casefile.ClientAction) which lists the saved browser interactions.
//
// Returns error which reports an invalid recipe, browser failure or receipt-write
// failure, or nil on success.
func probe(ctx context.Context, address, tag, result string, actions []casefile.ClientAction) error {
	recipe, err := readProbeRecipe()
	if err != nil {
		return err
	}
	origin, err := url.Parse(address)
	if err != nil {
		return err
	}
	ctx, session, closeBrowser, err := openBrowser(ctx, recipe.budget(len(actions)))
	if err != nil {
		return err
	}
	defer closeBrowser()
	scripts := newProbeScripts(tag)
	if recipe.coveragePath != "" {
		if err := startCoverage(ctx, session); err != nil {
			return fmt.Errorf("starting %w: %w", errRuntimeCoverage, err)
		}
	}
	var resultData casefile.ClientTransition
	if err := primaryVisit(ctx, session, origin, address, recipe.routed, &scripts, &resultData); err != nil {
		return err
	}
	if err := runActions(ctx, session, &scripts, actions, &resultData); err != nil {
		return err
	}
	runJourneys(ctx, session, &recipe, origin, address, scripts.settle, &resultData)
	if err := writeProbeReceipt(result, &resultData); err != nil {
		return err
	}
	if recipe.coveragePath != "" {
		if err := captureCoverage(ctx, session, address, recipe.coveragePath); err != nil {
			return fmt.Errorf("collecting %w: %w", errRuntimeCoverage, err)
		}
	}
	return nil
}

// newProbeScripts builds the scripts that drive the instances of one custom element.
//
// Takes tag (string) which identifies the custom element.
//
// Returns probeScripts which holds the element's selection, readiness and snapshot
// scripts.
func newProbeScripts(tag string) probeScripts {
	quoted := strconv.Quote(tag)
	hosts := "Array.from(document.querySelectorAll(" + quoted + "))"
	return probeScripts{
		tag:   quoted,
		hosts: hosts,
		ready: hosts + ".length > 0 && " + hosts +
			".every(h => h.shadowRoot?.querySelector('#count')?.textContent.trim() && h.shadowRoot?.querySelector('#increment'))",
		snapshot: hosts + probeSnapshot,
		settle:   await(settleScript, nil),
	}
}

// primaryVisit loads the primary page, waits for the instances, snapshots them and clicks
// the first instance's increment control.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes origin (*url.URL) which is the parsed application URL.
// Takes address (string) which is the application URL to visit.
// Takes routed (bool) which records the primary visit as a program route.
// Takes scripts (*probeScripts) which holds the element's scripts.
// Takes resultData (*casefile.ClientTransition) which receives the observations.
//
// Returns error which reports a navigation, readiness or increment failure, or nil on
// success.
func primaryVisit(ctx context.Context, session *browser.Session, origin *url.URL, address string, routed bool, scripts *probeScripts, resultData *casefile.ClientTransition) error {
	if routed {
		primary := navigateRoute(ctx, session, origin, origin.RequestURI())
		if primary.DriverError != "" {
			return fmt.Errorf("primary navigation: %s", primary.DriverError)
		}
		resultData.BrowserRoutes = append(resultData.BrowserRoutes, primary)
	} else if err := session.NavigateURL(ctx, address); err != nil {
		return fmt.Errorf("primary navigation: %s", sessionFailure(ctx, "", err))
	}
	if err := runBrowser(ctx, session, waitExpression(scripts.ready, 0), scripts.settle,
		evaluate("navigator.userAgent", &resultData.PrimaryBrowserVersion), evaluate(scripts.snapshot, &resultData.Before)); err != nil {
		return fmt.Errorf("primary readiness: %s", sessionFailure(ctx, "", err))
	}
	if len(resultData.Before) == 0 {
		return fmt.Errorf("%w for %s", errNoInstances, scripts.tag)
	}
	increment := "document.querySelector(" + scripts.tag + ").shadowRoot.querySelector('#increment').click()"
	if err := runBrowser(ctx, session, evaluate(increment, nil), scripts.settle, evaluate(scripts.snapshot, &resultData.After)); err != nil {
		return fmt.Errorf("increment: %s", sessionFailure(ctx, "", err))
	}
	return nil
}

// runActions performs the saved interactions and records the state after each one.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes scripts (*probeScripts) which holds the element's scripts.
// Takes actions ([]casefile.ClientAction) which lists the saved browser interactions.
// Takes resultData (*casefile.ClientTransition) which receives the steps.
//
// Returns error which reports an out-of-bounds or failed interaction, or nil on success.
func runActions(ctx context.Context, session *browser.Session, scripts *probeScripts, actions []casefile.ClientAction, resultData *casefile.ClientTransition) error {
	for index, action := range actions {
		if action.Instance < 0 || action.Instance >= len(resultData.Before) || len(action.Value) > maxActionValueBytes {
			return fmt.Errorf("interaction %d outside recipe bounds", index)
		}
		var step casefile.ClientStep
		step.Action = action
		drive, err := interaction(scripts.hosts, action, &step.Prevented)
		if err != nil {
			return err
		}
		if err := runBrowser(ctx, session, drive, scripts.settle, evaluate(scripts.snapshot, &step.After)); err != nil {
			return fmt.Errorf("interaction %d (%s): %s", index, action.Op, sessionFailure(ctx, "", err))
		}
		resultData.Steps = append(resultData.Steps, step)
	}
	return nil
}

// runJourneys runs the optional journeys the recipe selects and records their traces.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes recipe (*probeRecipe) which holds the selected journeys.
// Takes origin (*url.URL) which is the parsed application URL.
// Takes address (string) which is the application URL to visit.
// Takes settle (browserAction) which waits for queued browser rendering.
// Takes resultData (*casefile.ClientTransition) which receives the traces.
func runJourneys(ctx context.Context, session *browser.Session, recipe *probeRecipe, origin *url.URL, address string, settle browserAction, resultData *casefile.ClientTransition) {
	if recipe.intake {
		resultData.FormShim = driveIntake(ctx, session, origin)
	}
	if recipe.checkout != nil {
		resultData.Checkout = driveCheckout(ctx, session, address, *recipe.checkout)
	}
	if recipe.partialComponents != nil {
		resultData.PartialComponents = browserPartialComponents(ctx, session, address, *recipe.partialComponents)
	}
	if recipe.events != nil {
		resultData.Events = browserEvents(ctx, session, address, *recipe.events)
	}
	if recipe.lifecycle != nil {
		resultData.Lifecycle = browserLifecycle(ctx, session, address, *recipe.lifecycle)
	}
	if recipe.dom != nil {
		resultData.DOM = domJourney(ctx, session, address, *recipe.dom, settle)
	}
	resultData.BrowserRoutes = append(resultData.BrowserRoutes, visitBrowserPrograms(ctx, session, origin, recipe.routePaths)...)
}

// writeProbeReceipt encodes and writes a bounded browser receipt.
//
// Takes result (string) which is the receipt output path.
// Takes resultData (*casefile.ClientTransition) which holds the observations.
//
// Returns error which reports an encoding, size or write failure, or nil on success.
func writeProbeReceipt(result string, resultData *casefile.ClientTransition) error {
	data, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("encoding browser receipt: %w", err)
	}
	if len(data) > casefile.MaxBrowserReceiptBytes {
		return fmt.Errorf("browser-result exceeds %d bytes", casefile.MaxBrowserReceiptBytes)
	}
	return writePrivateFile(result, data)
}

// interaction selects a native browser action for the saved operation.
//
// Takes hosts (string) which selects the component instances.
// Takes action (casefile.ClientAction) which selects the browser operation and target
// instance.
// Takes prevented (*bool) which receives whether the event's default action was
// prevented.
//
// Returns browserAction which is the selected native browser action.
// Returns error which reports an unsupported browser operation, or nil on success.
func interaction(hosts string, action casefile.ClientAction, prevented *bool) (browserAction, error) {
	host := fmt.Sprintf("%s[%d].shadowRoot", hosts, action.Instance)

	if action.Op == casefile.ToggleCheck || action.Op == casefile.SelectOption {
		selector, key := "#client-check", "Space"
		if action.Op == casefile.SelectOption {
			selector = "#client-select"
			switch action.Value {
			case "first":
				key = "Home"
			case "last":
				key = "End"
			default:
				return nil, fmt.Errorf("unsupported selection %q", action.Value)
			}
		}
		return browserActions{evaluate(host+".querySelector('"+selector+"').focus()", nil), pressKey(key)}.run, nil
	}
	drive, advanced, err := advancedInteraction(host, action)
	if err != nil {
		return nil, err
	}
	if advanced {
		return drive, nil
	}
	value, err := scriptString(action.Value)
	if err != nil {
		return nil, err
	}
	var command string
	switch action.Op {
	case casefile.SetInput:
		command = "const input = root.querySelector('#client-input'); input.value = " + value + "; input.dispatchEvent(new Event('input', {bubbles:true,composed:true}));"
	case casefile.SetLabel:
		command = "host.setAttribute('label'," + value + ");"
	case casefile.ReverseRows:
		command = "root.querySelector('#client-reverse').click();"
	case casefile.RemoveRow:
		command = "root.querySelector('#client-remove').click();"
	case casefile.ModifiedClick:
		command = "return !root.querySelector('#client-modified').dispatchEvent(new MouseEvent('click',{bubbles:true,cancelable:true,composed:true}));"
	case casefile.ResetControls:
		command = "root.querySelector('#client-reset').click();"
	case casefile.ParentClick:
		command = "root.querySelector('#client-parent').click();"
	default:
		return nil, fmt.Errorf("unknown client operation %q", action.Op)
	}
	return evaluate(fmt.Sprintf("(() => {const host=%s[%d], root=host.shadowRoot; %s return false; })()", hosts, action.Instance, command), prevented), nil
}

// readProbeRecipe decodes the optional journeys selected by the environment.
//
// Returns probeRecipe which holds the selected journeys.
// Returns error which reports a malformed recipe variable, or nil on success.
func readProbeRecipe() (probeRecipe, error) {
	var recipe probeRecipe
	recipe.coveragePath = os.Getenv(envJSCoverage)
	recipe.intake = os.Getenv(envIntake) == "true"
	var err error
	if raw := os.Getenv(envProgramPaths); raw != "" {
		recipe.routed = true
		if recipe.routePaths, err = browserProgramPaths(raw); err != nil {
			return recipe, err
		}
	}
	if recipe.checkout, err = decodeRecipe[casefile.CheckoutInput](envCheckout); err != nil {
		return recipe, err
	}
	if recipe.partialComponents, err = decodeRecipe[casefile.PartialComponentsInput](envPartialComponents); err != nil {
		return recipe, err
	}
	if recipe.events, err = decodeRecipe[casefile.EventsInput](envEvents); err != nil {
		return recipe, err
	}
	if recipe.lifecycle, err = decodeRecipe[casefile.LifecycleInput](envLifecycle); err != nil {
		return recipe, err
	}
	if recipe.dom, err = decodeRecipe[casefile.DOMPlan](envDOM); err != nil {
		return recipe, err
	}
	return recipe, nil
}

// decodeRecipe decodes one JSON recipe variable.
//
// Takes name (string) which names the environment variable.
//
// Returns *T which is the decoded recipe, or nil when the variable is empty.
// Returns error which reports malformed JSON, or nil on success.
func decodeRecipe[T any](name string) (*T, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return nil, nil
	}
	value := new(T)
	if err := json.Unmarshal([]byte(raw), value); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", name, err)
	}
	return value, nil
}

// parseActions decodes the optional saved interaction recipe.
//
// Takes raw (string) which holds the JSON-encoded interactions.
//
// Returns []casefile.ClientAction which lists the saved interactions.
// Returns error which reports an oversized or malformed recipe, or nil on success.
func parseActions(raw string) ([]casefile.ClientAction, error) {
	if len(raw) > maxActionRecipeBytes {
		return nil, fmt.Errorf("invalid browser action recipe: exceeds %d bytes", maxActionRecipeBytes)
	}
	var actions []casefile.ClientAction
	if err := json.Unmarshal([]byte(raw), &actions); err != nil {
		return nil, fmt.Errorf("invalid browser action recipe: %w", err)
	}
	if len(actions) > maxActions {
		return nil, fmt.Errorf("invalid browser action recipe: more than %d actions", maxActions)
	}
	return actions, nil
}

// main runs the browser worker selected by its command arguments.
func main() {
	os.Exit(run())
}

// run executes the worker under a context that ends when the process is interrupted.
//
// Returns int which is the process exit status.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return execute(ctx, os.Args[1:])
}

// execute runs the worker mode its arguments select and reports any failure on stderr.
//
// Takes arguments ([]string) which are the command arguments after the program name.
//
// Returns int which is the process exit status.
func execute(ctx context.Context, arguments []string) int {
	if len(arguments) == modeArgumentCount {
		mode := [modeArgumentCount]string(arguments)
		switch mode[1] {
		case "--firefox-dom":
			return report(firefoxDOM(ctx, mode[0], mode[2], mode[jsonInputIndex]), 1)
		case "--form-workflow":
			err := formProbe(ctx, mode[0], mode[2], mode[jsonInputIndex])
			return report(err, exitCode(err))
		}
	}
	if len(arguments) != probeArgumentCount && len(arguments) != modeArgumentCount {
		_, _ = fmt.Fprintln(os.Stderr, "usage: browser-worker URL TAG RESULT [JSON_ACTIONS]")
		return usageExitCode
	}
	var actions []casefile.ClientAction
	if len(arguments) == modeArgumentCount {
		var err error
		if actions, err = parseActions(arguments[jsonInputIndex]); err != nil {
			return report(err, usageExitCode)
		}
	}
	err := probe(ctx, arguments[0], arguments[1], arguments[2], actions)
	return report(err, exitCode(err))
}

// report writes a worker failure to stderr and selects the exit status.
//
// Takes err (error) which is the worker failure, or nil on success.
// Takes failure (int) which is the exit status for a failure.
//
// Returns int which is zero on success, or failure after reporting err.
func report(err error, failure int) int {
	if err == nil {
		return 0
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	return failure
}

// exitCode selects the worker's exit status for an error, so the runner can tell a
// runtime-coverage failure apart from other failures without reading the log.
//
// Takes err (error) which is the worker failure.
//
// Returns int which is the process exit status.
func exitCode(err error) int {
	if errors.Is(err, errRuntimeCoverage) {
		return casefile.RuntimeCoverageExitCode
	}
	return 1
}
