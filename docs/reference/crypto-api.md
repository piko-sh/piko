---
title: Crypto API
description: Envelope encryption, key rotation, streaming, and KMS-backed providers.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 210
---

# Crypto API

Piko's crypto service provides authenticated encryption with automatic key rotation. Ciphertext envelopes are self-describing (key ID and provider type embedded) so decryption works across rotations without migration. Backends are local AES-GCM, AWS KMS, GCP KMS, `Azure Key Vault`, and HashiCorp Vault. Source file: [`wdk/crypto/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/crypto/facade.go).

## Top-level operations

```go
func Encrypt(ctx context.Context, plaintext string) (string, error)
func Decrypt(ctx context.Context, ciphertext string) (string, error)
func EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)
func DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)
```

Batch calls wrap one data key across all inputs, which is cheaper for KMS-backed providers.

## Builders

```go
func NewEncryptBuilder(service ServicePort) *EncryptBuilder
func NewDecryptBuilder(service ServicePort) *DecryptBuilder
func NewBatchEncryptBuilder(service ServicePort) *BatchEncryptBuilder
func NewBatchDecryptBuilder(service ServicePort) *BatchDecryptBuilder
func NewStreamEncryptBuilder(service ServicePort) *StreamEncryptBuilder
func NewStreamDecryptBuilder(service ServicePort) *StreamDecryptBuilder
```

Each has a `*FromDefault()` variant that resolves the bootstrap-configured service. Streaming builders operate in O(64 KB) memory regardless of input size.

## Service

```go
func NewService(ctx context.Context, cache cache_domain.Service, config *ServiceConfig, opts ...ServiceOption) (ServicePort, error)
func GetDefaultService() (ServicePort, error)
func DefaultServiceConfig() *ServiceConfig
func DefaultKeyRotationPolicy() *KeyRotationPolicy
func WithLocalProviderFactory(factory LocalProviderFactory) ServiceOption
```

The cache parameter backs the data-key cache (required for KMS backends to avoid per-call KMS round trips).

## Types

| Type | Purpose |
|---|---|
| `ServicePort` | Service interface. |
| `EncryptionProvider` | Backend interface. |
| `LocalProviderFactory` | Factory for local-only encryption. |
| `ServiceConfig` | Service configuration. |
| `KeyInfo`, `KeyStatus`, `DataKey` | Key-management types. |
| `KeyRotationPolicy` | Rotation rules. |
| `EncryptRequest`, `DecryptRequest`, `EncryptResponse`, `DecryptResponse`, `GenerateDataKeyRequest` | DTOs used by providers. |
| `EncryptionError` | Typed error. |

## Constants

| Group | Values |
|---|---|
| Provider | `ProviderTypeLocalAESGCM`, `ProviderTypeAWSKMS`, `ProviderTypeGCPKMS`, `ProviderTypeAzureKeyVault`, `ProviderTypeHashiCorpVault` |
| Key status | `KeyStatusActive`, `KeyStatusDeprecated`, `KeyStatusDisabled`, `KeyStatusDestroyed` |

## Errors

`ErrKeyNotFound`, `ErrInvalidKey`, `ErrInvalidCiphertext`, `ErrDecryptionFailed`, `ErrEncryptionFailed`, `ErrProviderUnavailable`, `ErrInvalidProvider`, `ErrEmptyPlaintext`, `ErrEmptyCiphertext`, `ErrContextMismatch`, `ErrKeyRotationInProgress`.

## Providers

| Sub-package | Backend |
|---|---|
| `crypto_provider_local_aes_gcm` | Local AES-256-GCM. |
| `crypto_provider_aws_kms` | AWS KMS. |
| `crypto_provider_gcp_kms` | GCP KMS. |
| `crypto_provider_azure_keyvault` | `Azure Key Vault`. |
| `crypto_provider_hashicorp_vault` | HashiCorp Vault. |
| `crypto_streaming` | Streaming envelope helpers. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithCryptoService(service)` | Registers a fully configured service. |
| `piko.WithCryptoProvider(name, provider)` | Registers a provider. |
| `piko.WithDefaultCryptoProvider(name)` | Marks the default. |

## See also

- [Secrets API](secrets-api.md) for key material that crypto consumes.
- [How to security](../how-to/security.md) for encryption-at-rest patterns.
