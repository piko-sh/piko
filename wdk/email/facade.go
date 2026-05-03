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

package email

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
)

// Service provides email sending capabilities through multiple providers.
type Service = email_domain.Service

// ServiceOption configures the email service.
type ServiceOption = email_domain.ServiceOption

// ProviderPort represents the interface that all email providers must
// implement. Implement it to create custom or mock providers.
type ProviderPort = email_domain.EmailProviderPort

// SendParams holds all the data needed to send a single email.
type SendParams = email_dto.SendParams

// Attachment represents a file attached to an email.
type Attachment = email_dto.Attachment

// MultiError is an error type that collects multiple failures from a bulk send
// operation, allowing for partial success.
type MultiError = email_domain.MultiError

// DispatcherConfig holds the settings for the background email dispatcher.
// It controls batching, retries, and dead-lettering.
type DispatcherConfig = email_dto.DispatcherConfig

// DispatcherStats provides runtime statistics for monitoring the email
// dispatcher.
type DispatcherStats = email_domain.DispatcherStats

// DeadLetterEntry represents an email that has failed delivery for good.
type DeadLetterEntry = email_dto.DeadLetterEntry

// EmailBuilder provides a fluent interface for constructing email messages.
type EmailBuilder = email_domain.EmailBuilder

// TemplatedEmailBuilder provides a fluent interface for constructing templated
// emails.
type TemplatedEmailBuilder[PropsT any] = email_domain.TemplatedEmailBuilder[PropsT]

const (
	// EmailNameDefault is the default name used for email addresses.
	EmailNameDefault = email_dto.EmailNameDefault
)

// NewService creates a new email service instance.
//
// Takes defaultProviderName (string) which specifies the provider to use when
// none is specified.
// Takes opts (...ServiceOption) which configures service limits and behaviour.
//
// Returns Service which is the configured email service ready for use.
//
// Example:
//
//	service := email.NewService("smtp")
//	provider, _ := email_provider_smtp.NewProvider(ctx, config)
//	service.RegisterProvider("smtp", provider)
func NewService(defaultProviderName string, opts ...ServiceOption) Service {
	return email_domain.NewServiceWithDefaultProvider(defaultProviderName, opts...)
}

// GetDefaultService returns the email service initialised by the framework.
//
// Returns Service which is the service instance ready for use.
// Returns error when the framework has not been bootstrapped.
func GetDefaultService() (Service, error) {
	service, err := bootstrap.GetEmailService()
	if err != nil {
		return nil, fmt.Errorf("email: get default service: %w", err)
	}
	return service, nil
}

// NewEmailBuilder creates a new email builder for composing HTML or plain text
// emails.
//
// Takes service (Service) which is the email service to use for sending.
//
// Returns *EmailBuilder which provides a fluent interface for building emails.
// Returns error when service is nil.
//
// Example:
//
//	service := email.NewService("smtp")
//	builder, err := email.NewEmailBuilder(service)
//	if err != nil {
//	    return err
//	}
//	err = builder.To("user@example.com").Subject("Hello").BodyHTML("<p>Hi!</p>").Do(ctx)
func NewEmailBuilder(service Service) (*EmailBuilder, error) {
	if service == nil {
		return nil, errors.New("email: service must not be nil")
	}
	return service.NewEmail(), nil
}

// NewEmailBuilderFromDefault creates a new email builder using the framework's
// bootstrapped service.
//
// Returns *EmailBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
func NewEmailBuilderFromDefault() (*EmailBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("email: get default service: %w", err)
	}
	return NewEmailBuilder(service)
}

// NewTemplatedEmailBuilder creates a new type-safe templated email builder.
//
// Takes service (Service) which is the email service to use for sending.
//
// The generic type parameter PropsT represents the Props type of your email
// template.
//
// Returns *TemplatedEmailBuilder[PropsT] which provides a fluent interface for
// building templated emails.
// Returns error when service is nil.
//
// Example:
//
//	type WelcomeProps struct {
//	    Username  string
//	    TrialDays int
//	}
//	service := email.NewService("smtp")
//	builder, err := email.NewTemplatedEmailBuilder[WelcomeProps](service)
//	if err != nil {
//	    return err
//	}
//	err = builder.To("user@example.com").
//	    Subject("Welcome!").
//	    Props(WelcomeProps{Username: "john", TrialDays: 14}).
//	    BodyTemplate("emails/welcome.pk").
//	    Do(ctx)
func NewTemplatedEmailBuilder[PropsT any](service Service) (*TemplatedEmailBuilder[PropsT], error) {
	if service == nil {
		return nil, errors.New("email: service must not be nil")
	}
	builder, err := email_domain.NewTemplatedEmail[PropsT](service)
	if err != nil {
		return nil, fmt.Errorf("email: %w", err)
	}
	return builder, nil
}

// NewTemplatedEmailBuilderFromDefault creates a new templated email builder
// using the framework's bootstrapped service.
//
// The generic type parameter PropsT represents the Props type of your email
// template.
//
// Returns *TemplatedEmailBuilder[PropsT] which is the configured builder.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	type WelcomeProps struct {
//	    Username  string
//	    TrialDays int
//	}
//	builder, err := email.NewTemplatedEmailBuilderFromDefault[WelcomeProps]()
//	if err != nil {
//	    return err
//	}
//	err = builder.To("user@example.com").
//	    Subject("Welcome!").
//	    Props(WelcomeProps{Username: "john", TrialDays: 14}).
//	    BodyTemplate("emails/welcome.pk").
//	    Do(ctx)
func NewTemplatedEmailBuilderFromDefault[PropsT any]() (*TemplatedEmailBuilder[PropsT], error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf("email: get default service for template: %w", err)
	}
	return NewTemplatedEmailBuilder[PropsT](service)
}
