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

//go:build integration

package e2e_browser_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	browserpkg "piko.sh/piko/wdk/browser"
)

const (
	renderedFrameTimeout = 5 * time.Second
	renderedFrameScript  = `new Promise(resolve => {
		const afterFrame = () => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true)));
		if (document.readyState === 'complete') {
			afterFrame();
			return;
		}
		window.addEventListener('load', afterFrame, {once: true});
	})`
	clickOnAppearanceAction       = "clickOnAppearance"
	clickOnAppearanceTimeout      = 5 * time.Second
	clickOnAppearancePollInterval = 5 * time.Millisecond
	elementCentreScript           = `(() => {
		const element = document.querySelector(%q);
		if (!element) return {found: false, x: 0, y: 0};
		const rect = element.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) return {found: false, x: 0, y: 0};
		return {found: true, x: rect.x + rect.width / 2, y: rect.y + rect.height / 2};
	})()`
)

var (
	errRenderedFrameTimeout     = errors.New("page did not load and paint before the DOM capture")
	errClickOnAppearanceTimeout = errors.New("element to click did not appear in time")
)

type elementCentre struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Found bool    `json:"found"`
}

func executeStep(h *E2EBrowserHarness, step BrowserStep) error {

	if step.Action == "captureDOM" {
		return actionCaptureDOM(h, step)
	}

	if step.Action == runtimeDeployAction {
		return runRuntimeDeployStep(h, step)
	}

	if step.Action == checkRuntimeDeployAction {
		return runCheckRuntimeDeployStep(h, step)
	}

	if step.Action == clickOnAppearanceAction {
		return runClickOnAppearanceStep(h, step)
	}

	if browserpkg.IsAssertionAction(step.Action) {
		return browserpkg.ExecuteAssertion(h.ActionCtx, &step)
	}

	return browserpkg.ExecuteStep(h.ActionCtx, &step)
}

func actionCaptureDOM(h *E2EBrowserHarness, step BrowserStep) error {
	if err := waitForRenderedFrame(h); err != nil {
		return err
	}

	html, err := browserpkg.CaptureDOM(h.ActionCtx, step.Selector, !step.ExcludeShadowRoots)
	if err != nil {
		return err
	}

	normalised := browserpkg.NormaliseDOM(html, browserpkg.DefaultNormaliseOptions())
	goldenPath := h.GoldenPath(step.GoldenFile)

	if *updateGoldenFiles {
		h.T.Logf("  Updating golden file: %s", goldenPath)
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			return fmt.Errorf("creating golden directory: %w", err)
		}
		if err := os.WriteFile(goldenPath, []byte(normalised), 0644); err != nil {
			return fmt.Errorf("writing golden file: %w", err)
		}
	} else {
		expected, err := os.ReadFile(goldenPath)
		if err != nil {
			return fmt.Errorf("reading golden file %s: %w (run with -update flag to create it)", goldenPath, err)
		}

		if !bytes.Equal(expected, []byte(normalised)) {
			diffCmd := exec.Command("diff", "-u", goldenPath, "-")
			diffCmd.Stdin = strings.NewReader(normalised)
			diffOutput, _ := diffCmd.CombinedOutput()

			h.T.Errorf("DOM mismatch for %s:\n%s", step.GoldenFile, string(diffOutput))
		}
	}

	h.T.Logf("  Captured DOM to '%s'", step.GoldenFile)
	return nil
}

func waitForRenderedFrame(h *E2EBrowserHarness) error {
	ctx, cancel := context.WithTimeoutCause(h.ActionCtx.Ctx, renderedFrameTimeout, errRenderedFrameTimeout)
	defer cancel()
	frameCtx := *h.ActionCtx
	frameCtx.Ctx = ctx
	if err := browserpkg.Eval(&frameCtx, "window", renderedFrameScript); err != nil {
		return fmt.Errorf("waiting for the page to render: %w", err)
	}
	return nil
}

func runClickOnAppearanceStep(h *E2EBrowserHarness, step BrowserStep) error {
	ctx, cancel := context.WithTimeoutCause(h.PageHelper.Ctx(), clickOnAppearanceTimeout, errClickOnAppearanceTimeout)
	defer cancel()

	centre, err := waitForElementCentre(ctx, step.Selector)
	if err != nil {
		return fmt.Errorf("waiting for %s to click it: %w", step.Selector, err)
	}
	return chromedp.Run(ctx, chromedp.ActionFunc(func(actionCtx context.Context) error {
		if err := input.DispatchMouseEvent(input.MousePressed, centre.X, centre.Y).WithButton(input.Left).WithClickCount(1).Do(actionCtx); err != nil {
			return fmt.Errorf("pressing the mouse on %s: %w", step.Selector, err)
		}
		if err := input.DispatchMouseEvent(input.MouseReleased, centre.X, centre.Y).WithButton(input.Left).WithClickCount(1).Do(actionCtx); err != nil {
			return fmt.Errorf("releasing the mouse on %s: %w", step.Selector, err)
		}
		return nil
	}))
}

func waitForElementCentre(ctx context.Context, selector string) (elementCentre, error) {
	script := fmt.Sprintf(elementCentreScript, selector)
	ticker := time.NewTicker(clickOnAppearancePollInterval)
	defer ticker.Stop()

	for {
		var centre elementCentre
		if err := chromedp.Run(ctx, chromedp.Evaluate(script, &centre)); err != nil {
			return elementCentre{}, err
		}
		if centre.Found {
			return centre, nil
		}
		select {
		case <-ctx.Done():
			return elementCentre{}, context.Cause(ctx)
		case <-ticker.C:
		}
	}
}
