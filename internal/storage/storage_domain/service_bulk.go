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
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// logFieldTotal is the log field key for the total count of items in a batch.
	logFieldTotal = "total"

	// logFieldSuccessful is the log field key for the count of successful items.
	logFieldSuccessful = "successful"

	// logFieldFailed is the log field key for the count of failed operations.
	logFieldFailed = "failed"

	// logFieldIndex is the log field key for the object index in bulk operations.
	logFieldIndex = "index"
)

// PutObjects uploads multiple objects in a batch operation.
//
// Takes providerName (string) which identifies the storage provider to use.
// Takes params (*storage_dto.PutManyParams) which contains the objects to upload.
//
// Returns error when validation fails, the provider is not found, or the batch
// operation fails.
func (s *service) PutObjects(ctx context.Context, providerName string, params *storage_dto.PutManyParams) error {
	startTime := s.clock.Now()
	atomic.AddInt64(&s.stats.TotalOperations, 1)

	if len(params.Objects) == 0 {
		atomic.AddInt64(&s.stats.SuccessfulOperations, 1)
		duration := s.clock.Now().Sub(startTime).Milliseconds()
		operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
		batchOperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
		return nil
	}

	if err := validatePutManyParams(params, &s.config); err != nil {
		atomic.AddInt64(&s.stats.FailedOperations, 1)
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
		return fmt.Errorf("validating batch put params: %w", err)
	}

	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		atomic.AddInt64(&s.stats.FailedOperations, 1)
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
		return fmt.Errorf("resolving provider %q for batch put: %w", providerName, err)
	}

	var batchErr error
	if goroutine.SafeCallValue(ctx, "storage.SupportsBatchOperations", func() bool { return provider.SupportsBatchOperations() }) {
		batchErr = s.executeNativeBatchPut(ctx, provider, providerName, params)
	} else {
		batchErr = s.executeSequentialPut(ctx, providerName, params)
	}

	duration := s.clock.Now().Sub(startTime).Milliseconds()
	if batchErr != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
	}
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
	batchOperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))
	batchItemsTotal.Add(ctx, int64(len(params.Objects)), metric.WithAttributes(attribute.String(LogFieldOperation, OperationPutObjects)))

	return batchErr
}

// executeNativeBatchPut uploads multiple objects using the provider's native
// batch operation.
//
// Takes provider (StorageProviderPort) which runs the batch upload.
// Takes providerName (string) which names the provider for logging.
// Takes params (*storage_dto.PutManyParams) which holds the objects to upload.
//
// Returns error when the batch operation fails or any object fails to upload.
func (s *service) executeNativeBatchPut(ctx context.Context, provider StorageProviderPort, providerName string, params *storage_dto.PutManyParams) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Using native batch upload", logger_domain.String("provider", providerName), logger_domain.Int("object_count", len(params.Objects)))

	result, err := goroutine.SafeCall1(ctx, "storage.PutMany", func() (*storage_dto.BatchResult, error) { return provider.PutMany(ctx, params) })
	if err != nil {
		atomic.AddInt64(&s.stats.FailedOperations, 1)
		return fmt.Errorf("executing native batch put on provider %q: %w", providerName, err)
	}

	atomic.AddInt64(&s.stats.SuccessfulOperations, int64(result.TotalSuccessful))
	atomic.AddInt64(&s.stats.FailedOperations, int64(result.TotalFailed))

	logBatchResult(ctx, "upload", result)

	if result.HasErrors() {
		return batchResultToMultiError(params.Repository, result)
	}
	return nil
}

// executeSequentialPut uploads multiple objects one at a time for providers
// that do not support batch operations.
//
// Takes providerName (string) which identifies the storage provider.
// Takes params (*storage_dto.PutManyParams) which contains the objects to upload.
//
// Returns error when one or more objects fail to upload.
func (s *service) executeSequentialPut(ctx context.Context, providerName string, params *storage_dto.PutManyParams) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Provider doesn't support batch operations, using sequential upload",
		logger_domain.String("provider", providerName), logger_domain.Int("object_count", len(params.Objects)))

	multiErr := newMultiError()
	for i, storageObject := range params.Objects {
		putParam := &storage_dto.PutParams{
			Repository:           params.Repository,
			Key:                  storageObject.Key,
			Reader:               storageObject.Reader,
			Size:                 storageObject.Size,
			ContentType:          storageObject.ContentType,
			Metadata:             nil,
			MultipartConfig:      nil,
			TransformConfig:      params.TransformConfig,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			UseContentAddressing: false,
		}

		if err := s.PutObject(ctx, providerName, putParam); err != nil {
			multiErr.Add(params.Repository, storageObject.Key, err)
			l.ReportError(nil, err, "Failed to upload object in bulk operation", logger_domain.String("key", storageObject.Key), logger_domain.Int(logFieldIndex, i))
		}
	}

	if multiErr.HasErrors() {
		l.Warn("Bulk upload completed with partial success",
			logger_domain.Int(logFieldTotal, len(params.Objects)),
			logger_domain.Int(logFieldSuccessful, len(params.Objects)-len(multiErr.Errors)),
			logger_domain.Int(logFieldFailed, len(multiErr.Errors)))
		return multiErr
	}

	l.Trace("Bulk upload completed successfully", logger_domain.Int(logFieldTotal, len(params.Objects)))
	return nil
}

// RemoveObjects deletes several objects at once.
//
// Takes providerName (string) which names the storage provider to use.
// Takes params (storage_dto.RemoveManyParams) which lists the objects to
// remove.
//
// Returns error when validation fails or the provider cannot be found.
func (s *service) RemoveObjects(ctx context.Context, providerName string, params storage_dto.RemoveManyParams) error {
	startTime := s.clock.Now()

	if len(params.Keys) == 0 {
		duration := s.clock.Now().Sub(startTime).Milliseconds()
		operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
		batchOperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
		return nil
	}
	if err := validateRemoveManyParams(&params, &s.config); err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
		return fmt.Errorf("validating batch remove params: %w", err)
	}
	provider, err := s.getProvider(ctx, providerName)
	if err != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
		return fmt.Errorf("resolving provider %q for batch remove: %w", providerName, err)
	}

	var batchErr error
	if goroutine.SafeCallValue(ctx, "storage.SupportsBatchOperations", func() bool { return provider.SupportsBatchOperations() }) {
		batchErr = s.executeNativeBatchRemove(ctx, provider, providerName, params)
	} else {
		batchErr = s.executeSequentialRemove(ctx, provider, params)
	}

	duration := s.clock.Now().Sub(startTime).Milliseconds()
	if batchErr != nil {
		operationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
	}
	operationDuration.Record(ctx, float64(duration), metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
	batchOperationsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))
	batchItemsTotal.Add(ctx, int64(len(params.Keys)), metric.WithAttributes(attribute.String(LogFieldOperation, OperationRemoveObjects)))

	return batchErr
}

// executeNativeBatchRemove performs a batch delete using the provider's native
// batch removal capability.
//
// Takes provider (StorageProviderPort) which executes the batch removal.
// Takes providerName (string) which identifies the provider for logging.
// Takes params (storage_dto.RemoveManyParams) which specifies the keys to remove.
//
// Returns error when the provider fails or any individual deletions fail.
func (*service) executeNativeBatchRemove(ctx context.Context, provider StorageProviderPort, providerName string, params storage_dto.RemoveManyParams) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Using native batch delete", logger_domain.String("provider", providerName), logger_domain.Int("key_count", len(params.Keys)))

	result, err := goroutine.SafeCall1(ctx, "storage.RemoveMany", func() (*storage_dto.BatchResult, error) { return provider.RemoveMany(ctx, params) })
	if err != nil {
		return fmt.Errorf("executing native batch remove on provider %q: %w", providerName, err)
	}

	logBatchResult(ctx, "delete", result)

	if result.HasErrors() {
		return batchResultToMultiError(params.Repository, result)
	}
	return nil
}

// executeSequentialRemove removes multiple objects one at a time when the
// storage provider does not support batch operations.
//
// Takes provider (StorageProviderPort) which performs the actual removals.
// Takes params (storage_dto.RemoveManyParams) which specifies the repository
// and keys to remove.
//
// Returns error when one or more removals fail, wrapped in a MultiError.
func (*service) executeSequentialRemove(ctx context.Context, provider StorageProviderPort, params storage_dto.RemoveManyParams) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Provider doesn't support batch operations, using sequential delete", logger_domain.Int("key_count", len(params.Keys)))

	multiErr := newMultiError()
	for _, key := range params.Keys {
		if ctx.Err() != nil {
			break
		}
		removeParam := storage_dto.GetParams{
			Repository:      params.Repository,
			Key:             key,
			ByteRange:       nil,
			TransformConfig: nil,
		}
		if err := goroutine.SafeCall(ctx, "storage.Remove", func() error { return provider.Remove(ctx, removeParam) }); err != nil {
			multiErr.Add(params.Repository, key, err)
			l.ReportError(nil, err, "Failed to remove object in bulk operation", logger_domain.String("key", key))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// logBatchResult logs the result of a batch operation at the right level.
//
// When the result is a partial success, logs a warning with counts. When the
// result has errors, logs an error. Otherwise, logs a trace message.
//
// Takes operation (string) which names the batch operation for the message.
// Takes result (*storage_dto.BatchResult) which contains the batch result.
func logBatchResult(ctx context.Context, operation string, result *storage_dto.BatchResult) {
	_, l := logger_domain.From(ctx, log)
	if result.IsPartialSuccess() {
		l.Warn("Batch "+operation+" completed with partial success",
			logger_domain.Int(logFieldTotal, result.TotalRequested),
			logger_domain.Int(logFieldSuccessful, result.TotalSuccessful),
			logger_domain.Int(logFieldFailed, result.TotalFailed),
			logger_domain.Duration("duration", result.ProcessingTime))
	} else if result.HasErrors() {
		l.Error("Batch "+operation+" failed",
			logger_domain.Int(logFieldTotal, result.TotalRequested),
			logger_domain.Int(logFieldFailed, result.TotalFailed))
	} else {
		l.Trace("Batch "+operation+" completed successfully",
			logger_domain.Int(logFieldTotal, result.TotalRequested),
			logger_domain.Duration("duration", result.ProcessingTime))
	}
}
