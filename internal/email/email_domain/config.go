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

	// defaultServiceRetryHeapMax is the largest number of items the retry heap
	// can hold.
	defaultServiceRetryHeapMax = 50000
)

// ServiceConfig holds configuration for the email service, including limits for
// DoS protection.
type ServiceConfig struct {
	// MaxTotalRecipients is the maximum total number of recipients
	// (To + Cc + Bcc) allowed per email, protecting against memory
	// exhaustion attacks (default: 100).
	MaxTotalRecipients int

	// MaxPayloadSizeBytes is the maximum total size in bytes for email
	// content including BodyHTML, BodyPlain, and all attachment contents,
	// protecting against memory exhaustion attacks (default: 26214400 /
	// 25 MB).
	MaxPayloadSizeBytes int64

	// MaxRetryHeapSize is the maximum number of emails that can be queued
	// for retry, where new attempts are rejected when full, protecting
	// against memory exhaustion from unbounded retry queues (default:
	// 50000).
	MaxRetryHeapSize int
}

// ServiceOption is a function that changes a ServiceConfig.
// It allows the functional options pattern for service creation.
type ServiceOption func(*ServiceConfig)

// WithMaxTotalRecipients sets the maximum total number of recipients
// (To + Cc + Bcc) allowed per email.
//
// Takes limit (int) which specifies the maximum number of recipients. Values
// of zero or less are ignored.
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
// This includes the email body (HTML and plain text) and all attachments.
//
// Takes limit (int64) which specifies the maximum size. Values of zero or
// less are ignored.
//
// Returns ServiceOption which sets the payload size limit.
func WithMaxPayloadSizeBytes(limit int64) ServiceOption {
	return func(config *ServiceConfig) {
		if limit > 0 {
			config.MaxPayloadSizeBytes = limit
		}
	}
}

// WithMaxRetryHeapSize sets the maximum number of emails that can be queued
// for retry. When the retry heap is full, new retry attempts are rejected and
// fail quickly.
//
// Takes limit (int) which specifies the maximum heap size. Values less than or
// equal to zero are ignored.
//
// Returns ServiceOption which sets the retry heap size limit.
func WithMaxRetryHeapSize(limit int) ServiceOption {
	return func(config *ServiceConfig) {
		if limit > 0 {
			config.MaxRetryHeapSize = limit
		}
	}
}

// defaultServiceConfig returns a ServiceConfig with safe defaults for
// denial-of-service protection.
//
// Returns ServiceConfig which contains safe default limits for recipients,
// payload size, and retry heap.
func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxTotalRecipients:  defaultMaxTotalRecipients,
		MaxPayloadSizeBytes: defaultMaxPayloadSizeBytes,
		MaxRetryHeapSize:    defaultServiceRetryHeapMax,
	}
}
