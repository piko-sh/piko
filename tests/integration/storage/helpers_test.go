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

//go:build integration

package storage_integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/storage/storage_provider_s3"
	"piko.sh/piko/wdk/storage/storage_transformer_crypto"
	"piko.sh/piko/wdk/storage/storage_transformer_gzip"
	"piko.sh/piko/wdk/storage/storage_transformer_zstd"
)

func newS3Provider(ctx context.Context, t *testing.T, env *testEnv) storage_domain.StorageProviderPort {
	t.Helper()

	s3Config := &storage_provider_s3.Config{
		RepositoryMappings: map[string]string{
			testRepoPrimary:   testBucketPrimary,
			testRepoSecondary: testBucketSecondary,
		},
		Region:          testRegion,
		AccessKey:       testAccessKey,
		SecretKey:       testSecretKey,
		EndpointURL:     env.endpointURL,
		UsePathStyle:    true,
		DisableChecksum: true,
	}

	provider, err := storage_provider_s3.NewS3Provider(ctx, s3Config)
	require.NoError(t, err)
	return provider
}

func newGzipTransformer(t *testing.T, level int) storage_domain.StreamTransformerPort {
	t.Helper()

	transformer, err := storage_transformer_gzip.NewGzipTransformer(storage_transformer_gzip.Config{
		Name:     "gzip",
		Priority: 100,
		Level:    level,
	})
	require.NoError(t, err)
	return transformer
}

func newZstdTransformer(t *testing.T) storage_domain.StreamTransformerPort {
	t.Helper()

	transformer, err := storage_transformer_zstd.NewZstdTransformer(storage_transformer_zstd.Config{
		Name:     "zstd",
		Priority: 100,
	})
	require.NoError(t, err)
	return transformer
}

type cryptoSetup struct {
	service     crypto_domain.CryptoServicePort
	transformer storage_domain.StreamTransformerPort
}

func newCryptoSetup(t *testing.T) *cryptoSetup {
	t.Helper()

	key := make([]byte, local_aes_gcm.KeySize)
	copy(key, "integration-test-key-material!!")

	provider, err := local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: "test-key-1",
	})
	require.NoError(t, err)

	service, err := crypto_domain.NewCryptoService(context.Background(), nil, &crypto_dto.ServiceConfig{
		ActiveKeyID:              "test-key-1",
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		EnableEnvelopeEncryption: false,
	})
	require.NoError(t, err)

	err = service.RegisterProvider(context.Background(), "local-aes-gcm", provider)
	require.NoError(t, err)

	err = service.SetDefaultProvider("local-aes-gcm")
	require.NoError(t, err)

	transformer := storage_transformer_crypto.New(service, "crypto-service", 250)

	return &cryptoSetup{
		service:     service,
		transformer: transformer,
	}
}

func newTransformerWrapper(
	t *testing.T,
	provider storage_domain.StorageProviderPort,
	transformers []storage_domain.StreamTransformerPort,
	enabledNames []string,
) *storage_domain.TransformerWrapper {
	t.Helper()

	registry := storage_domain.NewTransformerRegistry()
	for _, transformer := range transformers {
		err := registry.Register(transformer)
		require.NoError(t, err)
	}

	defaultConfig := &storage_dto.TransformConfig{
		EnabledTransformers: enabledNames,
		TransformerOptions:  make(map[string]any),
	}

	return storage_domain.NewTransformerWrapper(provider, registry, defaultConfig, "s3-test")
}

func getRawBytesFromS3(ctx context.Context, t *testing.T, bucket, key string) []byte {
	t.Helper()

	output, err := globalEnv.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	require.NoError(t, err)
	defer func() { _ = output.Body.Close() }()

	data, err := io.ReadAll(output.Body)
	require.NoError(t, err)
	return data
}

func getObjectMetadataFromS3(ctx context.Context, t *testing.T, bucket, key string) map[string]string {
	t.Helper()

	output, err := globalEnv.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	require.NoError(t, err)
	return output.Metadata
}

func isValidGzip(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

func isValidZstd(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD
}

func isStreamingEnvelope(data []byte) bool {
	return len(data) >= 1 && data[0] == crypto_dto.StreamingEnvelopeVersion
}

func isNotPlaintext(stored, original []byte) bool {
	if bytes.Equal(stored, original) {
		return false
	}

	checkLen := min(64, len(original))
	if checkLen > 0 && bytes.Contains(stored, original[:checkLen]) {
		return false
	}
	return true
}

func generateRepeatableText(size int) []byte {
	const phrase = "The quick brown fox jumps over the lazy dog. "
	var buffer bytes.Buffer
	buffer.Grow(size)
	for buffer.Len() < size {
		buffer.WriteString(phrase)
	}
	return buffer.Bytes()[:size]
}

func generateRandomData(t *testing.T, size int) []byte {
	t.Helper()
	data := make([]byte, size)
	_, err := rand.Read(data)
	require.NoError(t, err)
	return data
}

func uniqueKey(t *testing.T, prefix string) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "-")
	return fmt.Sprintf("%s/%s-%d", prefix, name, time.Now().UnixNano())
}

func putObject(ctx context.Context, t *testing.T, wrapper *storage_domain.TransformerWrapper, key string, data []byte) {
	t.Helper()

	params := &storage_dto.PutParams{
		Repository:  testRepoPrimary,
		Key:         key,
		Reader:      bytes.NewReader(data),
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}

	err := wrapper.Put(ctx, params)
	require.NoError(t, err)
}

func getObject(ctx context.Context, t *testing.T, wrapper *storage_domain.TransformerWrapper, key string) []byte {
	t.Helper()

	params := storage_dto.GetParams{
		Repository: testRepoPrimary,
		Key:        key,
	}

	reader, err := wrapper.Get(ctx, params)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	return data
}

func putRawBytesToS3(ctx context.Context, t *testing.T, bucket, key string, data []byte, metadata map[string]string) {
	t.Helper()

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	if len(metadata) > 0 {
		input.Metadata = metadata
	}

	_, err := globalEnv.s3Client.PutObject(ctx, input)
	require.NoError(t, err)
}

func newCryptoSetupWithKey(t *testing.T, keyMaterial string, keyID string) *cryptoSetup {
	t.Helper()

	key := make([]byte, local_aes_gcm.KeySize)
	copy(key, keyMaterial)

	provider, err := local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: keyID,
	})
	require.NoError(t, err)

	service, err := crypto_domain.NewCryptoService(context.Background(), nil, &crypto_dto.ServiceConfig{
		ActiveKeyID:              keyID,
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		EnableEnvelopeEncryption: false,
	})
	require.NoError(t, err)

	err = service.RegisterProvider(context.Background(), "local-aes-gcm", provider)
	require.NoError(t, err)

	err = service.SetDefaultProvider("local-aes-gcm")
	require.NoError(t, err)

	transformer := storage_transformer_crypto.New(service, "crypto-service", 250)

	return &cryptoSetup{
		service:     service,
		transformer: transformer,
	}
}

func getObjectExpectError(ctx context.Context, t *testing.T, wrapper *storage_domain.TransformerWrapper, key string) error {
	t.Helper()

	params := storage_dto.GetParams{
		Repository: testRepoPrimary,
		Key:        key,
	}

	reader, err := wrapper.Get(ctx, params)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	return err
}

func encryptBytesDirectly(ctx context.Context, t *testing.T, crypto *cryptoSetup, plaintext []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer, err := crypto.service.EncryptStream(ctx, &buffer, "")
	require.NoError(t, err)

	_, err = writer.Write(plaintext)
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
