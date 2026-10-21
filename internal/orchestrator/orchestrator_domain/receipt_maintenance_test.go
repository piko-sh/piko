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

package orchestrator_domain

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestReceiptMaintenanceRotatesBeyondItsBatch(t *testing.T) {
	seen := make(map[string]int)
	store := &MockTaskStore{GetWorkflowStatusFunc: func(_ context.Context, id string) (bool, error) { seen[id]++; return false, nil }}
	service := &orchestratorService{taskStore: store, clock: clock.RealClock(), receipts: make(map[string][]*WorkflowReceipt), receiptsMutex: sync.Mutex{}}
	for i := range 250 {
		require.NoError(t, service.registerReceiptInMemory(t.Context(), newWorkflowReceipt(fmt.Sprint(i))))
	}
	for range 3 {
		service.maintainReceipts(t.Context())
	}
	require.Len(t, seen, 250)
	require.Equal(t, 2, seen["0"])
	require.Equal(t, 1, seen["249"])
}
func TestReceiptCannotResolveBeforeAdmission(t *testing.T) {
	service := &orchestratorService{receipts: make(map[string][]*WorkflowReceipt), receiptsMutex: sync.Mutex{}}
	receipt := newWorkflowReceipt("workflow")
	receipt.id = "receipt"
	receipt.awaitingAdmission = true
	require.NoError(t, service.registerReceiptInMemory(t.Context(), receipt))
	service.resolveReceipts(t.Context(), "workflow", nil)
	select {
	case <-receipt.Done():
		t.Fatal("receipt resolved before admission")
	default:
	}
	service.receiptAdmitted("workflow", "receipt")
	service.resolveReceipts(t.Context(), "workflow", nil)
	select {
	case <-receipt.Done():
	default:
		t.Fatal("accepted receipt did not resolve")
	}
	require.Zero(t, service.receiptWork.count)
	require.Zero(t, service.receiptWork.queue.Len())
	require.Empty(t, service.receiptWork.entries)
}

type pendingPageStore struct{ *MockTaskStore }

func (*pendingPageStore) PendingDispatchPage(context.Context, string, time.Time, int) ([]*Task, error) {
	return []*Task{{ID: "pending", WorkflowID: "pending", Status: StatusPending}}, nil
}
func TestMaintenanceResolvesReceiptsWithoutRedispatching(t *testing.T) {
	store := &pendingPageStore{MockTaskStore: &MockTaskStore{
		GetWorkflowStatusFunc: func(context.Context, string) (bool, error) { return true, nil },
		GetTasksByIDFunc: func(context.Context, []string) ([]*Task, error) {
			return []*Task{{ID: "done", Status: StatusComplete}}, nil
		},
	}}
	receipt := newWorkflowReceipt("done")
	dispatcher := NewMockTaskDispatcher()
	dispatched := false
	dispatcher.DispatchFunc = func(context.Context, *Task) error {
		dispatched = true
		return nil
	}
	service := &orchestratorService{taskStore: store, taskDispatcher: dispatcher, clock: clock.RealClock(), receipts: make(map[string][]*WorkflowReceipt)}
	require.NoError(t, service.registerReceiptInMemory(t.Context(), receipt))
	service.maintainReceipts(t.Context())

	select {
	case <-receipt.Done():
	default:
		t.Error("maintenance did not resolve the receipt of a completed workflow")
	}
	assert.False(t, dispatched, "re-dispatching lost messages is the dispatcher's sweep, not receipt maintenance")
}
