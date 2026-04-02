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

package piko

import (
	"piko.sh/piko/internal/config/config_domain"
)

// Secret provides lazy-loaded, secure secret handling for configuration values.
// Unlike regular config fields, Secret[T] stores the resolver reference and
// resolves on-demand when Acquire() is called rather than at startup.
//
// Type parameter T should be:
//   - string: For text secrets (memory zeroing is best-effort)
//   - []byte: For binary secrets (stored in SecureBytes with mmap+mlock)
//
// Security characteristics:
//   - Lazy loading: Secrets don't enter memory until needed
//   - Explicit lifecycle: Acquire() + Release() for scoped access
//   - Reference counting: Multiple concurrent Acquire() calls are safe
//   - Automatic registration: Secrets register with SecretManager for shutdown
//   - Finaliser safety net: Unreleased handles are cleaned up by GC
type Secret[T any] = config_domain.Secret[T]

// SecretHandle provides scoped access to a secret value.
// It must be released when no longer needed to allow the secret to be refreshed
// or cleaned up during shutdown.
//
// SecretHandle implements io.Closer for use with defer.
type SecretHandle[T any] = config_domain.SecretHandle[T]

// SecretManager coordinates the lifecycle of all Secret[T] instances, handling
// registration, stats tracking, and graceful shutdown. SecretManager is a
// singleton; use GetSecretManager to access it.
type SecretManager = config_domain.SecretManager

// SecretStats provides statistics about secret usage.
type SecretStats = config_domain.SecretStats

var (
	// ErrSecretNotSet is returned when trying to acquire a secret that was never
	// populated.
	ErrSecretNotSet = config_domain.ErrSecretNotSet

	// ErrSecretClosed is returned when trying to acquire a secret that has been
	// closed.
	ErrSecretClosed = config_domain.ErrSecretClosed

	// ErrSecretResolutionFailed is returned when the resolver fails to resolve the
	// secret.
	ErrSecretResolutionFailed = config_domain.ErrSecretResolutionFailed

	// ErrSecretHandleClosed is returned when trying to use a handle that has
	// already been released.
	ErrSecretHandleClosed = config_domain.ErrSecretHandleClosed

	// ErrNoResolver is returned when no resolver is available for the secret's
	// prefix.
	ErrNoResolver = config_domain.ErrNoResolver
)

// GetSecretManager returns the singleton secret manager.
// Use this to access secret statistics or manually trigger shutdown.
//
// Returns *SecretManager which provides access to secret management operations.
//
// Example:
// stats := piko.GetSecretManager().Stats()
// log.Info("Active secrets", "count", stats.TotalSecrets)
func GetSecretManager() *SecretManager {
	return config_domain.GetSecretManager()
}
