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
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/wdk/crypto"
)

// Provider implements cloud-backed encryption using AWS Key Management
// Service (KMS). The master encryption key never leaves AWS's Hardware
// Security Modules (HSMs).
//
// Features:
//   - Master key stored in FIPS 140-2 Level 2 validated HSMs
//   - Integrated AWS CloudTrail audit logging
//   - Automatic key rotation (when enabled on the KMS key)
//   - Fine-grained IAM access control
//   - Support for envelope encryption via GenerateDataKey
//
// Performance characteristics:
//   - Network latency: ~10-50ms per operation (region-dependent)
//   - Rate limits: 5,500 request/sec for Encrypt, 10,000 request/sec for Decrypt
//   - Cost: $0.03 per 10,000 requests
//
// Use envelope encryption for bulk operations to minimise KMS calls.
// Circuit breaker protection is provided by the crypto service layer,
// not by this provider directly.
type Provider struct {
	// client is the AWS KMS client used for encryption operations.
	client *kms.Client

	// keyID is the identifier returned in encryption responses.
	keyID string

	// maxRetries is the maximum number of retry attempts; 0 means no retries.
	maxRetries int
}

var _ crypto.EncryptionProvider = (*Provider)(nil)

// Type returns the provider type.
//
// Returns crypto.ProviderType which identifies this as an AWS KMS provider.
func (*Provider) Type() crypto.ProviderType {
	return crypto.ProviderTypeAWSKMS
}

// Encrypt encrypts plaintext using AWS KMS and returns base64-encoded
// ciphertext.
//
// The ciphertext is a self-contained blob that includes the encrypted data
// encryption key (DEK), the encrypted data, and metadata about the encryption
// context. The returned ciphertext is base64-encoded for safe storage in
// databases.
//
// Takes request (*crypto.EncryptRequest) which contains the plaintext to
// encrypt and an optional key ID override.
//
// Returns *crypto.EncryptResponse which contains the base64-encoded
// ciphertext, key ID used, and provider type.
// Returns error when the plaintext is empty or the KMS encryption fails.
//
// For bulk encryption, use GenerateDataKey and encrypt locally to
// reduce KMS calls.
func (p *Provider) Encrypt(ctx context.Context, request *crypto.EncryptRequest) (*crypto.EncryptResponse, error) {
	if request.Plaintext == "" {
		return nil, crypto.ErrEmptyPlaintext
	}

	keyID := p.keyID
	if request.KeyID != "" {
		keyID = request.KeyID
	}

	result, err := p.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     aws.String(keyID),
		Plaintext: []byte(request.Plaintext),
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS encryption failed: %w", err)
	}

	encodedCiphertext := base64.StdEncoding.EncodeToString(result.CiphertextBlob)

	return &crypto.EncryptResponse{
		Ciphertext: encodedCiphertext,
		KeyID:      aws.ToString(result.KeyId),
		Provider:   p.Type(),
	}, nil
}

// Decrypt decrypts a KMS ciphertext blob.
// The ciphertext must be a Base64-encoded AWS KMS ciphertext blob.
//
// Takes request (*crypto.DecryptRequest) which contains the ciphertext to
// decrypt.
//
// Returns *crypto.DecryptResponse which contains the decrypted plaintext.
// Returns error when the ciphertext is empty, has invalid Base64 encoding, or
// AWS KMS decryption fails.
//
// AWS KMS automatically identifies the correct key from the ciphertext,
// so the KeyID in the request is optional.
func (p *Provider) Decrypt(ctx context.Context, request *crypto.DecryptRequest) (*crypto.DecryptResponse, error) {
	if request.Ciphertext == "" {
		return nil, crypto.ErrEmptyCiphertext
	}

	ciphertextBlob, err := base64.StdEncoding.DecodeString(request.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 encoding: %w", crypto.ErrInvalidCiphertext, err)
	}

	result, err := p.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: ciphertextBlob,
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS decryption failed: %w", err)
	}

	return &crypto.DecryptResponse{
		Plaintext: string(result.Plaintext),
	}, nil
}

// GenerateDataKey creates a new data encryption key (DEK) for envelope
// encryption.
//
// This is the recommended pattern for encrypting large amounts of data:
//
//  1. Call GenerateDataKey once to get a plaintext DEK and its encrypted version
//  2. Use the plaintext DEK to encrypt your data locally (fast, no network calls)
//  3. Store the encrypted DEK alongside your encrypted data
//  4. Discard the plaintext DEK from memory after use
//
// When decrypting:
//
//  1. Call Decrypt on the encrypted DEK to get the plaintext DEK
//  2. Use the plaintext DEK to decrypt your data locally
//
// This pattern reduces KMS API calls from N (one per item) to 1 or 2 per batch,
// improving performance and reducing costs.
//
// Takes request (*crypto.GenerateDataKeyRequest) which specifies the key ID and
// key spec for the new data key.
//
// Returns *crypto.DataKey which contains the plaintext key in secure memory
// and the encrypted key for storage.
// Returns error when the AWS KMS API call fails or secure memory allocation
// fails.
func (p *Provider) GenerateDataKey(ctx context.Context, request *crypto.GenerateDataKeyRequest) (*crypto.DataKey, error) {
	keyID := p.keyID
	if request.KeyID != "" {
		keyID = request.KeyID
	}

	keySpec := types.DataKeySpecAes256
	if request.KeySpec == "AES_128" {
		keySpec = types.DataKeySpecAes128
	}

	result, err := p.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   aws.String(keyID),
		KeySpec: keySpec,
	})
	if err != nil {
		return nil, fmt.Errorf("AWS KMS GenerateDataKey failed: %w", err)
	}

	secureKey, err := crypto_dto.NewSecureBytesFromSlice(result.Plaintext, crypto_dto.WithID("aws-datakey-"+aws.ToString(result.KeyId)))
	if err != nil {
		zeroBytes(result.Plaintext)
		return nil, fmt.Errorf("failed to create secure bytes for AWS data key: %w", err)
	}

	zeroBytes(result.Plaintext)

	return &crypto.DataKey{
		PlaintextKey: secureKey,
		EncryptedKey: base64.StdEncoding.EncodeToString(result.CiphertextBlob),
		KeyID:        aws.ToString(result.KeyId),
		Provider:     p.Type(),
	}, nil
}

// GetKeyInfo retrieves metadata about the KMS key.
// This includes the key state, creation date, and rotation status.
//
// Takes keyID (string) which specifies the key to query. If empty, uses the
// provider's default key.
//
// Returns *crypto.KeyInfo which contains the key metadata.
// Returns error when the key is not found or the AWS KMS API call fails.
func (p *Provider) GetKeyInfo(ctx context.Context, keyID string) (*crypto.KeyInfo, error) {
	if keyID == "" {
		keyID = p.keyID
	}

	result, err := p.client.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		if _, ok := errors.AsType[*types.NotFoundException](err); ok {
			return nil, crypto.ErrKeyNotFound
		}
		return nil, fmt.Errorf("AWS KMS DescribeKey failed: %w", err)
	}

	meta := result.KeyMetadata
	return &crypto.KeyInfo{
		KeyID:       aws.ToString(meta.KeyId),
		Provider:    p.Type(),
		Algorithm:   "AWS_KMS_RSAES_OAEP_SHA_256",
		CreatedAt:   aws.ToTime(meta.CreationDate),
		Status:      mapKeyState(meta.KeyState),
		Origin:      string(meta.Origin),
		Description: aws.ToString(meta.Description),
		Metadata: map[string]string{
			"arn":         aws.ToString(meta.Arn),
			"account_id":  aws.ToString(meta.AWSAccountId),
			"key_manager": string(meta.KeyManager),
		},
	}, nil
}

// HealthCheck performs a lightweight roundtrip encryption and decryption test.
// This validates that the KMS key exists and is enabled, the caller has
// kms:Encrypt and kms:Decrypt permissions, and network connectivity to AWS KMS
// is working.
//
// Returns error when encryption, decryption, or roundtrip validation fails.
func (p *Provider) HealthCheck(ctx context.Context) error {
	plaintext := "piko-health-check-" + time.Now().Format("20060102-150405")

	encryptResp, err := p.Encrypt(ctx, &crypto.EncryptRequest{
		Plaintext: plaintext,
	})
	if err != nil {
		return fmt.Errorf("health check encryption failed: %w", err)
	}

	decryptResp, err := p.Decrypt(ctx, &crypto.DecryptRequest{
		Ciphertext: encryptResp.Ciphertext,
	})
	if err != nil {
		return fmt.Errorf("health check decryption failed: %w", err)
	}

	if decryptResp.Plaintext != plaintext {
		return fmt.Errorf("health check roundtrip validation failed: expected %q, got %q", plaintext, decryptResp.Plaintext)
	}

	return nil
}

// NewProvider creates a new AWS KMS encryption provider.
//
// It automatically loads AWS credentials from the environment:
//   - IAM role (recommended for EC2/ECS/Lambda)
//   - AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
//   - ~/.aws/credentials file
//   - ECS task role
//   - EC2 instance profile
//
// The provider performs a validation call to DescribeKey on initialisation
// to ensure the key exists and the caller has permission to use it.
//
// Takes kmsConfig (Config) which specifies the AWS KMS settings including region,
// key ID, and retry configuration.
//
// Returns crypto.EncryptionProvider which is ready to encrypt and
// decrypt data using the configured KMS key.
// Returns error when the configuration is invalid, AWS credentials cannot be
// loaded, or the KMS key is inaccessible.
func NewProvider(ctx context.Context, kmsConfig Config) (crypto.EncryptionProvider, error) {
	kmsConfig = kmsConfig.WithDefaults()
	if err := kmsConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid AWS KMS config: %w", err)
	}

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(kmsConfig.Region),
		config.WithRetryMaxAttempts(kmsConfig.MaxRetries),
	}

	if kmsConfig.EndpointURL != "" && kmsConfig.UseStaticCredentials {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	var clientOpts []func(*kms.Options)
	if kmsConfig.EndpointURL != "" {
		clientOpts = append(clientOpts, func(o *kms.Options) {
			o.BaseEndpoint = aws.String(kmsConfig.EndpointURL)
		})
	}

	kmsClient := kms.NewFromConfig(awsConfig, clientOpts...)

	if _, err := kmsClient.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(kmsConfig.KeyID),
	}); err != nil {
		return nil, fmt.Errorf("failed to describe KMS key '%s' (check permissions and key existence): %w", kmsConfig.KeyID, err)
	}

	return &Provider{
		client:     kmsClient,
		keyID:      kmsConfig.KeyID,
		maxRetries: kmsConfig.MaxRetries,
	}, nil
}

// mapKeyState converts an AWS KMS key state to the domain KeyStatus value.
//
// Takes state (types.KeyState) which is the AWS KMS key state to convert.
//
// Returns crypto.KeyStatus which is the matching domain key status.
func mapKeyState(state types.KeyState) crypto.KeyStatus {
	switch state {
	case types.KeyStateEnabled:
		return crypto.KeyStatusActive
	case types.KeyStatePendingDeletion:
		return crypto.KeyStatusDestroyed
	default:
		return crypto.KeyStatusDisabled
	}
}

// zeroBytes overwrites a byte slice with zeros to remove sensitive data from
// memory. This is a critical security measure for ephemeral encryption keys.
//
// This does not guarantee the data is unrecoverable (compiler optimisations,
// swap files, etc.), but it is a best-effort approach to minimise key exposure.
//
// Takes data ([]byte) which is the sensitive data to overwrite.
func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
