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

package crypto_provider_aws_kms

import (
	"errors"
)

// defaultMaxRetries is the default maximum number of retry attempts for
// transient failures.
const defaultMaxRetries = 3

// Config holds configuration for the AWS KMS encryption provider. This provider
// delegates all cryptographic operations to AWS Key Management Service,
// ensuring that master keys never leave AWS's Hardware Security Modules (HSMs).
type Config struct {
	// KeyID is the identifier for the KMS key. Can be one of:
	//   - Key ID: 1234abcd-12ab-34cd-56ef-1234567890ab
	//   - Key ARN, e.g.:
	//     arn:aws:kms:us-east-1:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab
	//   - Alias name: alias/my-key
	//   - Alias ARN: arn:aws:kms:us-east-1:123456789012:alias/my-key
	// Required.
	KeyID string

	// Region is the AWS region where the KMS key is located (e.g., "us-east-1",
	// "eu-west-1"). REQUIRED.
	Region string

	// MaxRetries is the maximum number of retry attempts for short-lived failures.
	// Default is 3; must be zero or greater.
	MaxRetries int

	// EnableMetrics enables CloudWatch metric publishing for KMS operations.
	// Default is false.
	EnableMetrics bool
}

// Validate reports whether the configuration is valid.
//
// Returns error when required fields are missing or values are invalid.
func (c *Config) Validate() error {
	if c.KeyID == "" {
		return errors.New("AWS KMS KeyID is required")
	}
	if c.Region == "" {
		return errors.New("AWS KMS Region is required")
	}
	if c.MaxRetries < 0 {
		return errors.New("maxRetries cannot be negative")
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
//
// Returns Config which has default values set for any fields that were not
// specified.
func (c Config) WithDefaults() Config {
	if c.MaxRetries == 0 {
		c.MaxRetries = defaultMaxRetries
	}
	return c
}
