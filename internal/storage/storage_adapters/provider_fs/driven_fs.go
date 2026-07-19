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
	"cmp"
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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// maxListKeysCount is the maximum number of keys returned by ListKeys to prevent
	// unbounded memory allocation from large embedded filesystems.
	maxListKeysCount = 1_000_000

	// maxHashFileSize is the maximum file size in bytes for GetHash to prevent reading
	// excessively large files into memory.
	maxHashFileSize = 512 * 1024 * 1024
)

var (
	// ErrReadOnly is returned by all write operations on the embedded fs provider.
	ErrReadOnly = errors.New("embedded fs provider is read-only")

	// ErrEmptyKey is returned when an object key is empty.
	ErrEmptyKey = errors.New("object key cannot be empty")

	// ErrInvalidPath is returned when a key resolves to an invalid fs.FS path.
	ErrInvalidPath = errors.New("invalid object path")

	// log is the package logger for the embedded fs provider.
	log = logger_domain.GetLogger("piko/internal/storage/storage_adapters/provider_fs")

	_ storage_domain.StorageProviderPort = (*FSProvider)(nil)

	_ provider_domain.ProviderMetadata = (*FSProvider)(nil)
)

// rangeReadCloser wraps a section reader and the underlying file so that closing the
// reader also closes the file.
type rangeReadCloser struct {
	io.Reader

	// closer is the underlying file closed when the reader is closed.
	closer io.Closer
}

// FSProvider implements StorageProviderPort using a read-only fs.FS. It is designed for
// serving pre-built assets from embedded filesystems.
type FSProvider struct {
	// fsys is the read-only filesystem backing all read operations.
	fsys fs.FS
}

// NewFSProvider creates a new read-only storage provider backed by the given filesystem.
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
// Takes params (storage_dto.GetParams) which specifies the object to retrieve and any
// byte range options.
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

// GetHash returns the SHA256 hash of an object. The file size is limited to
// maxHashFileSize to prevent excessive memory consumption.
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

// ListKeys returns the storage keys within the given repository directory.
//
// Metadata sidecars and temporary files are filtered out. An absent repository root
// yields an empty result rather than an error, mirroring a repository that has never been
// written; any other walk error propagates so garbage collection never mislabels base
// content. Results are capped at maxListKeysCount, and a truncation at the cap is logged
// rather than passing silently.
//
// Takes repository (string) which identifies the repository to scan.
//
// Returns []string which contains all discovered storage keys.
// Returns error when the directory walk fails for a reason other than an absent root.
func (p *FSProvider) ListKeys(ctx context.Context, repository string) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	root := cmp.Or(repository, ".")

	var keys []string
	truncated := false
	err := fs.WalkDir(p.fsys, root, func(filePath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return listKeysWalkError(root, filePath, walkError)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() || isIgnoredStorageKey(filePath) {
			return nil
		}
		if len(keys) >= maxListKeysCount {
			truncated = true
			return fs.SkipAll
		}
		keys = append(keys, strings.TrimPrefix(filePath, root+"/"))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking repository %q: %w", repository, err)
	}
	if truncated {
		l.Warn("ListKeys truncated at the maximum key count; some keys were omitted",
			logger_domain.String("repository", repository),
			logger_domain.Int("max_keys", maxListKeysCount))
	}
	return keys, nil
}

// listKeysWalkError decides how ListKeys handles a walk error. An absent root ends the
// walk with an empty result, mirroring a repository never written; every other error
// propagates so garbage collection never mislabels base content.
//
// Takes root (string) which is the walk root.
// Takes filePath (string) which is the path the walk error occurred at.
// Takes walkError (error) which is the error the walk reported.
//
// Returns error which is fs.SkipAll to end an absent-root walk, or the original error.
func listKeysWalkError(root, filePath string, walkError error) error {
	if filePath == root && errors.Is(walkError, fs.ErrNotExist) {
		return fs.SkipAll
	}
	return walkError
}

// isIgnoredStorageKey reports whether a walked path is a sidecar or temporary file that
// ListKeys must skip rather than return as a storage key.
//
// Takes filePath (string) which is the walked path.
//
// Returns bool which is true for a metadata sidecar, checksum, or temporary file.
func isIgnoredStorageKey(filePath string) bool {
	return strings.HasSuffix(filePath, ".metadata.json") ||
		strings.HasSuffix(filePath, ".md5") ||
		strings.HasSuffix(filePath, ".tmp")
}

// Put is not supported on a read-only filesystem.
//
// Returns ErrReadOnly always.
func (*FSProvider) Put(_ context.Context, _ *storage_dto.PutParams) error {
	return ErrReadOnly
}

// Copy is not supported on a read-only filesystem.
//
// Returns ErrReadOnly always.
func (*FSProvider) Copy(_ context.Context, _ string, _, _ string) error {
	return ErrReadOnly
}

// CopyToAnotherRepository is not supported on a read-only filesystem.
//
// Returns ErrReadOnly always.
func (*FSProvider) CopyToAnotherRepository(_ context.Context, _ string, _ string, _ string, _ string) error {
	return ErrReadOnly
}

// Remove is not supported on a read-only filesystem.
//
// Returns ErrReadOnly always.
func (*FSProvider) Remove(_ context.Context, _ storage_dto.GetParams) error {
	return ErrReadOnly
}

// Rename is not supported on a read-only filesystem.
//
// Returns ErrReadOnly always.
func (*FSProvider) Rename(_ context.Context, _ string, _, _ string) error {
	return ErrReadOnly
}

// PresignURL is not supported on a read-only filesystem.
//
// Returns string which is always empty.
// Returns error which is always ErrReadOnly.
func (*FSProvider) PresignURL(_ context.Context, _ storage_dto.PresignParams) (string, error) {
	return "", ErrReadOnly
}

// PresignDownloadURL is not supported on a read-only filesystem.
//
// Returns string which is always empty.
// Returns error which is always ErrReadOnly.
func (*FSProvider) PresignDownloadURL(_ context.Context, _ storage_dto.PresignDownloadParams) (string, error) {
	return "", ErrReadOnly
}

// PutMany is not supported on a read-only filesystem.
//
// Returns *storage_dto.BatchResult which is always nil.
// Returns error which is always ErrReadOnly.
func (*FSProvider) PutMany(_ context.Context, _ *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	return nil, ErrReadOnly
}

// RemoveMany is not supported on a read-only filesystem.
//
// Returns *storage_dto.BatchResult which is always nil.
// Returns error which is always ErrReadOnly.
func (*FSProvider) RemoveMany(_ context.Context, _ storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	return nil, ErrReadOnly
}

// Close is a no-op for the embedded fs provider.
//
// Returns nil always.
func (*FSProvider) Close(_ context.Context) error {
	return nil
}

// SupportsMultipart reports whether multipart uploads are supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsMultipart() bool { return false }

// SupportsBatchOperations reports whether batch operations are supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsBatchOperations() bool { return false }

// SupportsRetry reports whether automatic retries are supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsRetry() bool { return false }

// SupportsCircuitBreaking reports whether circuit breaking is supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsCircuitBreaking() bool { return false }

// SupportsRateLimiting reports whether rate limiting is supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsRateLimiting() bool { return false }

// SupportsPresignedURLs reports whether presigned URLs are supported.
//
// Returns bool which is always false.
func (*FSProvider) SupportsPresignedURLs() bool { return false }

// GetProviderType returns the provider implementation type.
//
// Returns string which identifies the embedded fs provider.
func (*FSProvider) GetProviderType() string {
	return "embedded-fs"
}

// GetProviderMetadata returns metadata about the embedded fs provider.
//
// Returns map[string]any which describes the provider type and read-only state.
func (*FSProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"type":      "embedded-fs",
		"read_only": true,
	}
}

// Close releases the underlying file handle.
//
// Returns error when closing the underlying file fails.
func (r *rangeReadCloser) Close() error {
	return r.closer.Close()
}

// fsPath builds a forward-slash path from repository and key, validating that the result
// is a valid fs.FS path (no ".." traversal). The fs.FS interface uses forward slashes
// regardless of operating system.
//
// Takes repository (string) which is the optional repository prefix.
// Takes key (string) which is the object key relative to the repository.
//
// Returns the validated forward-slash path, or an error when the key or result is
// invalid.
func fsPath(repository, key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}

	if strings.ContainsAny(key, "\x00\r\n") {
		return "", fmt.Errorf("%w: key contains control characters: %q", ErrInvalidPath, key)
	}

	if !fs.ValidPath(key) {
		return "", fmt.Errorf("%w: key is not a valid path: %q", ErrInvalidPath, key)
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

// handleRangeRequest returns a reader for the requested byte range.
//
// It checks whether the opened file supports io.ReaderAt (embed.FS files do) and uses
// io.NewSectionReader for efficient partial reads. The byte range is clamped to the
// actual file size to prevent out-of-bounds reads.
//
// Takes file (fs.File) which is the opened file to read from.
// Takes key (string) which identifies the object for error messages.
// Takes byteRange (*storage_dto.ByteRange) which specifies the requested start and end.
//
// Returns a reader scoped to the requested range, or an error when the range is invalid.
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

	start = max(start, 0)
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

// mimeTypeFromExtension returns the MIME type for the given file path based on its
// extension, defaulting to "application/octet-stream" if unknown.
//
// Takes filePath (string) which is the path whose extension determines the MIME type.
//
// Returns the resolved MIME type, or "application/octet-stream" when unknown.
func mimeTypeFromExtension(filePath string) string {
	return cmp.Or(mime.TypeByExtension(path.Ext(filePath)), "application/octet-stream")
}
