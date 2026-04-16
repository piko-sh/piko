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
	"fmt"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// defaultProviderName is the key used for the default provider in the
	// providers map.
	defaultProviderName = email_dto.EmailNameDefault

	// errProviderNotFoundFmt is the format string for the provider not found
	// error.
	errProviderNotFoundFmt = "provider '%s' not found"

	// serviceName is the name used to identify the email service in the registry.
	serviceName = "email"
)

var (
	// errProviderNameEmpty is returned when an email provider is registered
	// with an empty name.
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	// errProviderNil is returned when a nil email provider is supplied during
	// registration.
	errProviderNil = errors.New("provider cannot be nil")
)

// service provides email sending through configurable providers and templates.
// It implements the email.Service and healthprobe.Probe interfaces.
type service struct {
	// dispatcher queues emails for async delivery; nil means send immediately.
	dispatcher EmailDispatcherPort

	// templater converts email templates into HTML content.
	templater TemplaterAdapterPort

	// assetResolver finds assets for email templates.
	assetResolver AssetResolverPort

	// registry stores email providers and tracks which one is the default.
	registry *provider_domain.StandardRegistry[EmailProviderPort]

	// config holds the service settings for validation limits.
	config ServiceConfig

	// mu guards access to the providers map.
	mu sync.RWMutex
}

var _ Service = (*service)(nil)

// RegisterProvider registers a new email provider with the given name.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (EmailProviderPort) which handles email delivery.
//
// Returns error when name is empty or provider is nil.
//
// Default provider must be set explicitly with SetDefaultProvider.
func (s *service) RegisterProvider(ctx context.Context, name string, provider EmailProviderPort) error {
	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}

	return s.registry.RegisterProvider(ctx, name, provider)
}

// SetDefaultProvider sets the default provider for sending methods.
//
// Takes name (string) which specifies the provider to use as the default.
//
// Returns error when the named provider does not exist.
func (s *service) SetDefaultProvider(ctx context.Context, name string) error {
	return s.registry.SetDefaultProvider(ctx, name)
}

// GetProviders returns a sorted list of registered provider names.
//
// Returns []string which contains the provider names in alphabetical order.
func (s *service) GetProviders(ctx context.Context) []string {
	providers := s.registry.ListProviders(ctx)
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return names
}

// HasProvider checks if a provider with the given name is registered.
//
// Takes name (string) which specifies the provider name to look up.
//
// Returns bool which is true if the provider exists, false otherwise.
func (s *service) HasProvider(name string) bool {
	return s.registry.HasProvider(name)
}

// ListProviders returns detailed information about all registered providers.
//
// Returns []provider_domain.ProviderInfo which contains provider metadata,
// health status, and capabilities.
func (s *service) ListProviders(ctx context.Context) []provider_domain.ProviderInfo {
	return s.registry.ListProviders(ctx)
}

// NewEmail creates a builder for a simple HTML or plain text email.
// This is the entry point for the fluent builder API for emails without
// templates.
//
// Returns *EmailBuilder which provides methods to set up and send the email.
func (s *service) NewEmail() *EmailBuilder {
	return &EmailBuilder{
		baseEmailBuilder: &baseEmailBuilder{
			service: s,
			params:  &email_dto.SendParams{},
		},
	}
}

// SendBulk sends multiple emails using the default provider.
//
// It sends all valid emails and returns a MultiError containing any
// validation or sending failures.
//
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when validation fails or any email cannot be sent.
func (s *service) SendBulk(ctx context.Context, emails []*email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	provider, err := s.getProvider(ctx, "")
	if err != nil {
		return fmt.Errorf("resolving default email provider: %w", err)
	}

	for _, email := range emails {
		sanitiseRecipients(email)
	}

	validEmails, validationErrs := validateAndSplitBulk(emails, s.config)

	sendErrs := handleBulkSend(ctx, provider, validEmails)

	if validationErrs == nil && sendErrs == nil {
		return nil
	}

	combinedErrors := &MultiError{}
	if validationErrs != nil {
		combinedErrors.Errors = append(combinedErrors.Errors, validationErrs.Errors...)
	}

	if sendErrs != nil {
		if me, ok := errors.AsType[*MultiError](sendErrs); ok {
			combinedErrors.Errors = append(combinedErrors.Errors, me.Errors...)
		} else {
			l.ReportError(nil, sendErrs, "A non-MultiError was returned from bulk sending")
		}
	}

	if combinedErrors.HasErrors() {
		return combinedErrors
	}

	return nil
}

// SendBulkWithProvider sends multiple emails using the specified provider.
//
// Takes providerName (string) which identifies the email provider to use.
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when the provider is not found or sending fails.
func (s *service) SendBulkWithProvider(ctx context.Context, providerName string, emails []*email_dto.SendParams) error {
	ctx, _ = logger_domain.From(ctx, log)
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		return fmt.Errorf("resolving email provider %q: %w", providerName, err)
	}
	if err := handleBulkSend(ctx, provider, emails); err != nil {
		return fmt.Errorf("sending bulk email via provider %q: %w", providerName, err)
	}
	return nil
}

// RegisterDispatcher registers and starts an email dispatcher with the service.
//
// Takes dispatcher (EmailDispatcherPort) which handles email delivery.
//
// Returns error when dispatcher is nil or fails to start.
//
// Safe for concurrent use. Uses a mutex to protect dispatcher registration.
func (s *service) RegisterDispatcher(ctx context.Context, dispatcher EmailDispatcherPort) error {
	if dispatcher == nil {
		return errDispatcherNil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dispatcher = dispatcher

	if err := dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start email dispatcher: %w", err)
	}

	return nil
}

// FlushDispatcher sends all queued emails in the dispatcher at once.
//
// Returns error when no dispatcher is registered or the flush fails.
//
// Safe for concurrent use.
func (s *service) FlushDispatcher(ctx context.Context) error {
	s.mu.RLock()
	dispatcher := s.dispatcher
	s.mu.RUnlock()

	if dispatcher == nil {
		return errNoDispatcher
	}

	return dispatcher.Flush(ctx)
}

// Name returns the service identifier and implements the
// healthprobe_domain.Probe interface.
//
// Returns string which is the service name "EmailService".
func (*service) Name() string {
	return "EmailService"
}

// Check implements the healthprobe_domain.Probe interface.
// It checks whether email providers are available and working.
//
// Takes checkType (healthprobe_dto.CheckType) which specifies whether to
// run a liveness or readiness check.
//
// Returns healthprobe_dto.Status which shows the health state of the service.
func (s *service) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	providers := s.registry.ListProviders(ctx)
	providerCount := len(providers)

	if checkType == healthprobe_dto.CheckTypeLiveness {
		return s.checkLiveness(startTime, providerCount)
	}

	return s.checkReadiness(ctx, startTime, checkType, providerCount)
}

// getProvider fetches a provider by name, returning the default provider when
// name is empty.
//
// Takes name (string) which specifies the provider to fetch, or empty for the
// default.
//
// Returns EmailProviderPort which is the requested provider.
// Returns error when no default provider is set or fetching fails.
func (s *service) getProvider(ctx context.Context, name string) (EmailProviderPort, error) {
	providerName := name
	if providerName == "" {
		providerName = s.registry.GetDefaultProvider()
	}

	if providerName == "" {
		return nil, errors.New("no default provider configured")
	}

	return s.registry.GetProvider(ctx, providerName)
}

// sendImmediateWithProvider is a helper method for sending email immediately
// with a specific provider.
//
// Takes providerName (string) which identifies the provider to use.
// Takes params (*email_dto.SendParams) which contains the email to send.
//
// Returns error when the provider cannot be resolved or sending fails.
func (s *service) sendImmediateWithProvider(ctx context.Context, providerName string, params *email_dto.SendParams) error {
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		return fmt.Errorf("resolving email provider %q: %w", providerName, err)
	}
	if err := goroutine.SafeCall(ctx, "email.Send", func() error { return provider.Send(ctx, params) }); err != nil {
		return fmt.Errorf("sending email via provider %q: %w", providerName, err)
	}
	return nil
}

// checkLiveness returns liveness status based on provider initialisation.
//
// Takes startTime (time.Time) which marks when the health check began.
// Takes providerCount (int) which is the number of registered email providers.
//
// Returns healthprobe_dto.Status which contains the current liveness state.
func (s *service) checkLiveness(startTime time.Time, providerCount int) healthprobe_dto.Status {
	state := healthprobe_dto.StateHealthy
	message := "Email service is running"

	if providerCount == 0 {
		state = healthprobe_dto.StateUnhealthy
		message = "No email providers registered"
	}

	return healthprobe_dto.Status{
		Name:      s.Name(),
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// checkReadiness returns the readiness status by checking all providers.
//
// Takes startTime (time.Time) which marks when the check started.
// Takes checkType (healthprobe_dto.CheckType) which sets the type of health
// check to run.
// Takes providerCount (int) which is the number of configured providers.
//
// Returns healthprobe_dto.Status which holds the overall readiness state and
// any issues with dependencies.
func (s *service) checkReadiness(ctx context.Context, startTime time.Time, checkType healthprobe_dto.CheckType, providerCount int) healthprobe_dto.Status {
	dependencies, overallState := s.checkProviders(ctx, checkType)

	message := fmt.Sprintf("Email service operational with %d provider(s)", providerCount)
	if overallState != healthprobe_dto.StateHealthy {
		message = "Email service has provider issues"
	}

	return healthprobe_dto.Status{
		Name:         s.Name(),
		State:        overallState,
		Message:      message,
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: dependencies,
	}
}

// checkProviders checks all providers and returns their statuses.
//
// Takes checkType (healthprobe_dto.CheckType) which sets the type of health
// check to run.
//
// Returns []*healthprobe_dto.Status which holds the status of each provider.
// Returns healthprobe_dto.State which is the overall health state across all
// providers.
func (s *service) checkProviders(ctx context.Context, checkType healthprobe_dto.CheckType) ([]*healthprobe_dto.Status, healthprobe_dto.State) {
	providerInfos := s.registry.ListProviders(ctx)
	dependencies := make([]*healthprobe_dto.Status, 0, len(providerInfos))
	overallState := healthprobe_dto.StateHealthy
	defaultProvider := s.registry.GetDefaultProvider()

	for _, info := range providerInfos {
		provider, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			continue
		}

		isDefault := info.Name == defaultProvider
		status, state := s.checkSingleProvider(ctx, info.Name, provider, checkType, isDefault, overallState)
		dependencies = append(dependencies, status)
		overallState = state
	}

	return dependencies, overallState
}

// checkSingleProvider checks a single provider and returns its status.
//
// Takes name (string) which identifies the provider.
// Takes provider (EmailProviderPort) which is the provider to check.
// Takes checkType (healthprobe_dto.CheckType) which specifies the type of
// health check to run.
// Takes isDefault (bool) which shows whether this is the default provider.
// Takes currentState (healthprobe_dto.State) which is the current overall
// health state.
//
// Returns *healthprobe_dto.Status which contains the provider's health status.
// Returns healthprobe_dto.State which is the updated overall health state.
func (*service) checkSingleProvider(
	ctx context.Context,
	name string,
	provider EmailProviderPort,
	checkType healthprobe_dto.CheckType,
	isDefault bool,
	currentState healthprobe_dto.State,
) (*healthprobe_dto.Status, healthprobe_dto.State) {
	if probe, ok := provider.(interface {
		Name() string
		Check(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status
	}); ok {
		providerStatus := probe.Check(ctx, checkType)
		return &providerStatus, updateOverallStateFromProvider(currentState, providerStatus.State, isDefault)
	}

	providerLabel := fmt.Sprintf("EmailProvider (%s)", name)
	if isDefault {
		providerLabel += " [default]"
	}
	return &healthprobe_dto.Status{
		Name:    providerLabel,
		State:   healthprobe_dto.StateHealthy,
		Message: "Provider does not support health checks (skipped)",
	}, currentState
}

// NewService creates a new email service that supports multiple providers.
//
// Takes opts (...ServiceOption) which configures service limits and behaviour.
//
// Returns Service which is the configured email service ready for use.
func NewService(_ context.Context, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &service{
		registry: provider_domain.NewStandardRegistry[EmailProviderPort](serviceName),
		config:   config,
	}
}

// NewServiceWithDefaultProvider creates a new email service with a specified
// default provider name. The provider itself must be registered separately via
// RegisterProvider.
//
// Takes opts (...ServiceOption) which configures service limits and behaviour.
//
// Returns Service which is the configured email service ready for use.
func NewServiceWithDefaultProvider(_ string, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	s := &service{
		registry: provider_domain.NewStandardRegistry[EmailProviderPort](serviceName),
		config:   config,
	}
	return s
}

// NewServiceWithProvider creates a new email service with a single provider
// (for backward compatibility). It accepts functional options to configure
// service limits and behaviour.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes provider (EmailProviderPort) which is the email provider to use.
// Takes opts (...ServiceOption) which are optional functions to configure the
// service.
//
// Returns Service which is the configured email service ready for use.
func NewServiceWithProvider(ctx context.Context, provider EmailProviderPort, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	s := &service{
		registry: provider_domain.NewStandardRegistry[EmailProviderPort](serviceName),
		config:   config,
	}
	if err := s.registry.RegisterProvider(ctx, defaultProviderName, provider); err != nil {
		log.ReportError(nil, err, "Failed to register default email provider")
	}
	if err := s.registry.SetDefaultProvider(ctx, defaultProviderName); err != nil {
		log.ReportError(nil, err, "Failed to set default email provider")
	}
	return s
}

// NewServiceWithProviderAndDispatcher creates a new email service with the
// given provider and optional dispatcher. It accepts functional options to
// set service limits and behaviour.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes provider (EmailProviderPort) which handles sending emails.
// Takes dispatcher (EmailDispatcherPort) which manages async email delivery.
// Takes templater (TemplaterAdapterPort) which renders email templates.
// Takes assetResolver (AssetResolverPort) which resolves template assets.
// Takes opts (...ServiceOption) which sets service limits and behaviour.
//
// Returns Service which is the configured email service ready to use.
func NewServiceWithProviderAndDispatcher(
	ctx context.Context, provider EmailProviderPort, dispatcher EmailDispatcherPort,
	templater TemplaterAdapterPort, assetResolver AssetResolverPort, opts ...ServiceOption,
) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	s := &service{
		registry:      provider_domain.NewStandardRegistry[EmailProviderPort](serviceName),
		dispatcher:    dispatcher,
		config:        config,
		templater:     templater,
		assetResolver: assetResolver,
	}
	if err := s.registry.RegisterProvider(ctx, defaultProviderName, provider); err != nil {
		log.ReportError(nil, err, "Failed to register default email provider")
	}
	if err := s.registry.SetDefaultProvider(ctx, defaultProviderName); err != nil {
		log.ReportError(nil, err, "Failed to set default email provider")
	}

	if dispatcher != nil {
		dispatchCtx := context.WithoutCancel(ctx)
		if err := dispatcher.Start(dispatchCtx); err != nil {
			log.ReportError(nil, err, "Failed to start email dispatcher")
		}
	}

	return s
}

// NewTemplatedEmail creates a new templated email builder with type-safe props.
// This is a top-level generic function (methods cannot have type parameters in
// Go).
//
// Takes s (Service) which is the email service implementation.
//
// Returns *TemplatedEmailBuilder[PropsT] which is a builder configured
// for template-based email composition.
//
// Panics if s is not the default Service implementation.
func NewTemplatedEmail[PropsT any](s Service) *TemplatedEmailBuilder[PropsT] {
	serviceImpl, ok := s.(*service)
	if !ok {
		panic("email_domain.NewTemplatedEmail requires the default Service implementation")
	}

	return &TemplatedEmailBuilder[PropsT]{
		baseEmailBuilder: &baseEmailBuilder{
			service: serviceImpl,
			params:  &email_dto.SendParams{},
		},
		templater:     serviceImpl.templater,
		assetResolver: serviceImpl.assetResolver,
	}
}

// handleBulkSend sends a batch of emails using the given provider.
//
// When the batch is empty, returns nil without doing anything.
//
// If the provider supports bulk sending, it tries that first. If bulk sending
// fails, it falls back to sending each email one by one.
//
// Takes provider (EmailProviderPort) which handles the actual email sending.
// Takes emails ([]*email_dto.SendParams) which specifies the emails to send.
//
// Returns error when validation fails or all send attempts fail.
func handleBulkSend(ctx context.Context, provider EmailProviderPort, emails []*email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	if len(emails) == 0 {
		return nil
	}

	for i, email := range emails {
		if email == nil {
			return fmt.Errorf("email at index %d: email cannot be nil", i)
		}
		if email.BodyPlain == "" && email.BodyHTML == "" {
			return fmt.Errorf("email at index %d: %w", i, ErrBodyRequired)
		}
		if len(email.To) == 0 {
			return fmt.Errorf("email at index %d: %w", i, ErrRecipientRequired)
		}
	}

	if goroutine.SafeCallValue(ctx, "email.SupportsBulkSending", func() bool { return provider.SupportsBulkSending() }) {
		if err := goroutine.SafeCall(ctx, "email.SendBulk", func() error { return provider.SendBulk(ctx, emails) }); err != nil {
			l.ReportError(nil, err, "Bulk sending failed, falling back to individual sends",
				logger_domain.Int("email_count", len(emails)))
			return sendIndividuallyWithMultiError(ctx, provider, emails)
		}
		return nil
	}

	return sendIndividuallyWithMultiError(ctx, provider, emails)
}

// sendIndividuallyWithMultiError sends emails one by one and gathers any
// failures into a MultiError.
//
// Takes provider (EmailProviderPort) which handles email delivery.
// Takes emails ([]*email_dto.SendParams) which contains the emails to send.
//
// Returns error when one or more emails fail to send; the error is a
// MultiError with details of each failure.
func sendIndividuallyWithMultiError(ctx context.Context, provider EmailProviderPort, emails []*email_dto.SendParams) error {
	ctx, l := logger_domain.From(ctx, log)
	var multiError *MultiError

	for i, email := range emails {
		if err := goroutine.SafeCall(ctx, "email.Send", func() error { return provider.Send(ctx, email) }); err != nil {
			l.ReportError(nil, err, "Failed to send individual email in bulk operation",
				logger_domain.Int("email_index", i),
				logger_domain.String("subject", email.Subject))

			emailError := EmailError{
				Email:       *email,
				Error:       err,
				Attempt:     1,
				LastAttempt: time.Now(),
				NextRetry:   time.Time{},
			}

			if multiError == nil {
				multiError = &MultiError{}
			}
			multiError.Add(&emailError)
		}
	}

	if multiError != nil && multiError.HasErrors() {
		return multiError
	}

	return nil
}

// updateOverallStateFromProvider determines the new overall health state based
// on a provider's state. It follows these rules: 1) unhealthy default provider
// makes overall unhealthy, 2) any unhealthy or degraded provider makes overall
// at least degraded.
//
// Takes currentState (healthprobe_dto.State) which is the current overall
// health state.
// Takes providerState (healthprobe_dto.State) which is the state of the
// provider being checked.
// Takes isDefaultProvider (bool) which indicates if this is the default
// provider.
//
// Returns healthprobe_dto.State which is the updated overall health state.
func updateOverallStateFromProvider(currentState, providerState healthprobe_dto.State, isDefaultProvider bool) healthprobe_dto.State {
	if providerState == healthprobe_dto.StateUnhealthy {
		if isDefaultProvider {
			return healthprobe_dto.StateUnhealthy
		}
		if currentState == healthprobe_dto.StateHealthy {
			return healthprobe_dto.StateDegraded
		}
	}

	if providerState == healthprobe_dto.StateDegraded && currentState == healthprobe_dto.StateHealthy {
		return healthprobe_dto.StateDegraded
	}

	return currentState
}
