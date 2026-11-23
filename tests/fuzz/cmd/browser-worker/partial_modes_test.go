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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

func TestDriveReloadModesRunsEachModeInOrder(t *testing.T) {
	base, err := url.Parse("http://127.0.0.1:1")
	require.NoError(t, err)
	for _, reverse := range []bool{false, true} {
		input := casefile.PartialComponentsInput{Edited: "edited", ReverseReload: reverse}
		var steps []string
		var counts []int
		var waits []string
		run := func(step string, actions ...browserAction) bool {
			steps = append(steps, step)
			counts = append(counts, len(actions))
			return true
		}
		wait := func(expression string) browserAction {
			waits = append(waits, expression)
			return func(context.Context, *browser.Session) error { return nil }
		}
		require.NoError(t, driveReloadModes(base, input, run, wait))
		order := input.ReloadOrder()
		want := make([]string, 0, len(order))
		for _, mode := range order {
			want = append(want, "reload-"+mode)
		}
		assert.Equal(t, want, steps)
		assert.Equal(t, []int{5, 2, 2, 2}, counts)
		require.Len(t, waits, len(order)+1)
		assert.Contains(t, waits[1], `=== "`+order[0]+`"`)
	}
}

func TestDriveReloadModesStopsAtFirstFailedStep(t *testing.T) {
	base, err := url.Parse("http://127.0.0.1:1")
	require.NoError(t, err)
	calls := 0
	run := func(string, ...browserAction) bool {
		calls++
		return false
	}
	wait := func(string) browserAction { return nil }
	require.NoError(t, driveReloadModes(base, casefile.PartialComponentsInput{}, run, wait))
	assert.Equal(t, 1, calls)
}
