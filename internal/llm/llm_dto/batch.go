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

package llm_dto

import "time"

// BatchStatus represents the state of a batch processing job.
type BatchStatus string

const (
	// BatchStatusPending indicates the batch is waiting to be processed.
	BatchStatusPending BatchStatus = "pending"

	// BatchStatusProcessing indicates the batch is being processed.
	BatchStatusProcessing BatchStatus = "processing"

	// BatchStatusCompleted means all requests in the batch have finished.
	BatchStatusCompleted BatchStatus = "completed"

	// BatchStatusFailed indicates the batch did not finish successfully.
	BatchStatusFailed BatchStatus = "failed"

	// BatchStatusCancelled indicates the batch was stopped before it finished.
	BatchStatusCancelled BatchStatus = "cancelled"
)

// BatchRequest contains parameters for a batch processing job.
type BatchRequest struct {
	// Metadata holds extra data for tracking and logging.
	Metadata map[string]string

	// Requests holds the list of completion requests to process in the batch.
	Requests []CompletionRequest

	// CompletionWindow is the time window for batch completion.
	// Longer windows typically result in lower costs.
	CompletionWindow time.Duration
}

// BatchResponse holds the result of a batch processing job.
type BatchResponse struct {
	// CreatedAt is the time when the batch was created.
	CreatedAt time.Time

	// CompletedAt is when the batch finished; nil if still in progress.
	CompletedAt *time.Time

	// Errors maps request index to error for requests that failed.
	Errors map[int]error

	// ID is a unique identifier for this batch job.
	ID string

	// Status is the current state of the batch job.
	Status BatchStatus

	// Results holds the completion responses for each request.
	// Only filled when Status is BatchStatusCompleted.
	Results []CompletionResponse

	// RequestCounts holds the number of requests grouped by their status.
	RequestCounts BatchRequestCounts
}

// BatchRequestCounts contains counts of requests by status.
type BatchRequestCounts struct {
	// Total is the number of requests in the batch.
	Total int

	// Completed is the number of requests that finished successfully.
	Completed int

	// Failed is the number of requests that did not complete successfully.
	Failed int

	// Pending is the number of requests that have not yet been processed.
	Pending int
}

// IsComplete reports whether the batch has finished processing.
//
// Returns bool which is true if the batch is complete (success or failure).
func (r *BatchResponse) IsComplete() bool {
	return r.Status == BatchStatusCompleted ||
		r.Status == BatchStatusFailed ||
		r.Status == BatchStatusCancelled
}

// Progress returns the completion progress as a value from 0.0 to 1.0.
//
// Returns float64 which is the progress where 0.0 means no requests are done
// and 1.0 means all requests are done. Both completed and failed requests
// count towards progress.
func (r *BatchResponse) Progress() float64 {
	if r.RequestCounts.Total == 0 {
		return 0
	}
	completed := r.RequestCounts.Completed + r.RequestCounts.Failed
	return float64(completed) / float64(r.RequestCounts.Total)
}

// GetResult returns the completion response for a specific request index.
//
// Takes index (int) which is the request index.
//
// Returns *CompletionResponse if found, nil otherwise.
func (r *BatchResponse) GetResult(index int) *CompletionResponse {
	if index < 0 || index >= len(r.Results) {
		return nil
	}
	return &r.Results[index]
}

// GetError returns the error for a specific request index.
//
// Takes index (int) which is the request index.
//
// Returns error if the request failed, nil otherwise.
func (r *BatchResponse) GetError(index int) error {
	if r.Errors == nil {
		return nil
	}
	return r.Errors[index]
}
