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

package hmac_challenge

import (
	"errors"
	"time"

	"piko.sh/piko/wdk/clock"
)

const (
	// DefaultTTL is the default challenge validity duration.
	DefaultTTL = 5 * time.Minute

	// minSecretLength is the minimum required secret key length in bytes.
	minSecretLength = 16
)

var (
	// ErrSecretTooShort is returned when the HMAC secret key is shorter than
	// the minimum required length.
	ErrSecretTooShort = errors.New("hmac_challenge: secret must be at least 16 bytes")

	// ErrSecretEmpty is returned when the HMAC secret key is empty.
	ErrSecretEmpty = errors.New("hmac_challenge: secret cannot be empty")

	// ErrNegativeTTL is returned when the TTL is negative.
	ErrNegativeTTL = errors.New("hmac_challenge: TTL must not be negative")

	// ErrInvalidAction is returned when an action name contains the token
	// separator character.
	ErrInvalidAction = errors.New("hmac_challenge: action name contains invalid character")

	// errHealthCheckFailed is returned when the health check round-trip
	// verification returns false.
	errHealthCheckFailed = errors.New("hmac_challenge: health check verification returned false")
)

// Config holds configuration for the HMAC challenge captcha provider.
type Config struct {
	// Clock provides the time source for token generation and TTL checks.
	// Defaults to the real system clock when nil.
	Clock clock.Clock

	// Secret is the HMAC secret key used to sign and verify challenge tokens.
	// Must be at least 16 bytes.
	Secret []byte

	// TTL is how long a generated challenge token remains valid. Defaults to
	// 5 minutes if zero.
	TTL time.Duration
}

// validate checks that the configuration is valid.
//
// Returns error when validation fails.
func (c *Config) validate() error {
	if len(c.Secret) == 0 {
		return ErrSecretEmpty
	}
	if len(c.Secret) < minSecretLength {
		return ErrSecretTooShort
	}
	if c.TTL < 0 {
		return ErrNegativeTTL
	}
	return nil
}
