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
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"piko.sh/piko/wdk/crypto"
	"piko.sh/piko/wdk/crypto/crypto_streaming"
)

// EncryptStream implements streaming encryption using AWS KMS
// envelope encryption.
//
// This method generates a data encryption key (DEK) from KMS and
// uses it to perform local AES-GCM streaming encryption. The
// encrypted DEK is stored in the streaming header.
//
// Cost: 1 KMS API call (GenerateDataKey) per stream, regardless
// of file size.
// Memory: O(chunk_size) ~64KB, regardless of file size.
//
// Takes output (io.Writer) which receives the encrypted stream
// data.
// Takes request (*crypto.EncryptRequest) which specifies encryption
// options.
//
// Returns io.WriteCloser which wraps output and encrypts data as
// it is written.
// Returns error when key generation, cipher creation, or header
// writing fails.
//
// Example:
//
//	encryptingWriter, err := provider.EncryptStream(ctx, outputFile, &crypto.EncryptRequest{})
//	if err != nil { return err }
//	defer encryptingWriter.Close()
//	_, err = io.Copy(encryptingWriter, largeInputFile)
//	return err
func (p *Provider) EncryptStream(ctx context.Context, output io.Writer, request *crypto.EncryptRequest) (io.WriteCloser, error) {
	keyID := p.keyID
	if request.KeyID != "" {
		keyID = request.KeyID
	}

	dataKeyResp, err := p.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   &keyID,
		KeySpec: "AES_256",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate data key from AWS KMS: %w", err)
	}

	block, err := aes.NewCipher(dataKeyResp.Plaintext)
	if err != nil {
		zeroBytes(dataKeyResp.Plaintext)
		return nil, fmt.Errorf("failed to create AES cipher from data key: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		zeroBytes(dataKeyResp.Plaintext)
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	baseIV, err := crypto_streaming.GenerateIV()
	if err != nil {
		zeroBytes(dataKeyResp.Plaintext)
		return nil, fmt.Errorf("failed to generate base IV: %w", err)
	}

	header := &crypto_streaming.StreamingHeader{
		Version:          2,
		KeyID:            keyID,
		Provider:         string(p.Type()),
		IV:               base64.StdEncoding.EncodeToString(baseIV),
		EncryptedDataKey: base64.StdEncoding.EncodeToString(dataKeyResp.CiphertextBlob),
		Algorithm:        "AES-256-GCM",
	}

	if err := crypto_streaming.WriteStreamingHeader(output, header); err != nil {
		zeroBytes(dataKeyResp.Plaintext)
		return nil, fmt.Errorf("failed to write streaming header: %w", err)
	}

	writer := crypto_streaming.NewEncryptingWriter(output, aead, baseIV, crypto_streaming.DefaultChunkSize)

	zeroBytes(dataKeyResp.Plaintext)

	return writer, nil
}

// DecryptStream implements streaming decryption using AWS KMS envelope
// encryption.
//
// This method reads the streaming header, decrypts the data encryption key
// (DEK) using KMS, and uses it to perform local AES-GCM streaming decryption.
//
// Takes input (io.Reader) which provides the encrypted data stream.
//
// Returns io.ReadCloser which streams the decrypted plaintext.
// Returns error when the header is invalid, the data key cannot be decrypted,
// or the cipher cannot be initialised.
//
// Cost: 1 KMS API call (Decrypt) per stream, regardless of file size.
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

	encryptedDataKey, err := base64.StdEncoding.DecodeString(header.EncryptedDataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data key: %w", err)
	}

	decryptResp, err := p.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: encryptedDataKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key with AWS KMS: %w", err)
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

// readAndValidateDecryptHeader reads and validates the streaming header for
// decryption.
//
// Takes input (io.Reader) which provides the encrypted stream to read from.
// Takes providerType (crypto.ProviderType) which specifies the expected
// encryption provider.
//
// Returns *crypto_streaming.StreamingHeader which contains the validated header
// data.
// Returns error when the header cannot be read, the provider does not match,
// or the encrypted data key is missing.
func readAndValidateDecryptHeader(input io.Reader, providerType crypto.ProviderType) (*crypto_streaming.StreamingHeader, error) {
	header, err := crypto_streaming.ReadStreamingHeader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read streaming header: %w", err)
	}

	if header.Provider != string(providerType) {
		return nil, fmt.Errorf("provider mismatch: header says %q, but this is %q", header.Provider, providerType)
	}

	if header.EncryptedDataKey == "" {
		return nil, errors.New("missing encrypted data key in streaming header (required for AWS KMS envelope encryption)")
	}

	return header, nil
}

// createDecryptionCipher creates an AES-GCM AEAD cipher and decodes the base
// IV from its base64 representation.
//
// Takes plaintextKey ([]byte) which is the raw AES key material.
// Takes ivB64 (string) which is the base64-encoded IV.
//
// Returns cipher.AEAD which is the configured GCM cipher for decryption.
// Returns []byte which is the decoded base IV.
// Returns error when cipher creation fails or IV decoding is invalid.
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
