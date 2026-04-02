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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	browserpkg "piko.sh/piko/wdk/browser"
)

func executeStep(h *E2EBrowserHarness, step BrowserStep) error {

	if step.Action == "captureDOM" {
		return actionCaptureDOM(h, step)
	}

	if browserpkg.IsAssertionAction(step.Action) {
		return browserpkg.ExecuteAssertion(h.ActionCtx, &step)
	}

	return browserpkg.ExecuteStep(h.ActionCtx, &step)
}

func actionCaptureDOM(h *E2EBrowserHarness, step BrowserStep) error {

	time.Sleep(50 * time.Millisecond)

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
