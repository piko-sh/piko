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
	"fmt"
	"net/url"
	"os"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

const (
	// firefoxBudget bounds the whole Firefox DOM-plan session, including start-up.
	firefoxBudget = 45 * time.Second
)

var (
	// errFirefoxBudget is the cause recorded when the Firefox session runs out of time.
	errFirefoxBudget = fmt.Errorf("%s: Firefox session: %w", casefile.BrowserBudgetMarker, context.DeadlineExceeded)
)

// bidiDOMSession performs the DOM plan through WebDriver BiDi.
//
// Takes session (*browser.FirefoxSession) which is the case-private Firefox session.
// Takes address (string) which is the application URL to visit.
// Takes plan (casefile.DOMPlan) which holds the saved DOM operations and expected values.
// Takes trace (*casefile.DOMTrace) which receives the browser's observations, including
// partial progress.
//
// Returns error which reports a browser operation or DOM observation failure, or nil on
// success.
func bidiDOMSession(ctx context.Context, session *browser.FirefoxSession, address string, plan casefile.DOMPlan, trace *casefile.DOMTrace) error {
	trace.Engine, trace.Version = "firefox", session.Version()
	if err := session.NavigateURL(ctx, address); err != nil {
		return err
	}
	ready, err := domSelector(plan.ReadyHost, plan.ReadyElement)
	if err != nil {
		return err
	}
	if _, err := session.EvaluateString(ctx, "new Promise(ok=>{const rendered=()=>("+ready+")?ok('ready'):setTimeout(rendered,16);rendered()})"); err != nil {
		return err
	}
	for _, step := range plan.Steps {
		state, err := bidiDOMStep(ctx, session, step)
		if err != nil {
			return fmt.Errorf("%s: %w", step.ID, err)
		}
		trace.States = append(trace.States, state)
	}
	return nil
}

// bidiDOMStep performs one DOM plan step through WebDriver BiDi and observes the result.
//
// Takes session (*browser.FirefoxSession) which is the case-private Firefox session.
// Takes step (casefile.DOMStep) which is the operation and its queries.
//
// Returns casefile.DOMState which is the step's observation.
// Returns error which reports a browser operation or observation failure, or nil on
// success.
func bidiDOMStep(ctx context.Context, session *browser.FirefoxSession, step casefile.DOMStep) (casefile.DOMState, error) {
	var state casefile.DOMState
	target, err := domSelector(step.Host, step.Target)
	if err != nil {
		return state, err
	}
	switch step.Op {
	case "viewport":
		if err := session.SetViewport(ctx, step.ViewportWidth(), domViewportHeight); err != nil {
			return state, err
		}
	case "click", "focus":
		if _, err := session.EvaluateString(ctx, target+"."+step.Op+"();''"); err != nil {
			return state, err
		}
	case "type", "key":
		if _, err := session.EvaluateString(ctx, target+".focus();''"); err != nil {
			return state, err
		}
		if step.Op == "key" {
			err = session.PressKey(ctx, step.Value)
		} else {
			err = session.Key(ctx, step.Value)
		}
		if err != nil {
			return state, err
		}
	}
	raw, err := json.Marshal(step)
	if err != nil {
		return state, fmt.Errorf("encoding DOM step: %w", err)
	}
	value, err := session.EvaluateString(ctx, "new Promise(ok=>requestAnimationFrame(()=>requestAnimationFrame(()=>ok(JSON.stringify("+queryDOM+"("+string(raw)+"))))))")
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return state, fmt.Errorf("decoding DOM observation: %w", err)
	}
	return state, nil
}

// firefoxDOM launches Firefox and writes the DOM-plan receipt.
//
// Takes base (string) which is the application base URL.
// Takes output (string) which is the private receipt output path.
// Takes raw (string) which holds the saved DOM plan as JSON.
//
// Returns error which reports an invalid plan, Firefox startup, protocol or receipt-write
// failure, or nil on success.
func firefoxDOM(ctx context.Context, base, output, raw string) error {
	var input casefile.DOMPlan
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return fmt.Errorf("decoding DOM plan: %w", err)
	}
	plan, err := input.DriverPlan()
	if err != nil {
		return err
	}
	origin, err := url.Parse(base)
	if err != nil {
		return err
	}
	destination, err := origin.Parse(plan.Path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeoutCause(ctx, firefoxBudget, errFirefoxBudget)
	defer cancel()
	session, err := browser.OpenFirefoxSession(ctx)
	if err != nil {
		return fmt.Errorf("infrastructure: Firefox: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "closing Firefox:", err)
		}
	}()
	trace := casefile.DOMTrace{}
	if err := bidiDOMSession(ctx, session, destination.String(), plan, &trace); err != nil {
		trace.DriverError = err.Error()
		if cause := context.Cause(ctx); cause != nil {
			trace.DriverError = cause.Error() + ": " + trace.DriverError
		}
	}
	return writeFirefoxReceipt(output, &trace)
}

// writeFirefoxReceipt encodes and writes a bounded Firefox DOM receipt.
//
// Takes output (string) which is the receipt output path.
// Takes trace (*casefile.DOMTrace) which holds the Firefox observations.
//
// Returns error which reports an encoding, size or write failure, or nil on success.
func writeFirefoxReceipt(output string, trace *casefile.DOMTrace) error {
	data, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("encoding Firefox receipt: %w", err)
	}
	if len(data) > casefile.MaxBrowserReceiptBytes {
		return fmt.Errorf("firefox receipt exceeds %d bytes", casefile.MaxBrowserReceiptBytes)
	}
	return writePrivateFile(output, data)
}
