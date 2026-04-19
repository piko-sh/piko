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
	"testing"
	"testing/fstest"

	"piko.sh/piko/internal/storage/storage_dto"
)

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
	if err != nil {
		t.Fatalf("NewFSProvider: %v", err)
	}
	return provider
}

func TestNewFSProvider_NilReturnsError(t *testing.T) {
	_, err := NewFSProvider(nil)
	if err == nil {
		t.Fatal("expected error for nil fsys")
	}
}

func TestFSProvider_Get(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{Key: "hello.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}
}

func TestFSProvider_Get_WithRepository(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	reader, err := provider.Get(ctx, storage_dto.GetParams{
		Repository: "repo",
		Key:        "data.bin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	if string(data) != "binary content here" {
		t.Errorf("got %q, want %q", string(data), "binary content here")
	}
}

func TestFSProvider_Get_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: "missing.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFSProvider_Get_EmptyKey(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: ""})
	if !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey, got %v", err)
	}
}

func TestFSProvider_Get_PathTraversal(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Get(ctx, storage_dto.GetParams{Key: "../etc/passwd"})
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath for path traversal, got %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", string(data), "world")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", string(data), "world")
	}
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
	if err == nil {
		t.Fatal("expected error for start beyond file size")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", string(data), "world")
	}
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
	if err == nil {
		t.Fatal("expected error for end before start")
	}
}

func TestFSProvider_Stat(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	info, err := provider.Stat(ctx, storage_dto.GetParams{Key: "hello.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Size != 11 {
		t.Errorf("size = %d, want 11", info.Size)
	}
	if info.ContentType != "text/plain; charset=utf-8" {
		t.Errorf("content type = %q, want %q", info.ContentType, "text/plain; charset=utf-8")
	}
}

func TestFSProvider_Stat_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Stat(ctx, storage_dto.GetParams{Key: "missing.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFSProvider_Stat_CSS(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	info, err := provider.Stat(ctx, storage_dto.GetParams{
		Repository: "assets",
		Key:        "style.css",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ContentType != "text/css; charset=utf-8" {
		t.Errorf("content type = %q, want %q", info.ContentType, "text/css; charset=utf-8")
	}
}

func TestFSProvider_Exists(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	exists, err := provider.Exists(ctx, storage_dto.GetParams{Key: "hello.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	exists, err = provider.Exists(ctx, storage_dto.GetParams{Key: "nope.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected file not to exist")
	}
}

func TestFSProvider_Exists_EmptyKey(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.Exists(ctx, storage_dto.GetParams{Key: ""})
	if !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey, got %v", err)
	}
}

func TestFSProvider_GetHash(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	hash, err := provider.GetHash(ctx, storage_dto.GetParams{Key: "hello.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := sha256.Sum256([]byte("hello world"))
	expectedHex := hex.EncodeToString(expected[:])

	if hash != expectedHex {
		t.Errorf("hash = %q, want %q", hash, expectedHex)
	}
}

func TestFSProvider_GetHash_NotFound(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	_, err := provider.GetHash(ctx, storage_dto.GetParams{Key: "missing.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFSProvider_ListKeys(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	keys, err := provider.ListKeys(ctx, "assets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	if !keySet["style.css"] {
		t.Error("expected style.css in keys")
	}
	if !keySet["app.js"] {
		t.Error("expected app.js in keys")
	}
	if keySet[".metadata.json"] {
		t.Error("metadata sidecar should be filtered out")
	}
	if keySet["image.tmp"] {
		t.Error(".tmp files should be filtered out")
	}
}

func TestFSProvider_ListKeys_RootRepository(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	keys, err := provider.ListKeys(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) == 0 {
		t.Error("expected at least one key")
	}

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	if !keySet["hello.txt"] {
		t.Error("expected hello.txt in keys")
	}
}

func TestFSProvider_ListKeys_CancelledContext(t *testing.T) {
	provider := newTestProvider(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.ListKeys(ctx, "")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestFSProvider_WriteOps_ReturnReadOnly(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	if err := provider.Put(ctx, &storage_dto.PutParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Put: got %v, want ErrReadOnly", err)
	}
	if err := provider.Copy(ctx, "", "", ""); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Copy: got %v, want ErrReadOnly", err)
	}
	if err := provider.CopyToAnotherRepository(ctx, "", "", "", ""); !errors.Is(err, ErrReadOnly) {
		t.Errorf("CopyToAnotherRepository: got %v, want ErrReadOnly", err)
	}
	if err := provider.Remove(ctx, storage_dto.GetParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Remove: got %v, want ErrReadOnly", err)
	}
	if err := provider.Rename(ctx, "", "", ""); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Rename: got %v, want ErrReadOnly", err)
	}
	if _, err := provider.PresignURL(ctx, storage_dto.PresignParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("PresignURL: got %v, want ErrReadOnly", err)
	}
	if _, err := provider.PresignDownloadURL(ctx, storage_dto.PresignDownloadParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("PresignDownloadURL: got %v, want ErrReadOnly", err)
	}
	if _, err := provider.PutMany(ctx, &storage_dto.PutManyParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("PutMany: got %v, want ErrReadOnly", err)
	}
	if _, err := provider.RemoveMany(ctx, storage_dto.RemoveManyParams{}); !errors.Is(err, ErrReadOnly) {
		t.Errorf("RemoveMany: got %v, want ErrReadOnly", err)
	}
}

func TestFSProvider_Capabilities(t *testing.T) {
	provider := newTestProvider(t)

	if provider.SupportsMultipart() {
		t.Error("should not support multipart")
	}
	if provider.SupportsBatchOperations() {
		t.Error("should not support batch operations")
	}
	if provider.SupportsRetry() {
		t.Error("should not support retry")
	}
	if provider.SupportsCircuitBreaking() {
		t.Error("should not support circuit breaking")
	}
	if provider.SupportsRateLimiting() {
		t.Error("should not support rate limiting")
	}
	if provider.SupportsPresignedURLs() {
		t.Error("should not support presigned URLs")
	}
}

func TestFSProvider_Close(t *testing.T) {
	provider := newTestProvider(t)
	if err := provider.Close(context.Background()); err != nil {
		t.Errorf("Close: unexpected error: %v", err)
	}
}

func TestFSProvider_ProviderMetadata(t *testing.T) {
	provider := newTestProvider(t)

	if provider.GetProviderType() != "embedded-fs" {
		t.Errorf("type = %q, want %q", provider.GetProviderType(), "embedded-fs")
	}

	metadata := provider.GetProviderMetadata()
	if metadata["type"] != "embedded-fs" {
		t.Errorf("metadata type = %v, want %q", metadata["type"], "embedded-fs")
	}
	if metadata["read_only"] != true {
		t.Errorf("metadata read_only = %v, want true", metadata["read_only"])
	}
}

func TestFsPath_ValidPaths(t *testing.T) {
	tests := []struct {
		repository string
		key        string
		want       string
	}{
		{"", "file.txt", "file.txt"},
		{"repo", "file.txt", "repo/file.txt"},
		{"repo", "sub/file.txt", "repo/sub/file.txt"},
	}

	for _, test := range tests {
		got, err := fsPath(test.repository, test.key)
		if err != nil {
			t.Errorf("fsPath(%q, %q): unexpected error: %v", test.repository, test.key, err)
			continue
		}
		if got != test.want {
			t.Errorf("fsPath(%q, %q) = %q, want %q", test.repository, test.key, got, test.want)
		}
	}
}

func TestFsPath_InvalidPaths(t *testing.T) {
	tests := []struct {
		repository string
		key        string
		wantErr    error
	}{
		{"", "", ErrEmptyKey},
		{"repo", "", ErrEmptyKey},
		{"", "../escape", ErrInvalidPath},
		{"repo", "../escape", ErrInvalidPath},
		{"", "/absolute", ErrInvalidPath},
	}

	for _, test := range tests {
		_, err := fsPath(test.repository, test.key)
		if !errors.Is(err, test.wantErr) {
			t.Errorf("fsPath(%q, %q): got %v, want %v", test.repository, test.key, err, test.wantErr)
		}
	}
}
