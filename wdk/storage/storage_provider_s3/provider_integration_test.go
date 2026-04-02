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

package storage_provider_s3_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/goleak"

	"piko.sh/piko/wdk/storage"
	"piko.sh/piko/wdk/storage/storage_provider_s3"
)

const (
	testBucketPrimary   = "test-bucket-primary"
	testBucketSecondary = "test-bucket-secondary"
	testRepoPrimary     = "primary"
	testRepoSecondary   = "secondary"
	testRegion          = "us-east-1"
	testAccessKey       = "test"
	testSecretKey       = "test"
)

type testEnv struct {
	container   testcontainers.Container
	endpointURL string
	s3Client    *s3.Client
}

var globalEnv *testEnv
var sharedProvider storage.ProviderPort

func setupTestEnvironment(ctx context.Context) (*testEnv, error) {
	request := testcontainers.ContainerRequest{
		Image:        "localstack/localstack:latest",
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":              "s3",
			"DEFAULT_REGION":        testRegion,
			"AWS_ACCESS_KEY_ID":     testAccessKey,
			"AWS_SECRET_ACCESS_KEY": testSecretKey,
		},
		WaitingFor: wait.ForHTTP("/_localstack/health").
			WithPort("4566/tcp").
			WithStatusCodeMatcher(func(status int) bool {
				return status == http.StatusOK
			}).
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start LocalStack container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	endpointURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpointURL)
		o.UsePathStyle = true
	})

	for _, bucket := range []string{testBucketPrimary, testBucketSecondary} {
		_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
	}

	return &testEnv{
		container:   container,
		endpointURL: endpointURL,
		s3Client:    s3Client,
	}, nil
}

func teardownTestEnvironment(ctx context.Context, env *testEnv) error {
	if env == nil || env.container == nil {
		return nil
	}
	return env.container.Terminate(ctx)
}

func newTestProvider(_ context.Context, _ *testEnv) (storage.ProviderPort, error) {
	return sharedProvider, nil
}

func createTestProvider(ctx context.Context, env *testEnv) (storage.ProviderPort, error) {
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
	return storage_provider_s3.NewS3Provider(ctx, s3Config)
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	globalEnv, err = setupTestEnvironment(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to setup test environment: %v", err))
	}

	sharedProvider, err = createTestProvider(ctx, globalEnv)
	if err != nil {
		panic(fmt.Sprintf("failed to create shared test provider: %v", err))
	}

	code := m.Run()

	if err := teardownTestEnvironment(ctx, globalEnv); err != nil {
		fmt.Printf("warning: failed to teardown test environment: %v\n", err)
	}

	if code == 0 {
		if err := goleak.Find(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}

	if code != 0 {
		panic(fmt.Sprintf("tests failed with code %d", code))
	}
}

func TestS3Provider_Put_SimpleUpload(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-simple-upload-%d", time.Now().UnixNano())
	testContent := []byte("Hello, S3 Integration Test!")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}

	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	}

	reader, err := provider.Get(ctx, getParams)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_Put_WithMetadata(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-metadata-%d", time.Now().UnixNano())
	testContent := []byte("Content with metadata")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "application/octet-stream",
		Metadata: map[string]string{
			"x-custom-header": "custom-value",
			"author":          "test-suite",
		},
	}

	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.Equal(t, "custom-value", info.Metadata["x-custom-header"])
	assert.Equal(t, "test-suite", info.Metadata["author"])
}

func TestS3Provider_Get_NonExistent(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-key-12345",
	}

	reader, err := provider.Get(ctx, getParams)
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "not found")
}

func TestS3Provider_Get_ByteRange(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-byte-range-%d", time.Now().UnixNano())
	testContent := []byte("0123456789ABCDEFGHIJ")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
		ByteRange: &storage.ByteRange{
			Start: 5,
			End:   9,
		},
	}

	reader, err := provider.Get(ctx, getParams)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("56789"), retrieved)
}

func TestS3Provider_Get_ByteRange_OpenEnded(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-byte-range-open-%d", time.Now().UnixNano())
	testContent := []byte("0123456789ABCDEFGHIJ")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
		ByteRange: &storage.ByteRange{
			Start: 15,
			End:   -1,
		},
	}

	reader, err := provider.Get(ctx, getParams)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("FGHIJ"), retrieved)
}

func TestS3Provider_Stat(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-stat-%d", time.Now().UnixNano())
	testContent := []byte("Content for stat test")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "application/json",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(len(testContent)), info.Size)
	assert.Equal(t, "application/json", info.ContentType)
	assert.NotEmpty(t, info.ETag)
	assert.False(t, info.LastModified.IsZero())
}

func TestS3Provider_Stat_NonExistent(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-stat-key",
	})
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not found")
}

func TestS3Provider_Exists(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-exists-%d", time.Now().UnixNano())
	testContent := []byte("Existence test")

	exists, err := provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.False(t, exists)

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	exists, err = provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestS3Provider_GetHash(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-hash-%d", time.Now().UnixNano())
	testContent := []byte("Hash test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	hash, err := provider.GetHash(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32)
}

func TestS3Provider_Copy_SameBucket(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	srcKey := fmt.Sprintf("test-copy-src-%d", time.Now().UnixNano())
	dstKey := fmt.Sprintf("test-copy-dst-%d", time.Now().UnixNano())
	testContent := []byte("Copy test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         srcKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	err = provider.Copy(ctx, testRepoPrimary, srcKey, dstKey)
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        dstKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_CopyToAnotherRepository(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	srcKey := fmt.Sprintf("test-cross-copy-src-%d", time.Now().UnixNano())
	dstKey := fmt.Sprintf("test-cross-copy-dst-%d", time.Now().UnixNano())
	testContent := []byte("Cross-repository copy test")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         srcKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	err = provider.CopyToAnotherRepository(ctx, testRepoPrimary, srcKey, testRepoSecondary, dstKey)
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoSecondary,
		Key:        dstKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_Remove(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-remove-%d", time.Now().UnixNano())
	testContent := []byte("Remove test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	exists, err := provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.True(t, exists)

	err = provider.Remove(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)

	exists, err = provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestS3Provider_Remove_Idempotent(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	err = provider.Remove(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-remove-key",
	})
	assert.NoError(t, err)
}

func TestS3Provider_Rename(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	oldKey := fmt.Sprintf("test-rename-old-%d", time.Now().UnixNano())
	newKey := fmt.Sprintf("test-rename-new-%d", time.Now().UnixNano())
	testContent := []byte("Rename test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         oldKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	err = provider.Rename(ctx, testRepoPrimary, oldKey, newKey)
	require.NoError(t, err)

	exists, err := provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        oldKey,
	})
	require.NoError(t, err)
	assert.False(t, exists)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        newKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_PutMany(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	baseKey := fmt.Sprintf("test-put-many-%d", time.Now().UnixNano())
	numObjects := 5

	objects := make([]storage.PutObjectSpec, numObjects)
	for i := range numObjects {
		content := fmt.Sprintf("Batch upload content %d", i)
		objects[i] = storage.PutObjectSpec{
			Key:         fmt.Sprintf("%s/%d.txt", baseKey, i),
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		}
	}

	params := &storage.PutManyParams{
		Repository:      testRepoPrimary,
		Objects:         objects,
		Concurrency:     3,
		ContinueOnError: true,
	}

	result, err := provider.PutMany(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, numObjects, result.TotalRequested)
	assert.Equal(t, numObjects, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.SuccessfulKeys, numObjects)

	for i := range numObjects {
		exists, err := provider.Exists(ctx, storage.GetParams{
			Repository: testRepoPrimary,
			Key:        fmt.Sprintf("%s/%d.txt", baseKey, i),
		})
		require.NoError(t, err)
		assert.True(t, exists)
	}
}

func TestS3Provider_RemoveMany(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	baseKey := fmt.Sprintf("test-remove-many-%d", time.Now().UnixNano())
	numObjects := 5

	keys := make([]string, numObjects)
	for i := range numObjects {
		key := fmt.Sprintf("%s/%d.txt", baseKey, i)
		keys[i] = key
		content := fmt.Sprintf("Remove many content %d", i)

		putParams := &storage.PutParams{
			Repository:  testRepoPrimary,
			Key:         key,
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		}
		err = provider.Put(ctx, putParams)
		require.NoError(t, err)
	}

	params := storage.RemoveManyParams{
		Repository:      testRepoPrimary,
		Keys:            keys,
		Concurrency:     3,
		ContinueOnError: true,
	}

	result, err := provider.RemoveMany(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, numObjects, result.TotalRequested)
	assert.Equal(t, numObjects, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)

	for _, key := range keys {
		exists, err := provider.Exists(ctx, storage.GetParams{
			Repository: testRepoPrimary,
			Key:        key,
		})
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestS3Provider_RemoveMany_LargeBatch(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	baseKey := fmt.Sprintf("test-remove-many-large-%d", time.Now().UnixNano())
	numObjects := 50

	keys := make([]string, numObjects)
	for i := range numObjects {
		key := fmt.Sprintf("%s/%d.txt", baseKey, i)
		keys[i] = key
		content := fmt.Sprintf("Large batch content %d", i)

		putParams := &storage.PutParams{
			Repository:  testRepoPrimary,
			Key:         key,
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		}
		err = provider.Put(ctx, putParams)
		require.NoError(t, err)
	}

	params := storage.RemoveManyParams{
		Repository:      testRepoPrimary,
		Keys:            keys,
		Concurrency:     10,
		ContinueOnError: true,
	}

	result, err := provider.RemoveMany(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, numObjects, result.TotalRequested)
	assert.Equal(t, numObjects, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
}

func TestS3Provider_Put_MultipartUpload(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-multipart-%d", time.Now().UnixNano())

	fileSize := 6 * 1024 * 1024
	testContent := make([]byte, fileSize)
	_, err = rand.Read(testContent)
	require.NoError(t, err)

	multipartConfig := storage.MultipartUploadConfig{
		PartSize:       5 * 1024 * 1024,
		Concurrency:    2,
		EnableChecksum: true,
		MaxRetries:     3,
	}

	putParams := &storage.PutParams{
		Repository:      testRepoPrimary,
		Key:             testKey,
		Reader:          bytes.NewReader(testContent),
		Size:            int64(fileSize),
		ContentType:     "application/octet-stream",
		MultipartConfig: &multipartConfig,
	}

	err = provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(fileSize), info.Size)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_PresignURL(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-presign-%d", time.Now().UnixNano())

	params := storage.PresignParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	}

	presignedURL, err := provider.PresignURL(ctx, params)
	require.NoError(t, err)
	assert.NotEmpty(t, presignedURL)
	assert.Contains(t, presignedURL, testKey)
	assert.Contains(t, presignedURL, "X-Amz-Signature")
}

func TestS3Provider_PresignURL_Upload(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	testKey := fmt.Sprintf("test-presign-upload-%d", time.Now().UnixNano())
	testContent := []byte("Uploaded via presigned URL")

	params := storage.PresignParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	}

	presignedURL, err := provider.PresignURL(ctx, params)
	require.NoError(t, err)

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(testContent))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "text/plain")

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	assert.Equal(t, http.StatusOK, response.StatusCode)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestS3Provider_Capabilities(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	assert.True(t, provider.SupportsMultipart(), "S3 should support multipart uploads")
	assert.True(t, provider.SupportsBatchOperations(), "S3 should support batch operations")
	assert.True(t, provider.SupportsRetry(), "S3 provider should have built-in retry")
	assert.False(t, provider.SupportsCircuitBreaking(), "Circuit breaking is at service layer")
	assert.True(t, provider.SupportsRateLimiting(), "S3 provider has built-in rate limiting")
}

func TestS3Provider_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	putParams := &storage.PutParams{
		Repository:  "non-existent-repo",
		Key:         "test-key",
		Reader:      strings.NewReader("test"),
		Size:        4,
		ContentType: "text/plain",
	}

	err = provider.Put(ctx, putParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestS3Provider_Close(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	err = provider.Close(ctx)
	assert.NoError(t, err)
}

func TestS3Provider_ConcurrentOperations(t *testing.T) {
	ctx := t.Context()
	provider, err := newTestProvider(ctx, globalEnv)
	require.NoError(t, err)

	baseKey := fmt.Sprintf("test-concurrent-%d", time.Now().UnixNano())
	numGoroutines := 10

	errChan := make(chan error, numGoroutines)

	for i := range numGoroutines {
		go func(index int) {
			key := fmt.Sprintf("%s/%d.txt", baseKey, index)
			content := fmt.Sprintf("Concurrent content %d", index)

			putParams := &storage.PutParams{
				Repository:  testRepoPrimary,
				Key:         key,
				Reader:      strings.NewReader(content),
				Size:        int64(len(content)),
				ContentType: "text/plain",
			}

			if err := provider.Put(ctx, putParams); err != nil {
				errChan <- fmt.Errorf("put %d: %w", index, err)
				return
			}

			reader, err := provider.Get(ctx, storage.GetParams{
				Repository: testRepoPrimary,
				Key:        key,
			})
			if err != nil {
				errChan <- fmt.Errorf("get %d: %w", index, err)
				return
			}

			retrieved, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				errChan <- fmt.Errorf("read %d: %w", index, err)
				return
			}

			if string(retrieved) != content {
				errChan <- fmt.Errorf("content mismatch %d: got %q, want %q", index, retrieved, content)
				return
			}

			errChan <- nil
		}(i)
	}

	for range numGoroutines {
		err := <-errChan
		assert.NoError(t, err)
	}
}
