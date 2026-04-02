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
	"errors"
	"time"

	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// checkAndResolveWorkflow checks if a workflow has finished and resolves its
// receipt.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes workflowID (string) which identifies the workflow to check.
// Takes latestErr (error) which is the most recent error from the workflow.
func (s *orchestratorService) checkAndResolveWorkflow(ctx context.Context, workflowID string, latestErr error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "OrchestratorService.checkAndResolveWorkflow",
		logger_domain.String(attributeKeyWorkflowID, workflowID),
	)
	defer span.End()

	sfKey := "workflow-check:" + workflowID
	_, _, _ = s.workflowCheckGroup.Do(sfKey, func() (any, error) {
		isComplete, err := s.taskStore.GetWorkflowStatus(ctx, workflowID)
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				l.Error("Singleflight: Failed to check workflow status", logger_domain.Error(err))
				WorkflowStatusCheckErrorCount.Add(ctx, 1)
			}
			return nil, err
		}

		if isComplete {
			l.Trace("Workflow completed", logger_domain.String(attributeKeyWorkflowID, workflowID))
			if latestErr != nil {
				WorkflowFailureCount.Add(ctx, 1)
			} else {
				WorkflowSuccessCount.Add(ctx, 1)
			}
			s.resolveReceipts(ctx, workflowID, latestErr)
			ActiveWorkflowsCount.Add(ctx, -1)
		}
		return nil, nil
	})
}

// registerReceipt adds a new receipt to the internal map and persists to
// database.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached internally so persistence completes independently.
// Takes receiptID (string) which is the unique identifier for this receipt.
// Takes receipt (*WorkflowReceipt) which is the workflow receipt to store.
func (s *orchestratorService) registerReceipt(ctx context.Context, receiptID string, receipt *WorkflowReceipt) {
	ctx, l := logger_domain.From(ctx, log)
	s.registerReceiptInMemory(ctx, receipt)

	if s.taskStore != nil {
		ctx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), 5*time.Second,
			errors.New("workflow receipt persistence exceeded 5s timeout"))
		if err := s.taskStore.CreateWorkflowReceipt(ctx, receiptID, receipt.WorkflowID, s.nodeID); err != nil {
			l.Warn("Failed to persist workflow receipt",
				logger_domain.Error(err),
				logger_domain.String("receiptID", receiptID),
				logger_domain.String(attributeKeyWorkflowID, receipt.WorkflowID))
		}
		cancel()
	}
}

// registerReceiptInMemory adds a receipt to the in-memory map without
// persisting to the store.
//
// Takes ctx (context.Context) which carries tracing values for metrics.
// Takes receipt (*WorkflowReceipt) which is the workflow receipt to track.
//
// Safe for concurrent use. Uses a mutex to protect the receipts map.
func (s *orchestratorService) registerReceiptInMemory(ctx context.Context, receipt *WorkflowReceipt) {
	s.receiptsMutex.Lock()
	s.receipts[receipt.WorkflowID] = append(s.receipts[receipt.WorkflowID], receipt)
	isFirstReceipt := len(s.receipts[receipt.WorkflowID]) == 1
	s.receiptsMutex.Unlock()

	if isFirstReceipt {
		ActiveWorkflowsCount.Add(context.WithoutCancel(ctx), 1)
	}
}

// removeReceipt removes a specific receipt from the waitlist.
//
// Takes receiptToRemove (*WorkflowReceipt) which specifies the receipt to
// remove from the workflow's waitlist.
//
// Safe for concurrent use. Acquires the receipts mutex for the duration of the
// operation.
func (s *orchestratorService) removeReceipt(receiptToRemove *WorkflowReceipt) {
	s.receiptsMutex.Lock()
	defer s.receiptsMutex.Unlock()

	waiters, ok := s.receipts[receiptToRemove.WorkflowID]
	if !ok {
		return
	}

	for i, r := range waiters {
		if r == receiptToRemove {
			s.receipts[receiptToRemove.WorkflowID] = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}

	if len(s.receipts[receiptToRemove.WorkflowID]) == 0 {
		delete(s.receipts, receiptToRemove.WorkflowID)
	}
}

// resolveReceipts resolves all waiting receipts for a completed workflow. Also
// resolves receipts in the database for cross-node coordination.
//
// Takes ctx (context.Context) which carries tracing values; cancellation is
// detached internally so resolution completes independently.
// Takes workflowID (string) which identifies the workflow whose receipts to
// resolve.
// Takes err (error) which is passed to each receipt to indicate success or
// failure.
//
// Safe for concurrent use. Acquires receiptsMutex to retrieve and remove
// waiters, then releases the lock before resolving receipts.
func (s *orchestratorService) resolveReceipts(ctx context.Context, workflowID string, err error) {
	ctx, l := logger_domain.From(ctx, log)
	s.receiptsMutex.Lock()
	waiters := s.receipts[workflowID]
	delete(s.receipts, workflowID)
	s.receiptsMutex.Unlock()

	if len(waiters) > 0 {
		l.Trace("Resolving workflow receipts",
			logger_domain.String(attributeKeyWorkflowID, workflowID),
			logger_domain.Int("receiptCount", len(waiters)),
			logger_domain.Bool("hasError", err != nil))

		for _, receipt := range waiters {
			receipt.resolve(err)
		}
	}

	if s.taskStore != nil {
		ctx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), 5*time.Second,
			errors.New("workflow receipt resolution exceeded 5s timeout"))
		errMessage := ""
		if err != nil {
			errMessage = err.Error()
		}
		if _, dbErr := s.taskStore.ResolveWorkflowReceipts(ctx, workflowID, errMessage); dbErr != nil {
			l.Warn("Failed to resolve workflow receipts in database",
				logger_domain.Error(dbErr),
				logger_domain.String(attributeKeyWorkflowID, workflowID))
		}
		cancel()
	}
}

// subscribeToCompletionEvents subscribes to task.completed events to resolve
// workflow receipts. When all tasks in a workflow complete, the associated
// receipts are resolved.
func (s *orchestratorService) subscribeToCompletionEvents() {
	defer s.wg.Done()
	defer goroutine.RecoverPanic(s.runCtx, "orchestrator.subscribeToCompletionEvents")

	ctx, scl := logger_domain.From(s.runCtx, log)
	ctx, span, l := scl.Span(ctx, "OrchestratorService.subscribeToCompletionEvents")
	defer span.End()

	if s.eventBus == nil {
		l.Internal("EventBus not configured, workflow receipt resolution unavailable")
		span.SetStatus(codes.Ok, "No event bus configured")
		return
	}

	l.Internal("Subscribing to task.completed events via handler")
	err := s.eventBus.SubscribeWithHandler(ctx, TopicTaskCompleted,
		func(ctx context.Context, event Event) error {
			s.handleCompletionEvent(ctx, event)
			return nil
		})

	if err != nil {
		l.Error("Failed to subscribe to task.completed events", logger_domain.Error(err))
		span.SetStatus(codes.Error, "Subscribe failed")
		return
	}

	l.Internal("Subscribed to task.completed events")
	span.SetStatus(codes.Ok, "Subscribed")

	<-ctx.Done()
	l.Internal("Completion event subscription shutting down")
}

// handleCompletionEvent processes a task.completed event to resolve workflow
// receipts.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes event (Event) which contains the completion payload with workflow ID
// and status.
func (s *orchestratorService) handleCompletionEvent(ctx context.Context, event Event) {
	workflowID := getPayloadString(event.Payload, "workflowId")
	status := getPayloadString(event.Payload, "status")
	errorMessage := getPayloadString(event.Payload, "error")

	if workflowID == "" {
		return
	}

	var latestErr error
	if status == "failure" && errorMessage != "" {
		latestErr = errors.New(errorMessage)
	}

	s.checkAndResolveWorkflow(ctx, workflowID, latestErr)
}
