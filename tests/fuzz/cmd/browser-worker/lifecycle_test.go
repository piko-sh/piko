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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestBrowserLifecycleRecordsUnparseableAddress(t *testing.T) {
	for _, input := range []casefile.LifecycleInput{{Draft: "draft", Reverse: true}, {ElementReuse: true}} {
		trace := browserLifecycle(t.Context(), nil, "://missing-scheme", input)
		assert.Equal(t, casefile.ExpectedLifecycle(input).States[0].Step, trace.FailedStep)
		assert.Contains(t, trace.DriverError, "missing protocol scheme")
		assert.Equal(t, input, trace.Input)
		assert.Empty(t, trace.States)
	}
}

func TestLifecycleBudgetCoversEveryModelledStep(t *testing.T) {
	for _, input := range []casefile.LifecycleInput{{}, {ElementReuse: true}} {
		steps := time.Duration(len(casefile.ExpectedLifecycle(input).States))
		assert.Equal(t, steps*lifecycleStepBudget, lifecycleBudget(input), "element reuse %t", input.ElementReuse)
	}
}
