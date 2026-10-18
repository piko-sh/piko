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
	"container/list"
	"context"
	"errors"
	"piko.sh/piko/wdk/goroutine"
	"time"
)

const (
	// receiptMaintenanceBatch caps workflows or pending tasks visited in one maintenance
	// pass.
	receiptMaintenanceBatch = 100

	// defaultReceiptCapacity bounds in-memory receipts when the store supplies no limit.
	defaultReceiptCapacity = 100000
)

// receiptWork provides bounded round-robin reconciliation of accepted workflows.
type receiptWork struct {
	// entries maps workflow IDs to their queue positions; receiptsMutex guards access.
	entries map[string]*list.Element

	// queue rotates workflows pending receipt resolution; receiptsMutex guards access.
	queue list.List

	// count tracks registered receipts for admission limits; receiptsMutex guards access.
	count int
}

// addReceiptWorkLocked counts a receipt and queues its workflow for reconciliation.
//
// Takes id (string) which identifies the workflow receiving a receipt.
//
// The caller must hold receiptsMutex.
func (s *orchestratorService) addReceiptWorkLocked(id string) {
	if s.receiptWork.entries == nil {
		s.receiptWork.entries = make(map[string]*list.Element)
	}
	if s.receiptWork.entries[id] == nil {
		s.receiptWork.entries[id] = s.receiptWork.queue.PushBack(id)
	}
	s.receiptWork.count++
}

// removeReceiptWorkLocked releases receipt capacity and removes empty workflows from the
// queue.
//
// Takes id (string) which identifies the workflow whose receipts were removed.
// Takes count (int) which is the number of receipts removed.
//
// The caller must hold receiptsMutex and update the receipts map before calling.
func (s *orchestratorService) removeReceiptWorkLocked(id string, count int) {
	s.receiptWork.count -= count
	if len(s.receipts[id]) == 0 {
		if e := s.receiptWork.entries[id]; e != nil {
			s.receiptWork.queue.Remove(e)
			delete(s.receiptWork.entries, id)
		}
	}
}

// receiptMaintenanceLoop reconciles receipts every thirty seconds until shutdown.
//
// Runs in the service's maintenance goroutine and owns the pending-dispatch cursor.
func (s *orchestratorService) receiptMaintenanceLoop() {
	defer goroutine.RecoverPanic(s.runCtx, "orchestrator.receiptMaintenanceLoop")
	ticker := s.clock.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.runCtx.Done():
			return
		case <-ticker.C():
			s.maintainReceipts(s.runCtx)
		}
	}
}

// maintainReceipts resolves a bounded batch of workflows and retries pending dispatches.
//
// Each pass has a five-second deadline and also cleans up old resolved receipts.
//
// Concurrency: receiptsMutex protects the receipt queue; the maintenance loop must
// serialise calls because it owns the pending-dispatch cursor.
func (s *orchestratorService) maintainReceipts(ctx context.Context) {
	if s.taskStore == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	s.receiptsMutex.Lock()
	ids := make([]string, 0, min(receiptMaintenanceBatch, s.receiptWork.queue.Len()))
	for range cap(ids) {
		e := s.receiptWork.queue.Front()
		ids = append(ids, e.Value.(string))
		s.receiptWork.queue.MoveToBack(e)
	}
	s.receiptsMutex.Unlock()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		complete, err := s.taskStore.GetWorkflowStatus(ctx, id)
		if err != nil || !complete {
			continue
		}
		outcome, err := s.workflowOutcome(ctx, id)
		if err != nil {
			continue
		}

		s.resolveReceipts(ctx, id, outcome)
	}
	if n, err := s.taskStore.CleanupOldResolvedReceipts(ctx, s.clock.Now().Add(-time.Hour)); err == nil {
		ReceiptCleanupCount.Add(ctx, int64(n))
	}
}

// receiptAdmitted marks an in-memory receipt as accepted by storage.
//
// Takes workflowID (string) which identifies the receipt's workflow.
// Takes receiptID (string) which identifies the admitted receipt.
//
// Safe for concurrent use; receiptsMutex guards the admission flag.
func (s *orchestratorService) receiptAdmitted(workflowID, receiptID string) {
	s.receiptsMutex.Lock()
	defer s.receiptsMutex.Unlock()
	for _, receipt := range s.receipts[workflowID] {
		if receipt.id == receiptID {
			receipt.awaitingAdmission = false
		}
	}
}

// workflowOutcome reads the persisted failure for a settled workflow.
//
// Stores without WorkflowError support fall back to looking up a task with the workflow
// ID.
//
// Takes id (string) which identifies the workflow to inspect.
//
// Returns outcome (error) which is the stored failure, or nil when no failure is found.
// Returns readErr (error) which reports a storage failure or an unavailable workflow
// outcome.
func (s *orchestratorService) workflowOutcome(ctx context.Context, id string) (outcome error, readErr error) {
	if reader, ok := s.taskStore.(interface {
		WorkflowError(context.Context, string) (error, error)
	}); ok {
		return reader.WorkflowError(ctx, id)
	}
	rows, err := s.taskStore.GetTasksByID(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, errors.New("task store cannot report the workflow outcome")
	}
	if rows[0].Status == StatusFailed {
		return errors.New(rows[0].LastError), nil
	}
	return nil, nil
}
