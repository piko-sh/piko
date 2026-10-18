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

package otter

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestCapacityRejectsActiveWorkAndReclaimsTerminalIndexes(t *testing.T) {
	raw, err := NewOtterDAL(Config{Capacity: 2})
	require.NoError(t, err)
	store, ok := raw.(*DAL)
	require.True(t, ok)
	defer store.Close()
	ctx := t.Context()
	first := makeTask("first", "first")
	first.DeduplicationKey = "dedup"
	require.NoError(t, store.CreateTaskWithDedup(ctx, first))
	require.NoError(t, store.CreateTask(ctx, makeTask("second", "second")))
	require.ErrorIs(t, store.CreateTask(ctx, makeTask("third", "third")), d.ErrTaskCapacity)
	rows, err := store.GetTasksByID(ctx, []string{"first", "second"})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	first.Status = d.StatusComplete
	require.NoError(t, store.UpdateTask(ctx, first))
	require.NoError(t, store.CreateTask(ctx, makeTask("third", "third")))
	require.Empty(t, store.workflowIndex.Get("first"))
	require.Empty(t, store.dedupIndex)
	require.Equal(t, 2, store.executeIndex.Size())
	require.Zero(t, store.terminalIndex.Size())
}
func TestJournalRestoresTaskReceiptAndIndexes(t *testing.T) {
	store := newTestDAL(t)
	ctx := t.Context()
	task := makeTask("task", "workflow")
	task.Status = d.StatusProcessing
	task.UpdatedAt = time.Now().Add(-time.Hour)
	task.DeduplicationKey = "dedup"
	require.NoError(t, store.CreateTaskWithDedup(ctx, task))
	require.NoError(t, store.CreateWorkflowReceipt(ctx, "receipt", "workflow", "node"))
	abort := errors.New("abort")
	err := store.RunAtomic(ctx, func(ctx context.Context, tx d.TaskStore) error {
		recovered := *task
		recovered.Status = d.StatusRetrying
		recovered.LastError = "recovery"
		recovered.ExecuteAt = time.Now()
		if _, err := tx.TransitionTask(ctx, d.Transition{Task: &recovered, From: []d.TaskStatus{d.StatusProcessing}}); err != nil {
			return err
		}
		if _, err := tx.ResolveWorkflowReceipts(ctx, "workflow", "failed"); err != nil {
			return err
		}
		if _, err := tx.CleanupOldResolvedReceipts(ctx, time.Now().Add(time.Hour)); err != nil {
			return err
		}
		return abort
	})
	require.ErrorIs(t, err, abort)
	rows, err := store.GetTasksByID(ctx, []string{"task"})
	require.NoError(t, err)
	require.Equal(t, d.StatusProcessing, rows[0].Status)
	require.Equal(t, 0, rows[0].Attempt)
	require.Equal(t, "task", store.dedupIndex["dedup"])
	require.Equal(t, 1, store.processingIndex.Size())
	require.Zero(t, store.executeIndex.Size())
	require.Equal(t, receiptPending, store.receipts["receipt"].status)
	require.Zero(t, store.resolvedIndex.Size())
	require.Len(t, store.receiptsByWorkflow.Get("workflow"), 1)
}
func TestResolvedReceiptCleanupIsBoundedAndPreservesPending(t *testing.T) {
	store := newTestDAL(t)
	ctx := t.Context()
	for i := range 250 {
		require.NoError(t, store.CreateWorkflowReceipt(ctx, fmt.Sprint(i), "done", "node"))
	}
	_, err := store.ResolveWorkflowReceipts(ctx, "done", "")
	require.NoError(t, err)
	require.NoError(t, store.CreateWorkflowReceipt(ctx, "pending", "running", "node"))
	for _, want := range []int{100, 100, 50} {
		count, err := store.CleanupOldResolvedReceipts(ctx, time.Now().Add(time.Second))
		require.NoError(t, err)
		require.Equal(t, want, count)
	}
	require.Len(t, store.receipts, 1)
	require.Empty(t, store.receiptsByWorkflow.Get("done"))
	require.Len(t, store.receiptsByNode.Get("node"), 1)
}
func BenchmarkAtomicReceiptHistory(b *testing.B) {
	for _, history := range []int{1000, 100000} {
		b.Run(fmt.Sprint(history), func(b *testing.B) {
			raw, err := NewOtterDAL(Config{Capacity: 200000})
			if err != nil {
				b.Fatal(err)
			}
			store, ok := raw.(*DAL)
			if !ok {
				b.Fatal("unexpected task store")
			}
			defer store.Close()
			ctx := context.Background()
			for i := range history {
				if err := store.CreateWorkflowReceipt(ctx, fmt.Sprint(i), "history", "node"); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := store.RunAtomic(ctx, func(ctx context.Context, tx d.TaskStore) error {
					return tx.CreateWorkflowReceipt(ctx, "one", "current", "node")
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestPendingDispatchCursorContinuesAfterClaimedEntriesDisappear(t *testing.T) {
	store := newTestDAL(t)
	now := time.Now().Add(-time.Minute)
	for i := range 250 {
		task := makeTask(fmt.Sprintf("%03d", i), "workflow")
		task.ExecuteAt = now
		require.NoError(t, store.CreateTask(t.Context(), task))
	}
	first, err := store.PendingDispatchPage(t.Context(), "", time.Time{}, 100)
	require.NoError(t, err)
	require.Len(t, first, 100)
	cursor := first[99]
	for _, task := range first {
		task.Status = d.StatusProcessing
		require.NoError(t, store.UpdateTask(t.Context(), task))
	}
	second, err := store.PendingDispatchPage(t.Context(), cursor.ID, cursor.ExecuteAt, 100)
	require.NoError(t, err)
	require.Len(t, second, 100)
	require.Equal(t, "100", second[0].ID)
	require.Equal(t, "199", second[99].ID)
}
