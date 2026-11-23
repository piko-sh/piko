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

package browser

import (
	"bufio"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func simpleRunnerReading(input string) *simpleRunner {
	runner := newSimpleRunner()
	runner.reader = bufio.NewReader(strings.NewReader(input))
	return runner
}

func TestSimpleRunnerRecordsSteps(t *testing.T) {
	runner := simpleRunnerReading("")
	require.NoError(t, runner.Start("TestCheckout"))
	assert.Equal(t, "TestCheckout", runner.testName)

	runner.BeforeStep("Click", "#buy")
	runner.AfterStep("Click", "#buy", false, time.Millisecond)
	runner.AfterStep("Fill", "#name", true, time.Millisecond)
	runner.Close()

	require.Len(t, runner.steps, 2)
	assert.Equal(t, Step{Action: "Click", Detail: "#buy", State: StepPassed, Duration: time.Millisecond}, runner.steps[0])
	assert.Equal(t, StepFailed, runner.steps[1].State)
}

func TestSimpleRunnerWaitForContinue(t *testing.T) {
	paused := simpleRunnerReading("ignored\n0\n\n")
	paused.WaitForContinue()
	assert.Equal(t, simpleModePaused, paused.mode)

	autoplay := simpleRunnerReading("3\n")
	autoplay.WaitForContinue()
	assert.Equal(t, simpleModeAutoplay, autoplay.mode)
	assert.Equal(t, 3*time.Second, autoplay.autoplayGap)
	autoplay.render("Navigate", "/")

	autoplay.autoplayGap = 0
	autoplay.WaitForContinue()
	assert.Equal(t, simpleModeAutoplay, autoplay.mode)
}

func TestPageEnableInteractiveSimple(t *testing.T) {
	page := &Page{t: t}
	page.enableInteractive(false)
	runner, ok := page.interactiveRunner.(*simpleRunner)
	require.True(t, ok)
	assert.Equal(t, t.Name(), runner.testName)

	runner.reader = bufio.NewReader(strings.NewReader("\n"))
	page.beforeAction("Click", "#go")
	page.afterAction("Click", "#go", false, time.Millisecond)
	require.Len(t, runner.steps, 1)
	assert.Equal(t, "Click", runner.steps[0].Action)
}

func TestPageWithoutRunnerSkipsInteractiveHooks(t *testing.T) {
	page := &Page{t: t}
	page.beforeAction("Click", "#go")
	page.afterAction("Click", "#go", true, time.Millisecond)
	assert.Nil(t, page.interactiveRunner)
}
