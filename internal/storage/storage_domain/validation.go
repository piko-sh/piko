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
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/storage/storage_dto"
)

// validatePutParams validates upload parameters.
// It checks for security issues (path traversal), valid keys, size limits,
// and content types.
//
// Takes params (*storage_dto.PutParams) which specifies the upload parameters
// to validate.
// Takes config (*ServiceConfig) which provides size limits and other
// validation settings.
//
// Returns error when validation fails due to invalid repository, key, size,
// content type, or multipart configuration.
func validatePutParams(params *storage_dto.PutParams, config *ServiceConfig) error {
	if params.Repository == "" {
		return errInvalidRepository
	}

	if params.UseContentAddressing {
		if err := validateCASParams(params); err != nil {
			return fmt.Errorf(ValidationFailedFmt, err)
		}
	} else {
		if err := validateKey(params.Key); err != nil {
			return fmt.Errorf(ValidationFailedFmt, err)
		}
	}

	if params.Size < 0 {
		return fmt.Errorf("validation failed: size cannot be negative, got %d", params.Size)
	}
	if config.MaxUploadSizeBytes > 0 && params.Size > config.MaxUploadSizeBytes {
		return fmt.Errorf("validation failed: file size %d bytes exceeds maximum allowed size of %d bytes",
			params.Size, config.MaxUploadSizeBytes)
	}

	if params.ContentType == "" {
		return errContentTypeEmpty
	}
	if !isValidContentType(params.ContentType) {
		return fmt.Errorf("validation failed: invalid content type: %s", params.ContentType)
	}

	if params.MultipartConfig != nil {
		if err := validateMultipartConfig(*params.MultipartConfig); err != nil {
			return fmt.Errorf(ValidationFailedFmt, err)
		}
	}

	return nil
}

// validateCASParams checks Content-Addressable Storage (CAS) parameters.
// This function changes params in place to set defaults if needed.
//
// Takes params (*storage_dto.PutParams) which holds the CAS parameters to
// check.
//
// Returns error when the hash algorithm is not supported, the expected hash
// format is not valid, or a key was given (CAS creates keys from content).
func validateCASParams(params *storage_dto.PutParams) error {
	if params.HashAlgorithm == "" {
		params.HashAlgorithm = "sha256"
	}

	params.HashAlgorithm = strings.ToLower(params.HashAlgorithm)
	switch params.HashAlgorithm {
	case "sha256", "md5":
	default:
		return fmt.Errorf("unsupported hash algorithm '%s', supported: sha256, md5", params.HashAlgorithm)
	}

	if params.ExpectedHash != "" {
		if err := validateHashFormat(params.ExpectedHash, params.HashAlgorithm); err != nil {
			return fmt.Errorf("validating expected hash format for %s: %w", params.HashAlgorithm, err)
		}
	}

	if params.Key != "" {
		return errKeyWithCAS
	}

	return nil
}

// validateHashFormat checks that a hash string is correct for the given
// algorithm.
//
// Takes hash (string) which is the hexadecimal hash value to check.
// Takes algorithm (string) which specifies the hash algorithm (sha256 or md5).
//
// Returns error when the hash is not valid hexadecimal, when the algorithm is
// not known, or when the hash length does not match the expected length for
// the algorithm.
func validateHashFormat(hash, algorithm string) error {
	hash = strings.TrimSpace(hash)

	if !isHexString(hash) {
		return fmt.Errorf("expected hash must be a hexadecimal string, got: %s", hash)
	}

	var expectedLength int
	switch algorithm {
	case "sha256":
		expectedLength = SHA256HexLength
	case "md5":
		expectedLength = MD5HexLength
	default:
		return fmt.Errorf("unknown hash algorithm: %s", algorithm)
	}

	if len(hash) != expectedLength {
		return fmt.Errorf("%s hash must be %d characters, got %d", algorithm, expectedLength, len(hash))
	}

	return nil
}

// isHexString checks if a string contains only hexadecimal characters.
//
// Takes s (string) which is the string to check.
//
// Returns bool which is true if the string is not empty and contains only
// valid hexadecimal characters (0-9, a-f, A-F).
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// validateMultipartConfig validates multipart upload configuration parameters.
//
// Takes config (storage_dto.MultipartUploadConfig) which specifies the
// multipart upload settings to validate.
//
// Returns error when the part size is outside the allowed range (5 MB to 5 GB),
// concurrency is less than 1 or exceeds MaxMultipartConcurrency, or max retries
// is negative or exceeds MaxRetries.
func validateMultipartConfig(config storage_dto.MultipartUploadConfig) error {
	const minPartSize = 5 * 1024 * 1024
	const maxPartSize = 5 * 1024 * 1024 * 1024

	if config.PartSize < minPartSize {
		return fmt.Errorf("multipart part size must be at least 5 MB, got %d bytes", config.PartSize)
	}

	if config.PartSize > maxPartSize {
		return fmt.Errorf("multipart part size cannot exceed 5 GB, got %d bytes", config.PartSize)
	}

	if config.Concurrency < 1 {
		return fmt.Errorf("multipart concurrency must be at least 1, got %d", config.Concurrency)
	}

	if config.Concurrency > MaxMultipartConcurrency {
		return fmt.Errorf("multipart concurrency cannot exceed %d, got %d", MaxMultipartConcurrency, config.Concurrency)
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("multipart max retries cannot be negative, got %d", config.MaxRetries)
	}

	if config.MaxRetries > MaxRetries {
		return fmt.Errorf("multipart max retries cannot exceed %d, got %d", MaxRetries, config.MaxRetries)
	}

	return nil
}

// validateByteRange checks that byte range values are valid for partial reads.
//
// Takes byteRange (storage_dto.ByteRange) which specifies the start and end
// positions for reading.
//
// Returns error when start is negative, or when end is less than start unless
// end is -1 to indicate end of file.
func validateByteRange(byteRange storage_dto.ByteRange) error {
	if byteRange.Start < 0 {
		return fmt.Errorf("byte range start must be non-negative, got %d", byteRange.Start)
	}

	if byteRange.End != -1 {
		if byteRange.End < byteRange.Start {
			return fmt.Errorf("byte range end (%d) must be >= start (%d) or -1 for end of file",
				byteRange.End, byteRange.Start)
		}
	}

	return nil
}

// validateGetParams checks parameters for read operations.
//
// Takes params (storage_dto.GetParams) which contains the repository, key, and
// optional byte range to check.
//
// Returns error when the repository is empty, the key is not valid, or the
// byte range check fails.
func validateGetParams(params storage_dto.GetParams) error {
	if params.Repository == "" {
		return errInvalidRepository
	}

	if err := validateKey(params.Key); err != nil {
		return fmt.Errorf(ValidationFailedFmt, err)
	}

	if params.ByteRange != nil {
		if err := validateByteRange(*params.ByteRange); err != nil {
			return fmt.Errorf(ValidationFailedFmt, err)
		}
	}

	return nil
}

// validateCopyParams checks that copy operation parameters are valid.
//
// Takes params (storage_dto.CopyParams) which specifies the source and
// destination repositories and keys.
//
// Returns error when the source or destination repository is empty, or when
// either key fails validation.
func validateCopyParams(params storage_dto.CopyParams) error {
	if params.SourceRepository == "" {
		return errInvalidSourceRepo
	}
	if params.DestinationRepository == "" {
		return errInvalidDestRepo
	}

	if err := validateKey(params.SourceKey); err != nil {
		return fmt.Errorf("validation failed for source: %w", err)
	}
	if err := validateKey(params.DestinationKey); err != nil {
		return fmt.Errorf("validation failed for destination: %w", err)
	}

	return nil
}

// validateKey checks that an object key is safe and well-formed.
// It prevents path traversal attacks and rejects unsafe input.
//
// Takes key (string) which is the object key to check.
//
// Returns error when the key is empty, an absolute path, contains path
// traversal sequences, contains dangerous characters, or is longer than
// MaxKeyLength.
func validateKey(key string) error {
	if key == "" {
		return errKeyEmpty
	}

	if filepath.IsAbs(key) {
		return fmt.Errorf("key cannot be an absolute path: %s", key)
	}

	cleanKey := filepath.Clean(key)
	if strings.Contains(cleanKey, "..") {
		return fmt.Errorf("key contains invalid path traversal: %s", key)
	}

	if containsDangerousChars(key) {
		return fmt.Errorf("key contains dangerous characters: %s", key)
	}

	if len(key) > MaxKeyLength {
		return fmt.Errorf("key exceeds maximum length of %d characters", MaxKeyLength)
	}

	return nil
}

// sanitiseKey cleans a key by removing or replacing unsafe characters and
// making paths consistent. This is a safety measure; validation should still
// be the main check.
//
// Takes key (string) which is the raw key to clean.
//
// Returns string which is the cleaned and consistent key.
func sanitiseKey(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")

	key = strings.TrimPrefix(key, PathSeparator)

	key = filepath.Clean(key)

	key = filepath.ToSlash(key)

	return key
}

// containsDangerousChars checks for characters that could be dangerous in
// file paths.
//
// Takes key (string) which is the path key to check for dangerous characters.
//
// Returns bool which is true if the key contains null bytes, carriage returns,
// or newlines.
func containsDangerousChars(key string) bool {
	dangerous := []string{
		"\x00",
		"\r",
		"\n",
	}

	for _, char := range dangerous {
		if strings.Contains(key, char) {
			return true
		}
	}

	return false
}

// isValidContentType checks whether a content type string has valid structure.
//
// Takes contentType (string) which is the MIME type to check.
//
// Returns bool which is true if the content type is valid.
func isValidContentType(contentType string) bool {
	if !strings.Contains(contentType, "/") {
		return false
	}

	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return false
	}

	if parts[0] == "" || parts[1] == "" {
		return false
	}

	if strings.ContainsAny(contentType, "\r\n\x00") {
		return false
	}

	return true
}

// validatePutManyParams validates parameters for batch upload operations.
//
// Takes params (*storage_dto.PutManyParams) which contains the batch upload
// specification including repository and objects.
// Takes config (*ServiceConfig) which provides validation limits such as
// maximum batch size.
//
// Returns error when the repository is empty, no objects are provided, the
// batch size exceeds the configured maximum, or any individual object fails
// validation.
func validatePutManyParams(params *storage_dto.PutManyParams, config *ServiceConfig) error {
	if params.Repository == "" {
		return errInvalidRepository
	}

	if len(params.Objects) == 0 {
		return errNoObjectsToUpload
	}

	if len(params.Objects) > config.MaxBatchSize {
		return fmt.Errorf("validation failed: batch size %d exceeds maximum of %d",
			len(params.Objects), config.MaxBatchSize)
	}

	if err := validateConcurrency(params.Concurrency); err != nil {
		return fmt.Errorf("validating batch put concurrency: %w", err)
	}

	for i, storageObject := range params.Objects {
		if err := validatePutObjectSpec(i, storageObject, config); err != nil {
			return fmt.Errorf("validating object at index %d: %w", i, err)
		}
	}

	return nil
}

// validateConcurrency checks that the concurrency value is valid for batch
// operations.
//
// Takes concurrency (int) which specifies the number of workers to run at once.
//
// Returns error when concurrency is negative or greater than MaxConcurrency.
func validateConcurrency(concurrency int) error {
	if concurrency < 0 {
		return errNegativeConcurrency
	}
	if concurrency > MaxConcurrency {
		return fmt.Errorf("validation failed: concurrency cannot exceed %d", MaxConcurrency)
	}
	return nil
}

// validatePutObjectSpec checks a single object in a batch upload.
//
// Takes index (int) which is the object's position in the batch.
// Takes storageObject (PutObjectSpec) which holds the object details to check.
// Takes config (*ServiceConfig) which sets the size limits.
//
// Returns error when the key is not valid, size is negative or too large,
// or the content type is empty or not valid.
func validatePutObjectSpec(index int, storageObject storage_dto.PutObjectSpec, config *ServiceConfig) error {
	if err := validateKey(storageObject.Key); err != nil {
		return fmt.Errorf("validation failed for object at index %d: %w", index, err)
	}

	if storageObject.Size < 0 {
		return fmt.Errorf("validation failed for object at index %d: size cannot be negative", index)
	}

	if config.MaxUploadSizeBytes > 0 && storageObject.Size > config.MaxUploadSizeBytes {
		return fmt.Errorf("validation failed for object at index %d: size %d exceeds maximum %d",
			index, storageObject.Size, config.MaxUploadSizeBytes)
	}

	if storageObject.ContentType == "" {
		return fmt.Errorf("validation failed for object at index %d: content type cannot be empty", index)
	}

	if !isValidContentType(storageObject.ContentType) {
		return fmt.Errorf("validation failed for object at index %d: invalid content type: %s",
			index, storageObject.ContentType)
	}

	return nil
}

// validateRemoveManyParams validates parameters for batch delete operations.
//
// Takes params (*storage_dto.RemoveManyParams) which contains the repository,
// keys, and concurrency settings for the batch removal.
// Takes config (*ServiceConfig) which provides limits such as maximum batch
// size.
//
// Returns error when the repository is empty, no keys are provided, batch size
// exceeds the configured maximum, concurrency is invalid, or any key fails
// validation.
func validateRemoveManyParams(params *storage_dto.RemoveManyParams, config *ServiceConfig) error {
	if params.Repository == "" {
		return errInvalidRepository
	}

	if len(params.Keys) == 0 {
		return errNoKeysToRemove
	}

	if len(params.Keys) > config.MaxBatchSize {
		return fmt.Errorf("validation failed: batch size %d exceeds maximum of %d",
			len(params.Keys), config.MaxBatchSize)
	}

	if err := validateConcurrency(params.Concurrency); err != nil {
		return fmt.Errorf("validating batch remove concurrency: %w", err)
	}

	for i, key := range params.Keys {
		if err := validateKey(key); err != nil {
			return fmt.Errorf("validation failed for key at index %d: %w", i, err)
		}
	}

	return nil
}
