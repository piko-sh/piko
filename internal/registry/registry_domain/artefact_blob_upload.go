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

package registry_domain

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/contextaware"
)

// uploadTempBlob stores source data and computes its content hash, returning the keys and
// hashes needed to finalise the blob.
//
// A seekable source is hashed first and skipped entirely when its content is already
// stored, avoiding a redundant temporary write on every restart for content already baked
// into the embedded base; every other source is streamed to a temporary key while it is
// hashed in one pass.
//
// Takes blobStore (BlobStore) which provides storage for the blob data.
// Takes sourceData (io.Reader) which supplies the data to upload.
// Takes sourcePath (string) which provides the original file path for extension
// extraction.
// Takes mimeType (string) which specifies the content type of the blob.
//
// Returns *blobUploadResult which contains the temporary key, hash, size, final key, MIME
// type, and whether the content was already present.
// Returns error when the blob cannot be hashed or saved to the store.
func uploadTempBlob(
	ctx context.Context,
	blobStore BlobStore,
	sourceData io.Reader,
	sourcePath string,
	mimeType string,
) (*blobUploadResult, error) {
	extension := filepath.Ext(sourcePath)
	if seeker, ok := sourceData.(io.ReadSeeker); ok {
		return uploadSeekableBlob(ctx, blobStore, seeker, extension, mimeType)
	}
	return uploadStreamingBlob(ctx, blobStore, sourceData, extension, mimeType)
}

// uploadSeekableBlob hashes a seekable source and, when the resulting content is already
// in the store, returns a reused result without writing anything; otherwise it rewinds
// and writes the source to a temporary key.
//
// Takes blobStore (BlobStore) which provides storage for the blob data.
// Takes source (io.ReadSeeker) which supplies the rewindable data to upload.
// Takes extension (string) which is the final key's file extension.
// Takes mimeType (string) which specifies the content type of the blob.
//
// Returns *blobUploadResult which describes the blob and whether it was reused.
// Returns error when hashing, rewinding, or writing fails.
func uploadSeekableBlob(
	ctx context.Context,
	blobStore BlobStore,
	source io.ReadSeeker,
	extension, mimeType string,
) (*blobUploadResult, error) {
	ctx, l := logger_domain.From(ctx, log)

	origin, err := source.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("reading source blob position: %w", err)
	}

	contentHash, sriHash, size, err := hashSource(ctx, source)
	if err != nil {
		return nil, err
	}
	if _, err := source.Seek(origin, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewinding source blob: %w", err)
	}

	result := &blobUploadResult{
		tempKey:  "",
		hash:     contentHash,
		sriHash:  sriHash,
		finalKey: fmt.Sprintf("source/%s%s", contentHash, extension),
		mimeType: mimeType,
		size:     size,
		reused:   false,
	}

	exists, existsErr := blobStore.Exists(ctx, result.finalKey)
	if existsErr != nil {
		l.Warn("Failed to check for an existing blob before upload, will write it",
			logger_domain.Error(existsErr), logger_domain.String(logKeyStorageKey, result.finalKey))
	} else if exists {
		result.reused = true
		registryServiceBlobDeduplicationHitCount.Add(ctx, 1)
		return result, nil
	}

	tempKey := "tmp/" + uuid.NewString()
	written := &writeCounter{}
	if err := blobStore.Put(ctx, tempKey, io.TeeReader(source, written)); err != nil {
		deleteTemporaryBlob(ctx, blobStore, tempKey)
		return nil, fmt.Errorf("failed to save source blob: %w", err)
	}

	if written.total != size {
		deleteTemporaryBlob(ctx, blobStore, tempKey)
		return nil, fmt.Errorf("source blob changed while being stored, hashed %d bytes but wrote %d: %w",
			size, written.total, ErrBlobChangedDuringUpload)
	}

	result.tempKey = tempKey
	return result, nil
}

// deleteTemporaryBlob removes a temporary blob after a failed upload.
//
// Removing it here stops a partial write lingering until the orphan sweep reclaims it.
//
// Takes blobStore (BlobStore) which holds the temporary blob.
// Takes tempKey (string) which is the temporary blob's key.
func deleteTemporaryBlob(ctx context.Context, blobStore BlobStore, tempKey string) {
	if tempKey == "" {
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	if err := blobStore.Delete(ctx, tempKey); err != nil {
		l.Warn("Failed to remove a temporary blob after a failed upload",
			logger_domain.Error(err), logger_domain.String(logKeyStorageKey, tempKey))
	}
}

// uploadStreamingBlob writes a non-seekable source to a temporary key while hashing it in
// a single pass, since such a source cannot be rewound to check for an existing copy
// first.
//
// Takes blobStore (BlobStore) which provides storage for the blob data.
// Takes sourceData (io.Reader) which supplies the data to upload.
// Takes extension (string) which is the final key's file extension.
// Takes mimeType (string) which specifies the content type of the blob.
//
// Returns *blobUploadResult which describes the written blob.
// Returns error when a hasher cannot be obtained or the store write fails.
func uploadStreamingBlob(
	ctx context.Context,
	blobStore BlobStore,
	sourceData io.Reader,
	extension, mimeType string,
) (*blobUploadResult, error) {
	tempKey := "tmp/" + uuid.NewString()

	sha256Hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		return nil, errors.New("sha256Pool returned unexpected type")
	}
	sha384Hasher, ok := sha384Pool.Get().(hash.Hash)
	if !ok {
		sha256Pool.Put(sha256Hasher)
		return nil, errors.New("sha384Pool returned unexpected type")
	}
	sha256Hasher.Reset()
	sha384Hasher.Reset()

	counter := &writeCounter{}
	teeReader := io.TeeReader(sourceData, io.MultiWriter(sha256Hasher, sha384Hasher, counter))

	if err := blobStore.Put(ctx, tempKey, teeReader); err != nil {
		sha256Pool.Put(sha256Hasher)
		sha384Pool.Put(sha384Hasher)
		deleteTemporaryBlob(ctx, blobStore, tempKey)
		return nil, fmt.Errorf("failed to save source blob: %w", err)
	}

	contentHash := hex.EncodeToString(sha256Hasher.Sum(nil))
	sriHash := encodeSRIHash(sha384Hasher)

	sha256Pool.Put(sha256Hasher)
	sha384Pool.Put(sha384Hasher)

	return &blobUploadResult{
		tempKey:  tempKey,
		hash:     contentHash,
		sriHash:  sriHash,
		finalKey: fmt.Sprintf("source/%s%s", contentHash, extension),
		mimeType: mimeType,
		size:     counter.total,
		reused:   false,
	}, nil
}

// hashSource reads a source to completion, returning its content hash, Subresource
// Integrity hash, and byte size without writing it anywhere. The caller rewinds a
// seekable source before reading it a second time.
//
// The read observes cancellation, so a large source cannot keep the enclosing registry
// transaction open past its deadline.
//
// Takes source (io.Reader) which supplies the data to hash.
//
// Returns contentHash (string) which is the hex-encoded SHA-256 of the content.
// Returns sriHash (string) which is the "sha384-<base64>" Subresource Integrity hash.
// Returns size (int64) which is the number of bytes read.
// Returns error when a hasher cannot be obtained or the source cannot be read.
func hashSource(ctx context.Context, source io.Reader) (contentHash string, sriHash string, size int64, err error) {
	sha256Hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		return "", "", 0, errors.New("sha256Pool returned unexpected type")
	}
	defer sha256Pool.Put(sha256Hasher)
	sha384Hasher, ok := sha384Pool.Get().(hash.Hash)
	if !ok {
		return "", "", 0, errors.New("sha384Pool returned unexpected type")
	}
	defer sha384Pool.Put(sha384Hasher)
	sha256Hasher.Reset()
	sha384Hasher.Reset()

	counter := &writeCounter{}
	if _, copyErr := io.Copy(io.MultiWriter(sha256Hasher, sha384Hasher, counter), contextaware.NewReader(ctx, source)); copyErr != nil {
		return "", "", 0, fmt.Errorf("hashing source blob: %w", copyErr)
	}

	return hex.EncodeToString(sha256Hasher.Sum(nil)), encodeSRIHash(sha384Hasher), counter.total, nil
}

// encodeSRIHash renders a SHA-384 digest as a Subresource Integrity value.
//
// Takes sha384Hasher (hash.Hash) which holds the completed SHA-384 digest.
//
// Returns string which is the "sha384-<base64>" Subresource Integrity hash.
func encodeSRIHash(sha384Hasher hash.Hash) string {
	return string(base64.StdEncoding.AppendEncode([]byte("sha384-"), sha384Hasher.Sum(nil)))
}

// finaliseBlobStorage moves a temporary blob to its final storage location. If a copy
// already exists, it removes the temporary blob instead.
//
// Takes blobStore (BlobStore) which provides blob storage operations.
// Takes tempKey (string) which is the temporary storage key for the blob.
// Takes finalKey (string) which is the target storage key for the blob.
//
// Returns error when the blob cannot be moved to its final location.
func finaliseBlobStorage(
	ctx context.Context,
	blobStore BlobStore,
	tempKey, finalKey string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	blobExists, err := blobStore.Exists(ctx, finalKey)
	if err != nil {
		l.Warn("Failed to check blob existence, will attempt upload",
			logger_domain.Error(err), logger_domain.String(logKeyStorageKey, finalKey))
		blobExists = false
	}

	if blobExists {
		l.Trace("Blob already exists (deduplication), reusing existing blob",
			logger_domain.String(logKeyStorageKey, finalKey))
		_ = blobStore.Delete(ctx, tempKey)
		registryServiceBlobDeduplicationHitCount.Add(ctx, 1)
		return nil
	}

	l.Trace("Blob doesn't exist, moving from temp to final location",
		logger_domain.String("from", tempKey), logger_domain.String("to", finalKey))
	if err := blobStore.Rename(ctx, tempKey, finalKey); err != nil {
		_ = blobStore.Delete(ctx, tempKey)
		return fmt.Errorf("failed to rename temp blob: %w", err)
	}
	return nil
}
