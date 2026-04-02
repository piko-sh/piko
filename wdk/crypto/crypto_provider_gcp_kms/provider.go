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
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/wdk/crypto"
)

const (
	// aes256KeySize is the key size in bytes for AES-256 encryption.
	aes256KeySize = 32

	// aes128KeySize is the key length in bytes for AES-128 encryption.
	aes128KeySize = 16
)

// Provider implements cloud-backed encryption using Google Cloud Key
// Management Service (KMS). The master encryption key never leaves Google's
// Hardware Security Modules (HSMs).
type Provider struct {
	// client is the GCP KMS client used for cryptographic operations.
	client *kms.KeyManagementClient

	// keyID is the full resource name of the key used in encryption responses.
	keyID string
}

var _ crypto.EncryptionProvider = (*Provider)(nil)

// Type returns the provider type.
//
// Returns crypto.ProviderType which identifies this as a GCP KMS provider.
func (*Provider) Type() crypto.ProviderType {
	return crypto.ProviderTypeGCPKMS
}

// Encrypt encrypts plaintext using Google Cloud KMS.
//
// The ciphertext is a self-contained blob that includes the encrypted data
// and metadata about the key and version used. The returned ciphertext is
// Base64-encoded for safe storage in databases. For bulk encryption, use
// GenerateDataKey and encrypt locally to reduce KMS calls.
//
// Takes request (*crypto.EncryptRequest) which specifies the plaintext to encrypt
// and optionally a key ID to override the default.
//
// Returns *crypto.EncryptResponse which contains the Base64-encoded ciphertext,
// key ID, and provider information.
// Returns error when plaintext is empty or KMS encryption fails.
func (p *Provider) Encrypt(ctx context.Context, request *crypto.EncryptRequest) (*crypto.EncryptResponse, error) {
	if request.Plaintext == "" {
		return nil, crypto.ErrEmptyPlaintext
	}

	keyName := p.keyID
	if request.KeyID != "" {
		keyName = request.KeyID
	}

	result, err := p.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      keyName,
		Plaintext: []byte(request.Plaintext),
	})
	if err != nil {
		return nil, fmt.Errorf("GCP KMS encryption failed: %w", err)
	}

	encodedCiphertext := base64.StdEncoding.EncodeToString(result.Ciphertext)

	return &crypto.EncryptResponse{
		Ciphertext: encodedCiphertext,
		KeyID:      result.Name,
		Provider:   p.Type(),
	}, nil
}

// Decrypt decrypts a GCP KMS ciphertext.
//
// The ciphertext must be a Base64-encoded GCP KMS ciphertext blob. GCP KMS
// automatically identifies the correct key version from the ciphertext, so
// the KeyID in the request is optional.
//
// Takes request (*crypto.DecryptRequest) which contains the ciphertext to
// decrypt.
//
// Returns *crypto.DecryptResponse which contains the decrypted plaintext.
// Returns error when the ciphertext is empty, has invalid Base64 encoding, or
// when the GCP KMS decryption fails.
func (p *Provider) Decrypt(ctx context.Context, request *crypto.DecryptRequest) (*crypto.DecryptResponse, error) {
	if request.Ciphertext == "" {
		return nil, crypto.ErrEmptyCiphertext
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(request.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 encoding: %w", crypto.ErrInvalidCiphertext, err)
	}

	result, err := p.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       p.keyID,
		Ciphertext: ciphertextBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("GCP KMS decryption failed: %w", err)
	}

	return &crypto.DecryptResponse{
		Plaintext: string(result.Plaintext),
	}, nil
}

// GenerateDataKey creates a new data encryption key (DEK) for
// envelope encryption.
//
// This implements the same pattern as AWS KMS for envelope
// encryption:
//
//  1. Call GenerateDataKey once to get a plaintext DEK and its encrypted version
//  2. Use the plaintext DEK to encrypt your data locally (fast, no network calls)
//  3. Store the encrypted DEK alongside your encrypted data
//  4. Discard the plaintext DEK from memory after use
//
// When decrypting:
//  1. Call Decrypt on the encrypted DEK to get the plaintext DEK
//  2. Use the plaintext DEK to decrypt your data locally
//
// This pattern reduces KMS API calls from N (one per item) to 1 or 2 per batch.
//
// GCP KMS doesn't have a native GenerateDataKey API like AWS KMS.
// We implement it by generating a random key locally and encrypting it with KMS.
//
// Takes request (*crypto.GenerateDataKeyRequest) which specifies the key ID and
// key spec for the generated data key.
//
// Returns *crypto.DataKey which contains the plaintext and encrypted key pair.
// Returns error when random key generation or KMS encryption fails.
func (p *Provider) GenerateDataKey(ctx context.Context, request *crypto.GenerateDataKeyRequest) (*crypto.DataKey, error) {
	keyName := p.keyID
	if request.KeyID != "" {
		keyName = request.KeyID
	}

	keySize := aes256KeySize
	if request.KeySpec == "AES_128" {
		keySize = aes128KeySize
	}

	plaintextKey := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, plaintextKey); err != nil {
		return nil, fmt.Errorf("failed to generate random data key: %w", err)
	}

	result, err := p.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      keyName,
		Plaintext: plaintextKey,
	})
	if err != nil {
		zeroBytes(plaintextKey)
		return nil, fmt.Errorf("GCP KMS failed to encrypt data key: %w", err)
	}

	encryptedKey := base64.StdEncoding.EncodeToString(result.Ciphertext)

	secureKey, err := crypto_dto.NewSecureBytesFromSlice(plaintextKey, crypto_dto.WithID("gcp-datakey-"+result.Name))
	if err != nil {
		zeroBytes(plaintextKey)
		return nil, fmt.Errorf("failed to create secure bytes for GCP data key: %w", err)
	}

	zeroBytes(plaintextKey)

	return &crypto.DataKey{
		PlaintextKey: secureKey,
		EncryptedKey: encryptedKey,
		KeyID:        result.Name,
		Provider:     p.Type(),
	}, nil
}

// GetKeyInfo retrieves metadata about the GCP KMS key.
// This includes the key state, creation date, and rotation schedule.
//
// Takes keyID (string) which specifies the key to query. If empty, uses the
// provider's default key.
//
// Returns *crypto.KeyInfo which contains the key metadata.
// Returns error when the key is not found or the API call fails.
func (p *Provider) GetKeyInfo(ctx context.Context, keyID string) (*crypto.KeyInfo, error) {
	if keyID == "" {
		keyID = p.keyID
	}

	result, err := p.client.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{
		Name: keyID,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, crypto.ErrKeyNotFound
		}
		return nil, fmt.Errorf("GCP KMS GetCryptoKey failed: %w", err)
	}
	var createdAt time.Time
	var keyState crypto.KeyStatus
	algorithm := "GOOGLE_SYMMETRIC_ENCRYPTION"

	if result.Primary != nil {
		createdAt = result.Primary.CreateTime.AsTime()
		keyState = mapKeyState(result.Primary.State)
		algorithm = result.Primary.Algorithm.String()
	}

	metadata := map[string]string{
		"purpose":          result.Purpose.String(),
		"primary_version":  result.Primary.Name,
		"version_template": result.VersionTemplate.Algorithm.String(),
	}

	if result.NextRotationTime != nil {
		metadata["next_rotation_time"] = result.NextRotationTime.AsTime().String()
	}

	return &crypto.KeyInfo{
		KeyID:       result.Name,
		Provider:    p.Type(),
		Algorithm:   algorithm,
		CreatedAt:   createdAt,
		Status:      keyState,
		Origin:      "GOOGLE_CLOUD_KMS",
		Description: "",
		Metadata:    metadata,
	}, nil
}

// HealthCheck performs a lightweight roundtrip encryption/decryption test.
//
// This validates that:
//   - The KMS key exists and is enabled
//   - The caller has cloudkms.cryptoKeyVersions.useToEncrypt and
//     useToDecrypt permissions
//   - Network connectivity to Google Cloud KMS is working
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

// NewProvider creates a new Google Cloud KMS encryption provider.
//
// It automatically loads credentials from the environment using Application
// Default Credentials (ADC):
//   - GOOGLE_APPLICATION_CREDENTIALS environment variable (path to service
//     account key)
//   - Compute Engine/Cloud Run/GKE service account
//   - gcloud auth application-default login credentials
//
// The provider performs a validation call to GetCryptoKey on initialisation
// to ensure the key exists and the caller has permission to use it.
//
// Takes config (Config) which specifies the KMS key location and settings.
// Takes opts (...option.ClientOption) which provides optional client settings.
//
// Returns crypto.EncryptionProvider which is ready for use.
// Returns error when the config is invalid, the KMS client cannot be created,
// or the key cannot be accessed.
func NewProvider(ctx context.Context, config Config, opts ...option.ClientOption) (crypto.EncryptionProvider, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid GCP KMS config: %w", err)
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP KMS client: %w", err)
	}

	keyResourceName := config.KeyResourceName()

	if _, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{
		Name: keyResourceName,
	}); err != nil {
		_ = kmsClient.Close()
		return nil, fmt.Errorf("failed to get GCP KMS key '%s' (check permissions and key existence): %w", keyResourceName, err)
	}

	return &Provider{
		client: kmsClient,
		keyID:  keyResourceName,
	}, nil
}

// mapKeyState translates GCP KMS key states to our domain's KeyStatus enum.
//
// Takes state (kmspb.CryptoKeyVersion_CryptoKeyVersionState) which is the GCP
// KMS key version state to translate.
//
// Returns crypto.KeyStatus which is the corresponding domain key status.
func mapKeyState(state kmspb.CryptoKeyVersion_CryptoKeyVersionState) crypto.KeyStatus {
	switch state {
	case kmspb.CryptoKeyVersion_ENABLED:
		return crypto.KeyStatusActive
	case kmspb.CryptoKeyVersion_DESTROYED:
		return crypto.KeyStatusDestroyed
	default:
		return crypto.KeyStatusDisabled
	}
}

// zeroBytes overwrites a byte slice with zeros to remove sensitive data from
// memory. This is a critical security measure for ephemeral encryption keys.
//
// Takes data ([]byte) which is the byte slice to overwrite with zeros.
func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
