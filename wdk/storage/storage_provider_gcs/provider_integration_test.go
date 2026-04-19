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

package storage_provider_gcs

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	gcsstorage "cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"piko.sh/piko/internal/testutil/leakcheck"
	"piko.sh/piko/wdk/storage"
)

const (
	testBucketPrimary   = "test-bucket-primary"
	testBucketSecondary = "test-bucket-secondary"
	testRepoPrimary     = "primary"
	testRepoSecondary   = "secondary"
)

type testEnv struct {
	server *fakestorage.Server
	client *gcsstorage.Client
}

var globalEnv *testEnv

func setupTestEnvironment() (*testEnv, error) {
	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		Scheme: "http",
		Host:   "127.0.0.1",
		Port:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start fake GCS server: %w", err)
	}

	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: testBucketPrimary})
	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: testBucketSecondary})

	return &testEnv{
		server: server,
		client: server.Client(),
	}, nil
}

func teardownTestEnvironment(env *testEnv) {
	if env == nil {
		return
	}
	if env.client != nil {
		_ = env.client.Close()
	}
	if env.server != nil {
		env.server.Stop()
	}
}

func newTestProvider() storage.ProviderPort {
	return newTestProviderWithClient(globalEnv.client)
}

func newTestProviderWithClient(client *gcsstorage.Client) storage.ProviderPort {
	rateLimiter := storage.ApplyProviderOptions(storage.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	})

	return &GCSProvider{
		client: client,
		repositoryBuckets: map[string]string{
			testRepoPrimary:   testBucketPrimary,
			testRepoSecondary: testBucketSecondary,
		},
		rateLimiter: rateLimiter,
	}
}

func TestMain(m *testing.M) {
	var err error
	globalEnv, err = setupTestEnvironment()
	if err != nil {
		panic(fmt.Sprintf("failed to setup test environment: %v", err))
	}

	code := m.Run()

	teardownTestEnvironment(globalEnv)

	if code == 0 {
		if err := leakcheck.FindLeaks(

			goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}

	if code != 0 {
		panic(fmt.Sprintf("tests failed with code %d", code))
	}
}

func TestGCSProvider_Put_SimpleUpload(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-simple-upload-%d", time.Now().UnixNano())
	testContent := []byte("Hello, GCS Integration Test!")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}

	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Put_WithMetadata(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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

	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.Equal(t, "custom-value", info.Metadata["x-custom-header"])
	assert.Equal(t, "test-suite", info.Metadata["author"])
}

func TestGCSProvider_Get_NonExistent(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-key-12345",
	}

	reader, err := provider.Get(ctx, getParams)
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "not found")
}

func TestGCSProvider_Get_ByteRange(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-byte-range-%d", time.Now().UnixNano())
	testContent := []byte("0123456789ABCDEFGHIJ")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Get_ByteRange_OpenEnded(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-byte-range-open-%d", time.Now().UnixNano())
	testContent := []byte("0123456789ABCDEFGHIJ")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Stat(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-stat-%d", time.Now().UnixNano())
	testContent := []byte("Content for stat test")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "application/json",
	}
	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(len(testContent)), info.Size)
	assert.Equal(t, "application/json", info.ContentType)
	assert.False(t, info.LastModified.IsZero())
}

func TestGCSProvider_Stat_NonExistent(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-stat-key",
	})
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "not found")
}

func TestGCSProvider_Exists(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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

func TestGCSProvider_GetHash(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-hash-%d", time.Now().UnixNano())
	testContent := []byte("Hash test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	hash, err := provider.GetHash(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestGCSProvider_Copy_SameBucket(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_CopyToAnotherRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Remove(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-remove-%d", time.Now().UnixNano())
	testContent := []byte("Remove test content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Remove_Idempotent(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.Remove(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "non-existent-remove-key",
	})
	assert.NoError(t, err)
}

func TestGCSProvider_Rename(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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
	err := provider.Put(ctx, putParams)
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

func TestGCSProvider_PutMany(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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

func TestGCSProvider_RemoveMany(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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
		err := provider.Put(ctx, putParams)
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

func TestGCSProvider_RemoveMany_LargeBatch(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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
		err := provider.Put(ctx, putParams)
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

func TestGCSProvider_Put_MultipartUpload(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-multipart-%d", time.Now().UnixNano())

	fileSize := 6 * 1024 * 1024
	testContent := make([]byte, fileSize)
	_, err := rand.Read(testContent)
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

func TestGCSProvider_Put_ZeroSizeFile(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-zero-size-%d", time.Now().UnixNano())

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(nil),
		Size:        0,
		ContentType: "application/octet-stream",
	}

	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, retrieved)
}

func TestGCSProvider_PutMany_EmptyBatch(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	params := &storage.PutManyParams{
		Repository:      testRepoPrimary,
		Objects:         nil,
		Concurrency:     3,
		ContinueOnError: true,
	}

	result, err := provider.PutMany(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, 0, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Nil(t, result.SuccessfulKeys)
	assert.Nil(t, result.FailedKeys)
}

func TestGCSProvider_RemoveMany_EmptyBatch(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	params := storage.RemoveManyParams{
		Repository:      testRepoPrimary,
		Keys:            nil,
		Concurrency:     3,
		ContinueOnError: true,
	}

	result, err := provider.RemoveMany(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, 0, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Nil(t, result.SuccessfulKeys)
	assert.Nil(t, result.FailedKeys)
}

func TestGCSProvider_RemoveMany_ContextCancelDoesNotLeakWorkers(t *testing.T) {
	provider := newTestProvider()

	ctx, cancel := context.WithCancelCause(t.Context())

	keys := make([]string, 200)
	for i := range keys {
		keys[i] = fmt.Sprintf("non-existent-cancel-test-%d-%d", time.Now().UnixNano(), i)
	}

	params := storage.RemoveManyParams{
		Repository:      testRepoPrimary,
		Keys:            keys,
		Concurrency:     8,
		ContinueOnError: true,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = provider.RemoveMany(ctx, params)
	}()

	cancel(errors.New("test cancelled context to detect worker leaks"))

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("RemoveMany did not return after context cancel; workers wedged")
	}
}

func TestGCSProvider_PutMany_ContextCancelDoesNotLeakWorkers(t *testing.T) {
	provider := newTestProvider()

	ctx, cancel := context.WithCancelCause(t.Context())

	objects := make([]storage.PutObjectSpec, 200)
	for i := range objects {
		content := fmt.Sprintf("ctx-cancel-test-%d", i)
		objects[i] = storage.PutObjectSpec{
			Key:         fmt.Sprintf("ctx-cancel-test-put-%d-%d", time.Now().UnixNano(), i),
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		}
	}

	params := &storage.PutManyParams{
		Repository:      testRepoPrimary,
		Objects:         objects,
		Concurrency:     8,
		ContinueOnError: true,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = provider.PutMany(ctx, params)
	}()

	cancel(errors.New("test cancelled context to detect worker leaks"))

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("PutMany did not return after context cancel; workers wedged")
	}
}

func TestGCSProvider_Get_ByteRange_SingleByte(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-byte-range-single-%d", time.Now().UnixNano())
	testContent := []byte("ABCDEFGHIJ")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	getParams := storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
		ByteRange: &storage.ByteRange{
			Start: 3,
			End:   3,
		},
	}

	reader, err := provider.Get(ctx, getParams)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("D"), retrieved)
}

func TestGCSProvider_Get_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	_, err := provider.Get(ctx, storage.GetParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Stat_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	_, err := provider.Stat(ctx, storage.GetParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Remove_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.Remove(ctx, storage.GetParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Exists_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	_, err := provider.Exists(ctx, storage.GetParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Copy_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.Copy(ctx, "non-existent-repo", "src-key", "dst-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_CopyToAnotherRepository_InvalidSourceRepo(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.CopyToAnotherRepository(ctx, "non-existent-repo", "src-key", testRepoSecondary, "dst-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_CopyToAnotherRepository_InvalidDestRepo(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.CopyToAnotherRepository(ctx, testRepoPrimary, "src-key", "non-existent-repo", "dst-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Copy_NonExistentSource(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	err := provider.Copy(ctx, testRepoPrimary, "non-existent-src", "dst-key")
	assert.Error(t, err)
}

func TestGCSProvider_GetHash_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	_, err := provider.GetHash(ctx, storage.GetParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_PresignURL_InvalidRepository(t *testing.T) {
	provider := newTestProvider()

	_, err := provider.PresignURL(t.Context(), storage.PresignParams{
		Repository:  "non-existent-repo",
		Key:         "test-key",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_PresignURL_NoCredentials(t *testing.T) {
	provider := newTestProvider()

	_, err := provider.PresignURL(t.Context(), storage.PresignParams{
		Repository:  testRepoPrimary,
		Key:         "test-key",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate signed URL")
}

func TestGCSProvider_PresignDownloadURL_InvalidRepository(t *testing.T) {
	provider := newTestProvider()

	_, err := provider.PresignDownloadURL(t.Context(), storage.PresignDownloadParams{
		Repository: "non-existent-repo",
		Key:        "test-key",
		ExpiresIn:  15 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_PresignDownloadURL_NoCredentials(t *testing.T) {
	provider := newTestProvider()

	_, err := provider.PresignDownloadURL(t.Context(), storage.PresignDownloadParams{
		Repository:  testRepoPrimary,
		Key:         "test-key",
		FileName:    "download.txt",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate signed download URL")
}

func TestGCSProvider_Capabilities(t *testing.T) {
	provider := newTestProvider()

	assert.True(t, provider.SupportsMultipart(), "GCS should support multipart uploads")
	assert.True(t, provider.SupportsBatchOperations(), "GCS should support batch operations")
	assert.False(t, provider.SupportsRetry(), "Retry is at service layer")
	assert.False(t, provider.SupportsCircuitBreaking(), "Circuit breaking is at service layer")
	assert.False(t, provider.SupportsRateLimiting(), "Rate limiting is at service layer")
	assert.True(t, provider.SupportsPresignedURLs(), "GCS should support presigned URLs")
}

func TestGCSProvider_InvalidRepository(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	putParams := &storage.PutParams{
		Repository:  "non-existent-repo",
		Key:         "test-key",
		Reader:      strings.NewReader("test"),
		Size:        4,
		ContentType: "text/plain",
	}

	err := provider.Put(ctx, putParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no bucket mapping found")
}

func TestGCSProvider_Close(t *testing.T) {
	ctx := t.Context()

	provider := newTestProviderWithClient(globalEnv.server.Client())

	err := provider.Close(ctx)
	assert.NoError(t, err)
}

func TestGCSProvider_ConcurrentOperations(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

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

func newSigningTestProvider(t *testing.T) *GCSProvider {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	saJSON, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"private_key":  string(keyPEM),
		"project_id":   "test-project",
	})
	require.NoError(t, err)

	creds := &google.Credentials{JSON: saJSON}
	client, err := gcsstorage.NewClient(
		context.Background(),
		option.WithHTTPClient(globalEnv.server.HTTPClient()),
		option.WithCredentials(creds),
	)
	require.NoError(t, err)

	rateLimiter := storage.ApplyProviderOptions(storage.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	})

	provider := &GCSProvider{
		client: client,
		repositoryBuckets: map[string]string{
			testRepoPrimary:   testBucketPrimary,
			testRepoSecondary: testBucketSecondary,
		},
		rateLimiter: rateLimiter,
	}

	t.Cleanup(func() { _ = provider.Close(context.Background()) })

	return provider
}

func TestGCSProvider_PresignURL_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newSigningTestProvider(t)

	testKey := fmt.Sprintf("test-presign-put-%d", time.Now().UnixNano())
	testContent := []byte("Presigned URL upload content")

	signedURL, err := provider.PresignURL(ctx, storage.PresignParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signedURL)
	assert.Contains(t, signedURL, testKey)

	httpClient := globalEnv.server.HTTPClient()
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, bytes.NewReader(testContent))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "text/plain")

	response, err := httpClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

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

func TestGCSProvider_PresignDownloadURL_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newSigningTestProvider(t)

	testKey := fmt.Sprintf("test-presign-download-%d", time.Now().UnixNano())
	testContent := []byte("Presigned URL download content")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "application/octet-stream",
	}
	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	signedURL, err := provider.PresignDownloadURL(ctx, storage.PresignDownloadParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		FileName:    "download.bin",
		ContentType: "application/octet-stream",
		ExpiresIn:   15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signedURL)
	assert.Contains(t, signedURL, testKey)

	httpClient := globalEnv.server.HTTPClient()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	require.NoError(t, err)

	response, err := httpClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	retrieved, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved)
}

func TestGCSProvider_PresignURL_WithContentType(t *testing.T) {
	ctx := t.Context()
	provider := newSigningTestProvider(t)

	signedURL, err := provider.PresignURL(ctx, storage.PresignParams{
		Repository:  testRepoPrimary,
		Key:         "test-content-type-key",
		ContentType: "image/png",
		ExpiresIn:   15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signedURL)
}

func TestGCSProvider_PresignDownloadURL_WithFileName(t *testing.T) {
	ctx := t.Context()
	provider := newSigningTestProvider(t)

	testKey := fmt.Sprintf("test-presign-filename-%d", time.Now().UnixNano())
	testContent := []byte("File name test")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        int64(len(testContent)),
		ContentType: "text/plain",
	}
	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	signedURL, err := provider.PresignDownloadURL(ctx, storage.PresignDownloadParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		FileName:    "my-file.txt",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	require.NoError(t, err)
	assert.Contains(t, signedURL, "response-content-disposition")
	assert.Contains(t, signedURL, "response-content-type")
}

func TestGCSProvider_PresignDownloadURL_NoFileName(t *testing.T) {
	ctx := t.Context()
	provider := newSigningTestProvider(t)

	signedURL, err := provider.PresignDownloadURL(ctx, storage.PresignDownloadParams{
		Repository: testRepoPrimary,
		Key:        "test-key",
		ExpiresIn:  15 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, signedURL)
	assert.NotContains(t, signedURL, "response-content-disposition")
}

func TestGCSProvider_Put_RateLimiterCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	provider := newTestProvider()
	err := provider.Put(ctx, &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         "should-not-upload",
		Reader:      strings.NewReader("test"),
		Size:        4,
		ContentType: "text/plain",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter wait failed")
}

func TestGCSProvider_Get_RateLimiterCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	provider := newTestProvider()
	_, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "should-not-get",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter wait failed")
}

func TestGCSProvider_Remove_RateLimiterCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	provider := newTestProvider()
	err := provider.Remove(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "should-not-remove",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter wait failed")
}

func TestGCSProvider_Exists_RateLimiterCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	provider := newTestProvider()
	_, err := provider.Exists(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        "should-not-check",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter wait failed")
}

func TestGCSProvider_Put_NonSeekableStream(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-non-seekable-%d", time.Now().UnixNano())
	testContent := []byte("This data comes from a non-seekable io.Pipe reader, simulating transformer pipeline output.")

	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(testContent)
		_ = pw.Close()
	}()

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      pr,
		Size:        int64(len(testContent)),
		ContentType: "application/octet-stream",
	}

	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved,
		"non-seekable io.Pipe reader should upload correctly via io.Copy")
}

func TestGCSProvider_Put_NonSeekableStream_UnknownSize(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-non-seekable-unknown-%d", time.Now().UnixNano())
	testContent := []byte("Unknown-size stream data from a transformer pipeline.")

	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(testContent)
		_ = pw.Close()
	}()

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      pr,
		Size:        -1,
		ContentType: "application/octet-stream",
	}

	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage.GetParams{
		Repository: testRepoPrimary,
		Key:        testKey,
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, retrieved,
		"non-seekable stream with unknown size should upload correctly")
}

func TestGCSProvider_Put_LargeFileAutoChunked(t *testing.T) {
	ctx := t.Context()
	provider := newTestProvider()

	testKey := fmt.Sprintf("test-large-auto-chunk-%d", time.Now().UnixNano())

	testContent := []byte("small content but large declared size")

	putParams := &storage.PutParams{
		Repository:  testRepoPrimary,
		Key:         testKey,
		Reader:      bytes.NewReader(testContent),
		Size:        largeFileThreshold + 1,
		ContentType: "application/octet-stream",
	}

	err := provider.Put(ctx, putParams)
	require.NoError(t, err)

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
