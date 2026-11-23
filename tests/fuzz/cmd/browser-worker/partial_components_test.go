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

func TestBrowserPartialComponentsRecordsUnparseableAddress(t *testing.T) {
	input := casefile.PartialComponentsInput{Edited: "edited"}
	trace := browserPartialComponents(t.Context(), nil, "://missing-scheme", input)
	assert.Contains(t, trace.DriverError, "missing protocol scheme")
	assert.Equal(t, input, trace.Input)
	assert.Empty(t, trace.States)
}

func TestPartialComponentsBudgetAddsReloadModes(t *testing.T) {
	assert.Equal(t, partialStepCount*partialStepBudget, partialComponentsBudget(casefile.PartialComponentsInput{}))
	reloading := casefile.PartialComponentsInput{ReloadModes: true}
	want := time.Duration(partialStepCount+len(reloading.ReloadOrder())) * partialStepBudget
	assert.Equal(t, want, partialComponentsBudget(reloading))
}

func TestPartialHostSelectsRecordChip(t *testing.T) {
	assert.Equal(t, "document.querySelector('article[data-record=amber] record-chip')", partialHost("amber"))
}
