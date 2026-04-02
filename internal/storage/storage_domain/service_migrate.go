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
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

// validateMigrationParams validates migration parameters and returns the source
// and destination providers.
//
// Takes params (*storage_dto.MigrateParams) which specifies the source and
// destination providers, keys, and concurrency settings.
//
// Returns srcProvider (StorageProviderPort) which is the resolved source
// provider.
// Returns dstProvider (StorageProviderPort) which is the resolved destination
// provider.
// Returns err (error) when providers are the same or cannot be resolved.
func (s *service) validateMigrationParams(
	ctx context.Context, params *storage_dto.MigrateParams,
) (srcProvider, dstProvider StorageProviderPort, err error) {
	if params.SourceProvider == params.DestinationProvider {
		return nil, nil, errors.New("source and destination providers cannot be the same")
	}

	srcProvider, err = s.getProvider(ctx, params.SourceProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get source provider: %w", err)
	}

	dstProvider, err = s.getProvider(ctx, params.DestinationProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get destination provider: %w", err)
	}

	if params.Concurrency <= 0 {
		params.Concurrency = DefaultMigrationBatchSize
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Starting migration",
		logger_domain.String("source", params.SourceProvider),
		logger_domain.String("destination", params.DestinationProvider),
		logger_domain.Int("object_count", len(params.Keys)),
		logger_domain.Int("concurrency", params.Concurrency),
		logger_domain.Bool("move_files", params.RemoveSourceAfterSuccess))

	return srcProvider, dstProvider, nil
}

// Migrate orchestrates the migration of objects between
// providers, processing jobs simultaneously.
//
// Takes params (*storage_dto.MigrateParams) which specifies
// the source and destination providers, keys, throughput,
// and error handling behaviour.
//
// Returns *storage_dto.BatchResult which contains counts and
// details of successful and failed migrations.
// Returns error when parameter validation fails or migration
// errors occur.
//
// Concurrent worker goroutines are spawned based on params.Concurrency,
// processing migration jobs in parallel via channels.
func (s *service) Migrate(ctx context.Context, params *storage_dto.MigrateParams) (*storage_dto.BatchResult, error) {
	srcProvider, dstProvider, err := s.validateMigrationParams(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("validating migration params from %q to %q: %w", params.SourceProvider, params.DestinationProvider, err)
	}

	if len(params.Keys) == 0 {
		return &storage_dto.BatchResult{}, nil
	}

	startTime := s.clock.Now()
	jobs := make(chan string, len(params.Keys))
	results := make(chan *storage_dto.BatchFailure, len(params.Keys))
	var wg sync.WaitGroup

	workerCtx := &migrationWorkerContext{
		params:      params,
		srcProvider: srcProvider,
		dstProvider: dstProvider,
		jobs:        jobs,
		results:     results,
	}
	for range params.Concurrency {
		wg.Add(1)
		go s.migrationWorker(ctx, &wg, workerCtx)
	}

	for _, key := range params.Keys {
		jobs <- key
	}
	close(jobs)

	wg.Wait()
	close(results)

	result := collectMigrationResults(params.Keys, results, startTime, s.clock.Now())
	logBatchResult(ctx, "migration", result)

	if result.HasErrors() {
		return result, batchResultToMultiError(params.Repository, result)
	}
	return result, nil
}

// migrationWorkerContext holds the shared state for a migration worker.
type migrationWorkerContext struct {
	// params holds the migration settings passed to migrateObject.
	params *storage_dto.MigrateParams

	// srcProvider is the source storage provider from which to migrate objects.
	srcProvider StorageProviderPort

	// dstProvider is the storage provider where objects are copied to.
	dstProvider StorageProviderPort

	// jobs receives object keys to migrate from the dispatcher.
	jobs <-chan string

	// results is a channel that receives migration failures for processing.
	results chan<- *storage_dto.BatchFailure
}

// migrationWorker processes migration jobs for individual files.
//
// Takes wg (*sync.WaitGroup) which tracks worker completion.
// Takes workerCtx (*migrationWorkerContext) which provides the jobs channel,
// results channel, and migration settings.
func (s *service) migrationWorker(
	ctx context.Context, wg *sync.WaitGroup,
	workerCtx *migrationWorkerContext,
) {
	defer wg.Done()
	for key := range workerCtx.jobs {
		if ctx.Err() != nil {
			return
		}
		err := s.migrateObject(ctx, key, workerCtx.params, workerCtx.srcProvider, workerCtx.dstProvider)
		if err != nil {
			workerCtx.results <- &storage_dto.BatchFailure{
				Key:       key,
				Error:     err.Error(),
				ErrorCode: "",
				Retryable: IsRetryableError(err),
			}
			if !workerCtx.params.ContinueOnError {
				for range workerCtx.jobs {
					_ = 1
				}
				return
			}
		}
	}
}

// migrateObject handles the Get->Stat->Put->(optional)Remove sequence for
// one object.
//
// Takes key (string) which identifies the object to migrate.
// Takes params (*storage_dto.MigrateParams) which specifies the migration
// settings.
// Takes srcProvider (StorageProviderPort) which provides access to the source
// storage.
// Takes dstProvider (StorageProviderPort) which provides access to the
// destination storage.
//
// Returns error when stat, get, or put operations fail.
func (*service) migrateObject(
	ctx context.Context, key string,
	params *storage_dto.MigrateParams, srcProvider, dstProvider StorageProviderPort,
) error {
	ctx, l := logger_domain.From(ctx, log)
	getParams := storage_dto.GetParams{
		Repository:      params.Repository,
		Key:             key,
		ByteRange:       nil,
		TransformConfig: nil,
	}

	info, err := srcProvider.Stat(ctx, getParams)
	if err != nil {
		return fmt.Errorf("failed to stat source object: %w", err)
	}

	reader, err := srcProvider.Get(ctx, getParams)
	if err != nil {
		return fmt.Errorf("failed to get source object: %w", err)
	}
	defer func() { _ = reader.Close() }()

	putParams := &storage_dto.PutParams{
		Repository:           params.Repository,
		Key:                  key,
		Reader:               reader,
		Size:                 info.Size,
		ContentType:          info.ContentType,
		Metadata:             info.Metadata,
		MultipartConfig:      nil,
		TransformConfig:      nil,
		HashAlgorithm:        "",
		ExpectedHash:         "",
		UseContentAddressing: false,
	}
	if err := dstProvider.Put(ctx, putParams); err != nil {
		return fmt.Errorf("failed to put destination object: %w", err)
	}

	if params.RemoveSourceAfterSuccess {
		if err := srcProvider.Remove(ctx, getParams); err != nil {
			l.Warn("Migration successful, but failed to remove source object",
				logger_domain.String("key", key),
				logger_domain.String("source_provider", params.SourceProvider),
				logger_domain.Error(err),
			)
		}
	}

	l.Trace("Successfully migrated object", logger_domain.String("key", key))
	return nil
}

// collectMigrationResults aggregates migration results from the results
// channel.
//
// Takes keys ([]string) which lists all keys that were requested for migration.
// Takes results (<-chan *storage_dto.BatchFailure) which yields failures from
// the migration process.
// Takes startTime (time.Time) which marks when the migration began.
// Takes now (time.Time) which marks when the migration ended.
//
// Returns *storage_dto.BatchResult which contains the aggregated success and
// failure counts along with the processing duration.
func collectMigrationResults(
	keys []string, results <-chan *storage_dto.BatchFailure, startTime time.Time, now time.Time,
) *storage_dto.BatchResult {
	result := &storage_dto.BatchResult{
		TotalRequested:  len(keys),
		TotalSuccessful: 0,
		TotalFailed:     0,
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		ProcessingTime:  now.Sub(startTime),
	}

	failedKeysMap := make(map[string]bool)
	for failure := range results {
		result.FailedKeys = append(result.FailedKeys, *failure)
		failedKeysMap[failure.Key] = true
	}
	result.TotalFailed = len(result.FailedKeys)

	for _, key := range keys {
		if !failedKeysMap[key] {
			result.SuccessfulKeys = append(result.SuccessfulKeys, key)
		}
	}
	result.TotalSuccessful = len(result.SuccessfulKeys)

	return result
}
