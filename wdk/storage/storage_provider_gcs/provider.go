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

package storage_provider_gcs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	gcsstorage "cloud.google.com/go/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/api/option"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
)

const (
	// defaultCallsPerSecond is the default GCS rate limit in calls per second.
	defaultCallsPerSecond = 100.0

	// defaultBurst is the default burst size for rate limiting.
	defaultBurst = 200

	// largeFileThreshold is the file size above which chunked uploads are used.
	largeFileThreshold = 100 * 1024 * 1024

	// defaultMultipartChunk is the chunk size in bytes for multipart uploads.
	defaultMultipartChunk = 100 * 1024 * 1024

	// gcsMD5AttributeSize is the length in bytes of an MD5 hash from GCS.
	gcsMD5AttributeSize = 16

	// defaultDeleteConcurrency is the number of concurrent delete operations.
	defaultDeleteConcurrency = 10

	// defaultPutConcurrency is the number of concurrent uploads for PutMany.
	defaultPutConcurrency = 5

	// attributeKeyOperation is the metric attribute key that names the operation type.
	attributeKeyOperation = "operation"

	// attributeKeyName is the logging attribute key for object names.
	attributeKeyName = "key"

	// errRateLimiterWait is the error message format for rate limiter wait
	// failures.
	errRateLimiterWait = "rate limiter wait failed: %w"

	// operationPut is the operation name for object upload metrics.
	operationPut = "put"

	// operationGet is the operation name for object retrieval metrics.
	operationGet = "get"

	// operationStat is the operation name for getting object metadata.
	operationStat = "stat"

	// operationCopy is the metric attribute value for object copy operations.
	operationCopy = "copy"

	// operationRemove is the metric attribute value for object deletion
	// operations.
	operationRemove = "remove"

	// operationGetHash is the operation name for fetching object hashes.
	operationGetHash = "gethash"

	// operationRemoveMany is the metrics attribute value for batch delete
	// operations.
	operationRemoveMany = "remove_many"

	// operationPutMany is the metric attribute value for batch put operations.
	operationPutMany = "put_many"
)

// GCSProvider implements the StorageProviderPort interface using Google Cloud
// Storage.
type GCSProvider struct {
	// client provides access to Google Cloud Storage buckets and objects.
	client *gcsstorage.Client

	// repositoryBuckets maps repository names to their GCS bucket names.
	repositoryBuckets map[string]string

	// rateLimiter controls how often GCS operations can run.
	rateLimiter *storage.ProviderRateLimiter
}

var _ storage.ProviderPort = (*GCSProvider)(nil)

// Config holds the settings needed to create a GCSProvider.
type Config struct {
	// RepositoryMappings maps repository names to their storage bucket
	// identifiers.
	RepositoryMappings map[string]string

	// CredentialsJSON holds the GCP service account credentials in JSON format.
	CredentialsJSON []byte
}

// SupportsMultipart indicates whether multipart uploads are supported.
// The GCS client library handles multipart uploads transparently.
//
// Returns bool which is true as GCS supports multipart uploads.
func (*GCSProvider) SupportsMultipart() bool {
	return true
}

// Put uploads an object to GCS with automatic chunked upload for large files.
//
// Takes params (*storage.PutParams) which specifies the object key,
// content, metadata, and optional multipart configuration.
//
// Returns error when rate limiting fails, the repository bucket cannot be
// determined, data copying fails, or the writer cannot be closed.
func (g *GCSProvider) Put(ctx context.Context, params *storage.PutParams) error {
	ctx, l := logger.From(ctx, log)
	startTime := time.Now()

	if err := g.rateLimiter.Wait(ctx); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
		return fmt.Errorf(errRateLimiterWait, err)
	}

	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
		return fmt.Errorf("putting object to GCS: %w", err)
	}

	writer := g.client.Bucket(bucketName).Object(params.Key).NewWriter(ctx)
	writer.ContentType = params.ContentType
	if params.Size >= 0 {
		writer.Size = params.Size
	}

	if len(params.Metadata) > 0 {
		writer.Metadata = params.Metadata
	}

	if params.MultipartConfig != nil {
		writer.ChunkSize = int(params.MultipartConfig.PartSize)
		l.Trace("Using configured chunked upload for GCS",
			logger.String(attributeKeyName, params.Key),
			logger.Int("chunk_size", writer.ChunkSize))
	} else if params.Size > largeFileThreshold {
		writer.ChunkSize = defaultMultipartChunk
		l.Trace("Auto-enabling chunked upload for large file",
			logger.String(attributeKeyName, params.Key),
			logger.Int("chunk_size", writer.ChunkSize))
	}

	bytesWritten, err := io.Copy(writer, params.Reader)
	if err != nil {
		_ = writer.Close()
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
		return fmt.Errorf("failed to copy data to GCS object '%s': %w", params.Key, err)
	}

	if err := writer.Close(); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
		return fmt.Errorf("failed to close writer for GCS object '%s': %w", params.Key, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))
	BytesTransferred.Add(ctx, bytesWritten, metric.WithAttributes(attribute.String(attributeKeyOperation, operationPut)))

	return nil
}

// Get retrieves an object from GCS as a readable stream.
//
// Takes params (storage.GetParams) which specifies the bucket, key, and
// optional byte range.
//
// Returns io.ReadCloser which provides the object content as a stream.
// Returns error when the rate limiter times out, the repository bucket is not
// found, or the object does not exist.
func (g *GCSProvider) Get(ctx context.Context, params storage.GetParams) (io.ReadCloser, error) {
	startTime := time.Now()

	if err := g.rateLimiter.Wait(ctx); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGet)))
		return nil, fmt.Errorf(errRateLimiterWait, err)
	}

	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGet)))
		return nil, fmt.Errorf("reading GCS object: %w", err)
	}

	objHandle := g.client.Bucket(bucketName).Object(params.Key)
	var reader *gcsstorage.Reader

	if params.ByteRange != nil {
		reader, err = g.getRangeReader(ctx, objHandle, params)
	} else {
		reader, err = objHandle.NewReader(ctx)
	}

	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGet)))
		if errors.Is(err, gcsstorage.ErrObjectNotExist) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to get GCS object reader for '%s': %w", params.Key, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationGet)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGet)))

	return reader, nil
}

// Stat retrieves metadata for an object in GCS.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns *storage.ObjectInfo which contains the object's size, content
// type, last modified time, ETag, and metadata.
// Returns error when the object is not found or the API call fails.
func (g *GCSProvider) Stat(ctx context.Context, params storage.GetParams) (*storage.ObjectInfo, error) {
	startTime := time.Now()

	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationStat)))
		return nil, fmt.Errorf("statting GCS object: %w", err)
	}

	attrs, err := g.client.Bucket(bucketName).Object(params.Key).Attrs(ctx)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationStat)))
		if errors.Is(err, gcsstorage.ErrObjectNotExist) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to get GCS object attributes for '%s': %w", params.Key, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationStat)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationStat)))

	return &storage.ObjectInfo{
		Size:         attrs.Size,
		ContentType:  attrs.ContentType,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
		Metadata:     attrs.Metadata,
	}, nil
}

// Copy performs an efficient, server-side copy within the same bucket.
//
// Takes repo (string) which identifies the repository containing both
// objects.
// Takes srcKey (string) which specifies the source object key.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the copy operation fails.
func (g *GCSProvider) Copy(ctx context.Context, repo string, srcKey, dstKey string) error {
	startTime := time.Now()

	bucketName, err := g.getRepositoryBucket(repo)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
		return fmt.Errorf("copying GCS object: %w", err)
	}

	source := g.client.Bucket(bucketName).Object(srcKey)
	destination := g.client.Bucket(bucketName).Object(dstKey)

	if _, err := destination.CopierFrom(source).Run(ctx); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
		return fmt.Errorf("failed to copy GCS object from '%s' to '%s': %w", srcKey, dstKey, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))

	return nil
}

// CopyToAnotherRepository performs an efficient, server-side copy between
// different buckets.
//
// Takes srcRepo (string) which identifies the source repository.
// Takes srcKey (string) which specifies the source object key.
// Takes dstRepo (string) which identifies the destination repository.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the copy operation fails.
func (g *GCSProvider) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	startTime := time.Now()

	srcBucket, err := g.getRepositoryBucket(srcRepo)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
		return fmt.Errorf("copying GCS object cross-repository (source): %w", err)
	}
	dstBucket, err := g.getRepositoryBucket(dstRepo)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
		return fmt.Errorf("copying GCS object cross-repository (destination): %w", err)
	}

	source := g.client.Bucket(srcBucket).Object(srcKey)
	destination := g.client.Bucket(dstBucket).Object(dstKey)

	if _, err := destination.CopierFrom(source).Run(ctx); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
		return fmt.Errorf("failed to copy GCS object from '%s' to '%s': %w", srcKey, dstKey, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationCopy)))

	return nil
}

// Remove deletes an object from GCS. This operation is idempotent.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns error when rate limiting fails or the delete operation fails.
func (g *GCSProvider) Remove(ctx context.Context, params storage.GetParams) error {
	startTime := time.Now()

	if err := g.rateLimiter.Wait(ctx); err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationRemove)))
		return fmt.Errorf(errRateLimiterWait, err)
	}

	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationRemove)))
		return fmt.Errorf("removing GCS object: %w", err)
	}

	err = g.client.Bucket(bucketName).Object(params.Key).Delete(ctx)
	if err != nil && !errors.Is(err, gcsstorage.ErrObjectNotExist) {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationRemove)))
		return fmt.Errorf("failed to remove GCS object '%s': %w", params.Key, err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationRemove)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationRemove)))

	return nil
}

// Rename performs a copy-then-delete to simulate atomic rename.
//
// GCS does not have a native rename operation, so this uses CopierFrom
// followed by Delete. If the delete phase fails after a successful copy,
// the method logs a warning but does not return an error, which may leave
// an orphan object at the old key.
//
// Takes repo (string) which identifies the target repository.
// Takes oldKey (string) which specifies the current object key.
// Takes newKey (string) which specifies the destination object key.
//
// Returns error when the copy phase fails.
func (g *GCSProvider) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	ctx, l := logger.From(ctx, log)

	if err := g.Copy(ctx, repo, oldKey, newKey); err != nil {
		return fmt.Errorf("rename copy phase failed: %w", err)
	}

	params := storage.GetParams{
		Repository:      repo,
		Key:             oldKey,
		ByteRange:       nil,
		TransformConfig: nil,
	}
	if err := g.Remove(ctx, params); err != nil {
		l.Warn("Rename delete phase failed, orphan object may exist",
			logger.String("old_key", oldKey),
			logger.Error(err))
	}

	return nil
}

// Exists checks if an object exists in GCS.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns bool which is true if the object exists.
// Returns error when the rate limiter fails or the API call fails.
func (g *GCSProvider) Exists(ctx context.Context, params storage.GetParams) (bool, error) {
	if err := g.rateLimiter.Wait(ctx); err != nil {
		return false, fmt.Errorf(errRateLimiterWait, err)
	}

	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		return false, fmt.Errorf("checking GCS object existence: %w", err)
	}

	_, err = g.client.Bucket(bucketName).Object(params.Key).Attrs(ctx)
	if err != nil {
		if errors.Is(err, gcsstorage.ErrObjectNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of '%s': %w", params.Key, err)
	}

	return true, nil
}

// GetHash retrieves the object's MD5 hash from metadata or by streaming
// the content.
//
// Takes params (storage.GetParams) which specifies the object to hash.
//
// Returns string which is the hex-encoded MD5 hash of the object.
// Returns error when the object cannot be accessed or hashed.
func (g *GCSProvider) GetHash(ctx context.Context, params storage.GetParams) (string, error) {
	startTime := time.Now()

	attrs, err := g.Stat(ctx, params)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))
		return "", fmt.Errorf("getting GCS object hash: %w", err)
	}

	if len(attrs.ETag) == gcsMD5AttributeSize {
		duration := time.Since(startTime).Milliseconds()
		OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))
		OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))
		return hex.EncodeToString([]byte(attrs.ETag)), nil
	}

	hash, err := g.fallbackToStreamingHash(ctx, params)
	if err != nil {
		OperationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))
		return "", fmt.Errorf("computing GCS object hash via streaming: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()
	OperationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))
	OperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationGetHash)))

	return hash, nil
}

// PresignURL generates a signed URL for direct client-side uploads (HTTP PUT).
//
// Takes params (storage.PresignParams) which specifies the object key,
// repository, content type, and expiration duration.
//
// Returns string which is the signed URL for uploading.
// Returns error when URL generation fails.
func (g *GCSProvider) PresignURL(_ context.Context, params storage.PresignParams) (string, error) {
	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		return "", fmt.Errorf("presigning GCS upload URL: %w", err)
	}

	opts := &gcsstorage.SignedURLOptions{
		Method:  "PUT",
		Expires: time.Now().Add(params.ExpiresIn),
		Style:   gcsstorage.PathStyle(),
		Scheme:  gcsstorage.SigningSchemeV4,
	}
	if params.ContentType != "" {
		opts.Headers = []string{"Content-Type:" + params.ContentType}
	}

	url, err := g.client.Bucket(bucketName).SignedURL(params.Key, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for '%s': %w", params.Key, err)
	}
	return url, nil
}

// PresignDownloadURL generates a signed URL for direct client-side downloads
// (HTTP GET).
//
// Takes params (storage.PresignDownloadParams) which specifies the download
// details.
//
// Returns string which is the signed URL for downloading.
// Returns error when URL generation fails.
func (g *GCSProvider) PresignDownloadURL(_ context.Context, params storage.PresignDownloadParams) (string, error) {
	bucketName, err := g.getRepositoryBucket(params.Repository)
	if err != nil {
		return "", fmt.Errorf("presigning GCS download URL: %w", err)
	}

	opts := &gcsstorage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(params.ExpiresIn),
		Style:   gcsstorage.PathStyle(),
		Scheme:  gcsstorage.SigningSchemeV4,
	}

	if params.FileName != "" || params.ContentType != "" {
		queryParams := make(map[string][]string)
		if params.FileName != "" {
			queryParams["response-content-disposition"] = []string{fmt.Sprintf("attachment; filename=%q", params.FileName)}
		}
		if params.ContentType != "" {
			queryParams["response-content-type"] = []string{params.ContentType}
		}
		opts.QueryParameters = queryParams
	}

	url, err := g.client.Bucket(bucketName).SignedURL(params.Key, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed download URL for '%s': %w", params.Key, err)
	}
	return url, nil
}

// SupportsPresignedURLs reports whether the GCS provider supports native
// presigned URLs.
//
// Returns bool which is always true as GCS supports signed URLs natively.
func (*GCSProvider) SupportsPresignedURLs() bool {
	return true
}

// Close releases the GCS client connection.
//
// Returns error when the underlying GCS client fails to close.
func (g *GCSProvider) Close(ctx context.Context) error {
	_, l := logger.From(ctx, log)
	l.Internal("Closing GCS client")
	return g.client.Close()
}

// SupportsBatchOperations returns true because this provider offers an
// optimised concurrent implementation for batch operations, which is more
// efficient than the service's sequential fallback.
//
// Returns bool which is true when batch operations are supported.
func (*GCSProvider) SupportsBatchOperations() bool {
	return true
}

// SupportsRetry returns false; the service layer should handle retries.
//
// Returns bool which is always false for this provider.
func (*GCSProvider) SupportsRetry() bool {
	return false
}

// SupportsCircuitBreaking returns false; the service layer handles circuit
// breaking.
//
// Returns bool which indicates whether this provider supports circuit breaking.
func (*GCSProvider) SupportsCircuitBreaking() bool {
	return false
}

// SupportsRateLimiting reports whether GCS handles rate limiting.
//
// Returns false; rate limiting is handled at the service layer.
func (*GCSProvider) SupportsRateLimiting() bool {
	return false
}

// executeBatchOperation handles the shared scaffolding for batch GCS operations:
// empty-input short-circuit, concurrency defaulting, timing, worker dispatch,
// result collection, metrics recording, and logging.
//
// Takes itemCount (int) which is the number of items to process.
// Takes givenConcurrency (int) which is the caller-supplied concurrency value;
// values <= 0 are replaced by defaultConcurrency.
// Takes defaultConcurrency (int) which is the fallback concurrency level.
// Takes operationName (string) which is the metric attribute value for this
// operation.
// Takes logOperationName (string) which is the human-readable operation name
// used in log output.
// Takes runWorkers (func) which starts the operation-specific worker pool and
// returns the collected batch result.
//
// Returns *storage.BatchResult which contains the outcome of all operations.
// Returns error which is always nil; the signature is kept to satisfy the
// batch operation interface.
func executeBatchOperation(
	ctx context.Context,
	itemCount int,
	givenConcurrency int,
	defaultConcurrency int,
	operationName string,
	logOperationName string,
	runWorkers func(ctx context.Context, concurrency int) *storage.BatchResult,
) (*storage.BatchResult, error) {
	if itemCount == 0 {
		return &storage.BatchResult{
			TotalRequested:  0,
			SuccessfulKeys:  nil,
			FailedKeys:      nil,
			TotalSuccessful: 0,
			TotalFailed:     0,
			ProcessingTime:  0,
		}, nil
	}

	concurrency := givenConcurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	startTime := time.Now()
	result := runWorkers(ctx, concurrency)
	result.ProcessingTime = time.Since(startTime)

	BatchOperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(attributeKeyOperation, operationName)))
	OperationDuration.Record(ctx, float64(result.ProcessingTime.Milliseconds()), metric.WithAttributes(attribute.String(attributeKeyOperation, operationName)))
	OperationsTotal.Add(ctx, int64(len(result.SuccessfulKeys)), metric.WithAttributes(attribute.String(attributeKeyOperation, operationName)))
	if len(result.FailedKeys) > 0 {
		OperationErrorsTotal.Add(ctx, int64(len(result.FailedKeys)), metric.WithAttributes(attribute.String(attributeKeyOperation, operationName)))
	}

	logBatchResult(ctx, logOperationName, "gcs", concurrency, result)
	return result, nil
}

// RemoveMany implements batch delete for GCS.
//
// Takes params (storage.RemoveManyParams) which specifies the keys to delete
// and concurrency settings.
//
// Returns *storage.BatchResult which contains the outcome of each delete
// operation.
// Returns error when the batch operation fails.
func (g *GCSProvider) RemoveMany(ctx context.Context, params storage.RemoveManyParams) (*storage.BatchResult, error) {
	return executeBatchOperation(
		ctx,
		len(params.Keys),
		params.Concurrency,
		defaultDeleteConcurrency,
		operationRemoveMany,
		"delete",
		func(ctx context.Context, concurrency int) *storage.BatchResult {
			jobs := queueDeleteJobs(params.Keys)
			results := make(chan deleteJobResult, len(params.Keys))
			startDeleteWorkers(ctx, concurrency, jobs, results, g, params)
			return collectBatchResults(results, len(params.Keys))
		},
	)
}

// PutMany implements batch upload for GCS.
//
// Takes params (*storage.PutManyParams) which specifies the objects to upload
// and concurrency settings.
//
// Returns *storage.BatchResult which contains the outcome of all uploads.
// Returns error when the batch operation fails.
func (g *GCSProvider) PutMany(ctx context.Context, params *storage.PutManyParams) (*storage.BatchResult, error) {
	return executeBatchOperation(
		ctx,
		len(params.Objects),
		params.Concurrency,
		defaultPutConcurrency,
		operationPutMany,
		"upload",
		func(ctx context.Context, concurrency int) *storage.BatchResult {
			jobs := queueUploadJobs(params.Objects)
			results := make(chan uploadJobResult, len(params.Objects))
			startUploadWorkers(ctx, concurrency, jobs, results, g, params)
			return collectBatchResults(results, len(params.Objects))
		},
	)
}

// getRepositoryBucket safely retrieves the bucket name for a given repository.
//
// Takes repo (string) which is the repository name to look up.
//
// Returns string which is the bucket name mapped to the repository.
// Returns error when no bucket mapping exists for the given repository.
func (g *GCSProvider) getRepositoryBucket(repo string) (string, error) {
	bucket, ok := g.repositoryBuckets[repo]
	if !ok {
		return "", fmt.Errorf("no bucket mapping found for repository: %s", repo)
	}
	return bucket, nil
}

// getRangeReader creates and returns a reader for a specific byte range.
//
// Takes objHandle (*gcsstorage.ObjectHandle) which references the GCS object.
// Takes params (storage.GetParams) which specifies the byte range to read.
//
// Returns *gcsstorage.Reader which provides access to the requested byte range.
// Returns error when the range reader cannot be created.
func (*GCSProvider) getRangeReader(ctx context.Context, objHandle *gcsstorage.ObjectHandle, params storage.GetParams) (*gcsstorage.Reader, error) {
	_, l := logger.From(ctx, log)

	offset := params.ByteRange.Start
	length := int64(-1)

	if params.ByteRange.End != -1 {
		length = params.ByteRange.End - params.ByteRange.Start + 1
	}

	l.Trace("GCS range request",
		logger.String(attributeKeyName, params.Key),
		logger.Int64("offset", offset),
		logger.Int64("length", length))

	return objHandle.NewRangeReader(ctx, offset, length)
}

// fallbackToStreamingHash computes the hash by streaming the entire object.
//
// Takes params (storage.GetParams) which specifies the object to hash.
//
// Returns string which is the hex-encoded SHA256 hash of the object.
// Returns error when the object cannot be read or hashed.
func (g *GCSProvider) fallbackToStreamingHash(ctx context.Context, params storage.GetParams) (string, error) {
	ctx, l := logger.From(ctx, log)
	l.Warn("MD5 not available in GCS metadata, streaming object to compute hash", logger.String(attributeKeyName, params.Key))

	reader, err := g.Get(ctx, params)
	if err != nil {
		return "", fmt.Errorf("streaming GCS object for hash computation: %w", err)
	}
	defer func() { _ = reader.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", fmt.Errorf("failed to read GCS object data for hashing: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

type (
	// deleteJob represents a request to remove an object from
	// storage, identified by key.
	deleteJob struct {
		// key is the object identifier to delete.
		key string
	}

	// uploadJob represents a request to upload an object to
	// storage, carrying the file metadata and content spec.
	uploadJob struct {
		// spec holds the upload metadata and content.
		spec storage.PutObjectSpec
	}

	// jobResult represents the outcome of an upload or delete operation.
	jobResult interface {
		// Key returns the unique identifier for this configuration entry.
		Key() string
		// Error returns the underlying error that caused this failure.
		Error() error
	}

	// deleteJobResult holds the outcome of a GCS object deletion job.
	deleteJobResult struct {
		// err holds any error from the delete operation.
		err error
		// key is the object key that was deleted.
		key string
	}

	// uploadJobResult holds the outcome of an upload operation.
	uploadJobResult struct {
		// err holds any error that occurred during the upload.
		err error
		// key is the unique identifier for this upload job result.
		key string
	}
)

// Key returns the unique identifier for this delete job result.
//
// Returns string which is the key that identifies this result.
func (r deleteJobResult) Key() string { return r.key }

// Error returns the error for the delete job result.
//
// Returns error when the delete operation failed.
func (r deleteJobResult) Error() error { return r.err }

// Key returns the key for the upload job result.
//
// Returns string which is the unique identifier for this result.
func (r uploadJobResult) Key() string { return r.key }

// Error returns the error for the upload job result.
//
// Returns error when the upload job failed.
func (r uploadJobResult) Error() error { return r.err }

// NewGCSProvider creates a new Google Cloud Storage adapter.
//
// Takes config (Config) which specifies the GCS credentials and bucket
// mappings.
// Takes opts (...storage.ProviderOption) which provides optional rate
// limiting settings.
//
// Returns storage.ProviderPort which is the configured GCS
// provider ready for use.
// Returns error when the GCS client cannot be created.
func NewGCSProvider(ctx context.Context, config Config, opts ...storage.ProviderOption) (storage.ProviderPort, error) {
	client, err := createGCSClient(ctx, config.CredentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("initialising GCS provider: %w", err)
	}

	defaultConfig := storage.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := storage.ApplyProviderOptions(defaultConfig, opts...)

	return &GCSProvider{
		client:            client,
		repositoryBuckets: config.RepositoryMappings,
		rateLimiter:       rateLimiter,
	}, nil
}

// createGCSClient initialises the connection to Google Cloud Storage.
//
// Takes credentialsJSON ([]byte) which provides the service account
// credentials.
// When empty, the client uses default application credentials.
//
// Returns *gcsstorage.Client which is the configured GCS client ready for use.
// Returns error when the storage client cannot be created.
func createGCSClient(ctx context.Context, credentialsJSON []byte) (*gcsstorage.Client, error) {
	var clientOpts []option.ClientOption
	if len(credentialsJSON) > 0 {
		clientOpts = append(clientOpts, option.WithAuthCredentialsJSON(option.ServiceAccount, credentialsJSON))
	}

	client, err := gcsstorage.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Storage client: %w", err)
	}
	return client, nil
}

// queueDeleteJobs creates a buffered channel of delete jobs from the given
// keys.
//
// Takes keys ([]string) which specifies the keys to delete.
//
// Returns <-chan deleteJob which yields one job per key. The channel is
// returned already closed.
func queueDeleteJobs(keys []string) <-chan deleteJob {
	jobs := make(chan deleteJob, len(keys))
	for _, key := range keys {
		jobs <- deleteJob{key: key}
	}
	close(jobs)
	return jobs
}

// startDeleteWorkers spawns workers simultaneously to consume
// delete jobs and send results to the results channel.
//
// Takes ctx (context.Context) which controls cancellation of
// all workers.
// Takes concurrency (int) which sets the number of workers.
// Takes jobs (<-chan deleteJob) which provides delete jobs for
// workers.
// Takes results (chan<- deleteJobResult) which receives
// deletion outcomes.
// Takes g (*GCSProvider) which performs the actual GCS
// deletions.
// Takes params (storage.RemoveManyParams) which provides
// repository and error handling settings.
//
// Safe for concurrent use. Spawns worker goroutines tracked by a
// WaitGroup; closes the results channel when all workers finish.
func startDeleteWorkers(ctx context.Context, concurrency int, jobs <-chan deleteJob, results chan<- deleteJobResult, g *GCSProvider, params storage.RemoveManyParams) {
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			runDeleteWorker(ctx, jobs, results, g, params)
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
}

// runDeleteWorker consumes delete jobs and publishes results until either the
// jobs channel closes or ctx is cancelled.
//
// When a job returns an error and ContinueOnError is false, the worker drains
// the remaining jobs so the producer is not left blocked, then exits.
//
// Takes ctx (context.Context) which controls cancellation of the worker loop.
// Takes jobs (<-chan deleteJob) which provides delete jobs for the worker to
// consume.
// Takes results (chan<- deleteJobResult) which receives the outcome of each
// delete attempt.
// Takes g (*GCSProvider) which performs the underlying GCS delete operation.
// Takes params (storage.RemoveManyParams) which supplies repository and error
// handling settings.
func runDeleteWorker(ctx context.Context, jobs <-chan deleteJob, results chan<- deleteJobResult, g *GCSProvider, params storage.RemoveManyParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if !processDeleteJob(ctx, job, jobs, results, g, params) {
				return
			}
		}
	}
}

// processDeleteJob runs a single delete and forwards its result.
//
// Takes ctx (context.Context) which controls cancellation of the operation.
// Takes job (deleteJob) which is the delete job to execute.
// Takes jobs (<-chan deleteJob) which is drained on a fatal error when
// ContinueOnError is false.
// Takes results (chan<- deleteJobResult) which receives the outcome of the
// delete attempt.
// Takes g (*GCSProvider) which performs the underlying GCS delete operation.
// Takes params (storage.RemoveManyParams) which supplies repository and error
// handling settings.
//
// Returns bool which is true to keep the worker running, or false to stop
// (because of ctx cancellation or a fatal error when ContinueOnError is false).
func processDeleteJob(ctx context.Context, job deleteJob, jobs <-chan deleteJob, results chan<- deleteJobResult, g *GCSProvider, params storage.RemoveManyParams) bool {
	err := g.Remove(ctx, storage.GetParams{
		Repository:      params.Repository,
		Key:             job.key,
		ByteRange:       nil,
		TransformConfig: nil,
	})
	select {
	case results <- deleteJobResult{key: job.key, err: err}:
	case <-ctx.Done():
		return false
	}
	if err != nil && !params.ContinueOnError {
		discardRemaining(jobs)
		return false
	}
	return true
}

// discardRemaining drains the jobs channel without dispatching the entries.
//
// The worker calls this on a fatal error when ContinueOnError is false so
// the producer can finish closing the channel without blocking.
//
// Takes ch (<-chan T) which is the channel to drain.
func discardRemaining[T any](ch <-chan T) {
	for range ch {
		_ = 1
	}
}

// queueUploadJobs creates a buffered channel of upload jobs from the given
// object specifications.
//
// Takes objects ([]storage.PutObjectSpec) which specifies the objects to
// upload.
//
// Returns <-chan uploadJob which yields one job per object specification. The
// channel is closed before returning, so all jobs are immediately available.
func queueUploadJobs(objects []storage.PutObjectSpec) <-chan uploadJob {
	jobs := make(chan uploadJob, len(objects))
	for _, storageObject := range objects {
		jobs <- uploadJob{spec: storageObject}
	}
	close(jobs)
	return jobs
}

// startUploadWorkers spawns workers simultaneously to consume
// upload jobs and send results to the results channel.
//
// Takes ctx (context.Context) which controls cancellation of
// all workers.
// Takes concurrency (int) which sets the number of workers.
// Takes jobs (<-chan uploadJob) which provides upload jobs for
// workers.
// Takes results (chan<- uploadJobResult) which receives upload
// outcomes.
// Takes g (*GCSProvider) which performs the actual GCS uploads.
// Takes params (*storage.PutManyParams) which provides
// repository and error handling settings.
//
// Safe for concurrent use. Spawns worker goroutines tracked by a
// WaitGroup; closes the results channel when all workers finish.
func startUploadWorkers(ctx context.Context, concurrency int, jobs <-chan uploadJob, results chan<- uploadJobResult, g *GCSProvider, params *storage.PutManyParams) {
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			runUploadWorker(ctx, jobs, results, g, params)
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
}

// runUploadWorker consumes upload jobs and publishes results until either the
// jobs channel closes or ctx is cancelled.
//
// Takes ctx (context.Context) which controls cancellation of the worker loop.
// Takes jobs (<-chan uploadJob) which provides upload jobs for the worker to
// consume.
// Takes results (chan<- uploadJobResult) which receives the outcome of each
// upload attempt.
// Takes g (*GCSProvider) which performs the underlying GCS upload operation.
// Takes params (*storage.PutManyParams) which supplies repository and error
// handling settings.
func runUploadWorker(ctx context.Context, jobs <-chan uploadJob, results chan<- uploadJobResult, g *GCSProvider, params *storage.PutManyParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if !processUploadJob(ctx, job, jobs, results, g, params) {
				return
			}
		}
	}
}

// processUploadJob runs a single put and forwards its result.
//
// Takes ctx (context.Context) which controls cancellation of the operation.
// Takes job (uploadJob) which is the upload job to execute.
// Takes jobs (<-chan uploadJob) which is drained on a fatal error when
// ContinueOnError is false.
// Takes results (chan<- uploadJobResult) which receives the outcome of the
// upload attempt.
// Takes g (*GCSProvider) which performs the underlying GCS put operation.
// Takes params (*storage.PutManyParams) which supplies repository and error
// handling settings.
//
// Returns bool which is true to keep the worker running, or false to stop
// (because of ctx cancellation or a fatal error when ContinueOnError is false).
func processUploadJob(ctx context.Context, job uploadJob, jobs <-chan uploadJob, results chan<- uploadJobResult, g *GCSProvider, params *storage.PutManyParams) bool {
	putParam := &storage.PutParams{
		Repository:           params.Repository,
		Key:                  job.spec.Key,
		Reader:               job.spec.Reader,
		Size:                 job.spec.Size,
		ContentType:          job.spec.ContentType,
		TransformConfig:      params.TransformConfig,
		MultipartConfig:      nil,
		Metadata:             nil,
		HashAlgorithm:        "",
		ExpectedHash:         "",
		UseContentAddressing: false,
	}
	err := g.Put(ctx, putParam)
	select {
	case results <- uploadJobResult{key: job.spec.Key, err: err}:
	case <-ctx.Done():
		return false
	}
	if err != nil && !params.ContinueOnError {
		discardRemaining(jobs)
		return false
	}
	return true
}

// collectBatchResults gathers job results from a channel into a batch result.
//
// Takes resultsChan (<-chan T) which yields job results to gather.
// Takes total (int) which specifies the total number of requested items.
//
// Returns *storage.BatchResult which contains the gathered success and failure
// counts along with their keys.
func collectBatchResults[T jobResult](resultsChan <-chan T, total int) *storage.BatchResult {
	result := &storage.BatchResult{
		TotalRequested:  total,
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}
	for batchRes := range resultsChan {
		if batchRes.Error() == nil {
			result.SuccessfulKeys = append(result.SuccessfulKeys, batchRes.Key())
			result.TotalSuccessful++
		} else {
			result.FailedKeys = append(result.FailedKeys, storage.BatchFailure{
				Key:       batchRes.Key(),
				Error:     batchRes.Error().Error(),
				ErrorCode: "",
				Retryable: storage.IsRetryableError(batchRes.Error()),
			})
			result.TotalFailed++
		}
	}
	return result
}

// logBatchResult logs the result of a batch operation at trace level.
//
// Takes ctx (context.Context) which carries the logger.
// Takes op (string) which names the batch operation.
// Takes provider (string) which identifies the storage provider.
// Takes concurrency (int) which is the level of parallelism used.
// Takes result (*storage.BatchResult) which contains the batch statistics.
func logBatchResult(ctx context.Context, op, provider string, concurrency int, result *storage.BatchResult) {
	_, l := logger.From(ctx, log)
	l.Trace("Batch operation completed",
		logger.String("operation", op),
		logger.String("provider", provider),
		logger.Int("total", result.TotalRequested),
		logger.Int("successful", result.TotalSuccessful),
		logger.Int("failed", result.TotalFailed),
		logger.Int("concurrency", concurrency),
		logger.Duration("duration", result.ProcessingTime))
}
