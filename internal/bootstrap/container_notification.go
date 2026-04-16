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

package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"piko.sh/piko/internal/deadletter/deadletter_adapters"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/notification/notification_adapters/driver_providers"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultBatchSize is the number of notifications to process in one batch.
	defaultBatchSize = 10

	// defaultFlushInterval is the time between automatic batch flushes (30 seconds).
	defaultFlushInterval = 30_000_000_000

	// defaultMaxRetries is the maximum number of times to retry failed operations.
	defaultMaxRetries = 3

	// defaultInitialDelay is the starting wait time for retry backoff (5 seconds).
	defaultInitialDelay = 5_000_000_000

	// defaultMaxDelay is the maximum delay between retry attempts (300 seconds).
	defaultMaxDelay = 300_000_000_000

	// defaultBackoffFactor is the multiplier applied to the delay between
	// retries.
	defaultBackoffFactor = 2.0

	// defaultCircuitBreakerThreshold is the number of consecutive failures
	// before the circuit breaker opens.
	defaultCircuitBreakerThreshold = 5

	// defaultCircuitBreakerTimeout is the duration in nanoseconds before a tripped
	// circuit breaker attempts recovery.
	defaultCircuitBreakerTimeout = 30_000_000_000

	// defaultCircuitBreakerInterval is the interval in nanoseconds between circuit
	// breaker state checks.
	defaultCircuitBreakerInterval = 60_000_000_000
)

// AddNotificationProvider registers a named notification provider.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (NotificationProviderPort) which handles notification delivery.
func (c *Container) AddNotificationProvider(name string, provider notification_domain.NotificationProviderPort) {
	if c.notificationProviders == nil {
		c.notificationProviders = make(map[string]notification_domain.NotificationProviderPort)
	}
	c.notificationProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "NotificationProvider-"+name, provider)
}

// SetNotificationDefaultProvider sets the default notification provider.
//
// Takes name (string) which is the provider name to set as default.
func (c *Container) SetNotificationDefaultProvider(name string) {
	c.notificationDefaultProvider = name
}

var (
	// notificationService holds the lazily initialised notification service singleton.
	notificationService notification_domain.Service

	// notificationErr holds any error encountered during notification service creation.
	notificationErr error

	// notificationInitOnce guards one-time initialisation of the notification service.
	notificationInitOnce sync.Once
)

// GetNotificationService returns the notification service, initialising it if
// needed.
//
// Returns notification_domain.Service which is the notification service.
// Returns error when initialisation fails.
func (c *Container) GetNotificationService() (notification_domain.Service, error) {
	notificationInitOnce.Do(func() {
		notificationService, notificationErr = c.createNotificationService()
	})
	return notificationService, notificationErr
}

// createNotificationService creates and sets up the notification service.
//
// Returns notification_domain.Service which is the configured notification
// service ready for use.
// Returns error when provider registration fails or the default provider
// cannot be set.
func (c *Container) createNotificationService() (notification_domain.Service, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	service := notification_domain.NewService()

	for name, provider := range c.notificationProviders {
		if err := service.RegisterProvider(name, provider); err != nil {
			return nil, fmt.Errorf("registering notification provider %q: %w", name, err)
		}
	}

	if c.notificationDefaultProvider != "" {
		if err := service.SetDefaultProvider(c.notificationDefaultProvider); err != nil {
			return nil, fmt.Errorf("setting default notification provider: %w", err)
		}
	}

	if len(c.notificationProviders) == 0 {
		stdoutProvider := driver_providers.NewStdoutProvider()
		if err := service.RegisterProvider(notification_dto.NotificationNameStdout, stdoutProvider); err != nil {
			return nil, fmt.Errorf("registering stdout provider: %w", err)
		}
		if err := service.SetDefaultProvider(notification_dto.NotificationNameStdout); err != nil {
			return nil, fmt.Errorf("setting default provider: %w", err)
		}
		l.Internal("Using default stdout notification provider (no custom providers registered)")
	}

	if err := c.createAndRegisterNotificationDispatcher(service); err != nil {
		return nil, fmt.Errorf("creating notification dispatcher: %w", err)
	}

	shutdown.Register(c.GetAppContext(), "notification-service", func(ctx context.Context) error {
		return service.Close(ctx)
	})

	l.Internal("Notification service initialised")

	return service, nil
}

// createAndRegisterNotificationDispatcher creates and registers the
// notification dispatcher.
//
// Takes service (notification_domain.Service) which handles sending
// notifications.
//
// Returns error when dispatcher registration fails.
func (c *Container) createAndRegisterNotificationDispatcher(service notification_domain.Service) error {
	dispatcherConfig := &notification_dto.DispatcherConfig{
		Enabled:                 true,
		BatchSize:               defaultBatchSize,
		FlushInterval:           defaultFlushInterval,
		MaxRetries:              defaultMaxRetries,
		InitialDelay:            defaultInitialDelay,
		MaxDelay:                defaultMaxDelay,
		BackoffFactor:           defaultBackoffFactor,
		CircuitBreakerThreshold: defaultCircuitBreakerThreshold,
		CircuitBreakerTimeout:   defaultCircuitBreakerTimeout,
		CircuitBreakerInterval:  defaultCircuitBreakerInterval,
		DeadLetterPath:          "/tmp/piko-notifications-dlq.jsonl",
	}

	dlqDir := filepath.Dir(dispatcherConfig.DeadLetterPath)
	dlqSandbox, sandboxErr := c.createSandbox("deadletter-disk", dlqDir, safedisk.ModeReadWrite)
	var dlqOpts []deadletter_adapters.DiskDeadLetterOption[*notification_dto.DeadLetterEntry]
	if sandboxErr == nil {
		dlqOpts = append(dlqOpts, deadletter_adapters.WithDeadLetterSandbox[*notification_dto.DeadLetterEntry](dlqSandbox))
	} else {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Warn("Failed to create deadletter sandbox, using fallback",
			logger_domain.Error(sandboxErr))
	}
	dlq := deadletter_adapters.NewDiskDeadLetterQueue[*notification_dto.DeadLetterEntry](dispatcherConfig.DeadLetterPath, dlqOpts...)

	dispatcher := notification_domain.NewNotificationDispatcher(service, dlq, dispatcherConfig)
	if dispatcher == nil {
		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Warn("Could not create notification dispatcher, using synchronous sending")
		return nil
	}

	c.notificationDispatcher = dispatcher
	c.hasNotificationDispatcher = true

	if err := service.RegisterDispatcher(dispatcher); err != nil {
		return fmt.Errorf("registering dispatcher: %w", err)
	}
	return nil
}
