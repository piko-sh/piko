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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"piko.sh/piko/wdk/crypto"
	"piko.sh/piko/wdk/crypto/crypto_streaming"
)

// aes256KeyBytes is the key size in bytes for AES-256 encryption.
const aes256KeyBytes = 32

// EncryptStream implements streaming encryption using GCP KMS
// envelope encryption.
//
// This method generates a data encryption key (DEK) locally,
// encrypts it with GCP KMS, and uses it to perform local AES-GCM
// streaming encryption. The encrypted DEK is stored in the
// streaming header.
//
// Cost: 1 GCP KMS API call (Encrypt) per stream, regardless of
// file size.
// Memory: O(chunk_size) ~64KB, regardless of file size.
//
// Takes output (io.Writer) which receives the encrypted data
// stream.
// Takes request (*crypto.EncryptRequest) which specifies encryption
// options.
//
// Returns io.WriteCloser which wraps output and encrypts data
// written to it.
// Returns error when DEK generation or KMS encryption fails.
//
// Example:
//
//	encryptingWriter, err := provider.EncryptStream(ctx, outputFile, &crypto.EncryptRequest{})
//	if err != nil { return err }
//	defer encryptingWriter.Close()
//	_, err = io.Copy(encryptingWriter, largeInputFile)
//	return err
func (p *Provider) EncryptStream(ctx context.Context, output io.Writer, request *crypto.EncryptRequest) (io.WriteCloser, error) {
	keyName := p.keyID
	if request.KeyID != "" {
		keyName = request.KeyID
	}

	plaintextDEK, err := generateDataEncryptionKey()
	if err != nil {
		return nil, err
	}

	encryptResp, err := p.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      keyName,
		Plaintext: plaintextDEK,
	})
	if err != nil {
		zeroBytes(plaintextDEK)
		return nil, fmt.Errorf("failed to encrypt data key with GCP KMS: %w", err)
	}

	aead, baseIV, err := createEncryptionCipher(plaintextDEK)
	if err != nil {
		zeroBytes(plaintextDEK)
		return nil, err
	}

	if err := writeEncryptHeader(output, keyName, p.Type(), baseIV, encryptResp.Ciphertext); err != nil {
		zeroBytes(plaintextDEK)
		return nil, err
	}

	writer := crypto_streaming.NewEncryptingWriter(output, aead, baseIV, crypto_streaming.DefaultChunkSize)
	zeroBytes(plaintextDEK)

	return writer, nil
}

// DecryptStream implements streaming decryption using GCP KMS envelope
// encryption.
//
// This method reads the streaming header, decrypts the data encryption key
// (DEK) using GCP KMS, and uses it to perform local AES-GCM streaming
// decryption.
//
// Takes input (io.Reader) which provides the encrypted data stream.
//
// Returns io.ReadCloser which provides the decrypted plaintext stream.
// Returns error when the header is invalid, decryption fails, or the KMS
// call fails.
//
// Cost: 1 GCP KMS API call (Decrypt) per stream, regardless of file size.
// Memory: O(chunk_size) ~64KB, regardless of file size.
//
// Example:
// decryptingReader, err := provider.DecryptStream(ctx, encryptedFile)
// if err != nil { return err }
// defer decryptingReader.Close()
// plaintext, err := io.ReadAll(decryptingReader)
// return plaintext, err
func (p *Provider) DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error) {
	header, err := readAndValidateDecryptHeader(input, p.Type())
	if err != nil {
		return nil, err
	}

	encryptedDEK, err := base64.StdEncoding.DecodeString(header.EncryptedDataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data key: %w", err)
	}

	decryptResp, err := p.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       p.keyID,
		Ciphertext: encryptedDEK,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key with GCP KMS: %w", err)
	}

	aead, baseIV, err := createDecryptionCipher(decryptResp.Plaintext, header.IV)
	if err != nil {
		zeroBytes(decryptResp.Plaintext)
		return nil, err
	}

	reader := crypto_streaming.NewDecryptingReader(input, aead, baseIV)
	zeroBytes(decryptResp.Plaintext)

	return reader, nil
}

// generateDataEncryptionKey creates a random AES-256 data encryption key.
//
// Returns []byte which contains the 32-byte plaintext key.
// Returns error when random bytes cannot be read from the system source.
func generateDataEncryptionKey() ([]byte, error) {
	plaintextDEK := make([]byte, aes256KeyBytes)
	if _, err := io.ReadFull(rand.Reader, plaintextDEK); err != nil {
		return nil, fmt.Errorf("failed to generate data encryption key: %w", err)
	}
	return plaintextDEK, nil
}

// createEncryptionCipher creates an AES-GCM cipher and generates a base IV.
//
// Takes plaintextDEK ([]byte) which is the plaintext data encryption key.
//
// Returns cipher.AEAD which is the authenticated encryption cipher.
// Returns []byte which is the generated base IV.
// Returns error when creating the AES cipher, GCM mode, or IV generation
// fails.
func createEncryptionCipher(plaintextDEK []byte) (cipher.AEAD, []byte, error) {
	block, err := aes.NewCipher(plaintextDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher from data key: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	baseIV, err := crypto_streaming.GenerateIV()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate base IV: %w", err)
	}

	return aead, baseIV, nil
}

// writeEncryptHeader writes the v2 streaming header for encryption.
//
// Takes output (io.Writer) which receives the encoded header bytes.
// Takes keyName (string) which identifies the encryption key.
// Takes providerType (crypto.ProviderType) which specifies the crypto
// provider.
// Takes baseIV ([]byte) which contains the IV for encryption.
// Takes encryptedDEK ([]byte) which contains the encrypted data encryption
// key.
//
// Returns error when the header cannot be written to the output.
func writeEncryptHeader(output io.Writer, keyName string, providerType crypto.ProviderType, baseIV, encryptedDEK []byte) error {
	header := &crypto_streaming.StreamingHeader{
		Version:          2,
		KeyID:            keyName,
		Provider:         string(providerType),
		IV:               base64.StdEncoding.EncodeToString(baseIV),
		EncryptedDataKey: base64.StdEncoding.EncodeToString(encryptedDEK),
		Algorithm:        "AES-256-GCM",
	}

	if err := crypto_streaming.WriteStreamingHeader(output, header); err != nil {
		return fmt.Errorf("failed to write streaming header: %w", err)
	}
	return nil
}

// readAndValidateDecryptHeader reads and validates the streaming header for
// decryption.
//
// Takes input (io.Reader) which provides the encrypted data stream.
// Takes providerType (crypto.ProviderType) which specifies the expected
// encryption provider.
//
// Returns *crypto_streaming.StreamingHeader which contains the validated header
// data.
// Returns error when the header cannot be read, the provider type does not
// match, or the encrypted data key is missing.
func readAndValidateDecryptHeader(input io.Reader, providerType crypto.ProviderType) (*crypto_streaming.StreamingHeader, error) {
	header, err := crypto_streaming.ReadStreamingHeader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read streaming header: %w", err)
	}

	if header.Provider != string(providerType) {
		return nil, fmt.Errorf("provider mismatch: header says %q, but this is %q", header.Provider, providerType)
	}

	if header.EncryptedDataKey == "" {
		return nil, errors.New("missing encrypted data key in streaming header (required for GCP KMS envelope encryption)")
	}

	return header, nil
}

// createDecryptionCipher creates an AES-GCM AEAD cipher and decodes the base
// IV from its base64 representation.
//
// Takes plaintextKey ([]byte) which is the raw AES key bytes.
// Takes ivB64 (string) which is the base64-encoded IV.
//
// Returns cipher.AEAD which is the configured GCM cipher for decryption.
// Returns []byte which is the decoded IV bytes.
// Returns error when the cipher cannot be created or the IV is invalid.
func createDecryptionCipher(plaintextKey []byte, ivB64 string) (cipher.AEAD, []byte, error) {
	block, err := aes.NewCipher(plaintextKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher from data key: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	baseIV, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode base IV: %w", err)
	}

	if len(baseIV) != crypto_streaming.IVSize {
		return nil, nil, fmt.Errorf("invalid IV length: expected %d, got %d", crypto_streaming.IVSize, len(baseIV))
	}

	return aead, baseIV, nil
}
