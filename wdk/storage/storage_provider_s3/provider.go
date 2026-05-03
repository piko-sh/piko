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

package storage_provider_s3

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 for ETag, not cryptographic
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
)

const (
	// defaultMultipartThreshold is the file size in bytes above which multipart
	// uploads are used.
	defaultMultipartThreshold = 100 * 1024 * 1024

	// minPartSize is the smallest part size that S3 allows (5 MB).
	minPartSize = 5 * 1024 * 1024

	// md5HexLength is the number of characters in an MD5 hash in hexadecimal form.
	md5HexLength = 32

	// defaultCallsPerSecond is the default AWS S3 rate limit in requests per
	// second.
	defaultCallsPerSecond = 100.0

	// defaultBurst is the maximum number of requests allowed in a single burst.
	defaultBurst = 200

	// s3MaxKeysPerDelete is the most keys allowed in one S3 DeleteObjects call.
	s3MaxKeysPerDelete = 1000

	// defaultPutConcurrency is the number of concurrent uploads used when no
	// value is specified.
	defaultPutConcurrency = 5

	// errMessageRateLimiterWait is the format string for errors when the rate
	// limiter wait fails.
	errMessageRateLimiterWait = "rate limiter wait failed: %w"

	// attributeKeyName is the attribute key for logging S3 object key names.
	attributeKeyName = "key"
)

// errS3ProviderClosed is returned by Close when the provider has already been
// shut down.
var errS3ProviderClosed = errors.New("s3 provider already closed")

var _ storage.ProviderPort = (*S3Provider)(nil)

// S3Provider implements the StorageProviderPort interface using the AWS SDK for
// Go V2.
type S3Provider struct {
	// client is the AWS S3 client for storing and fetching objects.
	client *s3.Client

	// presignClient creates pre-signed URLs for S3 operations.
	presignClient *s3.PresignClient

	// rateLimiter controls how often S3 API requests are made.
	rateLimiter *storage.ProviderRateLimiter

	// repositoryBuckets maps repository names to their S3 bucket names.
	repositoryBuckets map[string]string

	// closed is set by Close to make subsequent calls a no-op that returns
	// errS3ProviderClosed.
	closed atomic.Bool
}

// Config holds the settings needed to create an S3Provider.
type Config struct {
	// RepositoryMappings maps repository names to their storage bucket names.
	RepositoryMappings map[string]string

	// Region is the AWS region for the service (e.g. "eu-west-1").
	Region string

	// AccessKey is the AWS access key ID for static credentials.
	AccessKey string

	// SecretKey is the AWS secret access key for static credentials.
	SecretKey string

	// EndpointURL is an optional custom S3 endpoint for S3-compatible services.
	EndpointURL string

	// UsePathStyle enables path-style S3 addressing instead of
	// virtual-hosted-style.
	UsePathStyle bool

	// DisableChecksum skips checksum checks for S3 requests and responses.
	DisableChecksum bool
}

// SupportsMultipart reports whether the S3 provider supports multipart uploads.
//
// Returns bool which is always true as S3 supports multipart uploads.
func (*S3Provider) SupportsMultipart() bool {
	return true
}

// SupportsBatchOperations reports whether the S3 provider supports batch
// operations.
//
// Returns bool which is true as S3 supports batch operations.
func (*S3Provider) SupportsBatchOperations() bool {
	return true
}

// SupportsRetry indicates whether the provider supports retry operations.
//
// Returns bool which is always true as retries are handled by the AWS SDK.
func (*S3Provider) SupportsRetry() bool {
	return true
}

// SupportsCircuitBreaking returns false; circuit breaking is handled at the
// service layer.
//
// Returns bool which is always false for this provider.
func (*S3Provider) SupportsCircuitBreaking() bool {
	return false
}

// SupportsRateLimiting reports whether the S3 provider implements rate
// limiting.
//
// Returns bool which is always true for this provider.
func (*S3Provider) SupportsRateLimiting() bool {
	return true
}

// Put uploads content to S3, choosing between simple or streaming multipart
// upload based on file size and settings.
//
// Takes params (*storage.PutParams) which specifies the upload details.
//
// Returns error when rate limiting fails or the upload cannot complete.
func (p *S3Provider) Put(ctx context.Context, params *storage.PutParams) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return fmt.Errorf("putting object to S3: %w", err)
	}

	shouldUseMultipart := params.MultipartConfig != nil ||
		params.Size < 0 ||
		(params.Size >= 0 && params.Size > defaultMultipartThreshold)

	if shouldUseMultipart {
		return p.streamingMultipartUpload(ctx, bucketName, params)
	}
	return p.simpleUpload(ctx, bucketName, params)
}

// Get retrieves an object from S3 as a readable stream.
//
// Supports byte range requests for partial content retrieval.
//
// Takes params (storage.GetParams) which specifies the object key, repository,
// and optional byte range.
//
// Returns io.ReadCloser which streams the object content. The caller must close
// the reader when finished.
// Returns error when the object is not found or the S3 request fails.
func (p *S3Provider) Get(ctx context.Context, params storage.GetParams) (io.ReadCloser, error) {
	ctx, l := logger.From(ctx, log)

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return nil, fmt.Errorf("reading S3 object: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(params.Key),
	}

	if params.ByteRange != nil {
		rangeString := buildRangeHeader(params.ByteRange)
		input.Range = aws.String(rangeString)
		l.Trace("S3 range request", logger.String(attributeKeyName, params.Key), logger.String("range", rangeString))
	}

	output, err := p.client.GetObject(ctx, input)
	if err != nil {
		if isS3NotFoundError(err) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to get S3 object for key '%s': %w", params.Key, err)
	}
	return output.Body, nil
}

// Stat retrieves metadata for an object in S3 using a HeadObject call.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns *storage.ObjectInfo which contains the object's size, content
// type, last modified time, ETag, and metadata.
// Returns error when the object is not found or the S3 request fails.
func (p *S3Provider) Stat(ctx context.Context, params storage.GetParams) (*storage.ObjectInfo, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return nil, fmt.Errorf("statting S3 object: %w", err)
	}

	output, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(params.Key),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return nil, fmt.Errorf("object not found at key '%s': %w", params.Key, err)
		}
		return nil, fmt.Errorf("failed to stat S3 object for key '%s': %w", params.Key, err)
	}

	return &storage.ObjectInfo{
		Size:         aws.ToInt64(output.ContentLength),
		ContentType:  aws.ToString(output.ContentType),
		LastModified: aws.ToTime(output.LastModified),
		ETag:         aws.ToString(output.ETag),
		Metadata:     output.Metadata,
	}, nil
}

// Copy performs an efficient, server-side copy of an object within the same
// bucket.
//
// Takes repo (string) which identifies the repository containing both objects.
// Takes srcKey (string) which specifies the source object key.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the copy operation fails.
func (p *S3Provider) Copy(ctx context.Context, repo string, srcKey, dstKey string) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(repo)
	if err != nil {
		return fmt.Errorf("copying S3 object: %w", err)
	}

	_, err = p.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucketName),
		Key:        aws.String(dstKey),
		CopySource: aws.String(fmt.Sprintf("%s/%s", bucketName, srcKey)),
	})
	if err != nil {
		return fmt.Errorf("failed to copy S3 object from '%s' to '%s': %w", srcKey, dstKey, err)
	}
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
func (p *S3Provider) CopyToAnotherRepository(ctx context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf(errMessageRateLimiterWait, err)
	}

	srcBucket, err := p.getRepositoryBucket(srcRepo)
	if err != nil {
		return fmt.Errorf("copying S3 object cross-repository (source): %w", err)
	}
	dstBucket, err := p.getRepositoryBucket(dstRepo)
	if err != nil {
		return fmt.Errorf("copying S3 object cross-repository (destination): %w", err)
	}

	_, err = p.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(fmt.Sprintf("%s/%s", srcBucket, srcKey)),
	})
	if err != nil {
		return fmt.Errorf("failed to copy S3 object from '%s' to '%s': %w", srcKey, dstKey, err)
	}
	return nil
}

// Remove deletes an object from S3. This operation is idempotent.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns error when the delete operation fails.
func (p *S3Provider) Remove(ctx context.Context, params storage.GetParams) error {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return fmt.Errorf("removing S3 object: %w", err)
	}

	_, err = p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(params.Key),
	})
	if err != nil {
		return fmt.Errorf("failed to remove S3 object '%s': %w", params.Key, err)
	}
	return nil
}

// Rename performs a copy-then-delete to simulate atomic rename. S3 does not
// have a native rename operation, so this uses CopyObject followed by
// DeleteObject.
//
// Takes repo (string) which identifies the target repository.
// Takes oldKey (string) which specifies the source object key.
// Takes newKey (string) which specifies the destination object key.
//
// Returns error when the copy phase fails. If the delete phase fails, a
// warning is logged but no error is returned since the copy succeeded.
func (p *S3Provider) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	ctx, l := logger.From(ctx, log)

	if err := p.Copy(ctx, repo, oldKey, newKey); err != nil {
		return fmt.Errorf("rename copy phase failed: %w", err)
	}

	params := storage.GetParams{
		Repository:      repo,
		Key:             oldKey,
		ByteRange:       nil,
		TransformConfig: nil,
	}
	if err := p.Remove(ctx, params); err != nil {
		l.Warn("Rename delete phase failed, orphan object may exist",
			logger.String("old_key", oldKey),
			logger.Error(err))
	}

	return nil
}

// Exists checks if an object exists in S3 using HeadObject.
//
// Takes params (storage.GetParams) which specifies the object key and
// repository.
//
// Returns bool which is true if the object exists.
// Returns error when the rate limiter fails or the API call fails.
func (p *S3Provider) Exists(ctx context.Context, params storage.GetParams) (bool, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return false, fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return false, fmt.Errorf("checking S3 object existence: %w", err)
	}

	_, err = p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(params.Key),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of '%s': %w", params.Key, err)
	}

	return true, nil
}

// GetHash retrieves the MD5 hash of an object from S3 storage. It uses the
// ETag when available, or falls back to streaming the object content.
//
// Takes params (storage.GetParams) which specifies the object to retrieve.
//
// Returns string which is the MD5 hash of the object.
// Returns error when rate limiting fails or the object cannot be read.
func (p *S3Provider) GetHash(ctx context.Context, params storage.GetParams) (string, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf(errMessageRateLimiterWait, err)
	}

	info, err := p.Stat(ctx, params)
	if err != nil {
		return "", fmt.Errorf("getting S3 object hash: %w", err)
	}

	etag := strings.Trim(info.ETag, `"`)
	if isLikelyMD5(etag) {
		return etag, nil
	}

	return p.fallbackToStreamingHash(ctx, params, info.ETag)
}

// PresignURL generates a signed URL for direct client-side uploads (HTTP PUT).
//
// Takes params (storage.PresignParams) which specifies the object key,
// repository, content type, and expiration duration.
//
// Returns string which is the pre-signed URL for the upload.
// Returns error when URL generation fails.
func (p *S3Provider) PresignURL(ctx context.Context, params storage.PresignParams) (string, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return "", fmt.Errorf("presigning S3 upload URL: %w", err)
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(params.Key),
		ContentType: aws.String(params.ContentType),
	}

	request, err := p.presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = params.ExpiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for key '%s': %w", params.Key, err)
	}

	return request.URL, nil
}

// PresignDownloadURL generates a signed URL for direct client-side downloads
// (HTTP GET).
//
// Takes params (storage.PresignDownloadParams) which specifies the download
// details.
//
// Returns string which is the pre-signed URL for downloading.
// Returns error when URL generation fails.
func (p *S3Provider) PresignDownloadURL(ctx context.Context, params storage.PresignDownloadParams) (string, error) {
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return "", fmt.Errorf("presigning S3 download URL: %w", err)
	}

	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(params.Key),
	}

	if params.FileName != "" {
		disposition := fmt.Sprintf("attachment; filename=%q", params.FileName)
		getObjectInput.ResponseContentDisposition = aws.String(disposition)
	}
	if params.ContentType != "" {
		getObjectInput.ResponseContentType = aws.String(params.ContentType)
	}

	request, err := p.presignClient.PresignGetObject(ctx, getObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = params.ExpiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL for key '%s': %w", params.Key, err)
	}

	return request.URL, nil
}

// SupportsPresignedURLs reports whether the S3 provider supports presigned
// URLs.
//
// Returns bool which is always true as S3 supports presigned URLs natively.
func (*S3Provider) SupportsPresignedURLs() bool {
	return true
}

// Close marks the provider as shut down. The AWS SDK manages connection
// pools so there is nothing to release; the closed flag exists so callers
// can detect double-close.
//
// Takes ctx (context.Context) which is unused but required by the interface.
//
// Returns error which is errS3ProviderClosed when invoked more than once,
// otherwise nil.
func (p *S3Provider) Close(context.Context) error {
	if !p.closed.CompareAndSwap(false, true) {
		return errS3ProviderClosed
	}
	return nil
}

// RemoveMany implements native batch delete using S3's DeleteObjects API.
//
// Takes params (storage.RemoveManyParams) which specifies the keys to
// delete and error handling behaviour.
//
// Returns *storage.BatchResult which contains deletion outcomes for each
// key.
// Returns error when a batch request fails and ContinueOnError is false.
func (p *S3Provider) RemoveMany(ctx context.Context, params storage.RemoveManyParams) (*storage.BatchResult, error) {
	if len(params.Keys) == 0 {
		return &storage.BatchResult{
			TotalRequested:  0,
			SuccessfulKeys:  nil,
			FailedKeys:      nil,
			TotalSuccessful: 0,
			TotalFailed:     0,
			ProcessingTime:  0,
		}, nil
	}

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf(errMessageRateLimiterWait, err)
	}

	bucketName, err := p.getRepositoryBucket(params.Repository)
	if err != nil {
		return nil, fmt.Errorf("deleting S3 objects in batch: %w", err)
	}

	startTime := time.Now()
	result := &storage.BatchResult{
		TotalRequested:  len(params.Keys),
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}

	for i := 0; i < len(params.Keys); i += s3MaxKeysPerDelete {
		end := min(i+s3MaxKeysPerDelete, len(params.Keys))

		batchKeys := params.Keys[i:end]
		batchErr := p.deleteS3Batch(ctx, bucketName, batchKeys, result)

		if batchErr != nil && !params.ContinueOnError {
			result.ProcessingTime = time.Since(startTime)
			return result, fmt.Errorf("batch delete request failed, halting operation: %w", batchErr)
		}
	}

	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// PutMany uploads multiple objects to S3 in a batch with internal concurrency,
// since S3 has no native batch upload API. The context is threaded through to
// each Put call for cancellation support.
//
// Takes params (*storage.PutManyParams) which specifies the objects to upload
// and concurrency settings.
//
// Returns *storage.BatchResult which contains upload outcomes for each object.
// Returns error which is always nil; individual failures are recorded in the
// result.
func (p *S3Provider) PutMany(ctx context.Context, params *storage.PutManyParams) (*storage.BatchResult, error) {
	if len(params.Objects) == 0 {
		return &storage.BatchResult{
			TotalRequested:  0,
			SuccessfulKeys:  nil,
			FailedKeys:      nil,
			TotalSuccessful: 0,
			TotalFailed:     0,
			ProcessingTime:  0,
		}, nil
	}

	concurrency := params.Concurrency
	if concurrency <= 0 {
		concurrency = defaultPutConcurrency
	}

	startTime := time.Now()

	workerFunc := func(workerCtx context.Context, job storage.PutObjectSpec) (string, error) {
		putParam := &storage.PutParams{
			Repository:           params.Repository,
			Key:                  job.Key,
			Reader:               job.Reader,
			Size:                 job.Size,
			ContentType:          job.ContentType,
			TransformConfig:      params.TransformConfig,
			MultipartConfig:      nil,
			Metadata:             nil,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			UseContentAddressing: false,
		}
		return job.Key, p.Put(workerCtx, putParam)
	}

	result := runBatchWorkers(ctx, params.Objects, concurrency, params.ContinueOnError, workerFunc)
	result.ProcessingTime = time.Since(startTime)

	logBatchResult(ctx, "upload", "s3", concurrency, result)
	return result, nil
}

// getRepositoryBucket returns the bucket name for a given repository.
//
// Takes repo (string) which identifies the repository to look up.
//
// Returns string which is the bucket name for the repository.
// Returns error when no bucket mapping exists for the repository.
func (p *S3Provider) getRepositoryBucket(repo string) (string, error) {
	bucket, ok := p.repositoryBuckets[repo]
	if !ok {
		return "", fmt.Errorf("no bucket mapping found for repository: %s", repo)
	}
	return bucket, nil
}

// simpleUpload performs a standard single-request S3 PutObject operation.
//
// Takes bucketName (string) which specifies the target S3 bucket.
// Takes params (*storage.PutParams) which provides the object key, content,
// and optional metadata.
//
// Returns error when the upload fails.
func (p *S3Provider) simpleUpload(ctx context.Context, bucketName string, params *storage.PutParams) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(params.Key),
		Body:        params.Reader,
		ContentType: aws.String(params.ContentType),
	}

	if len(params.Metadata) > 0 {
		input.Metadata = params.Metadata
	}

	_, err := p.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3 for key '%s': %w", params.Key, err)
	}
	return nil
}

// streamingMultipartUpload uploads large files to S3 using the AWS SDK's
// high-level Uploader.
//
// Takes bucketName (string) which specifies the target S3 bucket.
// Takes params (*storage.PutParams) which contains the upload settings
// including the reader, key, content type, and multipart options.
//
// Returns error when the upload fails.
func (p *S3Provider) streamingMultipartUpload(ctx context.Context, bucketName string, params *storage.PutParams) error {
	ctx, l := logger.From(ctx, log)

	multipartConfig := params.MultipartConfig
	if multipartConfig == nil {
		multipartConfig = new(storage.DefaultMultipartConfig())
	}
	if multipartConfig.PartSize < minPartSize {
		multipartConfig.PartSize = minPartSize
	}

	l.Trace("Starting streaming multipart upload",
		logger.String(attributeKeyName, params.Key),
		logger.Int64("total_size", params.Size),
		logger.Int64("part_size", multipartConfig.PartSize),
		logger.Int("concurrency", multipartConfig.Concurrency))

	uploader := transfermanager.New(p.client, func(o *transfermanager.Options) {
		o.PartSizeBytes = multipartConfig.PartSize
		o.Concurrency = multipartConfig.Concurrency
	})

	input := &transfermanager.UploadObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(params.Key),
		Body:        params.Reader,
		ContentType: aws.String(params.ContentType),
	}

	if len(params.Metadata) > 0 {
		input.Metadata = params.Metadata
	}

	_, err := uploader.UploadObject(ctx, input)
	if err != nil {
		return fmt.Errorf("multipart upload failed for key '%s': %w", params.Key, err)
	}

	l.Trace("Multipart upload completed successfully", logger.String(attributeKeyName, params.Key))
	return nil
}

// fallbackToStreamingHash computes the hash by downloading the object.
//
// Takes params (storage.GetParams) which specifies the object to retrieve.
// Takes etag (string) which is the non-MD5 ETag that triggered the fallback.
//
// Returns string which is the computed MD5 hash as a hex string.
// Returns error when the object cannot be retrieved or read.
func (p *S3Provider) fallbackToStreamingHash(ctx context.Context, params storage.GetParams, etag string) (string, error) {
	ctx, l := logger.From(ctx, log)
	l.Warn("ETag is not a simple MD5, streaming object to compute hash",
		logger.String(attributeKeyName, params.Key),
		logger.String("etag", etag))

	body, err := p.Get(ctx, params)
	if err != nil {
		return "", fmt.Errorf("streaming S3 object for hash computation: %w", err)
	}
	defer func() { _ = body.Close() }()

	//nolint:gosec // MD5 for ETag, not cryptographic
	hasher := md5.New()
	if _, err := io.Copy(hasher, body); err != nil {
		return "", fmt.Errorf("failed to read S3 object for hashing: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// deleteS3Batch performs a single S3 DeleteObjects API call for a chunk of
// keys.
//
// Takes bucketName (string) which specifies the target S3 bucket.
// Takes keys ([]string) which contains the object keys to delete.
// Takes result (*storage.BatchResult) which accumulates deletion outcomes.
//
// Returns error when the entire API request fails.
func (p *S3Provider) deleteS3Batch(ctx context.Context, bucketName string, keys []string, result *storage.BatchResult) error {
	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(false)},
	}

	output, err := p.client.DeleteObjects(ctx, input)
	if err != nil {
		for _, key := range keys {
			result.FailedKeys = append(result.FailedKeys, storage.BatchFailure{
				Key:       key,
				Error:     err.Error(),
				ErrorCode: "",
				Retryable: storage.IsRetryableError(err),
			})
		}
		result.TotalFailed += len(keys)
		return fmt.Errorf("deleting S3 objects batch request: %w", err)
	}

	for _, deleted := range output.Deleted {
		result.SuccessfulKeys = append(result.SuccessfulKeys, aws.ToString(deleted.Key))
		result.TotalSuccessful++
	}
	for _, s3Err := range output.Errors {
		result.FailedKeys = append(result.FailedKeys, storage.BatchFailure{
			Key:       aws.ToString(s3Err.Key),
			Error:     aws.ToString(s3Err.Message),
			ErrorCode: aws.ToString(s3Err.Code),
			Retryable: isRetryableS3ErrorCode(aws.ToString(s3Err.Code)),
		})
		result.TotalFailed++
	}

	return nil
}

// batchJob represents a single item to be processed as part of a batch.
type batchJob[T any] struct {
	// Item is the value to be processed in this batch job.
	Item T
}

// batchResult holds the outcome of processing a single job.
type batchResult struct {
	// Err holds any error from the operation; nil means success.
	Err error

	// Key is the object key that was processed.
	Key string
}

// NewS3Provider creates a new S3-compatible storage adapter.
//
// Takes s3Config (*Config) which specifies the S3 connection settings and bucket
// mappings.
// Takes opts (...storage.ProviderOption) which configures rate limiting
// behaviour.
//
// Returns storage.ProviderPort which is the configured storage
// provider ready for use.
// Returns error when s3Config is nil or the AWS configuration cannot be loaded.
func NewS3Provider(ctx context.Context, s3Config *Config, opts ...storage.ProviderOption) (storage.ProviderPort, error) {
	if s3Config == nil {
		return nil, errors.New("config is required")
	}

	awsConfig, err := loadAWSConfig(ctx, s3Config)
	if err != nil {
		return nil, fmt.Errorf("initialising S3 provider: %w", err)
	}

	client := createS3Client(&awsConfig, s3Config.EndpointURL, s3Config.UsePathStyle)

	defaultConfig := storage.ProviderRateLimitConfig{
		CallsPerSecond: defaultCallsPerSecond,
		Burst:          defaultBurst,
	}
	rateLimiter := storage.ApplyProviderOptions(defaultConfig, opts...)

	return &S3Provider{
		client:            client,
		presignClient:     s3.NewPresignClient(client),
		repositoryBuckets: s3Config.RepositoryMappings,
		rateLimiter:       rateLimiter,
	}, nil
}

// isS3NotFoundError checks if an error from the AWS SDK is a not found type.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true if the error is NotFound or NoSuchKey.
func isS3NotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return true
		}
	}
	return false
}

// loadAWSConfig loads the AWS SDK settings from the given configuration.
//
// Takes s3Config (*Config) which provides the S3 settings including region,
// credentials, and checksum options.
//
// Returns aws.Config which contains the AWS SDK settings ready for use.
// Returns error when the AWS settings cannot be loaded.
func loadAWSConfig(ctx context.Context, s3Config *Config) (aws.Config, error) {
	var awsOpts []func(*config.LoadOptions) error

	if s3Config.Region != "" {
		awsOpts = append(awsOpts, config.WithRegion(s3Config.Region))
	}
	if s3Config.AccessKey != "" && s3Config.SecretKey != "" {
		awsOpts = append(awsOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3Config.AccessKey, s3Config.SecretKey, ""),
		))
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config for S3: %w", err)
	}
	if s3Config.DisableChecksum {
		awsConfig.RequestChecksumCalculation = 0
		awsConfig.ResponseChecksumValidation = 0
	}
	return awsConfig, nil
}

// createS3Client creates an S3 client with the given settings.
//
// Takes awsConfig (*aws.Config) which provides the base AWS settings.
// Takes endpointURL (string) which sets a custom endpoint, or empty for the
// default.
// Takes usePathStyle (bool) which enables path-style addressing when using a
// custom endpoint.
//
// Returns *s3.Client which is the configured S3 client with retry settings.
func createS3Client(awsConfig *aws.Config, endpointURL string, usePathStyle bool) *s3.Client {
	sdkRetryer := retry.NewStandard(func(o *retry.StandardOptions) {
		o.MaxAttempts = 2
	})

	return s3.NewFromConfig(*awsConfig, func(o *s3.Options) {
		if endpointURL != "" {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = usePathStyle
		}
		o.Retryer = sdkRetryer
	})
}

// buildRangeHeader builds an HTTP Range header string from a ByteRange.
//
// Takes byteRange (*storage.ByteRange) which specifies the byte range for the
// request.
//
// Returns string which is the formatted Range header value.
func buildRangeHeader(byteRange *storage.ByteRange) string {
	if byteRange.End == -1 {
		return fmt.Sprintf("bytes=%d-", byteRange.Start)
	}
	return fmt.Sprintf("bytes=%d-%d", byteRange.Start, byteRange.End)
}

// isLikelyMD5 checks whether a string looks like an MD5 hash.
//
// Takes s (string) which is the value to check.
//
// Returns bool which is true if the string has 32 hexadecimal characters.
func isLikelyMD5(s string) bool {
	if len(s) != md5HexLength {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-fA-F0-9]{32}$`, s)
	return matched
}

// isRetryableS3ErrorCode checks whether an S3 error code from a batch
// operation can be retried.
//
// Takes code (string) which is the S3 error code to check.
//
// Returns bool which is true if the error code may succeed on retry.
func isRetryableS3ErrorCode(code string) bool {
	switch code {
	case "InternalError", "ServiceUnavailable", "SlowDown", "RequestTimeout":
		return true
	default:
		return false
	}
}

// startBatchWorkers launches workers simultaneously to process
// jobs from the jobs channel.
//
// Takes ctx (context.Context) which is threaded to each worker for
// cancellation.
// Takes wg (*sync.WaitGroup) which tracks worker completion.
// Takes concurrency (int) which specifies the number of
// workers.
// Takes jobs (<-chan batchJob[T]) which provides jobs for
// workers to process.
// Takes results (chan<- batchResult) which receives processing
// outcomes.
// Takes continueOnError (bool) which controls whether workers
// stop on failure.
// Takes workerFunc (func(...)) which processes each job item.
//
// Safe for concurrent use. Spawns worker goroutines tracked by the
// provided WaitGroup.
func startBatchWorkers[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	concurrency int,
	jobs <-chan batchJob[T],
	results chan<- batchResult,
	continueOnError bool,
	workerFunc func(context.Context, T) (string, error),
) {
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			processBatchJobs(ctx, jobs, results, continueOnError, workerFunc)
		}()
	}
}

// processBatchJobs reads jobs from a channel and processes each one until the
// channel is closed or an error occurs (when continueOnError is false).
//
// Takes ctx (context.Context) which is passed to each worker function call.
// Takes jobs (<-chan batchJob[T]) which provides the jobs to process.
// Takes results (chan<- batchResult) which receives the processing results.
// Takes continueOnError (bool) which controls whether to stop on the first
// error.
// Takes workerFunc (func(context.Context, T) (string, error)) which processes
// each job item.
func processBatchJobs[T any](ctx context.Context, jobs <-chan batchJob[T], results chan<- batchResult, continueOnError bool, workerFunc func(context.Context, T) (string, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			key, err := workerFunc(ctx, job.Item)
			select {
			case results <- batchResult{Key: key, Err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil && !continueOnError {
				drainJobsChannel(jobs)
				return
			}
		}
	}
}

// drainJobsChannel discards remaining jobs to unblock senders.
//
// Takes jobs (<-chan batchJob[T]) which is the channel to drain.
func drainJobsChannel[T any](jobs <-chan batchJob[T]) {
	for range jobs {
		_ = 0
	}
}

// collectBatchResults gathers results from a channel into a batch result.
//
// Takes results (<-chan batchResult) which provides completed batch operation
// outcomes.
// Takes result (*storage.BatchResult) which collects the success and failure
// counts.
func collectBatchResults(results <-chan batchResult, result *storage.BatchResult) {
	for batchRes := range results {
		if batchRes.Err == nil {
			result.SuccessfulKeys = append(result.SuccessfulKeys, batchRes.Key)
			result.TotalSuccessful++
		} else {
			result.FailedKeys = append(result.FailedKeys, storage.BatchFailure{
				Key:       batchRes.Key,
				Error:     batchRes.Err.Error(),
				ErrorCode: "",
				Retryable: storage.IsRetryableError(batchRes.Err),
			})
			result.TotalFailed++
		}
	}
}

// runBatchWorkers creates a worker pool to process a slice of items at the
// same time.
//
// Takes ctx (context.Context) which is threaded to workers for cancellation.
// Takes items ([]T) which contains the items to process.
// Takes concurrency (int) which sets the number of workers to run in parallel.
// Takes continueOnError (bool) which controls whether processing continues
// after a failure.
// Takes workerFunc (func(...)) which processes each item and returns a key or
// an error.
//
// Returns *storage.BatchResult which holds the results of processing,
// including keys that succeeded and keys that failed.
//
// Safe for concurrent use. Spawns worker goroutines tracked by a
// WaitGroup; closes the results channel when all workers finish.
func runBatchWorkers[T any](ctx context.Context, items []T, concurrency int, continueOnError bool, workerFunc func(context.Context, T) (string, error)) *storage.BatchResult {
	result := &storage.BatchResult{
		TotalRequested:  len(items),
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}
	jobs := make(chan batchJob[T], len(items))
	results := make(chan batchResult, len(items))

	var wg sync.WaitGroup
	startBatchWorkers(ctx, &wg, concurrency, jobs, results, continueOnError, workerFunc)

	for _, item := range items {
		if ctx.Err() != nil {
			break
		}

		jobs <- batchJob[T]{Item: item}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	collectBatchResults(results, result)
	return result
}

// logBatchResult logs the result of a batch operation in a standard format.
//
// Takes ctx (context.Context) which carries the logger.
// Takes op (string) which names the batch operation that was run.
// Takes provider (string) which identifies the storage provider used.
// Takes concurrency (int) which shows how many workers ran at once.
// Takes result (*storage.BatchResult) which holds the operation results.
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
