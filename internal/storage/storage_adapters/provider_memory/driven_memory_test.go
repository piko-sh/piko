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

package provider_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const testRepository = "registry"

func newTestProvider(t *testing.T, maxBytes int64) *Provider {
	t.Helper()
	provider, err := New(Config{MaxBytes: maxBytes})
	require.NoError(t, err)
	require.NotNil(t, provider)
	t.Cleanup(func() {
		require.NoError(t, provider.Close(context.Background()))
	})
	return provider
}

func putObject(t *testing.T, provider *Provider, repository, key, contentType string, metadata map[string]string, content []byte) {
	t.Helper()
	err := provider.Put(context.Background(), &storage_dto.PutParams{
		Reader:               bytes.NewReader(content),
		MultipartConfig:      nil,
		TransformConfig:      nil,
		Metadata:             metadata,
		Key:                  key,
		ContentType:          contentType,
		HashAlgorithm:        "",
		ExpectedHash:         "",
		Repository:           repository,
		Size:                 int64(len(content)),
		UseContentAddressing: false,
	})
	require.NoError(t, err)
}

func getObject(t *testing.T, provider *Provider, params storage_dto.GetParams) []byte {
	t.Helper()
	reader, err := provider.Get(context.Background(), params)
	require.NoError(t, err)
	defer func() { require.NoError(t, reader.Close()) }()
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	return content
}

func TestNewRejectsNonPositiveBudget(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		maxBytes int64
	}{
		{name: "zero", maxBytes: 0},
		{name: "negative", maxBytes: -1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			provider, err := New(Config{MaxBytes: testCase.maxBytes})
			require.Error(t, err)
			assert.Nil(t, provider)
			assert.ErrorIs(t, err, ErrInvalidMaxBytes)
		})
	}
}

func TestInterfaceCompliance(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1024)
	require.Implements(t, (*storage_domain.StorageProviderPort)(nil), provider)
}

func TestPutGetRoundTrip(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("hello registry overlay")

	putObject(t, provider, testRepository, "blobs/a", "text/plain", nil, content)

	got := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "blobs/a",
		Repository:      testRepository,
	})
	assert.Equal(t, content, got)
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	_, err := provider.Get(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "absent",
		Repository:      testRepository,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestGetByteRange(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	putObject(t, provider, testRepository, "ranged", "application/octet-stream", nil, []byte("0123456789"))

	testCases := []struct {
		name     string
		expected string
		start    int64
		end      int64
	}{
		{name: "middle inclusive", expected: "2345", start: 2, end: 5},
		{name: "to end sentinel", expected: "3456789", start: 3, end: -1},
		{name: "single byte", expected: "0", start: 0, end: 0},
		{name: "end clamped beyond length", expected: "89", start: 8, end: 100},
		{name: "start beyond length yields empty", expected: "", start: 20, end: -1},
		{name: "negative start yields empty", expected: "", start: -1, end: 5},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := getObject(t, provider, storage_dto.GetParams{
				ByteRange:       &storage_dto.ByteRange{Start: testCase.start, End: testCase.end},
				TransformConfig: nil,
				Key:             "ranged",
				Repository:      testRepository,
			})
			assert.Equal(t, testCase.expected, string(got))
		})
	}
}

func TestStat(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("stat me")
	metadata := map[string]string{"origin": "seed"}
	before := time.Now()

	putObject(t, provider, testRepository, "stat/key", "text/markdown", metadata, content)

	info, err := provider.Stat(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "stat/key",
		Repository:      testRepository,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), info.Size)
	assert.Equal(t, "text/markdown", info.ContentType)
	assert.Equal(t, metadata, info.Metadata)
	assert.Empty(t, info.ETag)
	assert.False(t, info.LastModified.Before(before))
}

func TestStatMissingReturnsNotFound(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	_, err := provider.Stat(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "nope",
		Repository:      testRepository,
	})
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestPutCopiesMetadataDefensively(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	metadata := map[string]string{"key": "original"}

	putObject(t, provider, testRepository, "defensive", "text/plain", metadata, []byte("body"))
	metadata["key"] = "mutated"

	info, err := provider.Stat(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "defensive",
		Repository:      testRepository,
	})
	require.NoError(t, err)
	assert.Equal(t, "original", info.Metadata["key"])
}

func TestExists(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	putObject(t, provider, testRepository, "present", "text/plain", nil, []byte("x"))

	presentParams := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "present", Repository: testRepository}
	absentParams := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "absent", Repository: testRepository}

	present, err := provider.Exists(context.Background(), presentParams)
	require.NoError(t, err)
	assert.True(t, present)

	absent, err := provider.Exists(context.Background(), absentParams)
	require.NoError(t, err)
	assert.False(t, absent)
}

func TestGetHashMatchesSha256(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("content to hash")
	putObject(t, provider, testRepository, "hashed", "text/plain", nil, content)

	hash, err := provider.GetHash(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "hashed",
		Repository:      testRepository,
	})
	require.NoError(t, err)

	expected := sha256.Sum256(content)
	assert.Equal(t, hex.EncodeToString(expected[:]), hash)
}

func TestGetHashMissingReturnsNotFound(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	_, err := provider.GetHash(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "missing",
		Repository:      testRepository,
	})
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestRemoveIsIdempotent(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	params := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "removable", Repository: testRepository}
	putObject(t, provider, testRepository, "removable", "text/plain", nil, []byte("bye"))

	require.NoError(t, provider.Remove(context.Background(), params))
	require.NoError(t, provider.Remove(context.Background(), params))

	present, err := provider.Exists(context.Background(), params)
	require.NoError(t, err)
	assert.False(t, present)
}

func TestRenameMovesObject(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("atomic write")
	putObject(t, provider, testRepository, "temp/file", "text/plain", nil, content)

	require.NoError(t, provider.Rename(context.Background(), testRepository, "temp/file", "final/file"))

	oldParams := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "temp/file", Repository: testRepository}
	present, err := provider.Exists(context.Background(), oldParams)
	require.NoError(t, err)
	assert.False(t, present)

	moved := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "final/file",
		Repository:      testRepository,
	})
	assert.Equal(t, content, moved)
}

func TestRenameMissingSourceErrors(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	err := provider.Rename(context.Background(), testRepository, "ghost", "target")
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestCopyWithinRepository(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("copy source")
	putObject(t, provider, testRepository, "src", "text/plain", map[string]string{"a": "b"}, content)

	require.NoError(t, provider.Copy(context.Background(), testRepository, "src", "dst"))

	copied := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "dst",
		Repository:      testRepository,
	})
	assert.Equal(t, content, copied)

	original := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "src",
		Repository:      testRepository,
	})
	assert.Equal(t, content, original)
}

func TestCopyMissingSourceErrors(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	err := provider.Copy(context.Background(), testRepository, "ghost", "dst")
	assert.ErrorIs(t, err, ErrObjectNotFound)
}

func TestCopyToAnotherRepository(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("cross repo copy")
	putObject(t, provider, "source-repo", "key", "text/plain", nil, content)

	require.NoError(t, provider.CopyToAnotherRepository(context.Background(), "source-repo", "key", "dest-repo", "key"))

	copied := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "key",
		Repository:      "dest-repo",
	})
	assert.Equal(t, content, copied)
}

func TestCopyProducesIndependentSlice(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	content := []byte("mutate me later")
	putObject(t, provider, testRepository, "src", "text/plain", nil, content)

	require.NoError(t, provider.Copy(context.Background(), testRepository, "src", "dst"))
	require.NoError(t, provider.Remove(context.Background(), storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "src", Repository: testRepository}))

	copied := getObject(t, provider, storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "dst",
		Repository:      testRepository,
	})
	assert.Equal(t, content, copied)
}

func TestPutRejectsOversizedObject(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 100)

	err := provider.Put(context.Background(), &storage_dto.PutParams{
		Reader:               bytes.NewReader(bytes.Repeat([]byte("a"), 200)),
		MultipartConfig:      nil,
		TransformConfig:      nil,
		Metadata:             nil,
		Key:                  "oversized",
		ContentType:          "text/plain",
		HashAlgorithm:        "",
		ExpectedHash:         "",
		Repository:           testRepository,
		Size:                 200,
		UseContentAddressing: false,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrObjectTooLarge)

	present, existsErr := provider.Exists(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             "oversized",
		Repository:      testRepository,
	})
	require.NoError(t, existsErr)
	assert.False(t, present)
}

func TestPutFitsWhenFootprintEqualsBudget(t *testing.T) {
	t.Parallel()
	const (
		contentType = "text/plain"
		key         = "exact"
	)
	data := bytes.Repeat([]byte("b"), 100)
	footprint := entryFootprint(compositeKey(testRepository, key), &storedObject{
		lastModified: time.Time{},
		contentType:  contentType,
		metadata:     nil,
		data:         data,
	})

	atBudget := newTestProvider(t, footprint)
	putObject(t, atBudget, testRepository, key, contentType, nil, data)
	present, err := atBudget.Exists(context.Background(), storage_dto.GetParams{
		ByteRange:       nil,
		TransformConfig: nil,
		Key:             key,
		Repository:      testRepository,
	})
	require.NoError(t, err)
	assert.True(t, present, "an object whose full footprint equals the budget must fit")

	underBudget := newTestProvider(t, footprint-1)
	err = underBudget.Put(context.Background(), &storage_dto.PutParams{
		Reader:               bytes.NewReader(data),
		MultipartConfig:      nil,
		TransformConfig:      nil,
		Metadata:             nil,
		Key:                  key,
		ContentType:          contentType,
		HashAlgorithm:        "",
		ExpectedHash:         "",
		Repository:           testRepository,
		Size:                 int64(len(data)),
		UseContentAddressing: false,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrObjectTooLarge, "one byte under the footprint must be rejected")
}

func TestPutChargesOverheadKeyAndMetadata(t *testing.T) {
	t.Parallel()

	emptyProvider := newTestProvider(t, 1)
	emptyErr := emptyProvider.Put(context.Background(), &storage_dto.PutParams{
		Reader:               bytes.NewReader(nil),
		MultipartConfig:      nil,
		TransformConfig:      nil,
		Metadata:             nil,
		Key:                  "empty",
		ContentType:          "text/plain",
		HashAlgorithm:        "",
		ExpectedHash:         "",
		Repository:           testRepository,
		Size:                 0,
		UseContentAddressing: false,
	})
	require.Error(t, emptyErr)
	assert.ErrorIs(t, emptyErr, ErrObjectTooLarge, "an empty object is still charged the per-entry overhead and key length")

	const key = "withmeta"
	data := []byte("x")
	footprintWithoutMetadata := entryFootprint(compositeKey(testRepository, key), &storedObject{
		lastModified: time.Time{},
		contentType:  "text/plain",
		metadata:     nil,
		data:         data,
	})

	provider := newTestProvider(t, footprintWithoutMetadata+128)
	putObject(t, provider, testRepository, key, "text/plain", nil, data)

	metadataErr := provider.Put(context.Background(), &storage_dto.PutParams{
		Reader:               bytes.NewReader(data),
		MultipartConfig:      nil,
		TransformConfig:      nil,
		Metadata:             map[string]string{"k": strings.Repeat("m", 256)},
		Key:                  key,
		ContentType:          "text/plain",
		HashAlgorithm:        "",
		ExpectedHash:         "",
		Repository:           testRepository,
		Size:                 int64(len(data)),
		UseContentAddressing: false,
	})
	require.Error(t, metadataErr)
	assert.ErrorIs(t, metadataErr, ErrObjectTooLarge, "metadata counts toward the footprint")
}

func TestEvictionStaysWithinBudget(t *testing.T) {
	t.Parallel()
	const (
		maxBytes    = 50000
		objectBytes = 100
		objectCount = 500
	)
	provider := newTestProvider(t, maxBytes)
	ctx := context.Background()

	for index := range objectCount {
		putObject(t, provider, testRepository, fmt.Sprintf("blob-%03d", index), "application/octet-stream", nil, bytes.Repeat([]byte("z"), objectBytes))
	}

	earliestParams := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "blob-000", Repository: testRepository}
	require.Eventually(t, func() bool {
		present, err := provider.Exists(ctx, earliestParams)
		return err == nil && !present
	}, 2*time.Second, 10*time.Millisecond, "the earliest object should be evicted once the budget is exceeded")

	latestParams := storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: fmt.Sprintf("blob-%03d", objectCount-1), Repository: testRepository}
	require.Eventually(t, func() bool {
		present, err := provider.Exists(ctx, latestParams)
		return err == nil && present
	}, 2*time.Second, 10*time.Millisecond, "the most recently written object should survive once the write buffer settles")

	survivors := 0
	for index := range objectCount {
		present, existsErr := provider.Exists(ctx, storage_dto.GetParams{
			ByteRange:       nil,
			TransformConfig: nil,
			Key:             fmt.Sprintf("blob-%03d", index),
			Repository:      testRepository,
		})
		require.NoError(t, existsErr)
		if present {
			survivors++
		}
	}
	assert.Less(t, survivors, objectCount, "some objects should have been evicted to respect the budget")
}

func TestPresignURLUnsupported(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1024)

	uploadURL, err := provider.PresignURL(context.Background(), storage_dto.PresignParams{
		Key:         "k",
		ContentType: "text/plain",
		Repository:  testRepository,
		ExpiresIn:   time.Minute,
	})
	assert.Empty(t, uploadURL)
	assert.ErrorIs(t, err, ErrPresignUnsupported)

	downloadURL, err := provider.PresignDownloadURL(context.Background(), storage_dto.PresignDownloadParams{
		Key:         "k",
		Repository:  testRepository,
		FileName:    "file.txt",
		ContentType: "text/plain",
		ExpiresIn:   time.Minute,
	})
	assert.Empty(t, downloadURL)
	assert.ErrorIs(t, err, ErrPresignUnsupported)
}

func TestCapabilityFlagsAreFalse(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1024)

	assert.False(t, provider.SupportsMultipart())
	assert.False(t, provider.SupportsRetry())
	assert.False(t, provider.SupportsCircuitBreaking())
	assert.False(t, provider.SupportsRateLimiting())
	assert.False(t, provider.SupportsBatchOperations())
	assert.False(t, provider.SupportsPresignedURLs())
}

func TestPutMany(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)

	result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
		TransformConfig: nil,
		Repository:      testRepository,
		Objects: []storage_dto.PutObjectSpec{
			{Reader: strings.NewReader("first"), Key: "one", ContentType: "text/plain", Size: 5},
			{Reader: strings.NewReader("second"), Key: "two", ContentType: "text/plain", Size: 6},
		},
		Concurrency:     1,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalRequested)
	assert.Equal(t, 2, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.ElementsMatch(t, []string{"one", "two"}, result.SuccessfulKeys)

	got := getObject(t, provider, storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "two", Repository: testRepository})
	assert.Equal(t, "second", string(got))
}

func TestPutManyRecordsOversizedFailure(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 100)

	result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
		TransformConfig: nil,
		Repository:      testRepository,
		Objects: []storage_dto.PutObjectSpec{
			{Reader: strings.NewReader("ok"), Key: "small", ContentType: "text/plain", Size: 2},
			{Reader: bytes.NewReader(bytes.Repeat([]byte("x"), 500)), Key: "huge", ContentType: "text/plain", Size: 500},
		},
		Concurrency:     1,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalSuccessful)
	assert.Equal(t, 1, result.TotalFailed)
	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "huge", result.FailedKeys[0].Key)
}

func TestRemoveMany(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	putObject(t, provider, testRepository, "a", "text/plain", nil, []byte("1"))
	putObject(t, provider, testRepository, "b", "text/plain", nil, []byte("2"))

	result, err := provider.RemoveMany(context.Background(), storage_dto.RemoveManyParams{
		Repository:      testRepository,
		Keys:            []string{"a", "b", "never-existed"},
		Concurrency:     1,
		ContinueOnError: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalRequested)
	assert.Equal(t, 3, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)

	for _, key := range []string{"a", "b"} {
		present, existsErr := provider.Exists(context.Background(), storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: key, Repository: testRepository})
		require.NoError(t, existsErr)
		assert.False(t, present)
	}
}

func TestProviderTypeAndMetadata(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 2048)
	putObject(t, provider, testRepository, "k", "text/plain", nil, []byte("payload"))

	assert.Equal(t, "memory", provider.GetProviderType())

	metadata := provider.GetProviderMetadata()
	assert.Equal(t, "memory", metadata["type"])
	assert.Equal(t, true, metadata["writable"])
	assert.Equal(t, int64(2048), metadata["maxBytes"])
	weightedSize, ok := metadata["weightedSize"].(uint64)
	require.True(t, ok)
	assert.LessOrEqual(t, weightedSize, uint64(2048))
}

func TestClose(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1024)
	assert.NoError(t, provider.Close(context.Background()))
}

func TestKeysAreIsolatedByRepository(t *testing.T) {
	t.Parallel()
	provider := newTestProvider(t, 1<<20)
	putObject(t, provider, "repo-a", "same-key", "text/plain", nil, []byte("from a"))
	putObject(t, provider, "repo-b", "same-key", "text/plain", nil, []byte("from b"))

	fromA := getObject(t, provider, storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "same-key", Repository: "repo-a"})
	fromB := getObject(t, provider, storage_dto.GetParams{ByteRange: nil, TransformConfig: nil, Key: "same-key", Repository: "repo-b"})

	assert.Equal(t, "from a", string(fromA))
	assert.Equal(t, "from b", string(fromB))
}
