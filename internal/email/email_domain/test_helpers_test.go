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
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/premailer"
)

type emailDLQ = deadletter_domain.MockDeadLetterPort[*email_dto.DeadLetterEntry]

func newTestLogger() logger_domain.Logger {
	base := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	return logger_domain.New(base, "email-domain-test")
}

type fakeProvider struct {
	attempts     map[string]int
	failNTimes   map[string]int
	permanent    map[string]bool
	sendCalls    []*email_dto.SendParams
	bulkCalls    [][]*email_dto.SendParams
	mu           sync.Mutex
	supportsBulk bool
	closed       bool
}

func newFakeProvider(supportsBulk bool) *fakeProvider {
	return &fakeProvider{
		supportsBulk: supportsBulk,
		attempts:     make(map[string]int),
		failNTimes:   make(map[string]int),
		permanent:    make(map[string]bool),
	}
}

func keyFromEmail(p *email_dto.SendParams) string {
	if p.Subject != "" {
		return p.Subject
	}
	if len(p.To) > 0 {
		return p.To[0]
	}
	return "<unknown>"
}

func (f *fakeProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalls = append(f.sendCalls, params)
	key := keyFromEmail(params)
	f.attempts[key]++
	if f.permanent[key] {
		return errors.New("permanent failure")
	}
	if n := f.failNTimes[key]; n > 0 {
		f.failNTimes[key] = n - 1
		return errors.New("transient failure")
	}
	return nil
}

func (f *fakeProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bulkCalls = append(f.bulkCalls, emails)
	for _, e := range emails {
		key := keyFromEmail(e)
		f.attempts[key]++
	}
	return nil
}

func (f *fakeProvider) SupportsBulkSending() bool { return f.supportsBulk }
func (f *fakeProvider) Close(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeProvider) attemptsFor(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts[key]
}
func (f *fakeProvider) totalBulkCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bulkCalls)
}

type mockProvider struct {
	SendFunc         func(ctx context.Context, params *email_dto.SendParams) error
	SendBulkFunc     func(ctx context.Context, emails []*email_dto.SendParams) error
	SupportsBulkFunc func() bool
	CloseFunc        func(ctx context.Context) error
}

func (m *mockProvider) Send(ctx context.Context, params *email_dto.SendParams) error {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, params)
	}
	return nil
}

func (m *mockProvider) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	if m.SendBulkFunc != nil {
		return m.SendBulkFunc(ctx, emails)
	}
	return nil
}

func (m *mockProvider) SupportsBulkSending() bool {
	if m.SupportsBulkFunc != nil {
		return m.SupportsBulkFunc()
	}
	return false
}

func (m *mockProvider) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

type mockDispatcher struct {
	QueueFunc                func(ctx context.Context, params *email_dto.SendParams) error
	FlushFunc                func(ctx context.Context) error
	StartFunc                func(ctx context.Context) error
	StopFunc                 func(ctx context.Context) error
	SetBatchSizeFunc         func(size int)
	SetFlushIntervalFunc     func(interval time.Duration)
	SetRetryConfigFunc       func(config RetryConfig)
	GetRetryConfigFunc       func() RetryConfig
	GetDeadLetterQueueFunc   func() DeadLetterPort
	GetDeadLetterCountFunc   func(ctx context.Context) (int, error)
	ClearDeadLetterQueueFunc func(ctx context.Context) error
	GetRetryQueueSizeFunc    func(ctx context.Context) (int, error)
	GetProcessingStatsFunc   func(ctx context.Context) (DispatcherStats, error)
}

func (m *mockDispatcher) Queue(ctx context.Context, params *email_dto.SendParams) error {
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
	if m.ClearDeadLetterQueueFunc != nil {
		return m.ClearDeadLetterQueueFunc(ctx)
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

type mockTemplater struct {
	RenderFunc func(ctx context.Context, request *http.Request, templatePath string, props any, opts *premailer.Options) (*RenderedEmail, error)
}

func (m *mockTemplater) Render(ctx context.Context, request *http.Request, templatePath string, props any, opts *premailer.Options) (*RenderedEmail, error) {
	if m.RenderFunc != nil {
		return m.RenderFunc(ctx, request, templatePath, props, opts)
	}
	return &RenderedEmail{}, nil
}

type mockAssetResolver struct {
	ResolveAssetFunc  func(ctx context.Context, request *email_dto.EmailAssetRequest) (*email_dto.Attachment, error)
	ResolveAssetsFunc func(ctx context.Context, requests []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error)
}

func (m *mockAssetResolver) ResolveAsset(ctx context.Context, request *email_dto.EmailAssetRequest) (*email_dto.Attachment, error) {
	if m.ResolveAssetFunc != nil {
		return m.ResolveAssetFunc(ctx, request)
	}
	return nil, nil
}

func (m *mockAssetResolver) ResolveAssets(ctx context.Context, requests []*email_dto.EmailAssetRequest) ([]*email_dto.Attachment, []error) {
	if m.ResolveAssetsFunc != nil {
		return m.ResolveAssetsFunc(ctx, requests)
	}
	return nil, nil
}

type mockHealthProbeProvider struct {
	mockProvider
	NameFunc  func() string
	CheckFunc func(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
}

func (m *mockHealthProbeProvider) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock-provider"
}

func (m *mockHealthProbeProvider) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	if m.CheckFunc != nil {
		return m.CheckFunc(ctx, checkType)
	}
	return healthprobe_dto.Status{State: healthprobe_dto.StateHealthy}
}
