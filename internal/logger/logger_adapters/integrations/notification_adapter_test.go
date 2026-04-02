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

package integrations

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

type mockNotificationService struct {
	sendBulkFunc func(ctx context.Context, notifications []*notification_dto.SendParams) error
	calls        []*notification_dto.SendParams
}

var _ notification_domain.Service = (*mockNotificationService)(nil)

func (m *mockNotificationService) NewNotification() *notification_domain.NotificationBuilder {
	return nil
}

func (m *mockNotificationService) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	m.calls = append(m.calls, notifications...)
	if m.sendBulkFunc != nil {
		return m.sendBulkFunc(ctx, notifications)
	}
	return nil
}

func (*mockNotificationService) SendBulkWithProvider(_ context.Context, _ string, _ []*notification_dto.SendParams) error {
	return nil
}

func (*mockNotificationService) SendToProviders(_ context.Context, _ *notification_dto.SendParams, _ []string) error {
	return nil
}

func (*mockNotificationService) RegisterProvider(_ string, _ notification_domain.NotificationProviderPort) error {
	return nil
}

func (*mockNotificationService) SetDefaultProvider(_ string) error {
	return nil
}

func (*mockNotificationService) GetProviders() []string {
	return nil
}

func (*mockNotificationService) HasProvider(_ string) bool {
	return false
}

func (*mockNotificationService) RegisterDispatcher(_ notification_domain.NotificationDispatcherPort) error {
	return nil
}

func (*mockNotificationService) FlushDispatcher(_ context.Context) error {
	return nil
}

func (*mockNotificationService) Close(_ context.Context) error {
	return nil
}

func TestMapLogLevelToPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    slog.Level
		expected notification_dto.NotificationPriority
	}{
		{
			name:     "error level maps to critical priority",
			level:    slog.LevelError,
			expected: notification_dto.PriorityCritical,
		},
		{
			name:     "warn level maps to high priority",
			level:    slog.LevelWarn,
			expected: notification_dto.PriorityHigh,
		},
		{
			name:     "info level maps to normal priority",
			level:    slog.LevelInfo,
			expected: notification_dto.PriorityNormal,
		},
		{
			name:     "debug level maps to low priority",
			level:    slog.LevelDebug,
			expected: notification_dto.PriorityLow,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := mapLogLevelToPriority(testCase.level)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestSendGroupedErrors(t *testing.T) {
	t.Run("empty batch returns nil without calling service", func(t *testing.T) {
		t.Parallel()

		service := &mockNotificationService{}
		adapter := NewNotificationServiceAdapter(service)

		err := adapter.SendGroupedErrors(context.Background(), map[string]*logger_domain.GroupedError{})
		require.NoError(t, err)
		assert.Empty(t, service.calls)
	})

	t.Run("single error calls SendBulk with one notification", func(t *testing.T) {
		service := &mockNotificationService{}
		adapter := NewNotificationServiceAdapter(service)

		now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		record := slog.NewRecord(now, slog.LevelError, "something broke", 0)
		batch := map[string]*logger_domain.GroupedError{
			"key1": {
				LogRecord:  record,
				FirstSeen:  now,
				LastSeen:   now,
				SourceFile: "handler.go",
				SourceLine: 42,
				Count:      1,
			},
		}

		t.Setenv("ENVIRONMENT", "test")

		err := adapter.SendGroupedErrors(context.Background(), batch)
		require.NoError(t, err)
		require.Len(t, service.calls, 1)

		notification := service.calls[0]
		assert.Equal(t, "logger", notification.Context.Source)
		assert.Equal(t, "test", notification.Context.Environment)
		assert.Equal(t, notification_dto.PriorityCritical, notification.Context.Priority)
		assert.Equal(t, "something broke", notification.Content.Message)
		assert.Equal(t, notification_dto.NotificationTypeRich, notification.Content.Type)
	})

	t.Run("service error propagates", func(t *testing.T) {
		t.Parallel()

		expectedError := errors.New("send failed")
		service := &mockNotificationService{
			sendBulkFunc: func(_ context.Context, _ []*notification_dto.SendParams) error {
				return expectedError
			},
		}
		adapter := NewNotificationServiceAdapter(service)

		now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		record := slog.NewRecord(now, slog.LevelError, "failure", 0)
		batch := map[string]*logger_domain.GroupedError{
			"key1": {
				LogRecord: record,
				FirstSeen: now,
				LastSeen:  now,
				Count:     1,
			},
		}

		err := adapter.SendGroupedErrors(context.Background(), batch)
		require.ErrorIs(t, err, expectedError)
	})
}

func TestConvertGroupedErrorToNotification(t *testing.T) {
	t.Run("count one omits occurrences and last_seen fields", func(t *testing.T) {
		adapter := &NotificationServiceAdapter{}
		now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		record := slog.NewRecord(now, slog.LevelError, "single error", 0)

		grouped := &logger_domain.GroupedError{
			LogRecord:  record,
			FirstSeen:  now,
			LastSeen:   now,
			SourceFile: "",
			SourceLine: 0,
			Count:      1,
		}

		t.Setenv("ENVIRONMENT", "production")

		result := adapter.convertGroupedErrorToNotification(grouped)

		assert.NotContains(t, result.Content.Fields, "occurrences")
		assert.NotContains(t, result.Content.Fields, "last_seen")
		assert.Equal(t, now.Format(time.RFC3339), result.Content.Fields["first_seen"])
	})

	t.Run("count greater than one includes occurrences and last_seen", func(t *testing.T) {
		adapter := &NotificationServiceAdapter{}
		firstSeen := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		lastSeen := time.Date(2026, 3, 27, 10, 5, 0, 0, time.UTC)
		record := slog.NewRecord(firstSeen, slog.LevelError, "repeated error", 0)

		grouped := &logger_domain.GroupedError{
			LogRecord:  record,
			FirstSeen:  firstSeen,
			LastSeen:   lastSeen,
			SourceFile: "handler.go",
			SourceLine: 42,
			Count:      5,
		}

		t.Setenv("ENVIRONMENT", "staging")

		result := adapter.convertGroupedErrorToNotification(grouped)

		assert.Equal(t, "5", result.Content.Fields["occurrences"])
		assert.Equal(t, lastSeen.Format(time.RFC3339), result.Content.Fields["last_seen"])
		assert.Equal(t, firstSeen.Format(time.RFC3339), result.Content.Fields["first_seen"])
	})

	t.Run("source file set includes source field", func(t *testing.T) {
		adapter := &NotificationServiceAdapter{}
		now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		record := slog.NewRecord(now, slog.LevelError, "located error", 0)

		grouped := &logger_domain.GroupedError{
			LogRecord:  record,
			FirstSeen:  now,
			LastSeen:   now,
			SourceFile: "server.go",
			SourceLine: 99,
			Count:      1,
		}

		t.Setenv("ENVIRONMENT", "dev")

		result := adapter.convertGroupedErrorToNotification(grouped)

		assert.Equal(t, "server.go:99", result.Content.Fields["source"])
	})

	t.Run("empty source file omits source field", func(t *testing.T) {
		adapter := &NotificationServiceAdapter{}
		now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		record := slog.NewRecord(now, slog.LevelError, "no source", 0)

		grouped := &logger_domain.GroupedError{
			LogRecord:  record,
			FirstSeen:  now,
			LastSeen:   now,
			SourceFile: "",
			SourceLine: 0,
			Count:      1,
		}

		t.Setenv("ENVIRONMENT", "")

		result := adapter.convertGroupedErrorToNotification(grouped)

		assert.NotContains(t, result.Content.Fields, "source")
		assert.Equal(t, "unknown", result.Context.Environment)
	})
}
