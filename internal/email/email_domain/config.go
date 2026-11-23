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

const (
	// defaultMaxTotalRecipients is the default service-level limit for recipients.
	defaultMaxTotalRecipients = 100

	// defaultMaxPayloadSizeBytes is the largest allowed payload size (25 MiB).
	defaultMaxPayloadSizeBytes = 25 * 1024 * 1024
)

// ServiceConfig holds configuration for the email service, including limits for DoS
// protection.
type ServiceConfig struct {
	// initialProviderName is the registry key for a provider supplied at construction.
	initialProviderName string

	// MaxTotalRecipients caps To, Cc and Bcc recipients, defaulting to 100, protecting
	// against memory exhaustion.
	MaxTotalRecipients int

	// MaxPayloadSizeBytes caps body and attachment bytes, defaulting to 25 MiB.
	MaxPayloadSizeBytes int64
}

// ServiceOption is a function that changes a ServiceConfig.
type ServiceOption func(*ServiceConfig)

// WithInitialProviderName names the provider supplied at service construction.
//
// Takes name (string) which is the provider registry key, with an empty name ignored.
//
// Returns ServiceOption which sets the initial provider name.
func WithInitialProviderName(name string) ServiceOption {
	return func(config *ServiceConfig) {
		if name != "" {
			config.initialProviderName = name
		}
	}
}

// WithMaxTotalRecipients sets the maximum total number of recipients (To + Cc + Bcc)
// allowed per email.
//
// Takes limit (int) which specifies the maximum number of recipients. Values of zero or
// less are ignored.
//
// Returns ServiceOption which sets the recipient limit on a service.
func WithMaxTotalRecipients(limit int) ServiceOption {
	return func(config *ServiceConfig) {
		if limit > 0 {
			config.MaxTotalRecipients = limit
		}
	}
}

// WithMaxPayloadSizeBytes sets the maximum payload size in bytes.
//
// Takes limit (int64) which specifies the maximum size. Values of zero or less are
// ignored.
//
// Returns ServiceOption which sets the payload size limit.
func WithMaxPayloadSizeBytes(limit int64) ServiceOption {
	return func(config *ServiceConfig) {
		if limit > 0 {
			config.MaxPayloadSizeBytes = limit
		}
	}
}

// defaultServiceConfig returns a ServiceConfig with safe defaults for denial-of-service
// protection.
//
// Returns ServiceConfig which contains safe default limits for recipients and payload
// size. The retry heap is bounded by the dispatcher's own configuration.
func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		initialProviderName: defaultProviderName,
		MaxTotalRecipients:  defaultMaxTotalRecipients,
		MaxPayloadSizeBytes: defaultMaxPayloadSizeBytes,
	}
}
