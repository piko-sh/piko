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

package crypto_provider_gcp_kms

import (
	"errors"
	"fmt"
)

// defaultMaxRetries is the default maximum number of retry attempts for
// transient failures.
const defaultMaxRetries = 3

// Config holds settings for the Google Cloud KMS encryption provider.
// This provider sends all cryptographic work to Google Cloud KMS, so master
// keys never leave Google's Hardware Security Modules (HSMs).
type Config struct {
	// ProjectID is the Google Cloud project identifier.
	// Example: "my-project-123456"
	// Required.
	ProjectID string

	// Location is the regional or global location of the key ring
	// (e.g., "global", "us-central1", "europe-west1"). REQUIRED.
	Location string

	// KeyRing is the name of the key ring containing the key; key rings are
	// logical groupings of keys. REQUIRED.
	KeyRing string

	// KeyName is the name of the cryptographic key to use. Required.
	KeyName string

	// MaxRetries is the maximum number of times to retry after a short-lived
	// failure. Default is 3; must be zero or greater.
	MaxRetries int
}

// Validate reports whether the configuration is valid.
//
// Returns error when required fields are missing or MaxRetries is negative.
func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return errors.New("GCP KMS ProjectID is required")
	}
	if c.Location == "" {
		return errors.New("GCP KMS Location is required")
	}
	if c.KeyRing == "" {
		return errors.New("GCP KMS KeyRing is required")
	}
	if c.KeyName == "" {
		return errors.New("GCP KMS KeyName is required")
	}
	if c.MaxRetries < 0 {
		return errors.New("maxRetries cannot be negative")
	}
	return nil
}

// WithDefaults returns a copy of the config with sensible defaults applied.
//
// Returns Config which is a copy with any zero values set to defaults.
func (c Config) WithDefaults() Config {
	if c.MaxRetries == 0 {
		c.MaxRetries = defaultMaxRetries
	}
	return c
}

// KeyResourceName constructs the full resource name for the key.
// The format is:
// projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{key}.
//
// Returns string which is the fully qualified Cloud KMS key resource name.
func (c Config) KeyResourceName() string {
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		c.ProjectID, c.Location, c.KeyRing, c.KeyName)
}
