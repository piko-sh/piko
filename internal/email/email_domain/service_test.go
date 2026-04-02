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

package email_domain

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestNewService(t *testing.T) {
	service := NewService(context.Background())
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if len(service.GetProviders(context.Background())) != 0 {
		t.Error("expected no providers on fresh service")
	}
}

func TestNewService_WithOptions(t *testing.T) {
	emailService := NewService(context.Background(), WithMaxTotalRecipients(10))
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	if s.config.MaxTotalRecipients != 10 {
		t.Errorf("expected MaxTotalRecipients=10, got %d", s.config.MaxTotalRecipients)
	}
}

func TestNewServiceWithDefaultProvider(t *testing.T) {
	service := NewServiceWithDefaultProvider("my-provider")
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewServiceWithProvider(t *testing.T) {
	provider := &mockProvider{}
	service := NewServiceWithProvider(context.Background(), provider)
	if !service.HasProvider(defaultProviderName) {
		t.Error("expected default provider to be registered")
	}
}

func TestNewServiceWithProviderAndDispatcher(t *testing.T) {
	provider := &mockProvider{}
	startCalled := false
	disp := &mockDispatcher{
		StartFunc: func(_ context.Context) error {
			startCalled = true
			return nil
		},
	}
	service := NewServiceWithProviderAndDispatcher(context.Background(), provider, disp, nil, nil)
	if !service.HasProvider(defaultProviderName) {
		t.Error("expected default provider")
	}
	if !startCalled {
		t.Error("expected dispatcher Start to be called")
	}
}

func TestNewServiceWithProviderAndDispatcher_NilDispatcher(t *testing.T) {
	provider := &mockProvider{}
	service := NewServiceWithProviderAndDispatcher(context.Background(), provider, nil, nil, nil)
	if service == nil {
		t.Fatal("expected non-nil service even with nil dispatcher")
	}
}

func TestRegisterProvider_Success(t *testing.T) {
	service := NewService(context.Background())
	err := service.RegisterProvider(context.Background(), "smtp", &mockProvider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !service.HasProvider("smtp") {
		t.Error("expected provider to be registered")
	}
}

func TestRegisterProvider_EmptyName(t *testing.T) {
	service := NewService(context.Background())
	err := service.RegisterProvider(context.Background(), "", &mockProvider{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRegisterProvider_NilProvider(t *testing.T) {
	service := NewService(context.Background())
	err := service.RegisterProvider(context.Background(), "smtp", nil)
	if err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestSetDefaultProvider_Success(t *testing.T) {
	service := NewService(context.Background())
	_ = service.RegisterProvider(context.Background(), "smtp", &mockProvider{})
	err := service.SetDefaultProvider(context.Background(), "smtp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDefaultProvider_NotFound(t *testing.T) {
	service := NewService(context.Background())
	err := service.SetDefaultProvider(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent provider")
	}
}

func TestGetProviders_Sorted(t *testing.T) {
	service := NewService(context.Background())
	_ = service.RegisterProvider(context.Background(), "zeta", &mockProvider{})
	_ = service.RegisterProvider(context.Background(), "alpha", &mockProvider{})
	_ = service.RegisterProvider(context.Background(), "mu", &mockProvider{})
	names := service.GetProviders(context.Background())
	if len(names) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "mu" || names[2] != "zeta" {
		t.Errorf("expected sorted order, got %v", names)
	}
}

func TestGetProviders_Empty(t *testing.T) {
	service := NewService(context.Background())
	names := service.GetProviders(context.Background())
	if len(names) != 0 {
		t.Errorf("expected empty, got %v", names)
	}
}

func TestHasProvider(t *testing.T) {
	service := NewService(context.Background())
	_ = service.RegisterProvider(context.Background(), "smtp", &mockProvider{})
	if !service.HasProvider("smtp") {
		t.Error("expected true for registered provider")
	}
	if service.HasProvider("ses") {
		t.Error("expected false for unregistered provider")
	}
}

func TestListProviders(t *testing.T) {
	service := NewService(context.Background())
	_ = service.RegisterProvider(context.Background(), "smtp", &mockProvider{})
	infos := service.ListProviders(context.Background())
	if len(infos) != 1 {
		t.Fatalf("expected 1 provider info, got %d", len(infos))
	}
	if infos[0].Name != "smtp" {
		t.Errorf("expected name 'smtp', got %q", infos[0].Name)
	}
}

func TestSendBulk_Success(t *testing.T) {
	var sent int
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent++
			return nil
		},
	}
	service := NewServiceWithProvider(context.Background(), provider)
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, Subject: "s1", BodyPlain: "body1"},
		{To: []string{"b@x.com"}, Subject: "s2", BodyPlain: "body2"},
	}
	err := service.SendBulk(context.Background(), emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent != 2 {
		t.Errorf("expected 2 sends, got %d", sent)
	}
}

func TestSendBulk_NoDefaultProvider(t *testing.T) {
	service := NewService(context.Background())
	err := service.SendBulk(context.Background(), []*email_dto.SendParams{
		{To: []string{"a@x.com"}, BodyPlain: "body"},
	})
	if err == nil {
		t.Error("expected error when no default provider")
	}
}

func TestSendBulk_ValidationErrors(t *testing.T) {
	provider := &mockProvider{}
	service := NewServiceWithProvider(context.Background(), provider)
	emails := []*email_dto.SendParams{
		{To: []string{"valid@x.com"}, BodyPlain: "ok"},
		{To: nil, BodyPlain: "no recipient"},
	}
	err := service.SendBulk(context.Background(), emails)
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	_ = me
}

func TestSendBulk_SendErrors(t *testing.T) {
	provider := &mockProvider{
		SendFunc: func(_ context.Context, p *email_dto.SendParams) error {
			if p.Subject == "fail" {
				return errors.New("send failed")
			}
			return nil
		},
	}
	service := NewServiceWithProvider(context.Background(), provider)
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, Subject: "ok", BodyPlain: "body"},
		{To: []string{"b@x.com"}, Subject: "fail", BodyPlain: "body"},
	}
	err := service.SendBulk(context.Background(), emails)
	if err == nil {
		t.Fatal("expected error for failed send")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if me.Count() != 1 {
		t.Errorf("expected 1 error, got %d", me.Count())
	}
}

func TestSendBulkWithProvider_Success(t *testing.T) {
	var sent int
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent++
			return nil
		},
	}
	service := NewService(context.Background())
	_ = service.RegisterProvider(context.Background(), "custom", provider)
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, Subject: "s1", BodyPlain: "body"},
	}
	err := service.SendBulkWithProvider(context.Background(), "custom", emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent != 1 {
		t.Errorf("expected 1 send, got %d", sent)
	}
}

func TestSendBulkWithProvider_NotFound(t *testing.T) {
	service := NewService(context.Background())
	err := service.SendBulkWithProvider(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for non-existent provider")
	}
}

func TestHandleBulkSend_Empty(t *testing.T) {
	provider := &mockProvider{}
	err := handleBulkSend(context.Background(), provider, nil)
	if err != nil {
		t.Fatalf("expected nil for empty emails, got %v", err)
	}
}

func TestHandleBulkSend_NilEmail(t *testing.T) {
	provider := &mockProvider{}
	err := handleBulkSend(context.Background(), provider, []*email_dto.SendParams{nil})
	if err == nil || !strings.Contains(err.Error(), "email cannot be nil") {
		t.Fatalf("expected nil email error, got %v", err)
	}
}

func TestHandleBulkSend_NoBody(t *testing.T) {
	provider := &mockProvider{}
	err := handleBulkSend(context.Background(), provider, []*email_dto.SendParams{
		{To: []string{"a@x.com"}},
	})
	if err == nil || !errors.Is(err, ErrBodyRequired) {
		t.Fatalf("expected body error, got %v", err)
	}
}

func TestHandleBulkSend_NoRecipients(t *testing.T) {
	provider := &mockProvider{}
	err := handleBulkSend(context.Background(), provider, []*email_dto.SendParams{
		{BodyPlain: "body"},
	})
	if err == nil || !errors.Is(err, ErrRecipientRequired) {
		t.Fatalf("expected recipient error, got %v", err)
	}
}

func TestHandleBulkSend_BulkSupported(t *testing.T) {
	bulkCalled := false
	provider := &mockProvider{
		SupportsBulkFunc: func() bool { return true },
		SendBulkFunc: func(_ context.Context, _ []*email_dto.SendParams) error {
			bulkCalled = true
			return nil
		},
	}
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, BodyPlain: "body"},
	}
	err := handleBulkSend(context.Background(), provider, emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bulkCalled {
		t.Error("expected bulk send to be called")
	}
}

func TestHandleBulkSend_BulkFails_FallsBack(t *testing.T) {
	individualCalls := 0
	provider := &mockProvider{
		SupportsBulkFunc: func() bool { return true },
		SendBulkFunc: func(_ context.Context, _ []*email_dto.SendParams) error {
			return errors.New("bulk failed")
		},
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			individualCalls++
			return nil
		},
	}
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, BodyPlain: "body"},
		{To: []string{"b@x.com"}, BodyPlain: "body"},
	}
	err := handleBulkSend(context.Background(), provider, emails)
	if err != nil {
		t.Fatalf("unexpected error after fallback: %v", err)
	}
	if individualCalls != 2 {
		t.Errorf("expected 2 individual sends, got %d", individualCalls)
	}
}

func TestHandleBulkSend_NoBulk_SendsIndividually(t *testing.T) {
	var sent int
	provider := &mockProvider{
		SupportsBulkFunc: func() bool { return false },
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			sent++
			return nil
		},
	}
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, BodyPlain: "body"},
	}
	err := handleBulkSend(context.Background(), provider, emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent != 1 {
		t.Errorf("expected 1 individual send, got %d", sent)
	}
}

func TestSendIndividually_PartialFailure(t *testing.T) {
	call := 0
	provider := &mockProvider{
		SendFunc: func(_ context.Context, _ *email_dto.SendParams) error {
			call++
			if call == 2 {
				return errors.New("fail on second")
			}
			return nil
		},
	}
	emails := []*email_dto.SendParams{
		{To: []string{"a@x.com"}, BodyPlain: "body"},
		{To: []string{"b@x.com"}, BodyPlain: "body"},
		{To: []string{"c@x.com"}, BodyPlain: "body"},
	}
	err := sendIndividuallyWithMultiError(context.Background(), provider, emails)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}
	me, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if me.Count() != 1 {
		t.Errorf("expected 1 failure, got %d", me.Count())
	}
}

func TestRegisterDispatcher_Success(t *testing.T) {
	started := false
	disp := &mockDispatcher{
		StartFunc: func(_ context.Context) error {
			started = true
			return nil
		},
	}
	service := NewService(context.Background())
	err := service.RegisterDispatcher(context.Background(), disp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !started {
		t.Error("expected Start to be called")
	}
}

func TestRegisterDispatcher_Nil(t *testing.T) {
	service := NewService(context.Background())
	err := service.RegisterDispatcher(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil dispatcher")
	}
}

func TestFlushDispatcher_Success(t *testing.T) {
	flushed := false
	disp := &mockDispatcher{
		FlushFunc: func(_ context.Context) error {
			flushed = true
			return nil
		},
	}
	service := NewService(context.Background())
	_ = service.RegisterDispatcher(context.Background(), disp)
	err := service.FlushDispatcher(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !flushed {
		t.Error("expected Flush to be called")
	}
}

func TestFlushDispatcher_NoDispatcher(t *testing.T) {
	service := NewService(context.Background())
	err := service.FlushDispatcher(context.Background())
	if err == nil {
		t.Error("expected error when no dispatcher")
	}
}

func TestCheck_Liveness_Healthy(t *testing.T) {
	emailService := NewServiceWithProvider(context.Background(), &mockProvider{})
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected healthy, got %v", status.State)
	}
	if status.Name != "EmailService" {
		t.Errorf("expected name 'EmailService', got %q", status.Name)
	}
}

func TestCheck_Liveness_Unhealthy(t *testing.T) {
	emailService := NewService(context.Background())
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("expected unhealthy (no providers), got %v", status.State)
	}
}

func TestCheck_Readiness_AllHealthy(t *testing.T) {
	provider := &mockHealthProbeProvider{
		CheckFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{State: healthprobe_dto.StateHealthy}
		},
	}
	emailService := NewService(context.Background())
	_ = emailService.RegisterProvider(context.Background(), "smtp", provider)
	_ = emailService.SetDefaultProvider(context.Background(), "smtp")
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected healthy, got %v", status.State)
	}
	if len(status.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(status.Dependencies))
	}
}

func TestCheck_Readiness_DefaultUnhealthy(t *testing.T) {
	provider := &mockHealthProbeProvider{
		CheckFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{State: healthprobe_dto.StateUnhealthy}
		},
	}
	emailService := NewService(context.Background())
	_ = emailService.RegisterProvider(context.Background(), "smtp", provider)
	_ = emailService.SetDefaultProvider(context.Background(), "smtp")
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("expected unhealthy when default provider fails, got %v", status.State)
	}
}

func TestCheck_Readiness_NoHealthProbe(t *testing.T) {
	provider := &mockProvider{}
	emailService := NewService(context.Background())
	_ = emailService.RegisterProvider(context.Background(), "smtp", provider)
	_ = emailService.SetDefaultProvider(context.Background(), "smtp")
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	status := s.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected healthy for provider without health check (skipped), got %v", status.State)
	}
}

func TestUpdateOverallStateFromProvider(t *testing.T) {
	testCases := []struct {
		name      string
		current   healthprobe_dto.State
		provider  healthprobe_dto.State
		expected  healthprobe_dto.State
		isDefault bool
	}{
		{
			name:     "healthy stays healthy",
			current:  healthprobe_dto.StateHealthy,
			provider: healthprobe_dto.StateHealthy,
			expected: healthprobe_dto.StateHealthy,
		},
		{
			name:      "unhealthy default → unhealthy",
			current:   healthprobe_dto.StateHealthy,
			provider:  healthprobe_dto.StateUnhealthy,
			isDefault: true,
			expected:  healthprobe_dto.StateUnhealthy,
		},
		{
			name:     "unhealthy non-default → degraded",
			current:  healthprobe_dto.StateHealthy,
			provider: healthprobe_dto.StateUnhealthy,
			expected: healthprobe_dto.StateDegraded,
		},
		{
			name:     "degraded provider → degraded overall",
			current:  healthprobe_dto.StateHealthy,
			provider: healthprobe_dto.StateDegraded,
			expected: healthprobe_dto.StateDegraded,
		},
		{
			name:     "already degraded stays degraded",
			current:  healthprobe_dto.StateDegraded,
			provider: healthprobe_dto.StateDegraded,
			expected: healthprobe_dto.StateDegraded,
		},
		{
			name:     "already unhealthy stays unhealthy",
			current:  healthprobe_dto.StateUnhealthy,
			provider: healthprobe_dto.StateHealthy,
			expected: healthprobe_dto.StateUnhealthy,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := updateOverallStateFromProvider(tc.current, tc.provider, tc.isDefault)
			if got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestNewEmail(t *testing.T) {
	emailService := NewService(context.Background())
	s, ok := emailService.(*service)
	require.True(t, ok, "expected service to be *service")
	builder := s.NewEmail()
	if builder == nil {
		t.Fatal("expected non-nil builder")
	}
}
