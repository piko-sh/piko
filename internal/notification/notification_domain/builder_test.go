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

package notification_domain

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/notification/notification_dto"
)

func newTestService(providers map[string]*mockProvider) Service {
	service := NewService()
	for name, provider := range providers {
		_ = service.RegisterProvider(name, provider)
	}
	return service
}

func TestBuilder_Title(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Title("hello")
	if b.params.Content.Title != "hello" {
		t.Errorf("expected title %q, got %q", "hello", b.params.Content.Title)
	}
}

func TestBuilder_Message(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Message("world")
	if b.params.Content.Message != "world" {
		t.Errorf("expected message %q, got %q", "world", b.params.Content.Message)
	}
}

func TestBuilder_Field(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Field("k1", "v1").Field("k2", "v2")
	if len(b.params.Content.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(b.params.Content.Fields))
	}
	if b.params.Content.Fields["k1"] != "v1" {
		t.Errorf("expected k1=v1, got %q", b.params.Content.Fields["k1"])
	}
}

func TestBuilder_Fields(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Fields(map[string]string{"a": "1", "b": "2"})
	if len(b.params.Content.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(b.params.Content.Fields))
	}
}

func TestBuilder_Image(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Image("https://example.com/img.png")
	if b.params.Content.ImageURL != "https://example.com/img.png" {
		t.Errorf("unexpected image URL: %q", b.params.Content.ImageURL)
	}
}

func TestBuilder_Priority(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Priority(notification_dto.PriorityHigh)
	if b.params.Context.Priority != notification_dto.PriorityHigh {
		t.Errorf("expected PriorityHigh, got %v", b.params.Context.Priority)
	}
}

func TestBuilder_Source(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Source("my-source")
	if b.params.Context.Source != "my-source" {
		t.Errorf("expected %q, got %q", "my-source", b.params.Context.Source)
	}
}

func TestBuilder_Environment(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Environment("production")
	if b.params.Context.Environment != "production" {
		t.Errorf("expected %q, got %q", "production", b.params.Context.Environment)
	}
}

func TestBuilder_Service(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Service("my-service")
	if b.params.Context.Service != "my-service" {
		t.Errorf("expected %q, got %q", "my-service", b.params.Context.Service)
	}
}

func TestBuilder_TraceID(t *testing.T) {
	service := NewService()
	b := service.NewNotification().TraceID("trace-abc-123")
	if b.params.Context.TraceID != "trace-abc-123" {
		t.Errorf("expected %q, got %q", "trace-abc-123", b.params.Context.TraceID)
	}
}

func TestBuilder_Type(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Type(notification_dto.NotificationTypeRich)
	if b.params.Content.Type != notification_dto.NotificationTypeRich {
		t.Errorf("expected NotificationTypeRich, got %v", b.params.Content.Type)
	}
}

func TestBuilder_Provider(t *testing.T) {
	service := NewService()
	b := service.NewNotification().Provider("slack")
	if len(b.params.Providers) != 1 || b.params.Providers[0] != "slack" {
		t.Errorf("expected [slack], got %v", b.params.Providers)
	}
}

func TestBuilder_ToProviders(t *testing.T) {
	service := NewService()
	b := service.NewNotification().ToProviders("slack", "discord")
	if len(b.params.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(b.params.Providers))
	}
}

func TestBuilder_ProviderOption(t *testing.T) {
	service := NewService()
	b := service.NewNotification().ProviderOption("channel", "#alerts")
	if b.params.ProviderOptions["channel"] != "#alerts" {
		t.Errorf("expected #alerts, got %v", b.params.ProviderOptions["channel"])
	}
}

func TestBuilder_ProviderOptions(t *testing.T) {
	service := NewService()
	b := service.NewNotification().ProviderOptions(map[string]any{"a": 1, "b": "two"})
	if len(b.params.ProviderOptions) != 2 {
		t.Errorf("expected 2 options, got %d", len(b.params.ProviderOptions))
	}
}

func TestBuilder_Do_Success(t *testing.T) {
	service := newTestService(map[string]*mockProvider{
		"test": {},
	})

	ctx := context.Background()
	err := service.NewNotification().
		Title("Test").
		Message("Hello").
		Provider("test").
		Do(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuilder_Do_MultiCast(t *testing.T) {
	sendCount := 0
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			sendCount++
			return nil
		},
	}
	service := newTestService(map[string]*mockProvider{
		"a": provider,
		"b": provider,
	})

	ctx := context.Background()
	err := service.NewNotification().
		Title("Multi").
		ToProviders("a", "b").
		Do(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sendCount != 2 {
		t.Errorf("expected 2 sends, got %d", sendCount)
	}
}

func TestBuilder_Do_ValidationError(t *testing.T) {
	service := newTestService(map[string]*mockProvider{"test": {}})
	ctx := context.Background()

	err := service.NewNotification().Provider("test").Do(ctx)
	if err == nil {
		t.Error("expected validation error for empty title and message")
	}
}

func TestBuilder_Do_ProviderNotFound(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	err := service.NewNotification().
		Title("Test").
		Provider("nonexistent").
		Do(ctx)
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestBuilder_Do_SendError(t *testing.T) {
	service := newTestService(map[string]*mockProvider{
		"bad": {
			SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
				return errors.New("send failed")
			},
		},
	})

	ctx := context.Background()
	err := service.NewNotification().
		Title("Test").
		Provider("bad").
		Do(ctx)
	if err == nil {
		t.Error("expected error from send failure")
	}
}

func TestBuilder_Do_SetsTimestamp(t *testing.T) {
	service := newTestService(map[string]*mockProvider{"test": {}})
	ctx := context.Background()

	b := service.NewNotification().Title("Test").Provider("test")
	if !b.params.Context.Timestamp.IsZero() {
		t.Error("expected zero timestamp before Do")
	}

	_ = b.Do(ctx)
	if b.params.Context.Timestamp.IsZero() {
		t.Error("expected timestamp to be set after Do")
	}
}

func TestBuilder_Chaining(t *testing.T) {
	service := newTestService(map[string]*mockProvider{"test": {}})

	b := service.NewNotification().
		Title("t").
		Message("m").
		Field("k", "v").
		Fields(map[string]string{"a": "b"}).
		Image("https://img.com/test.png").
		Priority(notification_dto.PriorityCritical).
		Source("src").
		Environment("prod").
		Service("service").
		TraceID("trace").
		Type(notification_dto.NotificationTypeRich).
		Provider("test").
		ProviderOption("opt", "val").
		ProviderOptions(map[string]any{"x": 1})

	if b.params.Content.Title != "t" {
		t.Error("chaining broke Title")
	}
	if b.params.Content.Message != "m" {
		t.Error("chaining broke Message")
	}
	if b.params.Content.ImageURL != "https://img.com/test.png" {
		t.Error("chaining broke Image")
	}
	if b.params.Context.Priority != notification_dto.PriorityCritical {
		t.Error("chaining broke Priority")
	}
}
