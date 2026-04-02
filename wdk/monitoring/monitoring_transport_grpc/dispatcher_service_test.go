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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockDispatcherInspector struct {
	summariesError  error
	dlqEntriesError error
	dlqCountError   error
	clearDLQError   error
	summariesReturn []dispatcher_domain.DispatcherSummary
	dlqEntries      []dispatcher_domain.DLQEntry
	dlqCount        int
}

func (m *mockDispatcherInspector) GetDispatcherSummaries(_ context.Context) ([]dispatcher_domain.DispatcherSummary, error) {
	return m.summariesReturn, m.summariesError
}

func (m *mockDispatcherInspector) GetDLQEntries(_ context.Context, _ string, _ int) ([]dispatcher_domain.DLQEntry, error) {
	return m.dlqEntries, m.dlqEntriesError
}

func (m *mockDispatcherInspector) GetDLQCount(_ context.Context, _ string) (int, error) {
	return m.dlqCount, m.dlqCountError
}

func (m *mockDispatcherInspector) ClearDLQ(_ context.Context, _ string) error {
	return m.clearDLQError
}

var _ dispatcher_domain.DispatcherInspector = (*mockDispatcherInspector)(nil)

func TestNewDispatcherInspectorService(t *testing.T) {
	t.Parallel()

	mock := &mockDispatcherInspector{}
	service := NewDispatcherInspectorService(mock)
	require.NotNil(t, service)
}

func TestDispatcherInspectorService_GetDispatcherSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector     *mockDispatcherInspector
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns summaries successfully",
			inspector: &mockDispatcherInspector{
				summariesReturn: []dispatcher_domain.DispatcherSummary{
					{
						Type:            "email",
						QueuedItems:     5,
						RetryQueueSize:  2,
						DeadLetterCount: 1,
						TotalProcessed:  100,
						TotalSuccessful: 95,
						TotalFailed:     3,
						TotalRetries:    10,
						Uptime:          2 * time.Hour,
					},
					{
						Type:            "notification",
						QueuedItems:     10,
						RetryQueueSize:  0,
						DeadLetterCount: 0,
						TotalProcessed:  50,
						TotalSuccessful: 50,
						TotalFailed:     0,
						TotalRetries:    0,
						Uptime:          1 * time.Hour,
					},
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "returns empty list",
			inspector: &mockDispatcherInspector{
				summariesReturn: []dispatcher_domain.DispatcherSummary{},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockDispatcherInspector{
				summariesError: errors.New("connection failed"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewDispatcherInspectorService(tc.inspector)
			response, err := service.GetDispatcherSummary(context.Background(), &pb.GetDispatcherSummaryRequest{})

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "getting dispatcher summaries")
				return
			}

			require.NoError(t, err)
			require.Len(t, response.Summaries, tc.expectedCount)

			if tc.expectedCount > 0 {
				s := response.Summaries[0]
				assert.Equal(t, "email", s.Type)
				assert.Equal(t, int32(5), s.QueuedItems)
				assert.Equal(t, int32(2), s.RetryQueueSize)
				assert.Equal(t, int32(1), s.DeadLetterCount)
				assert.Equal(t, int64(100), s.TotalProcessed)
				assert.Equal(t, int64(95), s.TotalSuccessful)
				assert.Equal(t, int64(3), s.TotalFailed)
				assert.Equal(t, int64(10), s.TotalRetries)
				assert.Equal(t, (2 * time.Hour).Milliseconds(), s.UptimeMs)
			}
		})
	}
}

func TestDispatcherInspectorService_ListDLQEntries(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		inspector     *mockDispatcherInspector
		request       *pb.ListDLQEntriesRequest
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns entries successfully",
			inspector: &mockDispatcherInspector{
				dlqEntries: []dispatcher_domain.DLQEntry{
					{
						ID:            "entry-1",
						Type:          "email",
						OriginalError: "SMTP timeout",
						TotalAttempts: 3,
						FirstAttempt:  now.Add(-1 * time.Hour),
						LastAttempt:   now.Add(-5 * time.Minute),
						AddedAt:       now,
					},
				},
			},
			request:       &pb.ListDLQEntriesRequest{DispatcherType: "email", Limit: 10},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "uses default limit when zero",
			inspector: &mockDispatcherInspector{
				dlqEntries: []dispatcher_domain.DLQEntry{
					{ID: "entry-1", FirstAttempt: now, LastAttempt: now, AddedAt: now},
				},
			},
			request:       &pb.ListDLQEntriesRequest{DispatcherType: "email", Limit: 0},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "uses default limit when negative",
			inspector: &mockDispatcherInspector{
				dlqEntries: []dispatcher_domain.DLQEntry{
					{ID: "entry-1", FirstAttempt: now, LastAttempt: now, AddedAt: now},
				},
			},
			request:       &pb.ListDLQEntriesRequest{DispatcherType: "email", Limit: -1},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockDispatcherInspector{
				dlqEntriesError: errors.New("not found"),
			},
			request:     &pb.ListDLQEntriesRequest{DispatcherType: "unknown"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewDispatcherInspectorService(tc.inspector)
			response, err := service.ListDLQEntries(context.Background(), tc.request)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "listing DLQ entries")
				return
			}

			require.NoError(t, err)
			require.Len(t, response.Entries, tc.expectedCount)

			if tc.expectedCount > 0 {
				e := response.Entries[0]
				assert.Equal(t, "entry-1", e.Id)
			}
		})
	}
}

func TestDispatcherInspectorService_ListDLQEntries_FieldMapping(t *testing.T) {
	t.Parallel()

	now := time.Now()
	firstAttempt := now.Add(-2 * time.Hour)
	lastAttempt := now.Add(-10 * time.Minute)

	inspector := &mockDispatcherInspector{
		dlqEntries: []dispatcher_domain.DLQEntry{
			{
				ID:            "dlq-entry-42",
				Type:          "notification",
				OriginalError: "webhook returned 500",
				TotalAttempts: 5,
				FirstAttempt:  firstAttempt,
				LastAttempt:   lastAttempt,
				AddedAt:       now,
			},
		},
	}

	service := NewDispatcherInspectorService(inspector)
	response, err := service.ListDLQEntries(context.Background(), &pb.ListDLQEntriesRequest{
		DispatcherType: "notification",
		Limit:          10,
	})

	require.NoError(t, err)
	require.Len(t, response.Entries, 1)

	entry := response.Entries[0]
	assert.Equal(t, "dlq-entry-42", entry.Id)
	assert.Equal(t, "notification", entry.Type)
	assert.Equal(t, "webhook returned 500", entry.OriginalError)
	assert.Equal(t, int32(5), entry.TotalAttempts)
	assert.Equal(t, firstAttempt.UnixMilli(), entry.FirstAttemptMs)
	assert.Equal(t, lastAttempt.UnixMilli(), entry.LastAttemptMs)
	assert.Equal(t, now.UnixMilli(), entry.AddedAtMs)
}

func TestDispatcherInspectorService_GetDLQCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   *mockDispatcherInspector
		name        string
		expected    int32
		expectError bool
	}{
		{
			name: "returns count",
			inspector: &mockDispatcherInspector{
				dlqCount: 7,
			},
			expected:    7,
			expectError: false,
		},
		{
			name: "returns zero count",
			inspector: &mockDispatcherInspector{
				dlqCount: 0,
			},
			expected:    0,
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockDispatcherInspector{
				dlqCountError: errors.New("store error"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewDispatcherInspectorService(tc.inspector)
			response, err := service.GetDLQCount(context.Background(), &pb.GetDLQCountRequest{
				DispatcherType: "email",
			})

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "getting DLQ count")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, response.Count)
			assert.Equal(t, "email", response.DispatcherType)
		})
	}
}

func TestDispatcherInspectorService_ClearDLQ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   *mockDispatcherInspector
		name        string
		expectError bool
	}{
		{
			name:        "clears successfully",
			inspector:   &mockDispatcherInspector{},
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockDispatcherInspector{
				clearDLQError: errors.New("permission denied"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewDispatcherInspectorService(tc.inspector)
			response, err := service.ClearDLQ(context.Background(), &pb.ClearDLQRequest{
				DispatcherType: "email",
			})

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "clearing DLQ")
				return
			}

			require.NoError(t, err)
			assert.True(t, response.Success)
			assert.Equal(t, "email", response.DispatcherType)
		})
	}
}
