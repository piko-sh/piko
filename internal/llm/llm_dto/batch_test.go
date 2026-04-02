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

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchResponse_IsComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   BatchStatus
		expected bool
	}{
		{name: "pending", status: BatchStatusPending, expected: false},
		{name: "processing", status: BatchStatusProcessing, expected: false},
		{name: "completed", status: BatchStatusCompleted, expected: true},
		{name: "failed", status: BatchStatusFailed, expected: true},
		{name: "cancelled", status: BatchStatusCancelled, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := &BatchResponse{Status: tt.status}
			assert.Equal(t, tt.expected, response.IsComplete())
		})
	}
}

func TestBatchResponse_Progress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		completed int
		failed    int
		total     int
		expected  float64
	}{
		{name: "zero total", completed: 0, failed: 0, total: 0, expected: 0.0},
		{name: "none done", completed: 0, failed: 0, total: 10, expected: 0.0},
		{name: "partial completed", completed: 5, failed: 0, total: 10, expected: 0.5},
		{name: "partial with failures", completed: 3, failed: 2, total: 10, expected: 0.5},
		{name: "all completed", completed: 10, failed: 0, total: 10, expected: 1.0},
		{name: "all failed", completed: 0, failed: 10, total: 10, expected: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := &BatchResponse{
				RequestCounts: BatchRequestCounts{
					Total:     tt.total,
					Completed: tt.completed,
					Failed:    tt.failed,
				},
			}
			assert.InDelta(t, tt.expected, response.Progress(), 0.001)
		})
	}
}

func TestBatchResponse_GetResult(t *testing.T) {
	t.Parallel()

	response := &BatchResponse{
		Results: []CompletionResponse{
			{ID: "response-0", Model: "gpt-5"},
			{ID: "response-1", Model: "gpt-5"},
		},
	}

	t.Run("valid index", func(t *testing.T) {
		t.Parallel()

		result := response.GetResult(0)
		assert.NotNil(t, result)
		assert.Equal(t, "response-0", result.ID)
	})

	t.Run("second index", func(t *testing.T) {
		t.Parallel()

		result := response.GetResult(1)
		assert.NotNil(t, result)
		assert.Equal(t, "response-1", result.ID)
	})

	t.Run("negative index", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, response.GetResult(-1))
	})

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, response.GetResult(5))
	})
}

func TestBatchResponse_GetError(t *testing.T) {
	t.Parallel()

	t.Run("with error map", func(t *testing.T) {
		t.Parallel()

		testErr := errors.New("request failed")
		response := &BatchResponse{
			Errors: map[int]error{
				2: testErr,
			},
		}
		assert.Equal(t, testErr, response.GetError(2))
		assert.Nil(t, response.GetError(0))
	})

	t.Run("nil error map", func(t *testing.T) {
		t.Parallel()

		response := &BatchResponse{}
		assert.Nil(t, response.GetError(0))
	})
}
