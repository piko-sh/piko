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

// This file contains email service related container methods.

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/deadletter/deadletter_adapters"
	"piko.sh/piko/internal/email/email_adapters/asset_resolver"
	"piko.sh/piko/internal/email/email_adapters/provider_stdout"
	templater_email_adapter "piko.sh/piko/internal/email/email_adapters/templater_adapter"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

// AddEmailProvider registers a named email provider for sending emails.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (EmailProviderPort) which handles email delivery.
func (c *Container) AddEmailProvider(name string, provider email_domain.EmailProviderPort) {
	if c.emailProviders == nil {
		c.emailProviders = make(map[string]email_domain.EmailProviderPort)
	}
	c.emailProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "EmailProvider-"+name, provider)
}

// SetEmailDefaultProvider sets the default email provider to use when none is
// given.
//
// Takes name (string) which is the provider name to set as default.
func (c *Container) SetEmailDefaultProvider(name string) {
	c.emailDefaultProvider = name
}

// SetEmailDispatcherConfig configures the email dispatcher for async email
// sending.
//
// Takes config (*email_dto.DispatcherConfig) which specifies the dispatcher
// settings.
func (c *Container) SetEmailDispatcherConfig(config *email_dto.DispatcherConfig) {
	c.emailDispatcherConfig = config
	c.hasEmailDispatcher = true
}

// SetEmailDeadLetterAdapter sets the dead letter queue adapter for failed
// email deliveries.
//
// If the adapter implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes dlq (email_domain.DeadLetterPort) which handles failed email messages.
func (c *Container) SetEmailDeadLetterAdapter(dlq email_domain.DeadLetterPort) {
	c.emailDeadLetterAdapter = dlq
	registerCloseableForShutdown(c.GetAppContext(), "EmailDeadLetterAdapter", dlq)
}

// GetEmailService returns the email service, creating a default one if none
// was provided.
//
// Returns email_domain.Service which is the configured email service.
// Returns error when the default email service cannot be created.
func (c *Container) GetEmailService() (email_domain.Service, error) {
	c.emailOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.emailServiceOverride != nil {
			l.Internal("Using provided EmailService override.")
			c.emailService = c.emailServiceOverride
			return
		}
		c.createDefaultEmailService()
	})
	return c.emailService, c.emailErr
}

// createDefaultEmailService builds and sets up the default email service.
//
// It gets the email templater, picks a base provider, sets up an optional
// dispatcher, and creates an asset resolver for CID embedding. Any errors are
// stored in c.emailErr rather than returned.
func (c *Container) createDefaultEmailService() {
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default EmailService...")

	emailTemplater := c.GetEmailTemplateService()
	if emailTemplater == nil {
		c.emailErr = errors.New("email template service is not initialised; ensure the daemon builder sets it via SetEmailTemplateService")
		return
	}
	templaterAdapter := templater_email_adapter.New(emailTemplater)

	baseName, baseProvider, err := c.selectEmailBaseProvider(ctx)
	if err != nil {
		c.emailErr = err
		return
	}

	dispatcher := c.createEmailDispatcher(baseProvider)
	c.emailDispatcher = dispatcher

	registryService, err := c.GetRegistryService()
	if err != nil {
		c.emailErr = fmt.Errorf("failed to get registry service for email asset resolver: %w", err)
		return
	}
	assetResolver := asset_resolver.New(registryService)

	s := email_domain.NewServiceWithProviderAndDispatcher(c.GetAppContext(), baseProvider, dispatcher, templaterAdapter, assetResolver)

	if err := c.registerEmailProviders(s, baseName); err != nil {
		c.emailErr = err
		return
	}

	if dispatcher != nil {
		if err := s.RegisterDispatcher(ctx, dispatcher); err != nil {
			c.emailErr = fmt.Errorf("failed to register and start email dispatcher: %w", err)
			return
		}
	}

	c.emailService = s
}

// selectEmailBaseProvider selects the base email provider based on
// configuration.
//
// Returns string which is the name of the selected provider.
// Returns email_domain.EmailProviderPort which is the selected email
// provider.
// Returns error when the configured provider is not registered or the
// default provider fails to initialise.
func (c *Container) selectEmailBaseProvider(ctx context.Context) (baseName string, baseProvider email_domain.EmailProviderPort, err error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(c.emailProviders) > 0 {
		if c.emailDefaultProvider != "" {
			baseName = c.emailDefaultProvider
			baseProvider = c.emailProviders[baseName]
			if baseProvider == nil {
				return "", nil, fmt.Errorf("email default provider %q not registered", baseName)
			}
		} else if p, ok := c.emailProviders[email_dto.EmailNameDefault]; ok {
			baseName = email_dto.EmailNameDefault
			baseProvider = p
		} else {
			for n, p := range c.emailProviders {
				baseName, baseProvider = n, p
				break
			}
		}
	} else {
		baseName = email_dto.EmailNameDefault
		baseProvider, err = provider_stdout.New(ctx)
		if err != nil {
			return "", nil, fmt.Errorf("failed to initialise default stdout email provider: %w", err)
		}
		l.Internal("Using default stdout email provider (no custom providers registered)")
	}
	return baseName, baseProvider, nil
}

// createEmailDispatcher creates an email dispatcher if configured.
//
// Takes baseProvider (email_domain.EmailProviderPort) which provides the
// underlying email sending capability.
//
// Returns email_domain.EmailDispatcherPort which is the configured dispatcher,
// or nil if email dispatching is not enabled.
func (c *Container) createEmailDispatcher(baseProvider email_domain.EmailProviderPort) email_domain.EmailDispatcherPort {
	if !c.hasEmailDispatcher {
		return nil
	}

	config := c.emailDispatcherConfig
	if config == nil {
		config = new(email_dto.DefaultDispatcherConfig())
	}
	dlq := c.emailDeadLetterAdapter
	if dlq == nil {
		dlq = deadletter_adapters.NewMemoryDeadLetterQueue[*email_dto.DeadLetterEntry]()
	}
	return email_domain.NewEmailDispatcher(baseProvider, dlq, config)
}

// registerEmailProviders sets up all email providers with the service.
//
// Takes s (email_domain.Service) which receives the provider registrations.
// Takes baseName (string) which is the default provider name.
//
// Returns error when provider registration or default setup fails.
func (c *Container) registerEmailProviders(s email_domain.Service, baseName string) error {
	if err := c.registerAdditionalEmailProviders(s, baseName); err != nil {
		return fmt.Errorf("registering additional email providers: %w", err)
	}

	if err := c.ensureDefaultEmailProvider(s, baseName); err != nil {
		return fmt.Errorf("ensuring default email provider: %w", err)
	}

	return c.setDefaultEmailProvider(s, baseName)
}

// registerAdditionalEmailProviders registers all user-supplied email providers
// except the base provider which is already registered by the service constructor.
//
// Takes s (email_domain.Service) which receives the provider registrations.
// Takes baseName (string) which identifies the already-registered base provider.
//
// Returns error when a provider fails to register.
func (c *Container) registerAdditionalEmailProviders(s email_domain.Service, baseName string) error {
	ctx := c.GetAppContext()
	for name, provider := range c.emailProviders {
		if name == baseName {
			continue
		}
		if err := s.RegisterProvider(ctx, name, provider); err != nil {
			return fmt.Errorf("failed to register email provider %q: %w", name, err)
		}
	}
	return nil
}

// ensureDefaultEmailProvider creates and registers a default stdout provider
// if none were supplied.
//
// Takes s (email_domain.Service) which handles provider registration.
// Takes baseName (string) which is the base provider to try first.
//
// Returns error when provider creation or registration fails.
func (c *Container) ensureDefaultEmailProvider(s email_domain.Service, baseName string) error {
	if len(c.emailProviders) > 0 {
		return nil
	}

	ctx := c.GetAppContext()

	if err := s.RegisterProvider(ctx, email_dto.EmailNameDefault, c.emailProviders[baseName]); err == nil {
		return nil
	}

	stdoutProvider, providerErr := provider_stdout.New(ctx)
	if providerErr != nil {
		return fmt.Errorf("failed to create default stdout email provider: %w", providerErr)
	}
	if err := s.RegisterProvider(ctx, email_dto.EmailNameDefault, stdoutProvider); err != nil {
		return fmt.Errorf("failed to register default provider on default email service: %w", err)
	}
	return nil
}

// setDefaultEmailProvider sets the default email provider preference.
//
// Takes s (email_domain.Service) which provides the email service to set up.
// Takes baseName (string) which specifies the fallback provider name.
//
// Returns error when the provider cannot be set.
func (c *Container) setDefaultEmailProvider(s email_domain.Service, baseName string) error {
	defaultName := baseName
	if c.emailDefaultProvider != "" {
		defaultName = c.emailDefaultProvider
	}
	if err := s.SetDefaultProvider(c.GetAppContext(), defaultName); err != nil {
		return fmt.Errorf("failed to set default email provider to %q: %w", defaultName, err)
	}
	return nil
}

// SetEmailTemplateService sets the email template service.
// Called by the daemon builder to set the service built using the
// selected manifest runner.
//
// Takes s (EmailTemplateService) which provides email template operations.
func (c *Container) SetEmailTemplateService(s templater_domain.EmailTemplateService) {
	c.emailTemplateService = s
}

// GetEmailTemplateService returns the EmailTemplateService previously registered
// by the daemon builder.
//
// Returns templater_domain.EmailTemplateService which provides email template
// operations, or nil if not yet initialised.
func (c *Container) GetEmailTemplateService() templater_domain.EmailTemplateService {
	return c.emailTemplateService
}
