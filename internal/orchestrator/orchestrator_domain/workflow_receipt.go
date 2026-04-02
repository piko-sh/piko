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
	"sync"
)

// WorkflowReceipt provides a handle to wait for workflow completion.
// It is returned when dispatching or scheduling tasks and can be used
// to block until all tasks in the workflow have completed.
type WorkflowReceipt struct {
	// doneCh signals workflow completion; receives an error or nil then closes.
	doneCh chan error

	// WorkflowID is the unique identifier used to group and track related receipts.
	WorkflowID string

	// once restricts resolution to a single call.
	once sync.Once
}

// Done returns a channel that receives an error (or nil) when the workflow
// completes.
//
// Returns <-chan error which yields a single value when the workflow finishes.
func (r *WorkflowReceipt) Done() <-chan error {
	return r.doneCh
}

// Wait blocks until the workflow completes or the context is cancelled.
//
// Returns error when the workflow fails or the context is cancelled.
func (r *WorkflowReceipt) Wait(ctx context.Context) error {
	select {
	case err := <-r.doneCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// resolve completes the workflow receipt with the given error.
//
// Takes err (error) which is the error to send, or nil for success.
func (r *WorkflowReceipt) resolve(err error) {
	r.once.Do(func() {
		if err != nil {
			r.doneCh <- err
		}
		close(r.doneCh)
	})
}

// newWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// Takes workflowID (string) which identifies the workflow to track.
//
// Returns *WorkflowReceipt which provides a channel for waiting until the
// workflow finishes.
func newWorkflowReceipt(workflowID string) *WorkflowReceipt {
	return &WorkflowReceipt{
		doneCh:     make(chan error, 1),
		WorkflowID: workflowID,
		once:       sync.Once{},
	}
}
