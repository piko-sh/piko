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
	"time"

	"piko.sh/piko/internal/notification/notification_dto"
)

type mockProvider struct {
	SendFunc            func(ctx context.Context, params *notification_dto.SendParams) error
	SendBulkFunc        func(ctx context.Context, notifications []*notification_dto.SendParams) error
	SupportsBulkFunc    func() bool
	GetCapabilitiesFunc func() ProviderCapabilities
	CloseFunc           func(ctx context.Context) error
}

func (m *mockProvider) Send(ctx context.Context, params *notification_dto.SendParams) error {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, params)
	}
	return nil
}

func (m *mockProvider) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	if m.SendBulkFunc != nil {
		return m.SendBulkFunc(ctx, notifications)
	}
	return nil
}

func (m *mockProvider) SupportsBulkSending() bool {
	if m.SupportsBulkFunc != nil {
		return m.SupportsBulkFunc()
	}
	return false
}

func (m *mockProvider) GetCapabilities() ProviderCapabilities {
	if m.GetCapabilitiesFunc != nil {
		return m.GetCapabilitiesFunc()
	}
	return ProviderCapabilities{}
}

func (m *mockProvider) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

var _ NotificationProviderPort = (*mockProvider)(nil)

type mockDispatcher struct {
	StartFunc              func(ctx context.Context) error
	StopFunc               func(ctx context.Context) error
	QueueFunc              func(ctx context.Context, params *notification_dto.SendParams) error
	FlushFunc              func(ctx context.Context) error
	SetBatchSizeFunc       func(size int)
	SetFlushIntervalFunc   func(interval time.Duration)
	SetRetryConfigFunc     func(config RetryConfig)
	GetRetryConfigFunc     func() RetryConfig
	GetDeadLetterQueueFunc func() DeadLetterPort
	GetDeadLetterCountFunc func(ctx context.Context) (int, error)
	ClearDeadLetterFunc    func(ctx context.Context) error
	GetRetryQueueSizeFunc  func(ctx context.Context) (int, error)
	GetProcessingStatsFunc func(ctx context.Context) (DispatcherStats, error)
}

func (m *mockDispatcher) Start(ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return nil
}

func (m *mockDispatcher) Stop(ctx context.Context) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return nil
}

func (m *mockDispatcher) Queue(ctx context.Context, params *notification_dto.SendParams) error {
	if m.QueueFunc != nil {
		return m.QueueFunc(ctx, params)
	}
	return nil
}

func (m *mockDispatcher) Flush(ctx context.Context) error {
	if m.FlushFunc != nil {
		return m.FlushFunc(ctx)
	}
	return nil
}

func (m *mockDispatcher) SetBatchSize(size int) {
	if m.SetBatchSizeFunc != nil {
		m.SetBatchSizeFunc(size)
	}
}

func (m *mockDispatcher) SetFlushInterval(interval time.Duration) {
	if m.SetFlushIntervalFunc != nil {
		m.SetFlushIntervalFunc(interval)
	}
}

func (m *mockDispatcher) SetRetryConfig(config RetryConfig) {
	if m.SetRetryConfigFunc != nil {
		m.SetRetryConfigFunc(config)
	}
}

func (m *mockDispatcher) GetRetryConfig() RetryConfig {
	if m.GetRetryConfigFunc != nil {
		return m.GetRetryConfigFunc()
	}
	return RetryConfig{}
}

func (m *mockDispatcher) GetDeadLetterQueue() DeadLetterPort {
	if m.GetDeadLetterQueueFunc != nil {
		return m.GetDeadLetterQueueFunc()
	}
	return nil
}

func (m *mockDispatcher) GetDeadLetterCount(ctx context.Context) (int, error) {
	if m.GetDeadLetterCountFunc != nil {
		return m.GetDeadLetterCountFunc(ctx)
	}
	return 0, nil
}

func (m *mockDispatcher) ClearDeadLetterQueue(ctx context.Context) error {
	if m.ClearDeadLetterFunc != nil {
		return m.ClearDeadLetterFunc(ctx)
	}
	return nil
}

func (m *mockDispatcher) GetRetryQueueSize(ctx context.Context) (int, error) {
	if m.GetRetryQueueSizeFunc != nil {
		return m.GetRetryQueueSizeFunc(ctx)
	}
	return 0, nil
}

func (m *mockDispatcher) GetProcessingStats(ctx context.Context) (DispatcherStats, error) {
	if m.GetProcessingStatsFunc != nil {
		return m.GetProcessingStatsFunc(ctx)
	}
	return DispatcherStats{}, nil
}

var _ NotificationDispatcherPort = (*mockDispatcher)(nil)

func testParams(title string) *notification_dto.SendParams {
	return &notification_dto.SendParams{
		Content: notification_dto.NotificationContent{
			Title:   title,
			Message: "test message",
		},
	}
}

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	providers := service.GetProviders()
	if len(providers) != 0 {
		t.Errorf("expected no providers, got %d", len(providers))
	}
}

func TestNewServiceWithProvider(t *testing.T) {
	provider := &mockProvider{}
	service := NewServiceWithProvider(provider)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if !service.HasProvider(notification_dto.NotificationNameDefault) {
		t.Error("expected default provider to be registered")
	}
}

func TestNewServiceWithDispatcher(t *testing.T) {
	started := false
	disp := &mockDispatcher{
		StartFunc: func(_ context.Context) error {
			started = true
			return nil
		},
	}
	service := NewServiceWithDispatcher(disp)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if !started {
		t.Error("expected dispatcher to be started")
	}
}

func TestNewServiceWithDispatcher_Nil(t *testing.T) {
	service := NewServiceWithDispatcher(nil)
	if service == nil {
		t.Fatal("expected non-nil service even with nil dispatcher")
	}
}

func TestRegisterProvider_Success(t *testing.T) {
	service := NewService()
	provider := &mockProvider{}
	if err := service.RegisterProvider("test", provider); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !service.HasProvider("test") {
		t.Error("expected provider to be registered")
	}
}

func TestRegisterProvider_EmptyName(t *testing.T) {
	service := NewService()
	provider := &mockProvider{}
	if err := service.RegisterProvider("", provider); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterProvider_NilProvider(t *testing.T) {
	service := NewService()
	if err := service.RegisterProvider("test", nil); err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestRegisterProvider_FirstBecomesDefault(t *testing.T) {
	service := NewService()
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error { return nil },
	}
	if err := service.RegisterProvider("first", provider); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err := service.SendBulkWithProvider(ctx, "", []*notification_dto.SendParams{testParams("test")})
	if err != nil {
		t.Errorf("expected default provider to handle send, got error: %v", err)
	}
}

func TestRegisterProvider_SecondDoesNotOverrideDefault(t *testing.T) {
	service := NewService()
	firstCalled := false
	prov1 := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			firstCalled = true
			return nil
		},
	}
	prov2 := &mockProvider{}

	_ = service.RegisterProvider("first", prov1)
	_ = service.RegisterProvider("second", prov2)

	ctx := context.Background()
	_ = service.SendBulkWithProvider(ctx, "", []*notification_dto.SendParams{testParams("test")})
	if !firstCalled {
		t.Error("expected first provider to remain the default")
	}
}

func TestSetDefaultProvider_Success(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("a", &mockProvider{})
	_ = service.RegisterProvider("b", &mockProvider{})

	if err := service.SetDefaultProvider("b"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDefaultProvider_NotFound(t *testing.T) {
	service := NewService()
	err := service.SetDefaultProvider("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestGetProviders_Empty(t *testing.T) {
	service := NewService()
	providers := service.GetProviders()
	if len(providers) != 0 {
		t.Errorf("expected empty, got %v", providers)
	}
}

func TestGetProviders_Sorted(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("charlie", &mockProvider{})
	_ = service.RegisterProvider("alpha", &mockProvider{})
	_ = service.RegisterProvider("bravo", &mockProvider{})

	providers := service.GetProviders()
	if len(providers) != 3 {
		t.Fatalf("expected 3, got %d", len(providers))
	}
	if providers[0] != "alpha" || providers[1] != "bravo" || providers[2] != "charlie" {
		t.Errorf("expected sorted order, got %v", providers)
	}
}

func TestHasProvider_True(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("test", &mockProvider{})
	if !service.HasProvider("test") {
		t.Error("expected true")
	}
}

func TestHasProvider_False(t *testing.T) {
	service := NewService()
	if service.HasProvider("nonexistent") {
		t.Error("expected false")
	}
}

func TestSendBulk_EmptyList(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	err := service.SendBulk(ctx, nil)
	if err != nil {
		t.Errorf("expected nil error for empty list, got %v", err)
	}
}

func TestSendBulkWithProvider_BulkSupporting(t *testing.T) {
	bulkCalled := false
	service := NewService()
	provider := &mockProvider{
		SupportsBulkFunc: func() bool { return true },
		SendBulkFunc: func(_ context.Context, notifications []*notification_dto.SendParams) error {
			bulkCalled = true
			if len(notifications) != 2 {
				t.Errorf("expected 2 notifications, got %d", len(notifications))
			}
			return nil
		},
	}
	_ = service.RegisterProvider("bulk", provider)

	ctx := context.Background()
	err := service.SendBulkWithProvider(ctx, "bulk", []*notification_dto.SendParams{
		testParams("one"),
		testParams("two"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bulkCalled {
		t.Error("expected SendBulk to be called")
	}
}

func TestSendBulkWithProvider_NonBulk(t *testing.T) {
	sendCount := 0
	service := NewService()
	provider := &mockProvider{
		SupportsBulkFunc: func() bool { return false },
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			sendCount++
			return nil
		},
	}
	_ = service.RegisterProvider("single", provider)

	ctx := context.Background()
	err := service.SendBulkWithProvider(ctx, "single", []*notification_dto.SendParams{
		testParams("one"),
		testParams("two"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sendCount != 2 {
		t.Errorf("expected 2 individual sends, got %d", sendCount)
	}
}

func TestSendBulkWithProvider_PartialFailures(t *testing.T) {
	callCount := 0
	service := NewService()
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			callCount++
			if callCount == 2 {
				return errors.New("send failed")
			}
			return nil
		},
	}
	_ = service.RegisterProvider("test", provider)

	ctx := context.Background()
	err := service.SendBulkWithProvider(ctx, "test", []*notification_dto.SendParams{
		testParams("one"),
		testParams("two"),
		testParams("three"),
	})
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(me.Errors))
	}
}

func TestSendBulkWithProvider_ProviderNotFound(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	err := service.SendBulkWithProvider(ctx, "nonexistent", []*notification_dto.SendParams{testParams("test")})
	if err == nil {
		t.Error("expected error for nonexistent provider")
	}
}

func TestSendBulk_NoDefaultProvider(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	err := service.SendBulk(ctx, []*notification_dto.SendParams{testParams("test")})
	if !errors.Is(err, ErrNoDefaultProvider) {
		t.Errorf("expected ErrNoDefaultProvider, got %v", err)
	}
}

func TestSendToProviders_EmptyList(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	err := service.SendToProviders(ctx, testParams("test"), nil)
	if err != nil {
		t.Errorf("expected nil for empty providers list, got %v", err)
	}
}

func TestSendToProviders_AllSucceed(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("a", &mockProvider{})
	_ = service.RegisterProvider("b", &mockProvider{})

	ctx := context.Background()
	err := service.SendToProviders(ctx, testParams("test"), []string{"a", "b"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendToProviders_PartialFailure(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("good", &mockProvider{})
	_ = service.RegisterProvider("bad", &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			return errors.New("send failed")
		},
	})

	ctx := context.Background()
	err := service.SendToProviders(ctx, testParams("test"), []string{"good", "bad"})
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(me.Errors))
	}
	pe, ok := errors.AsType[*ProviderError](me.Errors[0])
	if !ok {
		t.Fatalf("expected ProviderError, got %T", me.Errors[0])
	}
	if pe.Provider != "bad" {
		t.Errorf("expected provider %q, got %q", "bad", pe.Provider)
	}
}

func TestSendToProviders_AllFail(t *testing.T) {
	service := NewService()
	failing := &mockProvider{
		SendFunc: func(_ context.Context, _ *notification_dto.SendParams) error {
			return errors.New("fail")
		},
	}
	_ = service.RegisterProvider("a", failing)
	_ = service.RegisterProvider("b", failing)

	ctx := context.Background()
	err := service.SendToProviders(ctx, testParams("test"), []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(me.Errors))
	}
}

func TestSendToProviders_ProviderNotFound(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("good", &mockProvider{})

	ctx := context.Background()
	err := service.SendToProviders(ctx, testParams("test"), []string{"good", "missing"})
	if err == nil {
		t.Fatal("expected error when provider not found")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	_ = me
}

func TestRegisterDispatcher_Nil(t *testing.T) {
	service := NewService()
	err := service.RegisterDispatcher(nil)
	if err == nil {
		t.Error("expected error for nil dispatcher")
	}
}

func TestRegisterDispatcher_Success(t *testing.T) {
	started := false
	service := NewService()
	disp := &mockDispatcher{
		StartFunc: func(_ context.Context) error {
			started = true
			return nil
		},
	}
	err := service.RegisterDispatcher(disp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Error("expected dispatcher to be started")
	}
}

func TestRegisterDispatcher_StartError(t *testing.T) {
	service := NewService()
	disp := &mockDispatcher{
		StartFunc: func(_ context.Context) error {
			return errors.New("start failed")
		},
	}
	err := service.RegisterDispatcher(disp)
	if err == nil {
		t.Error("expected error when dispatcher start fails")
	}
}

func TestFlushDispatcher_NoDispatcher(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	err := service.FlushDispatcher(ctx)
	if !errors.Is(err, ErrNoDispatcher) {
		t.Errorf("expected ErrNoDispatcher, got %v", err)
	}
}

func TestFlushDispatcher_Delegates(t *testing.T) {
	flushed := false
	service := NewService()
	disp := &mockDispatcher{
		FlushFunc: func(_ context.Context) error {
			flushed = true
			return nil
		},
	}
	_ = service.RegisterDispatcher(disp)

	ctx := context.Background()
	err := service.FlushDispatcher(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !flushed {
		t.Error("expected flush to be called")
	}
}

func TestClose_NoProviders(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	if err := service.Close(ctx); err != nil {
		t.Errorf("expected clean close, got %v", err)
	}
}

func TestClose_WithProviders(t *testing.T) {
	service := NewService()
	closed := false
	_ = service.RegisterProvider("test", &mockProvider{
		CloseFunc: func(_ context.Context) error {
			closed = true
			return nil
		},
	})

	ctx := context.Background()
	if err := service.Close(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !closed {
		t.Error("expected provider Close to be called")
	}
}

func TestClose_ProviderCloseError(t *testing.T) {
	service := NewService()
	_ = service.RegisterProvider("bad", &mockProvider{
		CloseFunc: func(_ context.Context) error {
			return errors.New("close failed")
		},
	})

	ctx := context.Background()
	err := service.Close(ctx)
	if err == nil {
		t.Fatal("expected error from failing provider close")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	_ = me
}

func TestClose_WithDispatcherStopError(t *testing.T) {
	service := NewService()
	disp := &mockDispatcher{
		StopFunc: func(_ context.Context) error {
			return errors.New("stop failed")
		},
	}
	_ = service.RegisterDispatcher(disp)

	ctx := context.Background()
	err := service.Close(ctx)
	if err == nil {
		t.Fatal("expected error from failing dispatcher stop")
	}
}

func TestClose_CombinedErrors(t *testing.T) {
	service := NewService()
	disp := &mockDispatcher{
		StopFunc: func(_ context.Context) error {
			return errors.New("stop failed")
		},
	}
	_ = service.RegisterDispatcher(disp)
	_ = service.RegisterProvider("bad", &mockProvider{
		CloseFunc: func(_ context.Context) error {
			return errors.New("close failed")
		},
	})

	ctx := context.Background()
	err := service.Close(ctx)
	if err == nil {
		t.Fatal("expected combined errors")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(me.Errors))
	}
}
