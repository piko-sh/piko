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

package goroutine_test

import (
	"context"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/goroutine"
)

func TestLabelAppearsInTheGoroutineTraceback(t *testing.T) {
	t.Parallel()

	var (
		waitGroup sync.WaitGroup
		stack     string
	)

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		goroutine.Label(context.Background(), "orchestrator.runHeartbeat", "task_id", "T-42")
		stack = string(debug.Stack())
	}()
	waitGroup.Wait()

	header, _, _ := strings.Cut(stack, "\n")
	assert.Contains(t, header, "orchestrator.runHeartbeat",
		"Go 1.27 prints goroutine labels in the traceback header, which is what makes a "+
			"panic log self-describing")
	assert.Contains(t, header, "T-42",
		"Additional label pairs must appear alongside the component")
}

func TestLabelDiscardsAnOddTrailingKey(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		goroutine.Label(context.Background(), "pkg.fn", "key_without_value")
	}, "pprof.Labels panics on an odd argument count, which this must absorb")
}

func TestLabelReturnsAContextCarryingTheLabels(t *testing.T) {
	t.Parallel()

	ctx := goroutine.Label(context.Background(), "pkg.fn", "task_id", "T-7")

	component, ok := pprof.Label(ctx, "component")
	require.True(t, ok)
	assert.Equal(t, "pkg.fn", component)

	taskID, ok := pprof.Label(ctx, "task_id")
	require.True(t, ok)
	assert.Equal(t, "T-7", taskID)
}
