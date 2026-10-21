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

package dalconformance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// receiptWorkflowID is the workflow the receipt lifecycle case queues a receipt for.
	receiptWorkflowID = "wf-r"
)

// newProcessing creates a task that is processing its given attempt, stamped now.
//
// Takes t (*testing.T) which reports test failures.
// Takes id (string) which is the task ID.
// Takes attempt (int) which is the attempt being processed.
//
// Returns *orchestrator_domain.Task which is the stored task.
func (h *Harness) newProcessing(t *testing.T, id string, attempt int) *orchestrator_domain.Task {
	t.Helper()
	task := h.newTask(id)
	task.Status = orchestrator_domain.StatusProcessing
	task.Attempt = attempt
	h.create(t, task)
	return task
}

// receiptCases checks workflow receipts.
//
// Returns []testCase which lists the cases.
func receiptCases() []testCase {
	return []testCase{
		{name: "receipts/lifecycle", run: func(t *testing.T, h *Harness) {
			ctx := t.Context()
			require.NoError(t, h.Store.CreateWorkflowReceipt(ctx, "receipt-1", receiptWorkflowID, nodeA))
			require.NoError(t, h.Store.CreateWorkflowReceipt(ctx, "receipt-2", "wf-other", "node-b"))

			pending, err := h.Store.GetPendingReceiptsByNode(ctx, nodeA)
			require.NoError(t, err)
			require.Len(t, pending, 1)
			assert.Equal(t, "receipt-1", pending[0].ID)
			assert.Equal(t, receiptWorkflowID, pending[0].WorkflowID)

			resolved, err := h.Store.ResolveWorkflowReceipts(ctx, receiptWorkflowID, "")
			require.NoError(t, err)
			assert.Equal(t, 1, resolved)
			resolved, err = h.Store.ResolveWorkflowReceipts(ctx, receiptWorkflowID, "")
			require.NoError(t, err)
			assert.Zero(t, resolved, "a resolved receipt is not resolved twice")

			pending, err = h.Store.GetPendingReceiptsByNode(ctx, nodeA)
			require.NoError(t, err)
			assert.Empty(t, pending)

			h.Clock.Advance(time.Hour)
			cleaned, err := h.Store.CleanupOldResolvedReceipts(ctx, h.Clock.Now().Add(-30*time.Minute))
			require.NoError(t, err)
			assert.Equal(t, 1, cleaned, "only the resolved receipt is old enough to clean up")

			pending, err = h.Store.GetPendingReceiptsByNode(ctx, "node-b")
			require.NoError(t, err)
			assert.Len(t, pending, 1, "another node's pending receipt is untouched")
		}},
		{name: "receipts/timeout-stale", run: func(t *testing.T, h *Harness) {
			ctx := t.Context()
			require.NoError(t, h.Store.CreateWorkflowReceipt(ctx, "old", "wf-old", nodeA))
			h.Clock.Advance(2 * time.Hour)
			require.NoError(t, h.Store.CreateWorkflowReceipt(ctx, "new", "wf-new", nodeA))

			timedOut, err := h.Store.TimeoutStaleReceipts(ctx, h.Clock.Now().Add(-time.Hour))
			require.NoError(t, err)
			assert.Equal(t, 1, timedOut)

			pending, err := h.Store.GetPendingReceiptsByNode(ctx, nodeA)
			require.NoError(t, err)
			require.Len(t, pending, 1)
			assert.Equal(t, "new", pending[0].ID)
		}},
	}
}

// semanticsCases checks time handling.
//
// Returns []testCase which lists the cases.
func semanticsCases() []testCase {
	return []testCase{
		{name: "time/stored-at-second-precision", run: func(t *testing.T, h *Harness) {
			task := h.newTask("precise")
			task.ExecuteAt = Epoch.Add(1500 * time.Millisecond)
			h.create(t, task)
			assert.True(t, h.get(t, "precise").ExecuteAt.Equal(Epoch.Add(time.Second)),
				"every store keeps whole seconds, so the backends order and compare times alike")
		}},
	}
}

// durabilityCases checks what survives a restart.
//
// Returns []testCase which lists the cases.
func durabilityCases() []testCase {
	return []testCase{
		{name: "durability/survives-reopen", run: func(t *testing.T, h *Harness) {
			if h.Reopen == nil {
				t.Skip("the store keeps nothing across a restart")
			}
			task := h.newTask("durable")
			task.DeduplicationKey = "dedup:durable"
			require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), task))

			h.Store = h.Reopen(t)
			got := h.get(t, "durable")
			assert.Equal(t, orchestrator_domain.StatusPending, got.Status)
			duplicate := h.newTask("after-restart")
			duplicate.DeduplicationKey = "dedup:durable"
			assert.ErrorIs(t, h.Store.CreateTaskWithDedup(t.Context(), duplicate), orchestrator_domain.ErrDuplicateTask,
				"the deduplication index is rebuilt on restart")
		}},
	}
}
