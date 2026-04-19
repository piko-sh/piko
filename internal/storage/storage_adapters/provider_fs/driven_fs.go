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
	"fmt"
	"io"
	"io/fs"
	"mime"
	"path"
	"strings"

	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// maxListKeysCount is the maximum number of keys returned by ListKeys
	// to prevent unbounded memory allocation from large embedded filesystems.
	maxListKeysCount = 1_000_000

	// maxHashFileSize is the maximum file size in bytes for GetHash
	// to prevent reading excessively large files into memory.
	maxHashFileSize = 512 * 1024 * 1024
)

// ErrReadOnly is returned by all write operations on the embedded fs provider.
var ErrReadOnly = errors.New("embedded fs provider is read-only")

// ErrEmptyKey is returned when an object key is empty.
var ErrEmptyKey = errors.New("object key cannot be empty")

// ErrInvalidPath is returned when a key resolves to an invalid fs.FS path.
var ErrInvalidPath = errors.New("invalid object path")

var (
	_ storage_domain.StorageProviderPort = (*FSProvider)(nil)

	_ provider_domain.ProviderMetadata   = (*FSProvider)(nil)
)

// rangeReadCloser wraps a section reader and the underlying file so that
// closing the reader also closes the file.
type rangeReadCloser struct {
	io.Reader

	closer io.Closer
}

// FSProvider implements StorageProviderPort using a read-only fs.FS.
// It is designed for serving pre-built assets from embedded filesystems.
type FSProvider struct {
	fsys fs.FS
}

// NewFSProvider creates a new read-only storage provider backed by the given
// filesystem.
//
// Takes fsys (fs.FS) which provides the underlying file access.
//
// Returns *FSProvider which implements StorageProviderPort for reading.
// Returns error when fsys is nil.
func NewFSProvider(fsys fs.FS) (*FSProvider, error) {
	if fsys == nil {
		return nil, errors.New("fsys must not be nil")
	}
	return &FSProvider{fsys: fsys}, nil
}

// Get retrieves an object as a readable stream.
//
// Takes params (storage_dto.GetParams) which specifies the object to retrieve
// and any byte range options.
//
// Returns io.ReadCloser which provides the object data as a stream.
// Returns error when the object cannot be retrieved.
func (p *FSProvider) Get(_ context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	filePath, err := fsPath(params.Repository, params.Key)
	if err != nil {
		return nil, err
	}

	file, err := p.fsys.Open(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}

	if params.ByteRange == nil {
		return file, nil
	}

	return handleRangeRequest(file, params.Key, params.ByteRange)
}

// Stat retrieves metadata for an object.
//
// Takes params (storage_dto.GetParams) which specifies which object to query.
//
// Returns *storage_domain.ObjectInfo which contains the object metadata.
// Returns error when the object cannot be found.
func (p *FSProvider) Stat(_ context.Context, params storage_dto.GetParams) (*storage_domain.ObjectInfo, error) {
	filePath, err := fsPath(params.Repository, params.Key)
	if err != nil {
		return nil, err
	}

	fileInfo, err := fs.Stat(p.fsys, filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to stat file '%s': %w", filePath, err)
	}

	return &storage_domain.ObjectInfo{
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		ContentType:  mimeTypeFromExtension(filePath),
	}, nil
}

// Exists checks if an object exists at the given key.
//
// Takes params (storage_dto.GetParams) which specifies the key to check.
//
// Returns bool which is true if the object exists, false otherwise.
// Returns error when the existence check fails unexpectedly.
func (p *FSProvider) Exists(_ context.Context, params storage_dto.GetParams) (bool, error) {
	filePath, err := fsPath(params.Repository, params.Key)
	if err != nil {
		return false, err
	}

	_, err = fs.Stat(p.fsys, filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat '%s': %w", filePath, err)
	}

	return true, nil
}

// GetHash returns the SHA256 hash of an object. The file size is
// limited to maxHashFileSize to prevent excessive memory consumption.
//
// Takes params (storage_dto.GetParams) which specifies the object to hash.
//
// Returns string which is the hex-encoded SHA256 hash.
// Returns error when the hash cannot be computed.
func (p *FSProvider) GetHash(_ context.Context, params storage_dto.GetParams) (string, error) {
	filePath, err := fsPath(params.Repository, params.Key)
	if err != nil {
		return "", err
	}

	file, err := p.fsys.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file for hash %q: %w", params.Key, err)
	}
	defer func() { _ = file.Close() }()

	limitedReader := io.LimitReader(file, maxHashFileSize+1)
	hasher := sha256.New()
	bytesWritten, err := io.Copy(hasher, limitedReader)
	if err != nil {
		return "", fmt.Errorf("hashing file %q: %w", params.Key, err)
	}
	if bytesWritten > maxHashFileSize {
		return "", fmt.Errorf("file %q exceeds maximum hash size of %d bytes", params.Key, maxHashFileSize)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ListKeys returns all storage keys within the given repository directory,
// filtering out metadata sidecars and temporary files. Results are capped
// at maxListKeysCount to prevent unbounded memory allocation.
//
// Takes repository (string) which identifies the repository to scan.
//
// Returns []string which contains all discovered storage keys.
// Returns error when the directory walk fails.
func (p *FSProvider) ListKeys(ctx context.Context, repository string) ([]string, error) {
	root := repository
	if root == "" {
		root = "."
	}

	var keys []string
	err := fs.WalkDir(p.fsys, root, func(filePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(filePath, ".metadata.json") ||
			strings.HasSuffix(filePath, ".md5") ||
			strings.HasSuffix(filePath, ".tmp") {
			return nil
		}
		if len(keys) >= maxListKeysCount {
			return fs.SkipAll
		}
		key := strings.TrimPrefix(filePath, root+"/")
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking repository %q: %w", repository, err)
	}
	return keys, nil
}

// Put is not supported on a read-only filesystem.
func (*FSProvider) Put(_ context.Context, _ *storage_dto.PutParams) error {
	return ErrReadOnly
}

// Copy is not supported on a read-only filesystem.
func (*FSProvider) Copy(_ context.Context, _ string, _, _ string) error {
	return ErrReadOnly
}

// CopyToAnotherRepository is not supported on a read-only filesystem.
func (*FSProvider) CopyToAnotherRepository(_ context.Context, _ string, _ string, _ string, _ string) error {
	return ErrReadOnly
}

// Remove is not supported on a read-only filesystem.
func (*FSProvider) Remove(_ context.Context, _ storage_dto.GetParams) error {
	return ErrReadOnly
}

// Rename is not supported on a read-only filesystem.
func (*FSProvider) Rename(_ context.Context, _ string, _, _ string) error {
	return ErrReadOnly
}

// PresignURL is not supported on a read-only filesystem.
func (*FSProvider) PresignURL(_ context.Context, _ storage_dto.PresignParams) (string, error) {
	return "", ErrReadOnly
}

// PresignDownloadURL is not supported on a read-only filesystem.
func (*FSProvider) PresignDownloadURL(_ context.Context, _ storage_dto.PresignDownloadParams) (string, error) {
	return "", ErrReadOnly
}

// PutMany is not supported on a read-only filesystem.
func (*FSProvider) PutMany(_ context.Context, _ *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	return nil, ErrReadOnly
}

// RemoveMany is not supported on a read-only filesystem.
func (*FSProvider) RemoveMany(_ context.Context, _ storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	return nil, ErrReadOnly
}

// Close is a no-op for the embedded fs provider.
func (*FSProvider) Close(_ context.Context) error {
	return nil
}

// SupportsMultipart returns false.
func (*FSProvider) SupportsMultipart() bool { return false }

// SupportsBatchOperations returns false.
func (*FSProvider) SupportsBatchOperations() bool { return false }

// SupportsRetry returns false.
func (*FSProvider) SupportsRetry() bool { return false }

// SupportsCircuitBreaking returns false.
func (*FSProvider) SupportsCircuitBreaking() bool { return false }

// SupportsRateLimiting returns false.
func (*FSProvider) SupportsRateLimiting() bool { return false }

// SupportsPresignedURLs returns false.
func (*FSProvider) SupportsPresignedURLs() bool { return false }

// GetProviderType returns the provider implementation type.
func (*FSProvider) GetProviderType() string {
	return "embedded-fs"
}

// GetProviderMetadata returns metadata about the embedded fs provider.
func (*FSProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"type":      "embedded-fs",
		"read_only": true,
	}
}

// Close releases the underlying file handle.
func (r *rangeReadCloser) Close() error {
	return r.closer.Close()
}

// fsPath builds a forward-slash path from repository and key, validating
// that the result is a valid fs.FS path (no ".." traversal). The fs.FS
// interface uses forward slashes regardless of operating system.
func fsPath(repository, key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("%w: key contains path traversal: %q", ErrInvalidPath, key)
	}
	var result string
	if repository == "" {
		result = key
	} else {
		result = path.Join(repository, key)
	}
	if !fs.ValidPath(result) {
		return "", fmt.Errorf("%w: %q", ErrInvalidPath, result)
	}
	return result, nil
}

// handleRangeRequest returns a reader for the requested byte range. It checks
// whether the opened file supports io.ReaderAt (embed.FS files do) and uses
// io.NewSectionReader for efficient partial reads. The byte range is clamped
// to the actual file size to prevent out-of-bounds reads.
func handleRangeRequest(file fs.File, key string, byteRange *storage_dto.ByteRange) (io.ReadCloser, error) {
	readerAt, ok := file.(io.ReaderAt)
	if !ok {
		_ = file.Close()
		return nil, fmt.Errorf("file %q does not support byte range reads", key)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stat for range request on %q: %w", key, err)
	}

	fileSize := fileInfo.Size()
	start := byteRange.Start
	end := byteRange.End

	if start < 0 {
		start = 0
	}
	if start >= fileSize {
		_ = file.Close()
		return nil, fmt.Errorf("byte range start %d is beyond file size %d for %q", byteRange.Start, fileSize, key)
	}
	if end == -1 || end >= fileSize {
		end = fileSize - 1
	}
	if end < start {
		_ = file.Close()
		return nil, fmt.Errorf("byte range end %d is before start %d for %q", end, start, key)
	}

	length := end - start + 1
	section := io.NewSectionReader(readerAt, start, length)
	return &rangeReadCloser{Reader: section, closer: file}, nil
}

// mimeTypeFromExtension returns the MIME type for the given file path based on
// its extension, defaulting to "application/octet-stream" if unknown.
func mimeTypeFromExtension(filePath string) string {
	ext := path.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
