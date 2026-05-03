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

package crypto_domain

import (
	"context"
	"io"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

// DisabledCryptoService implements CryptoServicePort but returns errors for
// all operations. It is used when no encryption key is set, allowing the
// application to start without encryption while giving clear errors if
// encryption is attempted.
type DisabledCryptoService struct{}

var _ CryptoServicePort = (*DisabledCryptoService)(nil)

// NewDisabledCryptoService creates a new disabled crypto service.
//
// Returns *DisabledCryptoService which is a no-op crypto implementation.
func NewDisabledCryptoService() *DisabledCryptoService {
	return &DisabledCryptoService{}
}

// Encrypt returns an empty string and ErrCryptoDisabled since
// encryption is not configured.
//
// Returns string which is always empty.
// Returns error when called, always ErrCryptoDisabled.
func (*DisabledCryptoService) Encrypt(_ context.Context, _ string) (string, error) {
	return "", crypto_dto.ErrCryptoDisabled
}

// Decrypt returns an empty string and ErrCryptoDisabled since encryption
// is not configured.
//
// Returns string which is always empty.
// Returns error when called, always ErrCryptoDisabled.
func (*DisabledCryptoService) Decrypt(_ context.Context, _ string) (string, error) {
	return "", crypto_dto.ErrCryptoDisabled
}

// EncryptWithKey returns an empty string and ErrCryptoDisabled since
// encryption is not set up.
//
// Returns string which is always empty.
// Returns error which is always ErrCryptoDisabled.
func (*DisabledCryptoService) EncryptWithKey(_ context.Context, _ string, _ string) (string, error) {
	return "", crypto_dto.ErrCryptoDisabled
}

// EncryptBatch returns ErrCryptoDisabled since encryption is not configured.
//
// Returns []string which is always nil.
// Returns error when called, as encryption is disabled.
func (*DisabledCryptoService) EncryptBatch(_ context.Context, _ []string) ([]string, error) {
	return nil, crypto_dto.ErrCryptoDisabled
}

// DecryptBatch returns nil and ErrCryptoDisabled since
// encryption is not configured.
//
// Returns []string which is always nil.
// Returns error when called, always returning ErrCryptoDisabled.
func (*DisabledCryptoService) DecryptBatch(_ context.Context, _ []string) ([]string, error) {
	return nil, crypto_dto.ErrCryptoDisabled
}

// RotateKey returns ErrCryptoDisabled since encryption is not configured.
//
// Returns error when called, as this service has encryption disabled.
func (*DisabledCryptoService) RotateKey(_ context.Context, _, _ string) error {
	return crypto_dto.ErrCryptoDisabled
}

// GetActiveKeyID returns an empty key identifier since encryption is not
// set up.
//
// Returns string which is always empty.
// Returns error which is always ErrCryptoDisabled.
func (*DisabledCryptoService) GetActiveKeyID(_ context.Context) (string, error) {
	return "", crypto_dto.ErrCryptoDisabled
}

// DecryptAndReEncrypt returns ErrCryptoDisabled since encryption is not
// configured.
//
// Returns ciphertext (string) which is always empty.
// Returns keyID (string) which is always empty.
// Returns rotated (bool) which is always false.
// Returns err (error) which is always ErrCryptoDisabled.
func (*DisabledCryptoService) DecryptAndReEncrypt(_ context.Context, _ string) (ciphertext, keyID string, rotated bool, err error) {
	return "", "", false, crypto_dto.ErrCryptoDisabled
}

// HealthCheck returns nil since the disabled service is healthy in the sense
// that it correctly reports its disabled state. The service is functioning as
// intended; encryption is not configured.
//
// Returns error when the health check fails, which never occurs for a disabled
// service.
func (*DisabledCryptoService) HealthCheck(_ context.Context) error {
	return nil
}

// EncryptStream returns a writer for encrypting data to the given output.
//
// Returns io.WriteCloser which wraps the output with encryption.
// Returns error when encryption is not configured (ErrCryptoDisabled).
func (*DisabledCryptoService) EncryptStream(_ context.Context, _ io.Writer, _ string) (io.WriteCloser, error) {
	return nil, crypto_dto.ErrCryptoDisabled
}

// DecryptStream returns ErrCryptoDisabled since encryption is not configured.
//
// Returns io.ReadCloser which is always nil.
// Returns error when called, as encryption is not configured.
func (*DisabledCryptoService) DecryptStream(_ context.Context, _ io.Reader) (io.ReadCloser, error) {
	return nil, crypto_dto.ErrCryptoDisabled
}

// NewEncrypt creates a new encryption builder that returns an error.
//
// Returns *EncryptBuilder which is configured with the disabled service.
func (d *DisabledCryptoService) NewEncrypt() *EncryptBuilder {
	return &EncryptBuilder{
		service: d,
	}
}

// NewDecrypt creates a new decryption builder that returns an error.
//
// Returns *DecryptBuilder which is configured with the disabled service.
func (d *DisabledCryptoService) NewDecrypt() *DecryptBuilder {
	return &DecryptBuilder{
		service: d,
	}
}

// NewBatchEncrypt creates a new batch encryption builder.
//
// Returns *BatchEncryptBuilder which is set up with the disabled service.
func (d *DisabledCryptoService) NewBatchEncrypt() *BatchEncryptBuilder {
	return &BatchEncryptBuilder{
		service: d,
	}
}

// NewBatchDecrypt creates a new batch decryption builder.
//
// Returns *BatchDecryptBuilder which is set up with the disabled service.
func (d *DisabledCryptoService) NewBatchDecrypt() *BatchDecryptBuilder {
	return &BatchDecryptBuilder{
		service: d,
	}
}

// NewStreamEncrypt creates a streaming encryption builder that returns an
// error.
//
// Returns *StreamEncryptBuilder which is configured with the disabled service.
func (d *DisabledCryptoService) NewStreamEncrypt() *StreamEncryptBuilder {
	return &StreamEncryptBuilder{
		service: d,
	}
}

// NewStreamDecrypt creates a streaming decryption builder configured to fail.
//
// Returns *StreamDecryptBuilder which is configured with the disabled service.
func (d *DisabledCryptoService) NewStreamDecrypt() *StreamDecryptBuilder {
	return &StreamDecryptBuilder{
		service: d,
	}
}

// RegisterProvider returns ErrCryptoDisabled since provider registration is not
// supported when encryption is disabled.
//
// Returns error always ErrCryptoDisabled.
func (*DisabledCryptoService) RegisterProvider(_ context.Context, _ string, _ EncryptionProvider) error {
	return crypto_dto.ErrCryptoDisabled
}

// SetDefaultProvider returns ErrCryptoDisabled since provider management is not
// supported when encryption is disabled.
//
// Returns error always ErrCryptoDisabled.
func (*DisabledCryptoService) SetDefaultProvider(_ string) error {
	return crypto_dto.ErrCryptoDisabled
}

// GetProviders returns an empty list since no providers are registered when
// encryption is disabled.
//
// Returns []string which is always empty.
func (*DisabledCryptoService) GetProviders(_ context.Context) []string {
	return []string{}
}

// HasProvider returns false since no providers are registered when encryption
// is disabled.
//
// Returns bool which is always false.
func (*DisabledCryptoService) HasProvider(_ string) bool {
	return false
}

// ListProviders returns an empty list since no providers are registered when
// encryption is disabled.
//
// Returns []provider_domain.ProviderInfo which is always empty.
func (*DisabledCryptoService) ListProviders(_ context.Context) []provider_domain.ProviderInfo {
	return []provider_domain.ProviderInfo{}
}

// Close releases resources held by the service.
//
// This is a no-op for the disabled service since there are no providers to
// shut down.
//
// Returns error which is always nil.
func (*DisabledCryptoService) Close(_ context.Context) error {
	return nil
}
