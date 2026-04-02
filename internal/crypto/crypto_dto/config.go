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

package crypto_dto

import "time"

const (
	// defaultDirectModeMaxConcurrency limits parallel KMS calls when envelope
	// encryption is disabled.
	defaultDirectModeMaxConcurrency = 50

	// defaultDataKeyCacheTTLMinutes is the default data key cache TTL in minutes.
	defaultDataKeyCacheTTLMinutes = 5

	// defaultDataKeyCacheMaxSize is the default maximum number of cached data
	// keys.
	defaultDataKeyCacheMaxSize = 100
)

// ServiceConfig holds configuration for the crypto service.
type ServiceConfig struct {
	// ActiveKeyID is the key identifier used for new encryptions.
	ActiveKeyID string

	// ProviderType specifies which encryption provider to use.
	ProviderType ProviderType

	// DeprecatedKeyIDs lists key IDs that can still decrypt data but should not
	// be used for new encryptions. These are kept during key rotation so that
	// older encrypted data can still be read.
	DeprecatedKeyIDs []string

	// EnableAutoReEncrypt enables transparent re-encryption of data when
	// decrypted using a deprecated key. When true, data is automatically
	// re-encrypted with the active key, supporting gradual key rotation
	// without explicit migration.
	EnableAutoReEncrypt bool

	// EnableEnvelopeEncryption controls whether batch operations use envelope
	// encryption.
	//
	// When TRUE (default, recommended for most use cases):
	//   - EncryptBatch generates ONE data key via KMS, encrypts all items locally
	//     with AES-GCM
	//   - Performance: ~1ms per batch (1 KMS call + fast local crypto)
	//   - Cost: $0.003 per batch of any size
	//   - Security: Data key is briefly in memory (in SecureBytes with mlock)
	//
	// When FALSE (maximum security mode for compliance-heavy scenarios):
	//   - EncryptBatch calls KMS directly for EACH item (parallelised)
	//   - Performance: ~50-200ms per batch (N parallel KMS calls, limited by
	//     DirectModeMaxConcurrency)
	//   - Cost: $0.003 x N items per batch
	//   - Security: Encryption keys NEVER enter application memory - all crypto in
	//     KMS HSM
	//
	// Set to false for HIPAA/PCI-DSS/SOC2 scenarios where auditors require
	// "encryption keys never in application memory" attestation.
	EnableEnvelopeEncryption bool

	// DirectModeMaxConcurrency limits parallel KMS calls when
	// EnableEnvelopeEncryption is false, preventing KMS rate limit
	// exhaustion (default 5,500 request/sec for Encrypt), where 1 means
	// strictly sequential encryption and the default is 50.
	DirectModeMaxConcurrency int

	// DataKeyCacheTTL specifies how long to cache decrypted data keys (for KMS
	// providers) Set to 0 to disable caching Only used when
	// EnableEnvelopeEncryption is true.
	DataKeyCacheTTL time.Duration

	// DataKeyCacheMaxSize sets the maximum number of data keys to cache.
	// Only used when EnableEnvelopeEncryption is true; 0 disables caching.
	DataKeyCacheMaxSize int
}

// DefaultServiceConfig returns a service config with sensible defaults.
//
// Returns *ServiceConfig which is set up with safe defaults including local
// AES-GCM encryption, envelope encryption turned on, and automatic
// re-encryption turned off.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		ActiveKeyID:              "default",
		ProviderType:             ProviderTypeLocalAESGCM,
		DeprecatedKeyIDs:         []string{},
		EnableAutoReEncrypt:      false,
		EnableEnvelopeEncryption: true,
		DirectModeMaxConcurrency: defaultDirectModeMaxConcurrency,
		DataKeyCacheTTL:          defaultDataKeyCacheTTLMinutes * time.Minute,
		DataKeyCacheMaxSize:      defaultDataKeyCacheMaxSize,
	}
}
