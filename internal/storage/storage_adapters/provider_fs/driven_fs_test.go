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

package provider_fs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_dto"
)

type erroringFS struct {
	err error
}

func (e erroringFS) Open(string) (fs.File, error) {
	return nil, e.err
}

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"hello.txt": &fstest.MapFile{
			Data: []byte("hello world"),
		},
		"repo/data.bin": &fstest.MapFile{
			Data: []byte("binary content here"),
		},
		"assets/style.css": &fstest.MapFile{
			Data: []byte("body { margin: 0; }"),
		},
		"assets/app.js": &fstest.MapFile{
			Data: []byte("console.log('piko');"),
		},
		"assets/.metadata.json": &fstest.MapFile{
			Data: []byte(`{"content-type":"text/css"}`),
		},
		"assets/image.tmp": &fstest.MapFile{
			Data: []byte("temporary"),
		},
	}
}

func newTestProvider(t *testing.T) *FSProvider {
	t.Helper()
	provider, err := NewFSProvider(testFS())
	require.NoError(t, err, "NewFSProvider")
	return provider
}

func TestNewFSProvider_NilReturnsError(t *testing.T) {
	_, err := NewFSProvider(nil)
	require.Error(t, err, "expected error for nil fsys")
}

func TestFSProvider_Get(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{Key: "hello.txt"})
	require.NoError(t, err, "unexpected error")
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err, "reading content")
	assert.Equal(t, "hello world", string(data))
}

func TestFSProvider_Get_WithRepository(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "data.bin",
	})
	require.NoError(t, err, "unexpected error")
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err, "reading content")
	assert.Equal(t, "binary content here", string(data))
}

func TestFSProvider_Get_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: "missing.txt"})
	require.Error(t, err, "expected error for missing file")
}

func TestFSProvider_Get_EmptyKey(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: ""})
	require.ErrorIs(t, err, ErrEmptyKey, "expected ErrEmptyKey")
}

func TestFSProvider_Get_PathTraversal(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: "../etc/passwd"})
	require.ErrorIs(t, err, ErrInvalidPath, "expected ErrInvalidPath for path traversal")
}

func TestFSProvider_Get_ByteRange(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Key: "hello.txt",
		ByteRange: &storage_dto.ByteRange{
			Start: 6,
			End:   10,
		},
	})
	require.NoError(t, err, "unexpected error")
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err, "reading content")
	assert.Equal(t, "world", string(data))
}

func TestFSProvider_Get_ByteRangeToEnd(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Key: "hello.txt",
		ByteRange: &storage_dto.ByteRange{
			Start: 6,
			End:   -1,
		},
	})
	require.NoError(t, err, "unexpected error")
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err, "reading content")
	assert.Equal(t, "world", string(data))
}

func TestFSProvider_Get_ByteRangeStartBeyondFileSize(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{
		Key: "hello.txt",
		ByteRange: &storage_dto.ByteRange{
			Start: 100,
			End:   -1,
		},
	})
	require.Error(t, err, "expected error for start beyond file size")
}

func TestFSProvider_Get_ByteRangeEndBeyondFileSize(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Key: "hello.txt",
		ByteRange: &storage_dto.ByteRange{
			Start: 6,
			End:   1000,
		},
	})
	require.NoError(t, err, "unexpected error")
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	require.NoError(t, err, "reading content")
	assert.Equal(t, "world", string(data))
}

func TestFSProvider_Get_ByteRangeEndBeforeStart(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{
		Key: "hello.txt",
		ByteRange: &storage_dto.ByteRange{
			Start: 5,
			End:   3,
		},
	})
	require.Error(t, err, "expected error for end before start")
}

func TestFSProvider_Stat(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	info, err := provider.Stat(ctx, storage_dto.GetParams{Key: "hello.txt"})
	require.NoError(t, err, "unexpected error")

	assert.Equal(t, int64(11), info.Size, "size")
	assert.Equal(t, "text/plain; charset=utf-8", info.ContentType, "content type")
}

func TestFSProvider_Stat_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Stat(ctx, storage_dto.GetParams{Key: "missing.txt"})
	require.Error(t, err, "expected error for missing file")
}

func TestFSProvider_Stat_CSS(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: "assets",
		Key:        "style.css",
	})
	require.NoError(t, err, "unexpected error")

	assert.Equal(t, "text/css; charset=utf-8", info.ContentType, "content type")
}

func TestFSProvider_Exists(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	exists, err := provider.Exists(ctx, storage_dto.GetParams{Key: "hello.txt"})
	require.NoError(t, err, "unexpected error")
	assert.True(t, exists, "expected file to exist")

	exists, err = provider.Exists(ctx, storage_dto.GetParams{Key: "nope.txt"})
	require.NoError(t, err, "unexpected error")
	assert.False(t, exists, "expected file not to exist")
}

func TestFSProvider_Exists_EmptyKey(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Exists(ctx, storage_dto.GetParams{Key: ""})
	require.ErrorIs(t, err, ErrEmptyKey, "expected ErrEmptyKey")
}

func TestFSProvider_GetHash(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	hash, err := provider.GetHash(ctx, storage_dto.GetParams{Key: "hello.txt"})
	require.NoError(t, err, "unexpected error")

	expected := sha256.Sum256([]byte("hello world"))
	expectedHex := hex.EncodeToString(expected[:])

	assert.Equal(t, expectedHex, hash, "hash")
}

func TestFSProvider_GetHash_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.GetHash(ctx, storage_dto.GetParams{Key: "missing.txt"})
	require.Error(t, err, "expected error for missing file")
}

func TestFSProvider_ListKeys(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	keys, err := provider.ListKeys(ctx, "assets")
	require.NoError(t, err, "unexpected error")

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	assert.True(t, keySet["style.css"], "expected style.css in keys")
	assert.True(t, keySet["app.js"], "expected app.js in keys")
	assert.False(t, keySet[".metadata.json"], "metadata sidecar should be filtered out")
	assert.False(t, keySet["image.tmp"], ".tmp files should be filtered out")
}

func TestFSProvider_ListKeys_RootRepository(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	keys, err := provider.ListKeys(ctx, "")
	require.NoError(t, err, "unexpected error")

	assert.NotEmpty(t, keys, "expected at least one key")

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	assert.True(t, keySet["hello.txt"], "expected hello.txt in keys")
}

func TestFSProvider_ListKeys_AbsentRepositoryReturnsEmpty(t *testing.T) {
	provider := newTestProvider(t)

	keys, err := provider.ListKeys(context.Background(), "no-such-repository")
	require.NoError(t, err, "an absent repository root must not be an error")
	assert.Empty(t, keys, "an absent repository root must yield an empty key set")
}

func TestFSProvider_ListKeys_PropagatesWalkError(t *testing.T) {
	sentinel := errors.New("permission denied")
	provider, err := NewFSProvider(erroringFS{err: sentinel})
	require.NoError(t, err)

	_, err = provider.ListKeys(context.Background(), "repo")
	require.Error(t, err, "a non-NotExist walk error must propagate, not be swallowed")
	require.ErrorIs(t, err, sentinel, "the underlying walk error must be wrapped, not discarded")
}

func TestFSProvider_ListKeys_CancelledContext(t *testing.T) {
	provider := newTestProvider(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.ListKeys(ctx, "")
	require.Error(t, err, "expected error for cancelled context")
}

func TestFSProvider_WriteOps_ReturnReadOnly(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	assert.ErrorIs(t, provider.Put(ctx, &storage_dto.PutParams{}), ErrReadOnly, "Put")
	assert.ErrorIs(t, provider.Copy(ctx, "", "", ""), ErrReadOnly, "Copy")
	assert.ErrorIs(t, provider.CopyToAnotherRepository(ctx, "", "", "", ""), ErrReadOnly,
		"CopyToAnotherRepository")
	assert.ErrorIs(t, provider.Remove(ctx, storage_dto.GetParams{}), ErrReadOnly, "Remove")
	assert.ErrorIs(t, provider.Rename(ctx, "", "", ""), ErrReadOnly, "Rename")

	_, err := provider.PresignURL(ctx, storage_dto.PresignParams{})
	assert.ErrorIs(t, err, ErrReadOnly, "PresignURL")
	_, err = provider.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{})
	assert.ErrorIs(t, err, ErrReadOnly, "PresignDownloadURL")
	_, err = provider.PutMany(ctx, &storage_dto.PutManyParams{})
	assert.ErrorIs(t, err, ErrReadOnly, "PutMany")
	_, err = provider.RemoveMany(ctx, storage_dto.RemoveManyParams{})
	assert.ErrorIs(t, err, ErrReadOnly, "RemoveMany")
}

func TestFSProvider_Capabilities(t *testing.T) {
	provider := newTestProvider(t)

	assert.False(t, provider.SupportsMultipart(), "should not support multipart")
	assert.False(t, provider.SupportsBatchOperations(), "should not support batch operations")
	assert.False(t, provider.SupportsRetry(), "should not support retry")
	assert.False(t, provider.SupportsCircuitBreaking(), "should not support circuit breaking")
	assert.False(t, provider.SupportsRateLimiting(), "should not support rate limiting")
	assert.False(t, provider.SupportsPresignedURLs(), "should not support presigned URLs")
}

func TestFSProvider_Close(t *testing.T) {
	provider := newTestProvider(t)
	assert.NoError(t, provider.Close(context.Background()), "Close: unexpected error")
}

func TestFSProvider_ProviderMetadata(t *testing.T) {
	provider := newTestProvider(t)

	assert.Equal(t, "embedded-fs", provider.GetProviderType(), "type")

	metadata := provider.GetProviderMetadata()
	assert.Equal(t, "embedded-fs", metadata["type"], "metadata type")
	assert.Equal(t, true, metadata["read_only"], "metadata read_only")
}

func TestFsPath_ValidPaths(t *testing.T) {
	tests := []struct {
		repository string
		key        string
		want       string
	}{
		{repository: "", key: "file.txt", want: "file.txt"},
		{repository: "repo", key: "file.txt", want: "repo/file.txt"},
		{repository: "repo", key: "sub/file.txt", want: "repo/sub/file.txt"},
		{repository: "", key: "archive..2026.tar", want: "archive..2026.tar"},
		{repository: "repo", key: "archive..2026.tar", want: "repo/archive..2026.tar"},
		{repository: "", key: "v1..2/file.txt", want: "v1..2/file.txt"},
	}

	for _, test := range tests {
		got, err := fsPath(test.repository, test.key)
		if !assert.NoErrorf(t, err, "fsPath(%q, %q): unexpected error", test.repository, test.key) {
			continue
		}
		assert.Equalf(t, test.want, got, "fsPath(%q, %q)", test.repository, test.key)
	}
}

func TestFsPath_InvalidPaths(t *testing.T) {
	tests := []struct {
		repository string
		key        string
		wantErr    error
	}{
		{repository: "", key: "", wantErr: ErrEmptyKey},
		{repository: "repo", key: "", wantErr: ErrEmptyKey},
		{repository: "", key: "../escape", wantErr: ErrInvalidPath},
		{repository: "repo", key: "../escape", wantErr: ErrInvalidPath},
		{repository: "", key: "/absolute", wantErr: ErrInvalidPath},
		{repository: "", key: "a/../b", wantErr: ErrInvalidPath},
		{repository: "repo", key: "sub/../../x", wantErr: ErrInvalidPath},
		{repository: "", key: "with\x00null", wantErr: ErrInvalidPath},
		{repository: "", key: "with\nnewline.txt", wantErr: ErrInvalidPath},
		{repository: "repo", key: "with\rcarriage", wantErr: ErrInvalidPath},
	}

	for _, test := range tests {
		_, err := fsPath(test.repository, test.key)
		assert.ErrorIsf(t, err, test.wantErr, "fsPath(%q, %q)", test.repository, test.key)
	}
}
