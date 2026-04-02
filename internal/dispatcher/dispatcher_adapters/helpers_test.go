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
	"time"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

func testContext() context.Context {
	return context.Background()
}

type mockEmailDLQ struct {
	AddFunc          func(ctx context.Context, entry *email_dto.DeadLetterEntry) error
	GetFunc          func(ctx context.Context, limit int) ([]*email_dto.DeadLetterEntry, error)
	RemoveFunc       func(ctx context.Context, entries []*email_dto.DeadLetterEntry) error
	CountFunc        func(ctx context.Context) (int, error)
	ClearFunc        func(ctx context.Context) error
	GetOlderThanFunc func(ctx context.Context, duration time.Duration) ([]*email_dto.DeadLetterEntry, error)
}

func (m *mockEmailDLQ) Add(ctx context.Context, entry *email_dto.DeadLetterEntry) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, entry)
	}
	return nil
}

func (m *mockEmailDLQ) Get(ctx context.Context, limit int) ([]*email_dto.DeadLetterEntry, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, limit)
	}
	return nil, nil
}

func (m *mockEmailDLQ) Remove(ctx context.Context, entries []*email_dto.DeadLetterEntry) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, entries)
	}
	return nil
}

func (m *mockEmailDLQ) Count(ctx context.Context) (int, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

func (m *mockEmailDLQ) Clear(ctx context.Context) error {
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx)
	}
	return nil
}

func (m *mockEmailDLQ) GetOlderThan(ctx context.Context, duration time.Duration) ([]*email_dto.DeadLetterEntry, error) {
	if m.GetOlderThanFunc != nil {
		return m.GetOlderThanFunc(ctx, duration)
	}
	return nil, nil
}

type mockNotificationDLQ struct {
	AddFunc          func(ctx context.Context, entry *notification_dto.DeadLetterEntry) error
	GetFunc          func(ctx context.Context, limit int) ([]*notification_dto.DeadLetterEntry, error)
	RemoveFunc       func(ctx context.Context, entries []*notification_dto.DeadLetterEntry) error
	CountFunc        func(ctx context.Context) (int, error)
	ClearFunc        func(ctx context.Context) error
	GetOlderThanFunc func(ctx context.Context, duration time.Duration) ([]*notification_dto.DeadLetterEntry, error)
}

func (m *mockNotificationDLQ) Add(ctx context.Context, entry *notification_dto.DeadLetterEntry) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, entry)
	}
	return nil
}

func (m *mockNotificationDLQ) Get(ctx context.Context, limit int) ([]*notification_dto.DeadLetterEntry, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, limit)
	}
	return nil, nil
}

func (m *mockNotificationDLQ) Remove(ctx context.Context, entries []*notification_dto.DeadLetterEntry) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, entries)
	}
	return nil
}

func (m *mockNotificationDLQ) Count(ctx context.Context) (int, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}
	return 0, nil
}

func (m *mockNotificationDLQ) Clear(ctx context.Context) error {
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx)
	}
	return nil
}

func (m *mockNotificationDLQ) GetOlderThan(ctx context.Context, duration time.Duration) ([]*notification_dto.DeadLetterEntry, error) {
	if m.GetOlderThanFunc != nil {
		return m.GetOlderThanFunc(ctx, duration)
	}
	return nil, nil
}

type mockEmailDispatcher struct {
	QueueFunc                func(ctx context.Context, params *email_dto.SendParams) error
	FlushFunc                func(ctx context.Context) error
	SetBatchSizeFunc         func(size int)
	SetFlushIntervalFunc     func(interval time.Duration)
	SetRetryConfigFunc       func(config email_domain.RetryConfig)
	GetRetryConfigFunc       func() email_domain.RetryConfig
	GetDeadLetterQueueFunc   func() email_domain.DeadLetterPort
	GetDeadLetterCountFunc   func(ctx context.Context) (int, error)
	ClearDeadLetterQueueFunc func(ctx context.Context) error
	GetRetryQueueSizeFunc    func(ctx context.Context) (int, error)
	GetProcessingStatsFunc   func(ctx context.Context) (email_domain.DispatcherStats, error)
	StartFunc                func(ctx context.Context) error
	StopFunc                 func(ctx context.Context) error
}

func (m *mockEmailDispatcher) Queue(ctx context.Context, params *email_dto.SendParams) error {
	if m.QueueFunc != nil {
		return m.QueueFunc(ctx, params)
	}
	return nil
}

func (m *mockEmailDispatcher) Flush(ctx context.Context) error {
	if m.FlushFunc != nil {
		return m.FlushFunc(ctx)
	}
	return nil
}

func (m *mockEmailDispatcher) SetBatchSize(size int) {
	if m.SetBatchSizeFunc != nil {
		m.SetBatchSizeFunc(size)
	}
}

func (m *mockEmailDispatcher) SetFlushInterval(interval time.Duration) {
	if m.SetFlushIntervalFunc != nil {
		m.SetFlushIntervalFunc(interval)
	}
}

func (m *mockEmailDispatcher) SetRetryConfig(config email_domain.RetryConfig) {
	if m.SetRetryConfigFunc != nil {
		m.SetRetryConfigFunc(config)
	}
}

func (m *mockEmailDispatcher) GetRetryConfig() email_domain.RetryConfig {
	if m.GetRetryConfigFunc != nil {
		return m.GetRetryConfigFunc()
	}
	return email_domain.RetryConfig{}
}

func (m *mockEmailDispatcher) GetDeadLetterQueue() email_domain.DeadLetterPort {
	if m.GetDeadLetterQueueFunc != nil {
		return m.GetDeadLetterQueueFunc()
	}
	return nil
}

func (m *mockEmailDispatcher) GetDeadLetterCount(ctx context.Context) (int, error) {
	if m.GetDeadLetterCountFunc != nil {
		return m.GetDeadLetterCountFunc(ctx)
	}
	return 0, nil
}

func (m *mockEmailDispatcher) ClearDeadLetterQueue(ctx context.Context) error {
	if m.ClearDeadLetterQueueFunc != nil {
		return m.ClearDeadLetterQueueFunc(ctx)
	}
	return nil
}

func (m *mockEmailDispatcher) GetRetryQueueSize(ctx context.Context) (int, error) {
	if m.GetRetryQueueSizeFunc != nil {
		return m.GetRetryQueueSizeFunc(ctx)
	}
	return 0, nil
}

func (m *mockEmailDispatcher) GetProcessingStats(ctx context.Context) (email_domain.DispatcherStats, error) {
	if m.GetProcessingStatsFunc != nil {
		return m.GetProcessingStatsFunc(ctx)
	}
	return email_domain.DispatcherStats{}, nil
}

func (m *mockEmailDispatcher) Start(ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return nil
}

func (m *mockEmailDispatcher) Stop(ctx context.Context) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return nil
}

type mockNotificationDispatcher struct {
	QueueFunc                func(ctx context.Context, params *notification_dto.SendParams) error
	FlushFunc                func(ctx context.Context) error
	SetBatchSizeFunc         func(size int)
	SetFlushIntervalFunc     func(interval time.Duration)
	SetRetryConfigFunc       func(config notification_domain.RetryConfig)
	GetRetryConfigFunc       func() notification_domain.RetryConfig
	GetDeadLetterQueueFunc   func() notification_domain.DeadLetterPort
	GetDeadLetterCountFunc   func(ctx context.Context) (int, error)
	ClearDeadLetterQueueFunc func(ctx context.Context) error
	GetRetryQueueSizeFunc    func(ctx context.Context) (int, error)
	GetProcessingStatsFunc   func(ctx context.Context) (notification_domain.DispatcherStats, error)
	StartFunc                func(ctx context.Context) error
	StopFunc                 func(ctx context.Context) error
}

func (m *mockNotificationDispatcher) Queue(ctx context.Context, params *notification_dto.SendParams) error {
	if m.QueueFunc != nil {
		return m.QueueFunc(ctx, params)
	}
	return nil
}

func (m *mockNotificationDispatcher) Flush(ctx context.Context) error {
	if m.FlushFunc != nil {
		return m.FlushFunc(ctx)
	}
	return nil
}

func (m *mockNotificationDispatcher) SetBatchSize(size int) {
	if m.SetBatchSizeFunc != nil {
		m.SetBatchSizeFunc(size)
	}
}

func (m *mockNotificationDispatcher) SetFlushInterval(interval time.Duration) {
	if m.SetFlushIntervalFunc != nil {
		m.SetFlushIntervalFunc(interval)
	}
}

func (m *mockNotificationDispatcher) SetRetryConfig(config notification_domain.RetryConfig) {
	if m.SetRetryConfigFunc != nil {
		m.SetRetryConfigFunc(config)
	}
}

func (m *mockNotificationDispatcher) GetRetryConfig() notification_domain.RetryConfig {
	if m.GetRetryConfigFunc != nil {
		return m.GetRetryConfigFunc()
	}
	return notification_domain.RetryConfig{}
}

func (m *mockNotificationDispatcher) GetDeadLetterQueue() notification_domain.DeadLetterPort {
	if m.GetDeadLetterQueueFunc != nil {
		return m.GetDeadLetterQueueFunc()
	}
	return nil
}

func (m *mockNotificationDispatcher) GetDeadLetterCount(ctx context.Context) (int, error) {
	if m.GetDeadLetterCountFunc != nil {
		return m.GetDeadLetterCountFunc(ctx)
	}
	return 0, nil
}

func (m *mockNotificationDispatcher) ClearDeadLetterQueue(ctx context.Context) error {
	if m.ClearDeadLetterQueueFunc != nil {
		return m.ClearDeadLetterQueueFunc(ctx)
	}
	return nil
}

func (m *mockNotificationDispatcher) GetRetryQueueSize(ctx context.Context) (int, error) {
	if m.GetRetryQueueSizeFunc != nil {
		return m.GetRetryQueueSizeFunc(ctx)
	}
	return 0, nil
}

func (m *mockNotificationDispatcher) GetProcessingStats(ctx context.Context) (notification_domain.DispatcherStats, error) {
	if m.GetProcessingStatsFunc != nil {
		return m.GetProcessingStatsFunc(ctx)
	}
	return notification_domain.DispatcherStats{}, nil
}

func (m *mockNotificationDispatcher) Start(ctx context.Context) error {
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return nil
}

func (m *mockNotificationDispatcher) Stop(ctx context.Context) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx)
	}
	return nil
}

var (
	_ email_domain.EmailDispatcherPort               = (*mockEmailDispatcher)(nil)
	_ notification_domain.NotificationDispatcherPort = (*mockNotificationDispatcher)(nil)
	_ email_domain.DeadLetterPort                    = (*mockEmailDLQ)(nil)
	_ notification_domain.DeadLetterPort             = (*mockNotificationDLQ)(nil)
)
