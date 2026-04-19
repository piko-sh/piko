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

package provider_union_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_adapters/provider_disk"
	"piko.sh/piko/internal/storage/storage_adapters/provider_fs"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_adapters/provider_union"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

func newUnion(t *testing.T) storage_domain.StorageProviderPort {
	t.Helper()

	base, err := provider_fs.NewFSProvider(fstest.MapFS{
		"baked.txt": &fstest.MapFile{Data: []byte("baked bytes")},
	})
	require.NoError(t, err, "creating base provider")

	sandbox := safedisk.NewMockSandbox("/overlay", safedisk.ModeReadWrite)
	t.Cleanup(func() { _ = sandbox.Close() })
	overlay, err := provider_disk.NewDiskProvider(provider_disk.Config{Sandbox: sandbox})
	require.NoError(t, err, "creating overlay provider")

	return provider_union.New(base, overlay)
}

func readAll(t *testing.T, reader io.ReadCloser) string {
	t.Helper()
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(data)
}

func TestReadsBakedAssetFromBase(t *testing.T) {
	store := newUnion(t)
	reader, err := store.Get(t.Context(), storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.Equal(t, "baked bytes", readAll(t, reader), "the baked asset must be served from the base")
}

func TestWritesGoToOverlayAndReadBack(t *testing.T) {
	store := newUnion(t)
	err := store.Put(t.Context(), &storage_dto.PutParams{
		Key:    "runtime.txt",
		Reader: bytes.NewReader([]byte("runtime bytes")),
		Size:   int64(len("runtime bytes")),
	})
	require.NoError(t, err)

	reader, err := store.Get(t.Context(), storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.Equal(t, "runtime bytes", readAll(t, reader), "a runtime write must read back through the union")
}

func TestExistsSpansBothLayers(t *testing.T) {
	store := newUnion(t)
	require.NoError(t, store.Put(t.Context(), &storage_dto.PutParams{
		Key: "runtime.txt", Reader: bytes.NewReader([]byte("x")), Size: 1,
	}))

	baked, err := store.Exists(t.Context(), storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.True(t, baked, "a baked asset must report as existing")

	runtime, err := store.Exists(t.Context(), storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.True(t, runtime, "a runtime asset must report as existing")

	absent, err := store.Exists(t.Context(), storage_dto.GetParams{Key: "nope.txt"})
	require.NoError(t, err)
	assert.False(t, absent, "an unknown key must report as absent")
}

func TestRemoveOfBaseObjectIsNoOp(t *testing.T) {
	store := newUnion(t)
	require.NoError(t, store.Remove(t.Context(), storage_dto.GetParams{Key: "baked.txt"}),
		"removing a base-owned object must be a safe no-op")

	reader, err := store.Get(t.Context(), storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err, "the baked asset must still be readable after a remove")
	assert.Equal(t, "baked bytes", readAll(t, reader))
}

func TestReadOnlyUnionRejectsWrites(t *testing.T) {
	base, err := provider_fs.NewFSProvider(fstest.MapFS{
		"baked.txt": &fstest.MapFile{Data: []byte("baked bytes")},
	})
	require.NoError(t, err)
	store := provider_union.New(base, nil)

	reader, err := store.Get(t.Context(), storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.Equal(t, "baked bytes", readAll(t, reader))

	err = store.Put(t.Context(), &storage_dto.PutParams{Key: "x", Reader: bytes.NewReader([]byte("x")), Size: 1})
	require.ErrorIs(t, err, provider_union.ErrReadOnly, "a read-only union must reject writes")
}

func TestRemoveManyFiltersBaseOwnedKeys(t *testing.T) {
	store := newUnion(t)
	require.NoError(t, store.Put(t.Context(), &storage_dto.PutParams{
		Key: "runtime.txt", Reader: bytes.NewReader([]byte("runtime")), Size: int64(len("runtime")),
	}))

	result, err := store.RemoveMany(t.Context(), storage_dto.RemoveManyParams{
		Keys:            []string{"baked.txt", "runtime.txt"},
		ContinueOnError: true,
	})
	require.NoError(t, err, "a mixed batch must not error on the baked key")
	require.NotNil(t, result)
	assert.Zero(t, result.TotalFailed, "the baked key must be filtered out, not forwarded and failed")

	reader, err := store.Get(t.Context(), storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err, "the baked asset must survive the batch remove")
	assert.Equal(t, "baked bytes", readAll(t, reader))

	exists, err := store.Exists(t.Context(), storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.False(t, exists, "the runtime key must be removed from the overlay")
}

func TestRemoveManyOfOnlyBaseOwnedKeysIsNoOp(t *testing.T) {
	base, err := provider_fs.NewFSProvider(fstest.MapFS{
		"baked.txt": &fstest.MapFile{Data: []byte("baked bytes")},
	})
	require.NoError(t, err)
	store := provider_union.New(base, nil)

	result, err := store.RemoveMany(t.Context(), storage_dto.RemoveManyParams{
		Keys:            []string{"baked.txt"},
		ContinueOnError: true,
	})
	require.NoError(t, err, "removing only base-owned keys must not error even with no overlay")
	require.NotNil(t, result)

	other, err := store.RemoveMany(t.Context(), storage_dto.RemoveManyParams{
		Keys:            []string{"runtime.txt"},
		ContinueOnError: true,
	})
	require.ErrorIs(t, err, provider_union.ErrReadOnly,
		"an overlay-owned key with no overlay must still return ErrReadOnly")
	assert.Nil(t, other)
}

func TestNewWithNoLayersIsReadOnly(t *testing.T) {
	store := provider_union.New(nil, nil)
	require.NotNil(t, store, "New(nil, nil) must never return a nil interface")

	_, err := store.Get(t.Context(), storage_dto.GetParams{Key: "anything"})
	require.ErrorIs(t, err, provider_union.ErrNoLayer, "a read with no layers must return ErrNoLayer")

	err = store.Put(t.Context(), &storage_dto.PutParams{Key: "x", Reader: bytes.NewReader([]byte("x")), Size: 1})
	require.ErrorIs(t, err, provider_union.ErrReadOnly, "a write with no layers must return ErrReadOnly")
}

func newBase(t *testing.T) storage_domain.StorageProviderPort {
	t.Helper()

	base, err := provider_fs.NewFSProvider(fstest.MapFS{
		"baked.txt": &fstest.MapFile{Data: []byte("baked bytes")},
	})
	require.NoError(t, err, "creating base provider")
	return base
}

func newUnionWithLayers(t *testing.T) (storage_domain.StorageProviderPort, storage_domain.StorageProviderPort, storage_domain.StorageProviderPort) {
	t.Helper()

	base := newBase(t)

	sandbox := safedisk.NewMockSandbox("/overlay", safedisk.ModeReadWrite)
	t.Cleanup(func() { _ = sandbox.Close() })
	overlay, err := provider_disk.NewDiskProvider(provider_disk.Config{Sandbox: sandbox})
	require.NoError(t, err, "creating overlay provider")

	return provider_union.New(base, overlay), base, overlay
}

func newReadOnlyUnion(t *testing.T) storage_domain.StorageProviderPort {
	t.Helper()
	return provider_union.New(newBase(t), nil)
}

func newMockOverlayUnion(t *testing.T) (storage_domain.StorageProviderPort, *provider_mock.MockStorageProvider) {
	t.Helper()

	overlay := provider_mock.NewMockStorageProvider()
	return provider_union.New(newBase(t), overlay), overlay
}

func asUnion(t *testing.T, store storage_domain.StorageProviderPort) *provider_union.UnionProvider {
	t.Helper()

	union, ok := store.(*provider_union.UnionProvider)
	require.True(t, ok, "the union must be a *UnionProvider")
	return union
}

type closeSpyProvider struct {
	*provider_mock.MockStorageProvider
	closeCount int
}

func (spy *closeSpyProvider) Close(ctx context.Context) error {
	spy.closeCount++
	return spy.MockStorageProvider.Close(ctx)
}

func TestStatPrefersBaseThenOverlay(t *testing.T) {
	store, _, _ := newUnionWithLayers(t)
	ctx := t.Context()

	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "baked.txt",
		Reader: bytes.NewReader([]byte("a much longer overlay copy")),
		Size:   int64(len("a much longer overlay copy")),
	}), "seeding an overlay copy of the baked key")
	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "runtime.txt",
		Reader: bytes.NewReader([]byte("runtime bytes")),
		Size:   int64(len("runtime bytes")),
	}), "seeding a runtime-only key")

	baked, err := store.Stat(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.Equal(t, int64(len("baked bytes")), baked.Size,
		"a key baked into the base must be stat-ed from the base, not the overlay copy")

	runtime, err := store.Stat(ctx, storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.Equal(t, int64(len("runtime bytes")), runtime.Size,
		"a runtime-only key must be stat-ed from the overlay")

	_, err = store.Stat(ctx, storage_dto.GetParams{Key: "absent.txt"})
	require.Error(t, err, "an absent key must error")
}

func TestGetHashPrefersBaseThenOverlay(t *testing.T) {
	store, base, overlay := newUnionWithLayers(t)
	ctx := t.Context()

	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "baked.txt",
		Reader: bytes.NewReader([]byte("an overlay copy with different bytes")),
		Size:   int64(len("an overlay copy with different bytes")),
	}), "seeding an overlay copy of the baked key")
	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "runtime.txt",
		Reader: bytes.NewReader([]byte("runtime bytes")),
		Size:   int64(len("runtime bytes")),
	}), "seeding a runtime-only key")

	baseHash, err := base.GetHash(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	unionBakedHash, err := store.GetHash(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.Equal(t, baseHash, unionBakedHash,
		"a baked key must be hashed from the base, not the overlay copy")

	overlayHash, err := overlay.GetHash(ctx, storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	unionRuntimeHash, err := store.GetHash(ctx, storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.Equal(t, overlayHash, unionRuntimeHash,
		"a runtime-only key must be hashed from the overlay")

	_, err = store.GetHash(ctx, storage_dto.GetParams{Key: "absent.txt"})
	require.Error(t, err, "an absent key must error")
}

func TestListKeysDeduplicatesAcrossLayers(t *testing.T) {
	store, _, _ := newUnionWithLayers(t)
	ctx := t.Context()

	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "baked.txt",
		Reader: bytes.NewReader([]byte("overlay copy")),
		Size:   int64(len("overlay copy")),
	}), "seeding an overlay copy so baked.txt lives in both layers")
	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key:    "runtime.txt",
		Reader: bytes.NewReader([]byte("runtime")),
		Size:   int64(len("runtime")),
	}), "seeding a runtime-only key")

	keys, err := asUnion(t, store).ListKeys(ctx, "")
	require.NoError(t, err)

	counts := make(map[string]int)
	for _, key := range keys {
		counts[key]++
	}
	assert.Equal(t, 1, counts["baked.txt"], "a key present in both layers must appear exactly once")
	assert.Equal(t, 1, counts["runtime.txt"], "a runtime-only key must appear exactly once")
}

func TestGetProviderTypeIsUnion(t *testing.T) {
	store, _, _ := newUnionWithLayers(t)
	assert.Equal(t, "union", asUnion(t, store).GetProviderType(), "the composed provider reports as a union")
}

func TestGetProviderMetadataReportsLayers(t *testing.T) {
	store, _, _ := newUnionWithLayers(t)
	metadata := asUnion(t, store).GetProviderMetadata()

	assert.Equal(t, "union", metadata["type"], "the metadata names the union type")
	assert.Equal(t, true, metadata["writable"], "a union with an overlay is writable")
	assert.Equal(t, "embedded-fs", metadata["base"], "the metadata reports the base provider type")
	assert.Equal(t, "disk", metadata["overlay"], "the metadata reports the overlay provider type")
}

func TestGetProviderMetadataReadOnlyUnionOmitsOverlay(t *testing.T) {
	metadata := asUnion(t, newReadOnlyUnion(t)).GetProviderMetadata()

	assert.Equal(t, false, metadata["writable"], "a union with no overlay is not writable")
	assert.Equal(t, "embedded-fs", metadata["base"], "the metadata still reports the base provider type")
	_, hasOverlay := metadata["overlay"]
	assert.False(t, hasOverlay, "with no overlay there is no overlay entry in the metadata")
}

func TestRemoveOfOverlayObjectDeletesIt(t *testing.T) {
	store, _, _ := newUnionWithLayers(t)
	ctx := t.Context()

	require.NoError(t, store.Put(ctx, &storage_dto.PutParams{
		Key: "runtime.txt", Reader: bytes.NewReader([]byte("runtime")), Size: int64(len("runtime")),
	}))
	require.NoError(t, store.Remove(ctx, storage_dto.GetParams{Key: "runtime.txt"}),
		"removing a runtime-only key must delegate to the overlay")

	exists, err := store.Exists(ctx, storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.False(t, exists, "the runtime key must be gone after a remove")
}

func TestReadOnlyUnionReadsFallThroughToBase(t *testing.T) {
	store := newReadOnlyUnion(t)
	ctx := t.Context()

	reader, err := store.Get(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err, "a baked key must read from the base with no overlay")
	assert.Equal(t, "baked bytes", readAll(t, reader))

	info, err := store.Stat(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err, "a baked key must stat from the base with no overlay")
	assert.Equal(t, int64(len("baked bytes")), info.Size)

	hash, err := store.GetHash(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err, "a baked key must hash from the base with no overlay")
	assert.NotEmpty(t, hash)

	_, err = store.Get(ctx, storage_dto.GetParams{Key: "absent.txt"})
	require.Error(t, err, "an absent key routed through the base alone must error")

	_, err = store.Stat(ctx, storage_dto.GetParams{Key: "absent.txt"})
	require.Error(t, err, "an absent Stat routed through the base alone must error")

	_, err = store.GetHash(ctx, storage_dto.GetParams{Key: "absent.txt"})
	require.Error(t, err, "an absent GetHash routed through the base alone must error")
}

func TestReadOnlyUnionExistsAcrossLayers(t *testing.T) {
	store := newReadOnlyUnion(t)
	ctx := t.Context()

	baked, err := store.Exists(ctx, storage_dto.GetParams{Key: "baked.txt"})
	require.NoError(t, err)
	assert.True(t, baked, "a baked key exists via the base")

	absent, err := store.Exists(ctx, storage_dto.GetParams{Key: "runtime.txt"})
	require.NoError(t, err)
	assert.False(t, absent, "with no overlay a non-base key does not exist")
}

func TestReadOnlyUnionMutatingMethodsReturnErrReadOnly(t *testing.T) {
	store := newReadOnlyUnion(t)
	ctx := t.Context()

	tests := []struct {
		call func() error
		name string
	}{
		{name: "Put", call: func() error {
			return store.Put(ctx, &storage_dto.PutParams{Key: "x", Reader: bytes.NewReader([]byte("x")), Size: 1})
		}},
		{name: "Copy", call: func() error { return store.Copy(ctx, "repo", "src", "dst") }},
		{name: "CopyToAnotherRepository", call: func() error {
			return store.CopyToAnotherRepository(ctx, "repo", "src", "other", "dst")
		}},
		{name: "Rename", call: func() error { return store.Rename(ctx, "repo", "src", "dst") }},
		{name: "Remove", call: func() error {
			return store.Remove(ctx, storage_dto.GetParams{Key: "runtime.txt"})
		}},
		{name: "PresignURL", call: func() error {
			_, err := store.PresignURL(ctx, storage_dto.PresignParams{Key: "x"})
			return err
		}},
		{name: "PresignDownloadURL", call: func() error {
			_, err := store.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{Key: "x"})
			return err
		}},
		{name: "PutMany", call: func() error {
			_, err := store.PutMany(ctx, &storage_dto.PutManyParams{
				Objects: []storage_dto.PutObjectSpec{{Key: "x", Reader: bytes.NewReader([]byte("x")), Size: 1, ContentType: "text/plain"}},
			})
			return err
		}},
		{name: "RemoveMany", call: func() error {
			_, err := store.RemoveMany(ctx, storage_dto.RemoveManyParams{Keys: []string{"runtime.txt"}})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorIs(t, test.call(), provider_union.ErrReadOnly,
				"a mutating method on a union with no overlay must return ErrReadOnly")
		})
	}
}

func TestMutatingMethodsDelegateToOverlay(t *testing.T) {
	seed := func(t *testing.T, overlay *provider_mock.MockStorageProvider) {
		t.Helper()
		require.NoError(t, overlay.Put(t.Context(), &storage_dto.PutParams{
			Key: "src.txt", Reader: bytes.NewReader([]byte("data")), Size: int64(len("data")),
		}), "seeding the overlay source object")
	}

	tests := []struct {
		run  func(t *testing.T, store storage_domain.StorageProviderPort, overlay *provider_mock.MockStorageProvider)
		name string
	}{
		{name: "Copy", run: func(t *testing.T, store storage_domain.StorageProviderPort, overlay *provider_mock.MockStorageProvider) {
			seed(t, overlay)
			require.NoError(t, store.Copy(t.Context(), "", "src.txt", "copy.txt"))
			_, ok := overlay.GetObjectData("", "copy.txt")
			assert.True(t, ok, "the copy must land in the overlay")
		}},
		{name: "CopyToAnotherRepository", run: func(t *testing.T, store storage_domain.StorageProviderPort, overlay *provider_mock.MockStorageProvider) {
			seed(t, overlay)
			require.NoError(t, store.CopyToAnotherRepository(t.Context(), "", "src.txt", "other", "dst.txt"))
			_, ok := overlay.GetObjectData("other", "dst.txt")
			assert.True(t, ok, "the cross-repository copy must land in the overlay")
		}},
		{name: "Rename", run: func(t *testing.T, store storage_domain.StorageProviderPort, overlay *provider_mock.MockStorageProvider) {
			seed(t, overlay)
			require.NoError(t, store.Rename(t.Context(), "", "src.txt", "renamed.txt"))
			_, renamed := overlay.GetObjectData("", "renamed.txt")
			assert.True(t, renamed, "the renamed object must exist in the overlay")
			_, original := overlay.GetObjectData("", "src.txt")
			assert.False(t, original, "the original object must be gone after a rename")
		}},
		{name: "PresignURL", run: func(t *testing.T, store storage_domain.StorageProviderPort, _ *provider_mock.MockStorageProvider) {
			url, err := store.PresignURL(t.Context(), storage_dto.PresignParams{Key: "src.txt"})
			require.NoError(t, err)
			assert.NotEmpty(t, url, "the overlay must produce a presigned upload URL")
		}},
		{name: "PresignDownloadURL", run: func(t *testing.T, store storage_domain.StorageProviderPort, _ *provider_mock.MockStorageProvider) {
			url, err := store.PresignDownloadURL(t.Context(), storage_dto.PresignDownloadParams{Key: "src.txt"})
			require.NoError(t, err)
			assert.NotEmpty(t, url, "the overlay must produce a presigned download URL")
		}},
		{name: "PutMany", run: func(t *testing.T, store storage_domain.StorageProviderPort, _ *provider_mock.MockStorageProvider) {
			result, err := store.PutMany(t.Context(), &storage_dto.PutManyParams{
				Objects: []storage_dto.PutObjectSpec{{Key: "a.txt", Reader: bytes.NewReader([]byte("a")), Size: 1, ContentType: "text/plain"}},
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, 1, result.TotalSuccessful, "the overlay batch put must record the object")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, overlay := newMockOverlayUnion(t)
			test.run(t, store, overlay)
		})
	}
}

func TestCloseDelegatesToOverlay(t *testing.T) {
	overlay := &closeSpyProvider{MockStorageProvider: provider_mock.NewMockStorageProvider()}
	store := provider_union.New(newBase(t), overlay)
	ctx := t.Context()

	require.NoError(t, store.Close(ctx), "closing a union with an overlay must succeed")
	assert.Equal(t, 1, overlay.closeCount, "Close must delegate to the overlay")

	require.NoError(t, store.Close(ctx), "closing again must remain safe")
	assert.Equal(t, 2, overlay.closeCount, "each Close must reach the overlay")
}

func TestCloseReadOnlyUnionIsNoOp(t *testing.T) {
	store := newReadOnlyUnion(t)
	ctx := t.Context()

	require.NoError(t, store.Close(ctx), "closing a union with no overlay is a no-op")
	require.NoError(t, store.Close(ctx), "closing again with no overlay stays a no-op")
}

func TestSupportsCapabilitiesReflectOverlay(t *testing.T) {
	withOverlay, _ := newMockOverlayUnion(t)
	assert.True(t, withOverlay.SupportsMultipart(), "the overlay supports multipart")
	assert.True(t, withOverlay.SupportsBatchOperations(), "the overlay supports batch operations")
	assert.True(t, withOverlay.SupportsPresignedURLs(), "the overlay supports presigned URLs")
	assert.False(t, withOverlay.SupportsRetry(), "the mock overlay does not support retry")
	assert.False(t, withOverlay.SupportsCircuitBreaking(), "the mock overlay does not support circuit breaking")
	assert.False(t, withOverlay.SupportsRateLimiting(), "the mock overlay does not support rate limiting")

	readOnly := newReadOnlyUnion(t)
	assert.False(t, readOnly.SupportsMultipart(), "with no overlay multipart is unsupported")
	assert.False(t, readOnly.SupportsBatchOperations(), "with no overlay batch operations are unsupported")
	assert.False(t, readOnly.SupportsPresignedURLs(), "with no overlay presigned URLs are unsupported")
	assert.False(t, readOnly.SupportsRetry(), "with no overlay retry is unsupported")
	assert.False(t, readOnly.SupportsCircuitBreaking(), "with no overlay circuit breaking is unsupported")
	assert.False(t, readOnly.SupportsRateLimiting(), "with no overlay rate limiting is unsupported")
}
