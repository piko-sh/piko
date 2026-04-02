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

package provider_disk

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

const testRepo = "test-repo"

func newRealProvider(t *testing.T) storage_domain.StorageProviderPort {
	t.Helper()

	directory := t.TempDir()
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	provider, err := NewDiskProvider(Config{Sandbox: sandbox})
	require.NoError(t, err)
	return provider
}

func TestIntegration_PutGet_RoundTrip(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("hello world - round-trip test content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "round-trip.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "round-trip.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_PutGet_EmptyFile(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "empty.bin",
		Reader:      bytes.NewReader(nil),
		Size:        0,
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "empty.bin",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestIntegration_PutGet_LargeFile(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	content := make([]byte, 2*1024*1024)
	_, err := rand.Read(content)
	require.NoError(t, err)

	err = provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "large-file.bin",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "large-file.bin",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_PutGet_NonSeekableStream(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("data from a non-seekable io.Pipe reader")

	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(content)
		_ = pw.Close()
	}()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "pipe-stream.bin",
		Reader:      pr,
		Size:        int64(len(content)),
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "pipe-stream.bin",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved,
		"non-seekable io.Pipe reader should upload correctly via io.Copy")
}

func TestIntegration_PutGet_NestedKey(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("deeply nested file content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "a/b/c/d/deep.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "a/b/c/d/deep.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_Get_ByteRange(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("0123456789ABCDEFGHIJ")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "range-test.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "range-test.txt",
		ByteRange:  &storage_dto.ByteRange{Start: 5, End: 9},
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("56789"), retrieved)
}

func TestIntegration_Get_ByteRange_OpenEnded(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("0123456789ABCDEFGHIJ")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "range-open.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "range-open.txt",
		ByteRange:  &storage_dto.ByteRange{Start: 15, End: -1},
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("FGHIJ"), retrieved)
}

func TestIntegration_Get_ByteRange_SingleByte(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("ABCDEFGHIJ")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "range-single.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "range-single.txt",
		ByteRange:  &storage_dto.ByteRange{Start: 3, End: 3},
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("D"), retrieved)
}

func TestIntegration_Stat_ReturnsCorrectInfo(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("stat test content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "stat-test.json",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "application/json",
	})
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "stat-test.json",
	})
	require.NoError(t, err)

	assert.Equal(t, int64(len(content)), info.Size)
	assert.False(t, info.LastModified.IsZero())
	assert.Contains(t, info.ContentType, "json",
		"MIME type should be inferred from .json extension")
}

func TestIntegration_Stat_NonExistent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	_, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "nonexistent.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegration_Stat_ZeroSizeFile(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "zero.bin",
		Reader:      bytes.NewReader(nil),
		Size:        0,
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "zero.bin",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size)
}

func TestIntegration_Metadata_RoundTrip(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	metadata := map[string]string{
		"x-piko-transformers": `["gzip","crypto-service"]`,
		"author":              "test-suite",
	}

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "meta-test.bin",
		Reader:      bytes.NewReader([]byte("metadata test")),
		Size:        13,
		ContentType: "application/octet-stream",
		Metadata:    metadata,
	})
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "meta-test.bin",
	})
	require.NoError(t, err)
	assert.Equal(t, `["gzip","crypto-service"]`, info.Metadata["x-piko-transformers"])
	assert.Equal(t, "test-suite", info.Metadata["author"])
}

func TestIntegration_Metadata_NilDoesNotCreateSidecar(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "no-meta.bin",
		Reader:      bytes.NewReader([]byte("no metadata")),
		Size:        11,
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "no-meta.bin",
	})
	require.NoError(t, err)
	assert.Nil(t, info.Metadata)
}

func TestIntegration_GetHash_ComputesHash(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "hash-test.txt",
		Reader:      bytes.NewReader([]byte("hash me")),
		Size:        7,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	hash, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "hash-test.txt",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64, "SHA256 hex string should be 64 characters")
}

func TestIntegration_GetHash_CacheHit(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "hash-cache.txt",
		Reader:      bytes.NewReader([]byte("cache test")),
		Size:        10,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	hash1, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "hash-cache.txt",
	})
	require.NoError(t, err)

	hash2, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "hash-cache.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2, "cached hash should match original computation")
}

func TestIntegration_GetHash_StaleCache(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "hash-stale.txt",
		Reader:      bytes.NewReader([]byte("version one")),
		Size:        11,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	hash1, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "hash-stale.txt",
	})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "hash-stale.txt",
		Reader:      bytes.NewReader([]byte("version two")),
		Size:        11,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	hash2, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "hash-stale.txt",
	})
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2,
		"hash should be recomputed when file is modified after cache")
}

func TestIntegration_GetHash_NonExistent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	_, err := provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "nonexistent.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegration_Exists(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	exists, err := provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "will-exist.txt",
	})
	require.NoError(t, err)
	assert.False(t, exists)

	err = provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "will-exist.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	exists, err = provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "will-exist.txt",
	})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIntegration_Remove(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "to-remove.txt",
		Reader:      bytes.NewReader([]byte("delete me")),
		Size:        9,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	exists, err := provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "to-remove.txt",
	})
	require.NoError(t, err)
	assert.True(t, exists)

	err = provider.Remove(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "to-remove.txt",
	})
	require.NoError(t, err)

	exists, err = provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "to-remove.txt",
	})
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIntegration_Remove_Idempotent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Remove(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "never-existed.txt",
	})
	assert.NoError(t, err, "removing non-existent file should succeed silently")
}

func TestIntegration_Remove_CleansSidecarFiles(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "sidecar-cleanup.txt",
		Reader:      bytes.NewReader([]byte("data")),
		Size:        4,
		ContentType: "text/plain",
		Metadata:    map[string]string{"key": "value"},
	})
	require.NoError(t, err)

	_, err = provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "sidecar-cleanup.txt",
	})
	require.NoError(t, err)

	err = provider.Remove(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "sidecar-cleanup.txt",
	})
	require.NoError(t, err)

	_, err = provider.GetHash(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "sidecar-cleanup.txt",
	})
	assert.Error(t, err)
}

func TestIntegration_Rename(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("rename test content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "old-name.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = provider.Rename(ctx, testRepo, "old-name.txt", "new-name.txt")
	require.NoError(t, err)

	exists, err := provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "old-name.txt",
	})
	require.NoError(t, err)
	assert.False(t, exists, "old key should no longer exist")

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "new-name.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_Rename_ToNestedPath(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "flat.txt",
		Reader:      bytes.NewReader([]byte("flat file")),
		Size:        9,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = provider.Rename(ctx, testRepo, "flat.txt", "nested/directory/renamed.txt")
	require.NoError(t, err)

	exists, err := provider.Exists(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "nested/directory/renamed.txt",
	})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIntegration_Copy_SameRepository(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("copy source content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "copy-src.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = provider.Copy(ctx, testRepo, "copy-src.txt", "copy-dst.txt")
	require.NoError(t, err)

	srcReader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "copy-src.txt",
	})
	require.NoError(t, err)
	srcData, _ := io.ReadAll(srcReader)
	_ = srcReader.Close()
	assert.Equal(t, content, srcData)

	dstReader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "copy-dst.txt",
	})
	require.NoError(t, err)
	dstData, _ := io.ReadAll(dstReader)
	_ = dstReader.Close()
	assert.Equal(t, content, dstData)
}

func TestIntegration_CopyToAnotherRepository(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("cross-repo copy content")

	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "cross-src.txt",
		Reader:      bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = provider.CopyToAnotherRepository(ctx, testRepo, "cross-src.txt", "other-repo", "cross-dst.txt")
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: "other-repo",
		Key:        "cross-dst.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_Copy_NonExistentSource(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	err := provider.Copy(ctx, testRepo, "nonexistent.txt", "dst.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")
}

func TestIntegration_Overwrite_SameKey_DifferentContent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	contentA := []byte("version A")
	err := provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "overwrite.txt",
		Reader:      bytes.NewReader(contentA),
		Size:        int64(len(contentA)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	contentB := []byte("version B - different length and content")
	err = provider.Put(ctx, &storage_dto.PutParams{
		Repository:  testRepo,
		Key:         "overwrite.txt",
		Reader:      bytes.NewReader(contentB),
		Size:        int64(len(contentB)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "overwrite.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, contentB, retrieved,
		"overwritten key should return the latest content")
}

func TestIntegration_Overwrite_Idempotent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()
	content := []byte("idempotent content")

	for range 3 {
		err := provider.Put(ctx, &storage_dto.PutParams{
			Repository:  testRepo,
			Key:         "idempotent.txt",
			Reader:      bytes.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)
	}

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "idempotent.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	retrieved, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, retrieved)
}

func TestIntegration_PutMany(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	objects := make([]storage_dto.PutObjectSpec, 5)
	for i := range 5 {
		content := fmt.Sprintf("batch content %d", i)
		objects[i] = storage_dto.PutObjectSpec{
			Key:         fmt.Sprintf("batch/%d.txt", i),
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		}
	}

	result, err := provider.PutMany(ctx, &storage_dto.PutManyParams{
		Repository:      testRepo,
		Objects:         objects,
		Concurrency:     3,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalRequested)
	assert.Equal(t, 5, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.SuccessfulKeys, 5)

	for i := range 5 {
		exists, err := provider.Exists(ctx, storage_dto.GetParams{
			Repository: testRepo,
			Key:        fmt.Sprintf("batch/%d.txt", i),
		})
		require.NoError(t, err)
		assert.True(t, exists)
	}
}

func TestIntegration_PutMany_EmptyBatch(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	result, err := provider.PutMany(ctx, &storage_dto.PutManyParams{
		Repository:      testRepo,
		Objects:         nil,
		Concurrency:     3,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
}

func TestIntegration_RemoveMany(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	keys := make([]string, 5)
	for i := range 5 {
		key := fmt.Sprintf("remove-batch/%d.txt", i)
		keys[i] = key
		content := fmt.Sprintf("content %d", i)

		err := provider.Put(ctx, &storage_dto.PutParams{
			Repository:  testRepo,
			Key:         key,
			Reader:      strings.NewReader(content),
			Size:        int64(len(content)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)
	}

	result, err := provider.RemoveMany(ctx, storage_dto.RemoveManyParams{
		Repository:      testRepo,
		Keys:            keys,
		Concurrency:     3,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalRequested)
	assert.Equal(t, 5, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)

	for _, key := range keys {
		exists, err := provider.Exists(ctx, storage_dto.GetParams{
			Repository: testRepo,
			Key:        key,
		})
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestIntegration_RemoveMany_EmptyBatch(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	result, err := provider.RemoveMany(ctx, storage_dto.RemoveManyParams{
		Repository:      testRepo,
		Keys:            nil,
		Concurrency:     3,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
}

func TestIntegration_Concurrent_PutGet(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	const goroutines = 10

	type result struct {
		key      string
		original []byte
	}
	results := make([]result, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			content := fmt.Appendf(nil, "concurrent content %d - unique data", i)
			key := fmt.Sprintf("concurrent/%d.txt", i)

			err := provider.Put(ctx, &storage_dto.PutParams{
				Repository:  testRepo,
				Key:         key,
				Reader:      bytes.NewReader(content),
				Size:        int64(len(content)),
				ContentType: "text/plain",
			})
			require.NoError(t, err)
			results[i] = result{key: key, original: content}
		})
	}
	wg.Wait()

	for i, uploadRes := range results {
		reader, err := provider.Get(ctx, storage_dto.GetParams{
			Repository: testRepo,
			Key:        uploadRes.key,
		})
		require.NoError(t, err)

		data, err := io.ReadAll(reader)
		_ = reader.Close()
		require.NoError(t, err)
		assert.Equal(t, uploadRes.original, data,
			"goroutine %d: concurrent upload should round-trip correctly", i)
	}
}

func TestIntegration_Concurrent_OverwriteSameKey(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	const goroutines = 10

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			content := fmt.Appendf(nil, "writer %d content", i)
			err := provider.Put(ctx, &storage_dto.PutParams{
				Repository:  testRepo,
				Key:         "contested.txt",
				Reader:      bytes.NewReader(content),
				Size:        int64(len(content)),
				ContentType: "text/plain",
			})
			require.NoError(t, err)
		})
	}
	wg.Wait()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "contested.txt",
	})
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	content := string(data)
	assert.True(t, strings.HasPrefix(content, "writer "),
		"content should be from a single, complete write (got: %q)", content)
	assert.True(t, strings.HasSuffix(content, " content"),
		"content should not be corrupted by concurrent writes (got: %q)", content)
}

func TestIntegration_Get_NonExistent(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	_, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: testRepo,
		Key:        "does-not-exist.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegration_Capabilities(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)

	assert.False(t, provider.SupportsMultipart())
	assert.False(t, provider.SupportsBatchOperations())
	assert.False(t, provider.SupportsRetry())
	assert.False(t, provider.SupportsCircuitBreaking())
	assert.False(t, provider.SupportsRateLimiting())
	assert.False(t, provider.SupportsPresignedURLs())
}

func TestIntegration_PresignURL_NotSupported(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	_, err := provider.PresignURL(ctx, storage_dto.PresignParams{
		Repository:  testRepo,
		Key:         "test.txt",
		ContentType: "text/plain",
		ExpiresIn:   15 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestIntegration_PresignDownloadURL_NotSupported(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)
	ctx := t.Context()

	_, err := provider.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{
		Repository: testRepo,
		Key:        "test.txt",
		ExpiresIn:  15 * time.Minute,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestIntegration_Close(t *testing.T) {
	t.Parallel()

	provider := newRealProvider(t)

	err := provider.Close(t.Context())
	assert.NoError(t, err)
}
