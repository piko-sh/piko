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

package notification_test

import (
	"context"
	"testing"
	"time"

	"piko.sh/piko/internal/notification/notification_adapters/driver_providers"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

func TestNotificationServiceBasic(t *testing.T) {

	service := notification_domain.NewService()

	provider := driver_providers.NewStdoutProvider()
	if err := service.RegisterProvider("stdout", provider); err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	if err := service.SetDefaultProvider("stdout"); err != nil {
		t.Fatalf("Failed to set default provider: %v", err)
	}

	ctx := context.Background()
	err := service.NewNotification().
		Title("Test Notification").
		Message("This is a test message").
		Priority(notification_dto.PriorityNormal).
		Source("test").
		Field("test_key", "test_value").
		Do(ctx)

	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	t.Log("Notification sent successfully")
}

func TestNotificationServiceMultiCast(t *testing.T) {

	service := notification_domain.NewService()

	provider1 := driver_providers.NewStdoutProvider()
	provider2 := driver_providers.NewStdoutProvider()

	if err := service.RegisterProvider("stdout1", provider1); err != nil {
		t.Fatalf("Failed to register provider 1: %v", err)
	}

	if err := service.RegisterProvider("stdout2", provider2); err != nil {
		t.Fatalf("Failed to register provider 2: %v", err)
	}

	ctx := context.Background()
	err := service.NewNotification().
		Title("Multi-Cast Test").
		Message("This should go to two providers").
		ToProviders("stdout1", "stdout2").
		Do(ctx)

	if err != nil {
		t.Fatalf("Failed to send multi-cast notification: %v", err)
	}

	t.Log("Multi-cast notification sent successfully")
}

func TestNotificationServiceProviderCapabilities(t *testing.T) {
	provider := driver_providers.NewStdoutProvider()
	caps := provider.GetCapabilities()

	if caps.SupportsBulkSending != true {
		t.Errorf("Expected stdout to support bulk sending")
	}

	if caps.MaxMessageLength != 0 {
		t.Errorf("Expected unlimited message length, got %d", caps.MaxMessageLength)
	}
}

func TestNotificationServiceBuilder(t *testing.T) {
	service := notification_domain.NewService()
	provider := driver_providers.NewStdoutProvider()

	if err := service.RegisterProvider("test", provider); err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	ctx := context.Background()

	err := service.NewNotification().
		Title("Builder Test").
		Message("Testing all builder methods").
		Priority(notification_dto.PriorityHigh).
		Source("unit-test").
		Environment("test").
		Service("notification-service").
		TraceID("test-trace-123").
		Field("key1", "value1").
		Field("key2", "value2").
		Provider("test").
		Do(ctx)

	if err != nil {
		t.Fatalf("Failed to send notification with builder: %v", err)
	}

	t.Log("Builder test passed")
}

func TestNotificationServiceClosesCleanly(t *testing.T) {
	service := notification_domain.NewService()
	provider := driver_providers.NewStdoutProvider()

	if err := service.RegisterProvider("test", provider); err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	ctx := context.Background()

	if err := service.Close(ctx); err != nil {
		t.Fatalf("Failed to close service: %v", err)
	}

	t.Log("Service closed cleanly")
}

func TestNotificationPriorityMapping(t *testing.T) {
	tests := []struct {
		expected string
		priority notification_dto.NotificationPriority
	}{
		{expected: "low", priority: notification_dto.PriorityLow},
		{expected: "normal", priority: notification_dto.PriorityNormal},
		{expected: "high", priority: notification_dto.PriorityHigh},
		{expected: "critical", priority: notification_dto.PriorityCritical},
	}

	for _, tt := range tests {
		if tt.priority.String() != tt.expected {
			t.Errorf("Priority.String() = %s, want %s", tt.priority.String(), tt.expected)
		}
	}
}

func TestNotificationTypeMapping(t *testing.T) {
	tests := []struct {
		expected string
		notType  notification_dto.NotificationType
	}{
		{expected: "plain", notType: notification_dto.NotificationTypePlain},
		{expected: "rich", notType: notification_dto.NotificationTypeRich},
		{expected: "templated", notType: notification_dto.NotificationTypeTemplated},
	}

	for _, tt := range tests {
		if tt.notType.String() != tt.expected {
			t.Errorf("NotificationType.String() = %s, want %s", tt.notType.String(), tt.expected)
		}
	}
}

func TestNotificationRetryConfig(t *testing.T) {
	config := notification_domain.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	baseTime := time.Now()

	nextRetry := config.CalculateNextRetry(1, baseTime)
	if nextRetry.Before(baseTime) {
		t.Error("Next retry should be in the future")
	}

	if !config.ShouldRetry(1) {
		t.Error("Should retry attempt 1")
	}

	if !config.ShouldRetry(3) {
		t.Error("Should retry attempt 3 (max retries)")
	}

	if config.ShouldRetry(4) {
		t.Error("Should not retry attempt 4 (exceeds max)")
	}
}
