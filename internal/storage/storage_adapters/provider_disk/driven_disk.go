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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/internal/json"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultCallsPerSecond is the default rate limit for the disk provider,
	// set to zero to disable rate limiting for local disk operations.
	defaultCallsPerSecond = 0

	// defaultBurst is the default burst size for rate limiting.
	defaultBurst = 0

	// metadataSidecarSuffix is the file extension for JSON metadata sidecar files.
	metadataSidecarSuffix = ".metadata.json"

	// md5SidecarSuffix is the file extension for MD5 hash sidecar files.
	md5SidecarSuffix = ".md5"

	// dirPermissions is the file mode used when creating directories.
	// Uses 0750: owner rwx, group rx, others none.
	dirPermissions = 0750

	// filePermissions is the Unix permission mode for sidecar cache files.
	// Uses 0640: owner rw, group r, others none.
	filePermissions = 0640

	// logOpPut is the operation name for put operations in logging and metrics.
	logOpPut = "put"

	// logOpGet is the metric label for get operations.
	logOpGet = "get"

	// logOpRemove is the operation name for file removal metrics.
	logOpRemove = "remove"

	// logOpRename is the operation name for rename operations in metrics.
	logOpRename = "rename"

	// logOpExists is the operation name for file existence checks.
	logOpExists = "exists"

	// logFieldPath is the structured log field name for file paths.
	logFieldPath = "path"

	// metricAttrOperation is the metric attribute key for the operation name.
	metricAttrOperation = "operation"

	// minDiskSpaceMB is the minimum required disk space in megabytes.
	minDiskSpaceMB = 1024

	// bytesPerMegabyte is the number of bytes in a megabyte (1024 x 1024).
	bytesPerMegabyte = 1024 * 1024

	// md5HashHexLength is the length of an MD5 hash in hexadecimal form (32
	// chars).
	md5HashHexLength = 32

	// defaultMaxSidecarBytes caps reads from sidecar files.
	//
	// Sidecars hold tiny payloads (an md5 digest or a JSON metadata map of opaque
	// keys/values), so the cap is generous (1 MiB) and just defends against a
	// corrupted or attacker-influenced sidecar that would otherwise be slurped
	// whole into memory.
	//
	// TODO(piko): expose as DiskProvider Config option once a broader storage
	// adapter config refactor lands.
	defaultMaxSidecarBytes int64 = 1 * 1024 * 1024
)

// DiskProvider implements StorageProviderPort using the local filesystem.
// It features atomic writes to prevent data corruption and sandboxes all
// file operations using safedisk for security.
type DiskProvider struct {
	// rateLimiter controls how often emails can be sent.
	rateLimiter *storage_domain.ProviderRateLimiter

	// sandbox provides file system operations for safe disk access.
	sandbox safedisk.Sandbox
}

var _ storage_domain.StorageProviderPort = (*DiskProvider)(nil)
var _ provider_domain.ProviderMetadata = (*DiskProvider)(nil)

// Config holds the necessary configuration for the DiskProvider.
type Config struct {
	// Sandbox is the filesystem sandbox used for all file operations.
	// If nil, a new sandbox is created using BaseDirectory.
	Sandbox safedisk.Sandbox

	// SandboxFactory creates sandboxes when Sandbox is nil. When non-nil and
	// Sandbox is nil, this factory is used instead of safedisk.NewSandbox.
	SandboxFactory safedisk.Factory

	// BaseDirectory is the root directory where all repositories and objects will
	// be stored.
	BaseDirectory string
}

// Put saves an object from a reader to the disk using an atomic write
// operation.
//
// Takes params (*storage_dto.PutParams) which specifies the repository, key,
// reader, and optional metadata for the object.
//
// Returns error when the path is invalid, directory creation fails, temporary
// file operations fail, or the atomic rename fails.
func (d *DiskProvider) Put(ctx context.Context, params *storage_dto.PutParams) error {
	startTime := time.Now()

	if params.Reader == nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return errors.New("reader cannot be nil for put operation")
	}

	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return fmt.Errorf("resolving path for put %q: %w", params.Key, err)
	}

	directory := filepath.Dir(path)
	if err := d.sandbox.MkdirAll(directory, dirPermissions); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return fmt.Errorf("failed to create directories for '%s': %w", path, err)
	}

	tmpFile, err := d.sandbox.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return fmt.Errorf("failed to create temporary file for upload: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() { _ = d.sandbox.Remove(tmpName) }()

	bytesWritten, err := io.Copy(tmpFile, contextaware.NewReader(ctx, params.Reader))
	if err == nil {
		if syncErr := tmpFile.Sync(); syncErr != nil {
			err = fmt.Errorf("failed to fsync temporary file '%s': %w", tmpName, syncErr)
		}
	}
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = fmt.Errorf("failed to close temporary file: %w", closeErr)
	}
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return fmt.Errorf("failed to write data to temporary file '%s': %w", tmpName, err)
	}

	if err := d.sandbox.Rename(tmpName, path); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
		return fmt.Errorf("failed to atomically move file to '%s': %w", path, err)
	}

	d.syncDirectoryAfterRename(ctx, directory)

	d.writeMetadataSidecar(ctx, path, params.Metadata)

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))
	BytesTransferred.Add(ctx, bytesWritten, metric.WithAttributes(attribute.String(metricAttrOperation, logOpPut)))

	return nil
}

// Get retrieves a file from disk as a readable stream, supporting partial
// reads.
//
// Takes params (storage_dto.GetParams) which specifies the file to retrieve
// and any byte range for partial reads.
//
// Returns io.ReadCloser which provides the file content as a stream.
// Returns error when the path is invalid or the file cannot be opened.
func (d *DiskProvider) Get(ctx context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	startTime := time.Now()

	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpGet)))
		return nil, fmt.Errorf("resolving path for get %q: %w", params.Key, err)
	}

	file, err := d.sandbox.Open(path)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpGet)))
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to open file '%s': %w", path, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(metricAttrOperation, logOpGet)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpGet)))

	if params.ByteRange == nil {
		return file, nil
	}

	return d.handleRangeRequest(ctx, file, params.Key, params.ByteRange)
}

// Stat retrieves file metadata from disk.
//
// Takes params (storage_dto.GetParams) which specifies the repository and key.
//
// Returns *storage_domain.ObjectInfo which contains the file metadata.
// Returns error when the path is invalid or the file does not exist.
func (d *DiskProvider) Stat(ctx context.Context, params storage_dto.GetParams) (*storage_domain.ObjectInfo, error) {
	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		return nil, fmt.Errorf("resolving path for stat %q: %w", params.Key, err)
	}

	fileInfo, err := d.sandbox.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to stat file '%s': %w", path, err)
	}

	objInfo := &storage_domain.ObjectInfo{
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		ContentType:  mimeTypeFromExtension(path),
		ETag:         "",
		Metadata:     nil,
	}

	if metadata, metaErr := d.readMetadataSidecar(ctx, path); metaErr == nil {
		objInfo.Metadata = metadata
	}

	return objInfo, nil
}

// Copy performs an intra-repository file copy atomically.
//
// Takes repo (string) which identifies the repository containing the objects.
// Takes srcKey (string) which specifies the source object key.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the source or destination key is invalid, or the copy
// fails.
func (d *DiskProvider) Copy(_ context.Context, repo string, srcKey, dstKey string) error {
	srcPath, err := repoPath(repo, srcKey)
	if err != nil {
		return fmt.Errorf("invalid source key: %w", err)
	}
	dstPath, err := repoPath(repo, dstKey)
	if err != nil {
		return fmt.Errorf("invalid destination key: %w", err)
	}
	if err := d.copyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("copying file from %q to %q: %w", srcKey, dstKey, err)
	}
	return nil
}

// CopyToAnotherRepository performs an inter-repository file copy atomically.
//
// Takes srcRepo (string) which identifies the source repository.
// Takes srcKey (string) which specifies the source object key.
// Takes dstRepo (string) which identifies the destination repository.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when a key is invalid or the copy fails.
func (d *DiskProvider) CopyToAnotherRepository(_ context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	srcPath, err := repoPath(srcRepo, srcKey)
	if err != nil {
		return fmt.Errorf("invalid source key: %w", err)
	}
	dstPath, err := repoPath(dstRepo, dstKey)
	if err != nil {
		return fmt.Errorf("invalid destination key: %w", err)
	}
	if err := d.copyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("copying file from %q/%q to %q/%q: %w", srcRepo, srcKey, dstRepo, dstKey, err)
	}
	return nil
}

// Remove deletes a file and its associated sidecar files from the disk.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key of the file to delete.
//
// Returns error when the path is invalid or the file cannot be removed.
func (d *DiskProvider) Remove(ctx context.Context, params storage_dto.GetParams) error {
	startTime := time.Now()

	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRemove)))
		return fmt.Errorf("resolving path for remove %q: %w", params.Key, err)
	}

	err = d.sandbox.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRemove)))
		return fmt.Errorf("failed to remove file '%s': %w", path, err)
	}

	d.removeSidecarFiles(path)

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(metricAttrOperation, logOpRemove)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRemove)))

	return nil
}

// Rename atomically moves a file from oldKey to newKey within the same
// repository. Uses os.Rename which is atomic on POSIX systems.
//
// Takes repo (string) which identifies the repository containing the file.
// Takes oldKey (string) which specifies the current file key.
// Takes newKey (string) which specifies the desired new file key.
//
// Returns error when a key is invalid, directory creation fails, or the
// rename operation fails.
func (d *DiskProvider) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	startTime := time.Now()

	oldPath, err := repoPath(repo, oldKey)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))
		return fmt.Errorf("invalid source key: %w", err)
	}

	newPath, err := repoPath(repo, newKey)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))
		return fmt.Errorf("invalid destination key: %w", err)
	}

	if err := d.sandbox.MkdirAll(filepath.Dir(newPath), dirPermissions); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))
		return fmt.Errorf("failed to create destination directories for '%s': %w", newPath, err)
	}

	if err := d.sandbox.Rename(oldPath, newPath); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))
		return fmt.Errorf("failed to rename '%s' to '%s': %w", oldPath, newPath, err)
	}

	d.removeSidecarFiles(oldPath)

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpRename)))

	return nil
}

// Exists checks if a file exists at the given key.
// Returns (true, nil) if exists, (false, nil) if not exists, (false, error) on
// failure.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key to check.
//
// Returns bool which is true if the file exists, false otherwise.
// Returns error when the path is invalid or the stat operation fails
// unexpectedly.
func (d *DiskProvider) Exists(ctx context.Context, params storage_dto.GetParams) (bool, error) {
	startTime := time.Now()

	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpExists)))
		return false, fmt.Errorf("resolving path for exists check %q: %w", params.Key, err)
	}

	_, err = d.sandbox.Stat(path)

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(metricAttrOperation, logOpExists)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpExists)))

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(metricAttrOperation, logOpExists)))
		return false, fmt.Errorf("failed to stat '%s': %w", path, err)
	}

	return true, nil
}

// ListKeys returns all non-sidecar, non-temporary storage keys within the
// given repository directory.
//
// Takes repository (string) which identifies the repository to scan.
//
// Returns []string which contains all discovered storage keys.
// Returns error when the directory walk fails.
func (d *DiskProvider) ListKeys(_ context.Context, repository string) ([]string, error) {
	root := repository
	if root == "" {
		root = "."
	}

	var keys []string
	err := d.sandbox.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, metadataSidecarSuffix) ||
			strings.HasSuffix(path, md5SidecarSuffix) ||
			strings.HasSuffix(path, ".tmp") {
			return nil
		}
		key := strings.TrimPrefix(path, root+string(filepath.Separator))
		keys = append(keys, filepath.ToSlash(key))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking repository %q: %w", repository, err)
	}
	return keys, nil
}

// GetHash returns the MD5 hash of a file, using a .md5 sidecar file for
// caching. It checks if the cached hash is still valid and recomputes it if
// the source file has changed.
//
// Takes params (storage_dto.GetParams) which specifies the repository and key.
//
// Returns string which is the MD5 hash of the file.
// Returns error when the path cannot be resolved or the hash cannot be
// computed.
func (d *DiskProvider) GetHash(ctx context.Context, params storage_dto.GetParams) (string, error) {
	path, err := repoPath(params.Repository, params.Key)
	if err != nil {
		return "", fmt.Errorf("resolving path for hash %q: %w", params.Key, err)
	}

	sidecarPath := path + md5SidecarSuffix

	if cachedHash, err := d.readAndValidateCache(ctx, path, sidecarPath); err == nil {
		return cachedHash, nil
	}

	return d.computeAndCacheHash(ctx, path, sidecarPath)
}

// PresignURL is not supported by the disk provider.
//
// Returns string which is always empty for this provider.
// Returns error which is always non-nil because this operation is not
// supported.
func (*DiskProvider) PresignURL(_ context.Context, _ storage_dto.PresignParams) (string, error) {
	return "", errors.New("presigned URLs are not supported by the disk provider")
}

// PresignDownloadURL is not supported by the disk provider.
// The service layer provides a fallback for disk providers.
//
// Returns string which is always empty for this provider.
// Returns error which is always non-nil as this operation is not supported.
func (*DiskProvider) PresignDownloadURL(_ context.Context, _ storage_dto.PresignDownloadParams) (string, error) {
	return "", errors.New("presigned download URLs are not supported by the disk provider")
}

// Close releases the resources held by the sandbox.
//
// Returns error when the sandbox fails to close.
func (d *DiskProvider) Close(_ context.Context) error {
	if err := d.sandbox.Close(); err != nil {
		return fmt.Errorf("closing disk provider sandbox: %w", err)
	}
	return nil
}

// SupportsMultipart reports whether multipart uploads are supported.
// The disk provider uses simple atomic writes instead.
//
// Returns bool which is always false for this provider.
func (*DiskProvider) SupportsMultipart() bool {
	return false
}

// SupportsBatchOperations returns false as the local disk has no native
// batch API.
//
// Returns bool which is always false for this provider.
func (*DiskProvider) SupportsBatchOperations() bool {
	return false
}

// SupportsRetry returns false; disk operations should be retried by the
// service layer.
//
// Returns bool which is always false for disk providers.
func (*DiskProvider) SupportsRetry() bool {
	return false
}

// SupportsCircuitBreaking returns false; the service layer handles circuit
// breaking.
//
// Returns bool which is always false for this provider.
func (*DiskProvider) SupportsCircuitBreaking() bool {
	return false
}

// SupportsRateLimiting returns whether rate limiting is needed for this
// provider.
//
// Returns bool which is always false as local disk operations do not need
// rate limiting.
func (*DiskProvider) SupportsRateLimiting() bool {
	return false
}

// SupportsPresignedURLs returns false as the disk provider cannot generate
// native presigned URLs. The storage service will use its fallback mechanism
// (HMAC-signed tokens + HTTP upload endpoint) instead.
//
// Returns bool which is always false for this provider.
func (*DiskProvider) SupportsPresignedURLs() bool {
	return false
}

// RemoveMany implements batch delete using sequential removal.
//
// Takes params (storage_dto.RemoveManyParams) which specifies the keys to
// delete and error handling behaviour.
//
// Returns *storage_dto.BatchResult which contains counts and details of
// successful and failed deletions.
// Returns error when the operation fails.
func (d *DiskProvider) RemoveMany(ctx context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	result := &storage_dto.BatchResult{
		TotalRequested:  len(params.Keys),
		TotalSuccessful: 0,
		TotalFailed:     0,
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		ProcessingTime:  0,
	}

	for _, key := range params.Keys {
		if ctx.Err() != nil {
			break
		}
		getParams := storage_dto.GetParams{
			Repository:      params.Repository,
			Key:             key,
			ByteRange:       nil,
			TransformConfig: nil,
		}
		err := d.Remove(ctx, getParams)
		if err != nil {
			result.FailedKeys = append(result.FailedKeys, storage_dto.BatchFailure{
				Key:       key,
				Error:     err.Error(),
				Retryable: false,
				ErrorCode: "",
			})
			result.TotalFailed++
			if !params.ContinueOnError {
				break
			}
		} else {
			result.SuccessfulKeys = append(result.SuccessfulKeys, key)
			result.TotalSuccessful++
		}
	}

	result.ProcessingTime = time.Since(startTime)
	l.Trace("Batch delete completed",
		logger_domain.String("provider", "disk"),
		logger_domain.Int("total", result.TotalRequested),
		logger_domain.Int("successful", result.TotalSuccessful),
		logger_domain.Int("failed", result.TotalFailed),
		logger_domain.Duration("duration", result.ProcessingTime))
	return result, nil
}

// PutMany implements batch upload using sequential uploads.
//
// Takes params (*storage_dto.PutManyParams) which specifies the objects to
// upload and batch settings.
//
// Returns *storage_dto.BatchResult which contains counts and details of
// successful and failed uploads.
// Returns error when the batch operation fails.
func (d *DiskProvider) PutMany(ctx context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()
	result := &storage_dto.BatchResult{
		TotalRequested:  len(params.Objects),
		TotalSuccessful: 0,
		TotalFailed:     0,
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		ProcessingTime:  0,
	}

	for _, storageObject := range params.Objects {
		if ctx.Err() != nil {
			break
		}
		putParam := &storage_dto.PutParams{
			Repository:           params.Repository,
			Key:                  storageObject.Key,
			Reader:               storageObject.Reader,
			Size:                 storageObject.Size,
			ContentType:          storageObject.ContentType,
			Metadata:             nil,
			TransformConfig:      params.TransformConfig,
			MultipartConfig:      nil,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			UseContentAddressing: false,
		}
		err := d.Put(ctx, putParam)
		if err != nil {
			result.FailedKeys = append(result.FailedKeys, storage_dto.BatchFailure{
				Key:       storageObject.Key,
				Error:     err.Error(),
				Retryable: false,
				ErrorCode: "",
			})
			result.TotalFailed++
			if !params.ContinueOnError {
				break
			}
		} else {
			result.SuccessfulKeys = append(result.SuccessfulKeys, storageObject.Key)
			result.TotalSuccessful++
		}
	}

	result.ProcessingTime = time.Since(startTime)
	l.Trace("Batch upload completed",
		logger_domain.String("provider", "disk"),
		logger_domain.Int("total", result.TotalRequested),
		logger_domain.Int("successful", result.TotalSuccessful),
		logger_domain.Int("failed", result.TotalFailed),
		logger_domain.Duration("duration", result.ProcessingTime))
	return result, nil
}

// Name returns the display name of this provider.
// Implements the healthprobe_domain.Probe interface.
//
// Returns string which is the human-readable name for this provider.
func (*DiskProvider) Name() string {
	return "StorageProvider (Disk)"
}

// Check implements the healthprobe_domain.Probe interface.
// It checks that the storage folder is reachable and has enough disk space.
//
// Takes checkType (healthprobe_dto.CheckType) which sets the type of health
// check to run.
//
// Returns healthprobe_dto.Status which holds the health state and details.
func (d *DiskProvider) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	rootPath := d.sandbox.Root()
	info, err := d.sandbox.Stat(".")
	if err != nil {
		state := healthprobe_dto.StateUnhealthy
		message := fmt.Sprintf("Storage directory not accessible: %v", err)

		return healthprobe_dto.Status{
			Name:         d.Name(),
			State:        state,
			Message:      message,
			Timestamp:    time.Now(),
			Duration:     time.Since(startTime).String(),
			Dependencies: nil,
		}
	}

	if !info.IsDir() {
		return healthprobe_dto.Status{
			Name:         d.Name(),
			State:        healthprobe_dto.StateUnhealthy,
			Message:      fmt.Sprintf("Storage path is not a directory: %s", rootPath),
			Timestamp:    time.Now(),
			Duration:     time.Since(startTime).String(),
			Dependencies: nil,
		}
	}

	if checkType == healthprobe_dto.CheckTypeReadiness {
		return d.checkDiskSpace(startTime)
	}

	return healthprobe_dto.Status{
		Name:         d.Name(),
		State:        healthprobe_dto.StateHealthy,
		Message:      "Disk storage directory accessible",
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// GetProviderType returns the provider implementation type.
//
// Returns string which identifies the provider type for monitoring.
func (*DiskProvider) GetProviderType() string {
	return "disk"
}

// GetProviderMetadata returns metadata about the disk storage provider.
//
// Returns map[string]any which contains provider details including version,
// base directory, supported features, and disk space information.
func (d *DiskProvider) GetProviderMetadata() map[string]any {
	rootPath := d.sandbox.Root()

	availableMB, totalMB, _ := getDiskSpace(rootPath)

	return map[string]any{
		"version":            "1.0.0",
		"environment":        "production",
		"description":        "Local disk storage provider",
		"base_directory":     rootPath,
		"supports_multipart": false,
		"supports_presigned": false,
		"available_space_mb": availableMB,
		"total_space_mb":     totalMB,
	}
}

// handleRangeRequest prepares a file reader for a specific byte range.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes file (safedisk.FileHandle) which is the open file to read from.
// Takes key (string) which identifies the file for logging purposes.
// Takes byteRange (*storage_dto.ByteRange) which specifies the range to read.
//
// Returns io.ReadCloser which provides the requested byte range content.
// Returns error when the file cannot be seeked to the start position.
func (*DiskProvider) handleRangeRequest(ctx context.Context, file safedisk.FileHandle, key string, byteRange *storage_dto.ByteRange) (io.ReadCloser, error) {
	if _, err := file.Seek(byteRange.Start, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to seek to position %d: %w", byteRange.Start, err)
	}

	_, l := logger_domain.From(ctx, log)

	var reader io.Reader = file
	if byteRange.End != -1 {
		bytesToRead := byteRange.End - byteRange.Start + 1
		reader = io.LimitReader(file, bytesToRead)
		l.Trace("Disk range request",
			logger_domain.String("key", key),
			logger_domain.Int64("start", byteRange.Start),
			logger_domain.Int64("end", byteRange.End),
			logger_domain.Int64("bytes_to_read", bytesToRead))
	} else {
		l.Trace("Disk range request (to end)", logger_domain.String("key", key), logger_domain.Int64("start", byteRange.Start))
	}

	return &rangeReadCloser{
		reader: reader,
		closer: file,
	}, nil
}

// copyFile copies a file using an atomic write pattern.
//
// Takes srcPath (string) which is the path to the source file.
// Takes dstPath (string) which is the path to the destination file.
//
// Returns error when the source file cannot be opened, destination
// directories cannot be created, or the atomic rename fails.
func (d *DiskProvider) copyFile(srcPath, dstPath string) error {
	srcFile, err := d.sandbox.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", srcPath, err)
	}
	defer func() { _ = srcFile.Close() }()

	if err := d.sandbox.MkdirAll(filepath.Dir(dstPath), dirPermissions); err != nil {
		return fmt.Errorf("failed to create destination directories for '%s': %w", dstPath, err)
	}

	tmpFile, err := d.sandbox.CreateTemp(filepath.Dir(dstPath), filepath.Base(dstPath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for copy: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() { _ = d.sandbox.Remove(tmpName) }()

	_, err = io.Copy(tmpFile, srcFile)
	if closeErr := tmpFile.Close(); closeErr != nil && err == nil {
		err = fmt.Errorf("failed to close temporary file: %w", closeErr)
	}
	if err != nil {
		return fmt.Errorf("failed to copy data to temporary file '%s': %w", tmpName, err)
	}

	if err := d.sandbox.Rename(tmpName, dstPath); err != nil {
		return fmt.Errorf("failed to atomically move file to '%s': %w", dstPath, err)
	}
	return nil
}

// readAndValidateCache attempts to read a hash from a sidecar file and
// validates its freshness.
//
// Takes objectPath (string) which is the path to the main object file.
// Takes sidecarPath (string) which is the path to the cache sidecar file.
//
// Returns string which is the cached MD5 hash if valid.
// Returns error when the cache is stale, inaccessible, or has invalid format.
func (d *DiskProvider) readAndValidateCache(ctx context.Context, objectPath, sidecarPath string) (string, error) {
	_, l := logger_domain.From(ctx, log)
	sidecarInfo, err := d.sandbox.Stat(sidecarPath)
	if err != nil {
		return "", fmt.Errorf("cache sidecar not found or inaccessible: %w", err)
	}

	objectInfo, err := d.sandbox.Stat(objectPath)
	if err != nil {
		return "", fmt.Errorf("object file not found or inaccessible: %w", err)
	}

	if objectInfo.ModTime().After(sidecarInfo.ModTime()) {
		l.Trace("MD5 cache is stale, recomputing hash", logger_domain.String(logFieldPath, objectPath))
		return "", errors.New("stale cache: object file modified after cache was written")
	}

	data, _, err := d.sandbox.ReadFileLimit(sidecarPath, defaultMaxSidecarBytes)
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			return "", fmt.Errorf("%w: hash sidecar %q: %w",
				storage_domain.ErrDiskObjectTooLarge, sidecarPath, err)
		}
		return "", fmt.Errorf("failed to read cache sidecar: %w", err)
	}

	hash := string(data)
	for len(hash) > 0 && (hash[len(hash)-1] == '\n' || hash[len(hash)-1] == '\r' || hash[len(hash)-1] == ' ') {
		hash = hash[:len(hash)-1]
	}

	if len(hash) != md5HashHexLength {
		return "", fmt.Errorf("invalid hash format in cache file: len is %d", len(hash))
	}

	l.Trace("MD5 cache hit", logger_domain.String(logFieldPath, objectPath))
	return hash, nil
}

// rangeReadCloser wraps an io.Reader with a Close method.
// It implements io.ReadCloser for range-limited reads.
type rangeReadCloser struct {
	// reader is the underlying data source for read operations.
	reader io.Reader

	// closer releases the underlying resource when reading is done.
	closer io.Closer
}

// Read reads up to len(p) bytes from the range-limited reader.
//
// Takes p ([]byte) which is the buffer to read into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) which is nil, io.EOF, or a read error.
func (r *rangeReadCloser) Read(p []byte) (n int, err error) { return r.reader.Read(p) }

// Close releases the resources held by the underlying reader.
//
// Returns error when the underlying closer fails.
func (r *rangeReadCloser) Close() error { return r.closer.Close() }

// computeAndCacheHash reads the object file to compute its SHA256 hash and
// writes it to a sidecar file.
//
// Takes objectPath (string) which specifies the path to the object file.
// Takes sidecarPath (string) which specifies the path for the hash cache file.
//
// Returns string which is the hex-encoded SHA256 hash of the object file.
// Returns error when the object file cannot be opened or read.
func (d *DiskProvider) computeAndCacheHash(ctx context.Context, objectPath, sidecarPath string) (string, error) {
	_, l := logger_domain.From(ctx, log)
	file, err := d.sandbox.Open(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("object not found at path '%s': %w", objectPath, err)
		}
		return "", fmt.Errorf("failed to open file for hashing '%s': %w", objectPath, err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file for hashing '%s': %w", objectPath, err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	if err := d.sandbox.WriteFile(sidecarPath, []byte(hash), filePermissions); err != nil {
		l.Warn("Failed to write hash sidecar cache file", logger_domain.String(logFieldPath, sidecarPath), logger_domain.Error(err))
	}

	return hash, nil
}

// writeMetadataSidecar serialises and writes object metadata to a JSON
// sidecar file.
//
// Takes objectPath (string) which is the path to the associated object.
// Takes metadata (map[string]string) which contains the metadata to write.
func (d *DiskProvider) writeMetadataSidecar(ctx context.Context, objectPath string, metadata map[string]string) {
	_, l := logger_domain.From(ctx, log)
	if len(metadata) == 0 {
		return
	}
	metadataPath := objectPath + metadataSidecarSuffix
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		l.Warn("Failed to encode metadata for sidecar file", logger_domain.String(logFieldPath, metadataPath), logger_domain.Error(err))
		return
	}
	if err := d.sandbox.WriteFile(metadataPath, metadataJSON, filePermissions); err != nil {
		l.Warn("Failed to write metadata sidecar file", logger_domain.String(logFieldPath, metadataPath), logger_domain.Error(err))
	}
}

// readMetadataSidecar reads and deserialises object metadata from a JSON
// sidecar file. The read is bounded by defaultMaxSidecarBytes; a sidecar
// larger than the cap is rejected with storage_domain.ErrDiskObjectTooLarge
// so a corrupted or attacker-influenced sidecar cannot dominate memory.
//
// Takes objectPath (string) which is the path to the object file.
//
// Returns map[string]string which contains the deserialised metadata.
// Returns error when the sidecar file cannot be read or parsed.
func (d *DiskProvider) readMetadataSidecar(ctx context.Context, objectPath string) (map[string]string, error) {
	_, l := logger_domain.From(ctx, log)
	metadataPath := objectPath + metadataSidecarSuffix
	metadataData, _, err := d.sandbox.ReadFileLimit(metadataPath, defaultMaxSidecarBytes)
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			return nil, fmt.Errorf("%w: metadata sidecar %q: %w",
				storage_domain.ErrDiskObjectTooLarge, metadataPath, err)
		}
		return nil, fmt.Errorf("reading metadata sidecar %q: %w", metadataPath, err)
	}

	var metadata map[string]string
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		l.Warn("Failed to parse metadata sidecar file", logger_domain.String(logFieldPath, metadataPath), logger_domain.Error(err))
		return nil, fmt.Errorf("unmarshalling metadata sidecar %q: %w", metadataPath, err)
	}
	return metadata, nil
}

// removeSidecarFiles tries to remove known sidecar files for an object.
// Errors are ignored as this is a best-effort operation.
//
// Takes objectPath (string) which is the base path of the object whose
// sidecars should be removed.
func (d *DiskProvider) removeSidecarFiles(objectPath string) {
	_ = d.sandbox.Remove(objectPath + md5SidecarSuffix)
	_ = d.sandbox.Remove(objectPath + metadataSidecarSuffix)
}

// checkDiskSpace verifies available disk space and returns appropriate health
// status.
//
// Takes startTime (time.Time) which records when the check began for duration
// calculation.
//
// Returns healthprobe_dto.Status which indicates healthy, degraded, or the
// current disk space state.
func (d *DiskProvider) checkDiskSpace(startTime time.Time) healthprobe_dto.Status {
	rootPath := d.sandbox.Root()
	availableMB, _, err := getDiskSpace(rootPath)
	if err != nil {
		return healthprobe_dto.Status{
			Name:         d.Name(),
			State:        healthprobe_dto.StateHealthy,
			Message:      "Disk storage directory accessible",
			Timestamp:    time.Now(),
			Duration:     time.Since(startTime).String(),
			Dependencies: nil,
		}
	}

	if availableMB < minDiskSpaceMB {
		return healthprobe_dto.Status{
			Name:         d.Name(),
			State:        healthprobe_dto.StateDegraded,
			Message:      fmt.Sprintf("Low disk space: %d MB available", availableMB),
			Timestamp:    time.Now(),
			Duration:     time.Since(startTime).String(),
			Dependencies: nil,
		}
	}

	return healthprobe_dto.Status{
		Name:         d.Name(),
		State:        healthprobe_dto.StateHealthy,
		Message:      fmt.Sprintf("Disk storage operational, %d MB available", availableMB),
		Timestamp:    time.Now(),
		Duration:     time.Since(startTime).String(),
		Dependencies: nil,
	}
}

// NewDiskProvider creates a new disk-based storage adapter.
//
// It validates the configuration and creates the necessary directory
// structure exists.
//
// Takes config (Config) which specifies the storage configuration.
// Takes opts (...storage_domain.ProviderOption) which provides optional
// rate limiting settings.
//
// Returns storage_domain.StorageProviderPort which is the configured storage
// provider ready for use.
// Returns error when BaseDirectory and Sandbox are both empty, or when the
// sandbox cannot be created.
func NewDiskProvider(config Config, opts ...storage_domain.ProviderOption) (storage_domain.StorageProviderPort, error) {
	if config.BaseDirectory == "" && config.Sandbox == nil {
		return nil, errors.New("baseDirectory or sandbox must be provided for disk provider")
	}

	sandbox := config.Sandbox
	if sandbox == nil && config.SandboxFactory != nil {
		var err error
		sandbox, err = config.SandboxFactory.Create("storage-disk", config.BaseDirectory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox for storage via factory: %w", err)
		}
	}
	if sandbox == nil {
		var err error
		sandbox, err = safedisk.NewSandbox(config.BaseDirectory, safedisk.ModeReadWrite)
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox for storage: %w", err)
		}
	}

	defaultConfig := storage_domain.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
		Clock:          nil,
	}
	rateLimiter := storage_domain.ApplyProviderOptions(defaultConfig, opts...)

	d := &DiskProvider{
		sandbox:     sandbox,
		rateLimiter: rateLimiter,
	}

	return d, nil
}

// syncDirectoryAfterRename fsyncs the parent directory after a rename.
//
// Required on filesystems where the metadata journal is flushed
// independently of file data. Without this step, a crash between the
// rename and the next metadata flush can leave the file invisible after
// recovery despite a successful rename return.
//
// The fsync is best-effort: any error is logged but not surfaced because the
// data itself was already fsynced and callers cannot recover meaningfully.
//
// Takes ctx (context.Context) which carries the logger.
// Takes directory (string) which is the relative path to the directory inside
// the sandbox.
func (d *DiskProvider) syncDirectoryAfterRename(ctx context.Context, directory string) {
	if directory == "" {
		directory = "."
	}
	dirHandle, err := d.sandbox.OpenFile(directory, os.O_RDONLY, 0)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("opening directory for fsync skipped",
			logger_domain.String(logFieldPath, directory),
			logger_domain.Error(err))
		return
	}
	if syncErr := dirHandle.Sync(); syncErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("fsync of directory failed",
			logger_domain.String(logFieldPath, directory),
			logger_domain.Error(syncErr))
	}
	if closeErr := dirHandle.Close(); closeErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Trace("closing directory handle failed",
			logger_domain.String(logFieldPath, directory),
			logger_domain.Error(closeErr))
	}
}

// repoPath returns the relative path for a repository and key within the
// sandbox.
//
// Takes repo (string) which specifies the repository name.
// Takes key (string) which specifies the object key within the repository.
//
// Returns string which is the joined path of repo and key.
// Returns error when key is empty.
func repoPath(repo string, key string) (string, error) {
	if key == "" {
		return "", errors.New("object key cannot be empty")
	}
	return filepath.Join(repo, key), nil
}

// mimeTypeFromExtension guesses the MIME type from a file extension.
//
// Takes path (string) which is the file path to extract the extension from.
//
// Returns string which is the MIME type, or "application/octet-stream" if
// the type cannot be determined.
func mimeTypeFromExtension(path string) string {
	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
