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
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestConcurrency_ParallelUploads(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	const goroutines = 10

	type uploadResult struct {
		key      string
		original []byte
	}
	results := make([]uploadResult, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			data := generateRepeatableText(10*1024 + i)
			key := uniqueKey(t, fmt.Sprintf("concurrent-upload-%d", i))
			putObject(ctx, t, wrapper, key, data)
			results[i] = uploadResult{key: key, original: data}
		})
	}
	wg.Wait()

	for i, uploadRes := range results {
		retrieved := getObject(ctx, t, wrapper, uploadRes.key)
		assert.Equal(t, uploadRes.original, retrieved,
			"goroutine %d: upload should round-trip correctly", i)
	}
}

func TestConcurrency_ParallelDownloads(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	zstdT := newZstdTransformer(t)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{zstdT, crypto.transformer},
		[]string{"zstd", "crypto-service"})

	original := generateRepeatableText(1024 * 1024)
	key := uniqueKey(t, "concurrent-download")
	putObject(ctx, t, wrapper, key, original)

	const goroutines = 10

	var wg sync.WaitGroup
	errors := make([]error, goroutines)
	results := make([][]byte, goroutines)

	for i := range goroutines {
		wg.Go(func() {
			params := storage_dto.GetParams{
				Repository: testRepoPrimary,
				Key:        key,
			}
			reader, err := wrapper.Get(ctx, params)
			if err != nil {
				errors[i] = err
				return
			}
			defer func() { _ = reader.Close() }()

			data, err := io.ReadAll(reader)
			if err != nil {
				errors[i] = err
				return
			}
			results[i] = data
		})
	}
	wg.Wait()

	for i := range goroutines {
		require.NoError(t, errors[i], "goroutine %d: download should not error", i)
		assert.Equal(t, original, results[i],
			"goroutine %d: concurrent download should return correct data", i)
	}
}

func TestConcurrency_ContextCancellationDuringUpload(t *testing.T) {
	provider := newS3Provider(t.Context(), t, globalEnv)
	crypto := newCryptoSetup(t)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	data := generateRepeatableText(5 * 1024 * 1024)

	ctx, cancel := context.WithCancelCause(t.Context())

	reader := &cancellingReader{
		inner:     bytes.NewReader(data),
		cancel:    cancel,
		remaining: 1024 * 1024,
	}

	params := &storage_dto.PutParams{
		Repository:  testRepoPrimary,
		Key:         uniqueKey(t, "concurrent-cancel-upload"),
		Reader:      reader,
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}

	err := wrapper.Put(ctx, params)
	assert.Error(t, err, "upload with cancelled context should fail")
}

func TestConcurrency_ContextCancellationDuringDownload(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	original := generateRepeatableText(5 * 1024 * 1024)
	key := uniqueKey(t, "concurrent-cancel-download")
	putObject(ctx, t, wrapper, key, original)

	shortCtx, cancel := context.WithTimeoutCause(ctx, 1*time.Millisecond, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	time.Sleep(5 * time.Millisecond)

	params := storage_dto.GetParams{
		Repository: testRepoPrimary,
		Key:        key,
	}

	reader, err := wrapper.Get(shortCtx, params)
	if err != nil {

		assert.Error(t, err)
		return
	}
	defer func() { _ = reader.Close() }()

	_, err = io.ReadAll(reader)
	assert.Error(t, err, "download with expired context should fail during read")
}

func TestOverwrite_SameKey_DifferentContent(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	key := uniqueKey(t, "overwrite-different-content")

	contentA := []byte("This is content version A - the original upload.")
	putObject(ctx, t, wrapper, key, contentA)

	contentB := []byte("This is content version B - the overwrite.")
	putObject(ctx, t, wrapper, key, contentB)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, contentB, retrieved,
		"overwritten key should return the latest content")
}

func TestOverwrite_SameKey_DifferentTransformers(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	key := uniqueKey(t, "overwrite-different-transformers")

	gzipWrapper := newTransformerWrapper(t, provider,
		singleTransformer(gzipT),
		[]string{"gzip"})
	contentA := []byte("First upload through gzip.")
	putObject(ctx, t, gzipWrapper, key, contentA)

	cryptoWrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})
	contentB := []byte("Second upload through crypto.")
	putObject(ctx, t, cryptoWrapper, key, contentB)

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	require.NoError(t, registry.Register(crypto.transformer))
	downloadWrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	retrieved := getObject(ctx, t, downloadWrapper, key)
	assert.Equal(t, contentB, retrieved,
		"overwritten key should use the latest transformer config from metadata")
}

func TestOverwrite_Idempotent(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	key := uniqueKey(t, "overwrite-idempotent")
	content := generateRepeatableText(10 * 1024)

	putObject(ctx, t, wrapper, key, content)
	putObject(ctx, t, wrapper, key, content)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, content, retrieved,
		"re-uploading same content should not corrupt state")
}

func TestS3Key_SpecialCharacters(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	original := generateRepeatableText(1024)

	testCases := []struct {
		name      string
		keySuffix string
	}{
		{name: "spaces", keySuffix: "hello world/test file.txt"},
		{name: "unicode", keySuffix: "data/\xe4\xb8\xad\xe6\x96\x87/test.bin"},
		{name: "deep_nesting", keySuffix: "a/b/c/d/e/f/g/h/file.txt"},
		{name: "url_special_chars", keySuffix: "data/file+name&param=value.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := fmt.Sprintf("s3key-special/%s/%d", tc.keySuffix, time.Now().UnixNano())
			putObject(ctx, t, wrapper, key, original)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved,
				"special characters in key should not affect round-trip")
		})
	}
}

func TestS3Key_VeryLongKey(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	original := generateRepeatableText(1024)

	segment := "long-segment/"
	var builder strings.Builder
	builder.WriteString("s3key-long/")
	for builder.Len() < 1024 {
		builder.WriteString(segment)
	}
	key := builder.String()[:1024]

	putObject(ctx, t, wrapper, key, original)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"1024-character key should round-trip correctly")
}

func TestReaderFailure_UploadPartialRead(t *testing.T) {
	provider := newS3Provider(t.Context(), t, globalEnv)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(gzipT),
		[]string{"gzip"})

	data := generateRepeatableText(10 * 1024)

	params := &storage_dto.PutParams{
		Repository:  testRepoPrimary,
		Key:         uniqueKey(t, "reader-failure-gzip"),
		Reader:      &errorReader{inner: bytes.NewReader(data), remaining: 512},
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}

	err := wrapper.Put(t.Context(), params)
	assert.Error(t, err, "partial reader failure should propagate as upload error")
}

func TestReaderFailure_UploadPartialRead_WithCrypto(t *testing.T) {
	provider := newS3Provider(t.Context(), t, globalEnv)
	crypto := newCryptoSetup(t)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	data := generateRepeatableText(10 * 1024)

	params := &storage_dto.PutParams{
		Repository:  testRepoPrimary,
		Key:         uniqueKey(t, "reader-failure-crypto"),
		Reader:      &errorReader{inner: bytes.NewReader(data), remaining: 512},
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	}

	err := wrapper.Put(t.Context(), params)
	assert.Error(t, err,
		"partial reader failure through crypto pipeline should propagate without deadlock")
}

type cancellingReader struct {
	inner     io.Reader
	cancel    context.CancelCauseFunc
	remaining int
	cancelled bool
}

func (r *cancellingReader) Read(p []byte) (n int, err error) {
	n, err = r.inner.Read(p)
	r.remaining -= n
	if r.remaining <= 0 && !r.cancelled {
		r.cancelled = true
		r.cancel(fmt.Errorf("test: simulating cancelled context"))
	}
	return n, err
}

type errorReader struct {
	inner     *bytes.Reader
	remaining int
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.remaining <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err = r.inner.Read(p)
	r.remaining -= n
	if r.remaining <= 0 && err == nil {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}
