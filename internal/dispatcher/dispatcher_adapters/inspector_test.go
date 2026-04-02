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

package dispatcher_adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

func TestGetDispatcherSummaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		emailDispatcher        email_domain.EmailDispatcherPort
		notificationDispatcher notification_domain.NotificationDispatcherPort
		expectedCount          int
		expectedTypes          []string
	}{
		{
			name:                   "both dispatchers nil returns empty slice",
			emailDispatcher:        nil,
			notificationDispatcher: nil,
			expectedCount:          0,
			expectedTypes:          nil,
		},
		{
			name: "email only returns one summary",
			emailDispatcher: &mockEmailDispatcher{
				GetProcessingStatsFunc: func(context.Context) (email_domain.DispatcherStats, error) {
					return email_domain.DispatcherStats{
						QueuedEmails:    5,
						RetryQueueSize:  2,
						DeadLetterCount: 1,
						TotalProcessed:  100,
						TotalSuccessful: 90,
						TotalFailed:     10,
						TotalRetries:    15,
						Uptime:          3 * time.Hour,
					}, nil
				},
			},
			notificationDispatcher: nil,
			expectedCount:          1,
			expectedTypes:          []string{"email"},
		},
		{
			name:            "notification only returns one summary",
			emailDispatcher: nil,
			notificationDispatcher: &mockNotificationDispatcher{
				GetProcessingStatsFunc: func(context.Context) (notification_domain.DispatcherStats, error) {
					return notification_domain.DispatcherStats{
						QueuedNotifications: 3,
						RetryQueueSize:      1,
						DeadLetterCount:     0,
						TotalProcessed:      50,
						TotalSuccessful:     48,
						TotalFailed:         2,
						TotalRetries:        4,
						Uptime:              time.Hour,
					}, nil
				},
			},
			expectedCount: 1,
			expectedTypes: []string{"notification"},
		},
		{
			name: "both present returns two summaries",
			emailDispatcher: &mockEmailDispatcher{
				GetProcessingStatsFunc: func(context.Context) (email_domain.DispatcherStats, error) {
					return email_domain.DispatcherStats{QueuedEmails: 1}, nil
				},
			},
			notificationDispatcher: &mockNotificationDispatcher{
				GetProcessingStatsFunc: func(context.Context) (notification_domain.DispatcherStats, error) {
					return notification_domain.DispatcherStats{QueuedNotifications: 2}, nil
				},
			},
			expectedCount: 2,
			expectedTypes: []string{"email", "notification"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inspector := NewInspector(testCase.emailDispatcher, testCase.notificationDispatcher)

			summaries, err := inspector.GetDispatcherSummaries(testContext())
			require.NoError(t, err)
			assert.Len(t, summaries, testCase.expectedCount)

			for index, expectedType := range testCase.expectedTypes {
				assert.Equal(t, expectedType, summaries[index].Type)
			}
		})
	}
}

func TestGetDispatcherSummariesEmailFieldMapping(t *testing.T) {
	t.Parallel()

	emailDispatcher := &mockEmailDispatcher{
		GetProcessingStatsFunc: func(context.Context) (email_domain.DispatcherStats, error) {
			return email_domain.DispatcherStats{
				QueuedEmails:    5,
				RetryQueueSize:  2,
				DeadLetterCount: 1,
				TotalProcessed:  100,
				TotalSuccessful: 90,
				TotalFailed:     10,
				TotalRetries:    15,
				Uptime:          3 * time.Hour,
			}, nil
		},
	}

	inspector := NewInspector(emailDispatcher, nil)

	summaries, err := inspector.GetDispatcherSummaries(testContext())
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	summary := summaries[0]
	assert.Equal(t, "email", summary.Type)
	assert.Equal(t, 5, summary.QueuedItems)
	assert.Equal(t, 2, summary.RetryQueueSize)
	assert.Equal(t, 1, summary.DeadLetterCount)
	assert.Equal(t, int64(100), summary.TotalProcessed)
	assert.Equal(t, int64(90), summary.TotalSuccessful)
	assert.Equal(t, int64(10), summary.TotalFailed)
	assert.Equal(t, int64(15), summary.TotalRetries)
	assert.Equal(t, 3*time.Hour, summary.Uptime)
}

func TestGetDispatcherSummariesNotificationFieldMapping(t *testing.T) {
	t.Parallel()

	notificationDispatcher := &mockNotificationDispatcher{
		GetProcessingStatsFunc: func(context.Context) (notification_domain.DispatcherStats, error) {
			return notification_domain.DispatcherStats{
				QueuedNotifications: 7,
				RetryQueueSize:      3,
				DeadLetterCount:     2,
				TotalProcessed:      200,
				TotalSuccessful:     195,
				TotalFailed:         5,
				TotalRetries:        8,
				Uptime:              5 * time.Hour,
			}, nil
		},
	}

	inspector := NewInspector(nil, notificationDispatcher)

	summaries, err := inspector.GetDispatcherSummaries(testContext())
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	summary := summaries[0]
	assert.Equal(t, "notification", summary.Type)
	assert.Equal(t, 7, summary.QueuedItems)
	assert.Equal(t, 3, summary.RetryQueueSize)
	assert.Equal(t, 2, summary.DeadLetterCount)
	assert.Equal(t, int64(200), summary.TotalProcessed)
	assert.Equal(t, int64(195), summary.TotalSuccessful)
	assert.Equal(t, int64(5), summary.TotalFailed)
	assert.Equal(t, int64(8), summary.TotalRetries)
	assert.Equal(t, 5*time.Hour, summary.Uptime)
}

func TestGetDispatcherSummariesEmailStatsError(t *testing.T) {
	t.Parallel()

	emailDispatcher := &mockEmailDispatcher{
		GetProcessingStatsFunc: func(context.Context) (email_domain.DispatcherStats, error) {
			return email_domain.DispatcherStats{}, errors.New("email stats unavailable")
		},
	}
	notificationDispatcher := &mockNotificationDispatcher{
		GetProcessingStatsFunc: func(context.Context) (notification_domain.DispatcherStats, error) {
			return notification_domain.DispatcherStats{
				QueuedNotifications: 4,
				TotalProcessed:      60,
			}, nil
		},
	}

	inspector := NewInspector(emailDispatcher, notificationDispatcher)

	summaries, err := inspector.GetDispatcherSummaries(testContext())
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "notification", summaries[0].Type)
	assert.Equal(t, 4, summaries[0].QueuedItems)
}

func TestGetDLQEntries(t *testing.T) {
	t.Parallel()

	firstAttempt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	lastAttempt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	addedAt := time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC)

	tests := []struct {
		name                   string
		emailDispatcher        email_domain.EmailDispatcherPort
		notificationDispatcher notification_domain.NotificationDispatcherPort
		dispatcherType         string
		limit                  int
		expectedError          string
		expectedCount          int
		validate               func(t *testing.T, entries []dispatcher_domain.DLQEntry)
	}{
		{
			name: "email type with entries",
			emailDispatcher: &mockEmailDispatcher{
				GetDeadLetterQueueFunc: func() email_domain.DeadLetterPort {
					return &mockEmailDLQ{
						GetFunc: func(_ context.Context, limit int) ([]*email_dto.DeadLetterEntry, error) {
							assert.Equal(t, 10, limit)
							return []*email_dto.DeadLetterEntry{
								{
									ID:                "email-dlq-1",
									OriginalError:     "connection refused",
									TotalAttempts:     3,
									FirstAttempt:      firstAttempt,
									LastAttempt:       lastAttempt,
									AddedToDeadLetter: addedAt,
								},
							}, nil
						},
					}
				},
			},
			dispatcherType: "email",
			limit:          10,
			expectedCount:  1,
			validate: func(t *testing.T, entries []dispatcher_domain.DLQEntry) {
				t.Helper()
				entry := entries[0]
				assert.Equal(t, "email-dlq-1", entry.ID)
				assert.Equal(t, "email", entry.Type)
				assert.Equal(t, "connection refused", entry.OriginalError)
				assert.Equal(t, 3, entry.TotalAttempts)
				assert.Equal(t, firstAttempt, entry.FirstAttempt)
				assert.Equal(t, lastAttempt, entry.LastAttempt)
				assert.Equal(t, addedAt, entry.AddedAt)
			},
		},
		{
			name:            "notification type with entries",
			emailDispatcher: nil,
			notificationDispatcher: &mockNotificationDispatcher{
				GetDeadLetterQueueFunc: func() notification_domain.DeadLetterPort {
					return &mockNotificationDLQ{
						GetFunc: func(_ context.Context, limit int) ([]*notification_dto.DeadLetterEntry, error) {
							assert.Equal(t, 5, limit)
							return []*notification_dto.DeadLetterEntry{
								{
									OriginalError: "webhook timeout",
									TotalAttempts: 5,
									FirstAttempt:  firstAttempt,
									LastAttempt:   lastAttempt,
									Providers:     []string{"slack"},
								},
							}, nil
						},
					}
				},
			},
			dispatcherType: "notification",
			limit:          5,
			expectedCount:  1,
			validate: func(t *testing.T, entries []dispatcher_domain.DLQEntry) {
				t.Helper()
				entry := entries[0]
				assert.Equal(t, "notification", entry.Type)
				assert.Equal(t, "webhook timeout", entry.OriginalError)
				assert.Equal(t, 5, entry.TotalAttempts)
				assert.Equal(t, firstAttempt, entry.FirstAttempt)
				assert.Equal(t, lastAttempt, entry.LastAttempt)
			},
		},
		{
			name:           "unknown type returns error",
			dispatcherType: "sms",
			expectedError:  "unknown dispatcher type",
		},
		{
			name:            "email type with nil dispatcher",
			emailDispatcher: nil,
			dispatcherType:  "email",
			expectedError:   errEmailDispatcherNotConfigured.Error(),
		},
		{
			name: "email type with nil DLQ returns nil",
			emailDispatcher: &mockEmailDispatcher{
				GetDeadLetterQueueFunc: func() email_domain.DeadLetterPort {
					return nil
				},
			},
			dispatcherType: "email",
			limit:          10,
			expectedCount:  0,
		},
		{
			name: "DLQ Get error propagates",
			emailDispatcher: &mockEmailDispatcher{
				GetDeadLetterQueueFunc: func() email_domain.DeadLetterPort {
					return &mockEmailDLQ{
						GetFunc: func(context.Context, int) ([]*email_dto.DeadLetterEntry, error) {
							return nil, errors.New("storage failure")
						},
					}
				},
			},
			dispatcherType: "email",
			limit:          10,
			expectedError:  "storage failure",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inspector := NewInspector(testCase.emailDispatcher, testCase.notificationDispatcher)

			entries, err := inspector.GetDLQEntries(testContext(), testCase.dispatcherType, testCase.limit)

			if testCase.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError)
				return
			}

			require.NoError(t, err)

			if testCase.expectedCount == 0 {
				assert.Empty(t, entries)
			} else {
				require.Len(t, entries, testCase.expectedCount)
			}

			if testCase.validate != nil {
				testCase.validate(t, entries)
			}
		})
	}
}

func TestGetDLQCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		emailDispatcher        email_domain.EmailDispatcherPort
		notificationDispatcher notification_domain.NotificationDispatcherPort
		dispatcherType         string
		expectedCount          int
		expectedError          string
	}{
		{
			name: "email type delegates to GetDeadLetterCount",
			emailDispatcher: &mockEmailDispatcher{
				GetDeadLetterCountFunc: func(context.Context) (int, error) {
					return 7, nil
				},
			},
			dispatcherType: "email",
			expectedCount:  7,
		},
		{
			name: "notification type delegates to GetDeadLetterCount",
			notificationDispatcher: &mockNotificationDispatcher{
				GetDeadLetterCountFunc: func(context.Context) (int, error) {
					return 3, nil
				},
			},
			dispatcherType: "notification",
			expectedCount:  3,
		},
		{
			name:            "email type nil dispatcher returns error",
			emailDispatcher: nil,
			dispatcherType:  "email",
			expectedError:   errEmailDispatcherNotConfigured.Error(),
		},
		{
			name:                   "notification type nil dispatcher returns error",
			notificationDispatcher: nil,
			dispatcherType:         "notification",
			expectedError:          errNotificationDispatcherNotConfigured.Error(),
		},
		{
			name:           "unknown type returns error",
			dispatcherType: "sms",
			expectedError:  "unknown dispatcher type",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inspector := NewInspector(testCase.emailDispatcher, testCase.notificationDispatcher)

			count, err := inspector.GetDLQCount(testContext(), testCase.dispatcherType)

			if testCase.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedCount, count)
		})
	}
}

func TestClearDLQ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		emailDispatcher        email_domain.EmailDispatcherPort
		notificationDispatcher notification_domain.NotificationDispatcherPort
		dispatcherType         string
		expectedError          string
	}{
		{
			name: "email type delegates to ClearDeadLetterQueue",
			emailDispatcher: &mockEmailDispatcher{
				ClearDeadLetterQueueFunc: func(context.Context) error {
					return nil
				},
			},
			dispatcherType: "email",
		},
		{
			name: "notification type delegates to ClearDeadLetterQueue",
			notificationDispatcher: &mockNotificationDispatcher{
				ClearDeadLetterQueueFunc: func(context.Context) error {
					return nil
				},
			},
			dispatcherType: "notification",
		},
		{
			name:            "email type nil dispatcher returns error",
			emailDispatcher: nil,
			dispatcherType:  "email",
			expectedError:   errEmailDispatcherNotConfigured.Error(),
		},
		{
			name:                   "notification type nil dispatcher returns error",
			notificationDispatcher: nil,
			dispatcherType:         "notification",
			expectedError:          errNotificationDispatcherNotConfigured.Error(),
		},
		{
			name:           "unknown type returns error",
			dispatcherType: "sms",
			expectedError:  "unknown dispatcher type",
		},
		{
			name: "email clear error propagates",
			emailDispatcher: &mockEmailDispatcher{
				ClearDeadLetterQueueFunc: func(context.Context) error {
					return errors.New("clear failed")
				},
			},
			dispatcherType: "email",
			expectedError:  "clear failed",
		},
		{
			name: "notification clear error propagates",
			notificationDispatcher: &mockNotificationDispatcher{
				ClearDeadLetterQueueFunc: func(context.Context) error {
					return errors.New("notification clear failed")
				},
			},
			dispatcherType: "notification",
			expectedError:  "notification clear failed",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inspector := NewInspector(testCase.emailDispatcher, testCase.notificationDispatcher)

			err := inspector.ClearDLQ(testContext(), testCase.dispatcherType)

			if testCase.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError)
				return
			}

			require.NoError(t, err)
		})
	}
}
