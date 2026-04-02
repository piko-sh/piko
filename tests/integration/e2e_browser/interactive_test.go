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
	"errors"
	"fmt"
	"time"

	browserpkg "piko.sh/piko/wdk/browser"
)

func runInteractiveMode(h *E2EBrowserHarness) (retErr error) {
	runner := browserpkg.NewTUIRunner()

	if err := runner.Start(h.TestCase.Name); err != nil {
		return fmt.Errorf("failed to start interactive runner: %w", err)
	}
	defer runner.Close()

	defer func() {
		if r := recover(); r != nil {
			if message, ok := r.(string); ok && message == "browser: test aborted by user" {
				retErr = errors.New("test aborted by user")
				return
			}
			panic(r)
		}
	}()

	for i, step := range h.Spec.BrowserSteps {
		detail := browserpkg.StepDetail(&step)

		runner.BeforeStep(step.Action, detail)
		runner.WaitForContinue()

		start := time.Now()
		err := executeStep(h, step)
		duration := time.Since(start)

		runner.AfterStep(step.Action, detail, err != nil, duration)

		if err != nil {
			return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Action, err)
		}
	}

	return nil
}
