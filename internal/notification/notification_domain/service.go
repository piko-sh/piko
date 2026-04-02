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
	"fmt"
	"slices"
	"sync"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/notification/notification_dto"
)

const (
	// defaultProviderName is the key used for the default provider in the
	// providers map.
	defaultProviderName = notification_dto.NotificationNameDefault
)

var (
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	errProviderNil = errors.New("provider cannot be nil")
)

// service provides notification sending through configurable providers.
// It implements notification.Service and io.Closer.
type service struct {
	// dispatcher queues notifications for async delivery; nil means send
	// immediately.
	dispatcher NotificationDispatcherPort

	// providers maps provider names to their notification provider instances.
	providers map[string]NotificationProviderPort

	// defaultProvider is the provider name used when none is given.
	defaultProvider string

	// mu guards access to providers and defaultProvider.
	mu sync.RWMutex
}

var _ Service = (*service)(nil)

// NewNotification creates a builder for composing and sending notifications.
//
// Returns *NotificationBuilder which provides a fluent interface for
// building notifications.
func (s *service) NewNotification() *NotificationBuilder {
	return &NotificationBuilder{
		service: s,
		params:  &notification_dto.SendParams{},
	}
}

// SendBulk sends multiple notifications in a single batch operation.
//
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notification details to send.
//
// Returns error when any notification fails to send.
func (s *service) SendBulk(ctx context.Context, notifications []*notification_dto.SendParams) error {
	return s.SendBulkWithProvider(ctx, "", notifications)
}

// SendBulkWithProvider sends multiple notifications using the specified
// provider.
//
// Takes providerName (string) which identifies which notification provider to
// use.
// Takes notifications ([]*notification_dto.SendParams) which contains the
// notifications to send.
//
// Returns error when sending fails or the provider is not found.
func (s *service) SendBulkWithProvider(ctx context.Context, providerName string, notifications []*notification_dto.SendParams) error {
	if len(notifications) == 0 {
		return nil
	}

	provider, err := s.getProvider(providerName)
	if err != nil {
		return fmt.Errorf("resolving notification provider %q: %w", providerName, err)
	}

	if goroutine.SafeCallValue(ctx, "notification.SupportsBulkSending", func() bool { return provider.SupportsBulkSending() }) {
		return goroutine.SafeCall(ctx, "notification.SendBulk", func() error { return provider.SendBulk(ctx, notifications) })
	}

	var sendErrors []error
	for _, params := range notifications {
		if err := goroutine.SafeCall(ctx, "notification.Send", func() error { return provider.Send(ctx, params) }); err != nil {
			sendErrors = append(sendErrors, err)
		}
	}

	if len(sendErrors) > 0 {
		return &MultiError{Errors: sendErrors}
	}

	return nil
}

// SendToProviders sends a single notification to multiple providers
// simultaneously (multi-cast).
//
// Takes params (*notification_dto.SendParams) which contains the notification
// details.
// Takes providers ([]string) which specifies the provider names to send to.
//
// Returns error when sending fails. Partial failures are reported via
// MultiError.
func (s *service) SendToProviders(ctx context.Context, params *notification_dto.SendParams, providers []string) error {
	if len(providers) == 0 {
		return nil
	}

	multiCastCount.Add(ctx, 1)

	var providerErrors []error
	successCount := 0

	for _, providerName := range providers {
		provider, err := s.getProvider(providerName)
		if err != nil {
			providerErrors = append(providerErrors, &ProviderError{
				Provider: providerName,
				Err:      err,
			})
			continue
		}

		if err := goroutine.SafeCall(ctx, "notification.Send", func() error { return provider.Send(ctx, params) }); err != nil {
			providerErrors = append(providerErrors, &ProviderError{
				Provider: providerName,
				Err:      err,
			})
		} else {
			successCount++
		}
	}

	if len(providerErrors) > 0 && successCount > 0 {
		partialFailureCount.Add(ctx, 1)
	}

	if len(providerErrors) > 0 {
		return &MultiError{Errors: providerErrors}
	}

	return nil
}

// RegisterProvider adds a notification provider to the registry.
//
// Takes name (string) which identifies the provider.
// Takes provider (NotificationProviderPort) which handles notification
// delivery.
//
// Returns error when registration fails.
//
// Safe for concurrent use. If this is the first provider registered, it becomes
// the default.
func (s *service) RegisterProvider(name string, provider NotificationProviderPort) error {
	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.providers[name] = provider

	if s.defaultProvider == "" {
		s.defaultProvider = name
	}

	return nil
}

// SetDefaultProvider sets the default provider by name.
//
// Takes name (string) which specifies the provider to set as default.
//
// Returns error when the named provider does not exist.
//
// Safe for concurrent use.
func (s *service) SetDefaultProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; !exists {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	s.defaultProvider = name
	return nil
}

// GetProviders returns the names of all registered providers.
//
// Returns []string which contains the provider names in alphabetical order.
//
// Safe for concurrent use.
func (s *service) GetProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// HasProvider reports whether a provider with the given name exists.
//
// Takes name (string) which identifies the provider to check.
//
// Returns bool which is true if the provider exists, false otherwise.
//
// Safe for concurrent use.
func (s *service) HasProvider(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.providers[name]
	return exists
}

// RegisterDispatcher sets the notification dispatcher for sending
// notifications.
//
// Takes dispatcher (NotificationDispatcherPort) which handles notification
// delivery.
//
// Returns error when the dispatcher is nil or cannot be started.
//
// Safe for concurrent use.
func (s *service) RegisterDispatcher(dispatcher NotificationDispatcherPort) error {
	if dispatcher == nil {
		return errDispatcherNil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dispatcher = dispatcher

	ctx := context.Background()
	if err := dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	return nil
}

// FlushDispatcher flushes any pending dispatches to their destinations.
//
// Returns error when the flush operation fails or no dispatcher is set.
//
// Safe for concurrent use. Uses a read lock to access the dispatcher.
func (s *service) FlushDispatcher(ctx context.Context) error {
	s.mu.RLock()
	dispatcher := s.dispatcher
	s.mu.RUnlock()

	if dispatcher == nil {
		return ErrNoDispatcher
	}

	return dispatcher.Flush(ctx)
}

// Close releases all resources held by the service.
//
// Returns error when the service cannot shut down cleanly.
//
// Safe for concurrent use. Uses a mutex to ensure only one caller closes the
// service at a time.
func (s *service) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var closeErrors []error

	if s.dispatcher != nil {
		if err := s.dispatcher.Stop(ctx); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("stopping dispatcher: %w", err))
		}
	}

	for name, provider := range s.providers {
		if err := provider.Close(ctx); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("closing provider %q: %w", name, err))
		}
	}

	if len(closeErrors) > 0 {
		return &MultiError{Errors: closeErrors}
	}

	return nil
}

// getProvider returns a notification provider by name. If the name is empty,
// it returns the default provider.
//
// Takes name (string) which specifies the provider to retrieve.
//
// Returns NotificationProviderPort which is the requested provider.
// Returns error when no default provider is set or the provider is not found.
//
// Safe for concurrent use; uses a read lock to protect the provider map.
func (s *service) getProvider(name string) (NotificationProviderPort, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providerName := name
	if providerName == "" {
		providerName = s.defaultProvider
	}

	if providerName == "" {
		return nil, ErrNoDefaultProvider
	}

	provider, exists := s.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerName)
	}

	return provider, nil
}

// NewService creates a notification service that supports multiple providers.
//
// Returns Service which is the configured notification service ready for use.
func NewService() Service {
	return &service{
		providers: make(map[string]NotificationProviderPort),
	}
}

// NewServiceWithProvider creates a new notification service with a single
// provider.
//
// Takes provider (NotificationProviderPort) which is the notification provider
// to use.
//
// Returns Service which is the configured notification service ready for use.
func NewServiceWithProvider(provider NotificationProviderPort) Service {
	s := &service{
		providers:       make(map[string]NotificationProviderPort),
		defaultProvider: defaultProviderName,
	}
	s.providers[defaultProviderName] = provider
	return s
}

// NewServiceWithDispatcher creates a new notification service with a
// dispatcher.
//
// Takes dispatcher (NotificationDispatcherPort) which manages async
// notification delivery.
//
// Returns Service which is the configured notification service ready for use.
func NewServiceWithDispatcher(dispatcher NotificationDispatcherPort) Service {
	s := &service{
		providers:  make(map[string]NotificationProviderPort),
		dispatcher: dispatcher,
	}

	if dispatcher != nil {
		ctx := context.Background()
		if err := dispatcher.Start(ctx); err != nil {
			log.ReportError(nil, err, "Failed to start notification dispatcher")
		}
	}

	return s
}
