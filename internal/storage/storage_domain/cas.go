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

package storage_domain

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 for checksums, not cryptographic
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// logFieldAlgorithm is the log field key for the hash algorithm name.
	logFieldAlgorithm = "algorithm"

	// logFieldHash is the log field name for content hash values.
	logFieldHash = "hash"
)

// errCASDeduplicated is a sentinel error used internally to signal that a CAS
// upload was successfully deduplicated and skipped, which is not a failure
// condition.
var errCASDeduplicated = errors.New("CAS object already exists, upload skipped")

// CASResult holds the results of a successful content-addressing operation.
// The caller is responsible for managing the returned resources.
type CASResult struct {
	// Reader provides access to the object content; the caller must close it.
	Reader io.ReadCloser

	// Cleanup removes the temporary file from disk.
	Cleanup func()

	// Key is the storage key for the object after processing.
	Key string
}

// cleanupTempFile closes and removes a temporary file, ignoring errors.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for
// removal.
// Takes tmpFile (safedisk.FileHandle) which is the temporary file to clean up.
func cleanupTempFile(ctx context.Context, sandbox safedisk.Sandbox, tmpFile safedisk.FileHandle) {
	_, l := logger_domain.From(ctx, log)
	if err := tmpFile.Close(); err != nil {
		l.Warn("closing temp file during cleanup", logger_domain.Error(err))
	}
	if err := sandbox.Remove(tmpFile.Name()); err != nil {
		l.Warn("removing temp file during cleanup", logger_domain.Error(err))
	}
}

// setupHasher creates a hash.Hash for the given algorithm.
//
// Takes algorithm (string) which specifies the hashing method to use.
// Supported values are "sha256" and "md5".
//
// Returns hash.Hash which is the hasher ready for use.
// Returns error when the algorithm is not supported.
func setupHasher(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "sha256":
		return sha256.New(), nil
	case "md5":
		//nolint:gosec // MD5 for CAS, not cryptographic
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

// streamAndComputeHash writes data to a writer while computing its hash at
// the same time.
//
// Takes w (io.Writer) which is the destination where data will be written.
// Takes hasher (hash.Hash) which computes the hash as data flows through.
// Takes reader (io.Reader) which provides the source data.
//
// Returns string which is the hex-encoded hash of all data written.
// Returns error when the data cannot be copied to the writer or hasher.
func streamAndComputeHash(w io.Writer, hasher hash.Hash, reader io.Reader) (string, error) {
	multiWriter := io.MultiWriter(w, hasher)
	if _, err := io.Copy(multiWriter, reader); err != nil {
		return "", fmt.Errorf("failed to process data for CAS hashing: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// checkCASDeduplication checks if an object with the given hash already
// exists in storage. Returns errCASDeduplicated if found, nil if not found,
// or an error if the stat operation fails.
//
// Takes provider (StorageProviderPort) which provides access to storage
// operations.
// Takes repository (string) which identifies the target repository.
// Takes generatedKey (string) which is the content-addressed key to check.
// Takes actualHash (string) which is the hash of the content being uploaded.
//
// Returns error when the object already exists (errCASDeduplicated) or when
// the stat operation fails.
func checkCASDeduplication(
	ctx context.Context, provider StorageProviderPort,
	repository, generatedKey, actualHash string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	statParams := storage_dto.GetParams{
		Repository:      repository,
		Key:             generatedKey,
		ByteRange:       nil,
		TransformConfig: nil,
	}
	if _, err := provider.Stat(ctx, statParams); err == nil {
		l.Trace("CAS deduplication: object with this hash already exists, skipping upload",
			logger_domain.String(logFieldKey, generatedKey),
			logger_domain.String(logFieldHash, actualHash))
		return errCASDeduplicated
	}
	return nil
}

// handleContentAddressing processes content-addressable storage by streaming
// input to a temporary file while computing its hash.
//
// Produces a CASResult holding the new reader, the generated key, and a
// cleanup function. The caller takes ownership of these resources and must
// call the cleanup function when done.
//
// Takes provider (StorageProviderPort) which handles storage operations.
// Takes params (*storage_dto.PutParams) which specifies the input and hash
// algorithm.
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem access
// for temporary files.
//
// Returns *CASResult which contains the reader, generated key, and cleanup
// function for the caller to manage.
// Returns error when temporary file creation fails, hash computation fails,
// hash verification fails, or the file cannot be rewound.
func handleContentAddressing(
	ctx context.Context, provider StorageProviderPort,
	params *storage_dto.PutParams, sandbox safedisk.Sandbox,
) (*CASResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Content-addressable storage enabled, computing hash via streaming",
		logger_domain.String(logFieldAlgorithm, params.HashAlgorithm))

	tmpFile, err := sandbox.CreateTemp(".", "cas-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for CAS: %w", err)
	}

	hasher, err := setupHasher(params.HashAlgorithm)
	if err != nil {
		cleanupTempFile(ctx, sandbox, tmpFile)
		return nil, fmt.Errorf("setting up hasher for CAS with algorithm %q: %w", params.HashAlgorithm, err)
	}

	actualHash, err := streamAndComputeHash(tmpFile, hasher, params.Reader)
	if err != nil {
		cleanupTempFile(ctx, sandbox, tmpFile)
		return nil, fmt.Errorf("streaming and computing hash for CAS: %w", err)
	}

	if params.ExpectedHash != "" && !strings.EqualFold(actualHash, params.ExpectedHash) {
		cleanupTempFile(ctx, sandbox, tmpFile)
		return nil, fmt.Errorf("hash mismatch: expected %s, got %s", params.ExpectedHash, actualHash)
	}

	generatedKey := fmt.Sprintf("cas/%s/%s", params.HashAlgorithm, actualHash)
	l.Trace("Generated CAS key from content hash", logger_domain.String(logFieldKey, generatedKey), logger_domain.String(logFieldHash, actualHash))

	if err := checkCASDeduplication(ctx, provider, params.Repository, generatedKey, actualHash); err != nil {
		cleanupTempFile(ctx, sandbox, tmpFile)
		if errors.Is(err, errCASDeduplicated) {
			return nil, err
		}
		return nil, fmt.Errorf("checking CAS deduplication for key %q: %w", generatedKey, err)
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		cleanupTempFile(ctx, sandbox, tmpFile)
		return nil, fmt.Errorf("failed to seek temp file for upload: %w", err)
	}

	l.Trace("CAS object does not exist, proceeding with upload from temp file", logger_domain.String(logFieldKey, generatedKey))

	return &CASResult{
		Reader: tmpFile,
		Key:    generatedKey,
		Cleanup: func() {
			_, cl := logger_domain.From(ctx, log)
			if err := sandbox.Remove(tmpFile.Name()); err != nil {
				cl.Warn("removing temp file during CAS cleanup", logger_domain.Error(err))
			}
		},
	}, nil
}
