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

package config_domain

import (
	"context"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

// SecretManager coordinates the lifecycle of all Secret[T] instances,
// providing registration, stats tracking, batch refresh, and graceful shutdown.
// It implements handlerShutdown and contextShutdown interfaces.
type SecretManager struct {
	// secrets tracks all registered secret holders for lifecycle management.
	secrets map[secretCloser]struct{}

	// mu protects the secrets map for concurrent access.
	mu sync.RWMutex
}

var (
	// globalSecretManager holds the lazily initialised singleton SecretManager.
	globalSecretManager *SecretManager

	// globalSecretManagerOnce guards one-time initialisation of globalSecretManager.
	globalSecretManagerOnce sync.Once
)

// SecretStats provides statistics about secret usage.
type SecretStats struct {
	// TotalSecrets is the number of secrets that have been registered.
	TotalSecrets int

	// ActiveSecrets is the number of secrets that have been resolved.
	ActiveSecrets int
}

// Stats returns current statistics about registered secrets.
//
// Returns SecretStats which contains counts of total and active secrets.
//
// Safe for concurrent use.
func (sm *SecretManager) Stats() SecretStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	active := 0
	for secret := range sm.secrets {
		if secret.isResolved() {
			active++
		}
	}

	return SecretStats{
		TotalSecrets:  len(sm.secrets),
		ActiveSecrets: active,
	}
}

// Shutdown closes all registered secrets, releasing their resources. This
// should be called during application shutdown.
//
// The context can be used to set a timeout for the shutdown process. If the
// context is cancelled, remaining secrets will still be force-closed.
//
// Returns error when the last secret fails to close.
//
// Safe for concurrent use. Collects secrets under lock, then closes each
// without holding the lock.
func (sm *SecretManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	secretLog.Internal("Shutting down secret manager",
		logger_domain.Int("total_secrets", len(sm.secrets)))

	secretsToClose := make([]secretCloser, 0, len(sm.secrets))
	for secret := range sm.secrets {
		secretsToClose = append(secretsToClose, secret)
	}
	sm.secrets = make(map[secretCloser]struct{})
	sm.mu.Unlock()

	var lastErr error
	for _, secret := range secretsToClose {
		select {
		case <-ctx.Done():
			secretLog.Warn("Secret manager shutdown interrupted, force-closing remaining secrets")
		default:
		}

		if err := secret.Close(); err != nil {
			secretLog.Warn("Failed to close secret during shutdown",
				logger_domain.Error(err))
			lastErr = err
		}
	}

	secretLog.Internal("Secret manager shutdown complete")
	return lastErr
}

// Count returns the number of registered secrets.
//
// Returns int which is the current count of secrets.
//
// Safe for concurrent use.
func (sm *SecretManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.secrets)
}

// register adds a secret to the manager.
// This is called automatically by Secret[T].UnmarshalText.
//
// Takes secret (secretCloser) which is the secret to track for cleanup.
//
// Safe for concurrent use; protects the secrets map with a mutex.
func (sm *SecretManager) register(secret secretCloser) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.secrets[secret] = struct{}{}
}

// unregister removes a secret from the manager.
// This is called automatically by Secret[T].Close.
//
// Takes secret (secretCloser) which is the secret to remove.
//
// Safe for concurrent use.
func (sm *SecretManager) unregister(secret secretCloser) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.secrets, secret)
}

// GetSecretManager returns the singleton secret manager.
//
// Returns *SecretManager which is the global secret manager instance,
// created on first call.
func GetSecretManager() *SecretManager {
	globalSecretManagerOnce.Do(func() {
		globalSecretManager = &SecretManager{
			secrets: make(map[secretCloser]struct{}),
		}
	})
	return globalSecretManager
}

// ResetSecretManager resets the global secret manager singleton. This is
// mainly for testing to ensure test isolation.
//
// Closes all registered secrets before resetting.
//
// Safe for concurrent use.
func ResetSecretManager() {
	if globalSecretManager != nil {
		globalSecretManager.mu.Lock()
		secretsToClose := make([]secretCloser, 0, len(globalSecretManager.secrets))
		for secret := range globalSecretManager.secrets {
			secretsToClose = append(secretsToClose, secret)
		}
		globalSecretManager.secrets = make(map[secretCloser]struct{})
		globalSecretManager.mu.Unlock()

		for _, secret := range secretsToClose {
			_ = secret.Close()
		}
	}
	globalSecretManagerOnce = sync.Once{}
	globalSecretManager = nil
}
