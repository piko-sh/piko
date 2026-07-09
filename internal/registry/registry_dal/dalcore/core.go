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

package dalcore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dal"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
	"piko.sh/piko/wdk/clock"
)

const (
	// maxTransactionTimeout is the maximum duration a RunAtomic transaction may hold before
	// being cancelled.
	maxTransactionTimeout = 30 * time.Second

	// defaultGCHintLimit is the default number of GC hints to fetch at once.
	defaultGCHintLimit = 100

	// maxGCHintLimit bounds a single PopGCHints call so the paired DeleteGCHints delete
	// stays within the tightest dialect bind-variable cap.
	maxGCHintLimit = 999

	// logKeyDurationMs is the log field key for operation duration in milliseconds.
	logKeyDurationMs = "durationMs"

	// logKeyStorageKey is the logging field name for blob storage keys.
	logKeyStorageKey = "storageKey"
)

var (
	// log is the package-level logger for the dalcore package.
	log = logger_domain.GetLogger("piko/internal/registry/registry_dal/dalcore")

	// errDALNotInitialised is returned when a transaction is attempted but the core has not
	// been initialised with a sql.DB connection.
	errDALNotInitialised = errors.New("cannot create transaction: DAL not initialised with a sql.DB connection")

	// errSearchQueryEmpty is returned when a search operation is attempted with an empty
	// query string.
	errSearchQueryEmpty = errors.New("search query is empty")

	_ registry_dal.RegistryDALWithTx = (*core)(nil)

	_ registry_domain.MetadataStore = (*core)(nil)

	_ registry_domain.RegistryInspector = (*core)(nil)
)

// core is the dialect-agnostic registry DAL. It satisfies RegistryDALWithTx,
// MetadataStore, and RegistryInspector, delegating every generated query to a per-dialect
// Driver while owning FlatBuffer serialisation, transaction lifecycle, and domain
// mapping.
type core struct {
	// sqlDB is the underlying database connection for health checks and transaction
	// creation.
	sqlDB *sql.DB

	// driver performs the dialect-specific generated-query calls.
	driver Driver

	// clock supplies the wall-clock time used for domain timestamps such as blob reference
	// and GC hint creation times. It defaults to the real clock and can be replaced with a
	// mock for deterministic tests.
	clock clock.Clock

	// inTransaction is true when this core is a transaction-scoped clone created by
	// withTransaction. It prevents nested transactions.
	inTransaction bool
}

// New creates a registry DAL backed by the given database connection and dialect driver.
//
// Takes database (*sql.DB) which provides the database connection for transactions and
// health checks.
// Takes driver (Driver) which performs the dialect-specific generated-query calls.
//
// Returns registry_dal.RegistryDALWithTx which is the configured DAL ready for use.
func New(database *sql.DB, driver Driver) registry_dal.RegistryDALWithTx {
	return &core{
		sqlDB:  database,
		driver: driver,
		clock:  clock.RealClock(),
	}
}

// HealthCheck performs a health check on the database connection.
//
// Returns error when the database ping fails.
func (c *core) HealthCheck(ctx context.Context) error {
	if c.sqlDB != nil {
		return c.sqlDB.PingContext(ctx)
	}
	return nil
}

// Close is a no-op because the caller owns the database connection.
//
// Returns error which is always nil.
func (*core) Close() error {
	return nil
}

// RunAtomic executes fn within a transaction.
//
// The provided MetadataStore is scoped to the transaction, so all reads and writes
// through it are atomic. If fn returns an error (or panics), all mutations are rolled
// back.
//
// Takes fn (func(ctx context.Context, transactionStore registry_domain.MetadataStore)
// error) which receives a transactional MetadataStore.
//
// Returns error when fn returns an error or the transaction fails to commit.
func (c *core) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	if c.inTransaction {
		return cache_domain.ErrNestedTransactionUnsupported
	}

	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	return c.withTransaction(ctx, func(ctx context.Context, transactionDAL registry_dal.RegistryDAL) error {
		store, ok := transactionDAL.(registry_domain.MetadataStore)
		if !ok {
			return errors.New("transaction DAL does not implement MetadataStore")
		}
		return fn(ctx, store)
	})
}

// GetArtefact retrieves a single artefact by ID with all its variants and profiles. Uses
// FlatBuffer blob for optimised reads.
//
// Takes artefactID (string) which specifies the unique identifier of the artefact to
// retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact with its variants and
// profiles.
// Returns error when the artefact is not found or the database query fails.
func (c *core) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.GetArtefact",
		logger_domain.String("artefactID", artefactID),
	)
	defer span.End()

	startTime := time.Now()

	dataFbs, err := c.driver.GetArtefactData(ctx, artefactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Trace("Artefact not found", logger_domain.String("artefactID", artefactID))
			return nil, registry_domain.ErrArtefactNotFound
		}
		l.ReportError(span, err, "Failed to get artefact")
		return nil, fmt.Errorf("failed to get artefact '%s': %w", artefactID, err)
	}

	artefact := registry_schema.ParseArtefactMeta(dataFbs)
	if artefact == nil {
		return nil, fmt.Errorf("failed to parse artefact '%s': corrupted or empty data", artefactID)
	}

	duration := time.Since(startTime)
	l.Trace("GetArtefact completed",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("variantCount", len(artefact.ActualVariants)),
		logger_domain.Int("profileCount", len(artefact.DesiredProfiles)))

	return artefact, nil
}

// GetMultipleArtefacts retrieves multiple artefacts by their IDs.
//
// Takes artefactIDs ([]string) which specifies the artefact IDs to retrieve.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts in the same
// order as the input IDs.
// Returns error when the database query fails.
func (c *core) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	if len(artefactIDs) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	blobs, err := c.driver.GetMultipleArtefactsData(ctx, artefactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple artefacts: %w", err)
	}

	artefactMap := make(map[string]*registry_dto.ArtefactMeta, len(blobs))
	for i := range blobs {
		artefact := registry_schema.ParseArtefactMeta(blobs[i])
		if artefact != nil {
			artefactMap[artefact.ID] = artefact
		}
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		if artefact, ok := artefactMap[id]; ok {
			results = append(results, artefact)
		}
	}

	return results, nil
}

// ListAllArtefactIDs returns all artefact IDs in the store.
//
// Returns []string which contains all artefact IDs currently stored.
// Returns error when the database query fails.
func (c *core) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	return c.driver.ListAllArtefactIDs(ctx)
}

// SearchArtefacts searches for artefacts matching the given tag query.
//
// Takes query (registry_domain.SearchQuery) which specifies the search criteria including
// simple tag queries.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the query is empty, uses unsupported RediSearch syntax, or when
// retrieval fails.
func (c *core) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.SearchArtefacts",
		logger_domain.Int("tagQueryCount", len(query.SimpleTagQuery)),
		logger_domain.Bool("hasRediSearch", query.RawRediSearchQuery != ""),
	)
	defer span.End()

	startTime := time.Now()

	if query.RawRediSearchQuery != "" {
		l.Trace("RediSearch query not supported")
		return nil, registry_dal.ErrSearchUnsupported
	}
	if len(query.SimpleTagQuery) == 0 {
		err := errSearchQueryEmpty
		l.ReportError(span, err, "Empty search query")
		return nil, fmt.Errorf("searching artefacts: %w", err)
	}

	finalIDs, err := c.processTagQueries(ctx, query.SimpleTagQuery)
	if err != nil {
		l.ReportError(span, err, "Failed to process tag queries")
		return nil, fmt.Errorf("processing tag queries: %w", err)
	}

	if len(finalIDs) == 0 {
		l.Trace("No matching artefacts found")
		return []*registry_dto.ArtefactMeta{}, nil
	}

	l.Trace("Found matching artefacts", logger_domain.Int("matchCount", len(finalIDs)))
	artefacts, err := c.GetMultipleArtefacts(ctx, finalIDs)

	duration := time.Since(startTime)
	if err != nil {
		l.ReportError(span, err, "Failed to get multiple artefacts")
		return nil, fmt.Errorf("retrieving matched artefacts: %w", err)
	}

	l.Trace("SearchArtefacts completed",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("resultCount", len(artefacts)))

	return artefacts, nil
}

// SearchArtefactsByTagValues searches for artefacts that have a specific tag key with any
// of the given values.
//
// Takes tagKey (string) which specifies the tag key to search for.
// Takes tagValues ([]string) which contains the tag values to match against.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the database query fails.
func (c *core) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.SearchArtefactsByTagValues",
		logger_domain.String("tagKey", tagKey),
		logger_domain.Int("valueCount", len(tagValues)),
	)
	defer span.End()

	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	artefactIDs, err := c.driver.FindArtefactIDsByTagValues(ctx, tagKey, tagValues)
	if err != nil {
		l.ReportError(span, err, "Failed to find artefact IDs by tag values")
		return nil, fmt.Errorf("failed to find artefact IDs for tag %s: %w", tagKey, err)
	}

	if len(artefactIDs) == 0 {
		l.Trace("No artefacts found for the given tag values")
		return []*registry_dto.ArtefactMeta{}, nil
	}

	l.Trace("Found matching artefact IDs, fetching full data", logger_domain.Int("idCount", len(artefactIDs)))

	return c.GetMultipleArtefacts(ctx, artefactIDs)
}

// FindArtefactByVariantStorageKey finds an artefact by the storage key of one of its
// variants.
//
// Takes storageKey (string) which identifies the variant's storage location.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact is not found or the query fails.
func (c *core) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	artefactID, err := c.driver.FindArtefactIDByVariantStorageKey(ctx, storageKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		return nil, fmt.Errorf("failed to find artefact by storage key '%s': %w", storageKey, err)
	}
	return c.GetArtefact(ctx, artefactID)
}

// PopGCHints retrieves and removes garbage collection hints from the store.
//
// Takes limit (int) which specifies the maximum number of hints to retrieve. Uses a
// default limit when limit is zero or negative.
//
// Returns []registry_dto.GCHint which contains the retrieved hints.
// Returns error when the database transaction fails.
func (c *core) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.PopGCHints",
		logger_domain.Int("limit", limit),
	)
	defer span.End()

	startTime := time.Now()
	if limit <= 0 {
		limit = defaultGCHintLimit
		l.Trace("Using default limit", logger_domain.Int("defaultLimit", limit))
	}
	if limit > maxGCHintLimit {
		limit = maxGCHintLimit
		l.Trace("Clamping limit to maximum", logger_domain.Int("maxLimit", limit))
	}

	var hints []registry_dto.GCHint
	err := l.RunInSpan(ctx, "PopGCHintsTransaction", func(ctx context.Context, _ logger_domain.Logger) error {
		return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
			dbHints, err := driver.PopGCHints(ctx, limit)
			if err != nil {
				return fmt.Errorf("failed to pop GC hints from DB: %w", err)
			}

			if len(dbHints) == 0 {
				l.Trace("No GC hints found")
				hints = []registry_dto.GCHint{}
				return nil
			}

			l.Trace("Processing GC hints", logger_domain.Int("hintCount", len(dbHints)))
			var idsToDelete []int64
			hints, idsToDelete = convertHintsToDTO(dbHints)
			if err := driver.DeleteGCHints(ctx, idsToDelete); err != nil {
				return fmt.Errorf("failed to delete popped GC hints: %w", err)
			}
			return nil
		})
	})

	duration := time.Since(startTime)
	if err != nil {
		l.ReportError(span, err, "Failed to pop GC hints")
		return nil, fmt.Errorf("popping GC hints: %w", err)
	}

	l.Trace("PopGCHints completed",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("hintCount", len(hints)))

	return hints, nil
}

// AtomicUpdate performs a batch of atomic operations within a single transaction.
//
// Takes actions ([]registry_dto.AtomicAction) which specifies the operations to execute
// atomically.
//
// Returns error when the transaction fails to begin, an action fails, or the commit
// fails.
func (c *core) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.AtomicUpdate",
		logger_domain.Int("actionCount", len(actions)),
	)
	defer span.End()

	startTime := time.Now()
	now := c.clock.Now().UTC()

	err := l.RunInSpan(ctx, "AtomicUpdateTransaction", func(ctx context.Context, _ logger_domain.Logger) error {
		return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
			for i, action := range actions {
				l.Trace("Processing atomic action",
					logger_domain.Int("actionIndex", i),
					logger_domain.String("actionType", string(action.Type)))

				if err := processAtomicAction(ctx, driver, action, now); err != nil {
					l.ReportError(span, err, "Atomic action failed")
					return err
				}
			}
			return nil
		})
	})

	duration := time.Since(startTime)
	if err != nil {
		return fmt.Errorf("executing atomic update: %w", err)
	}

	l.Trace("AtomicUpdate completed",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("actionCount", len(actions)))

	return nil
}

// IncrementBlobRefCount atomically increments the reference count for a blob. If the blob
// does not exist, it creates it with a reference count of one.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob to increment.
//
// Returns int which is the new reference count after the increment.
// Returns error when the database operation fails.
func (c *core) IncrementBlobRefCount(ctx context.Context, blob registry_domain.BlobReference) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.IncrementBlobRefCount",
		logger_domain.String(logKeyStorageKey, blob.StorageKey),
	)
	defer span.End()

	now := c.clock.Now().UTC()
	newRefCount, err := c.driver.IncrementBlobRefCount(ctx, IncrementBlobRefCountParams{
		StorageKey:       blob.StorageKey,
		StorageBackendID: blob.StorageBackendID,
		ContentHash:      blob.ContentHash,
		SizeBytes:        blob.SizeBytes,
		MimeType:         blob.MimeType,
		CreatedAt:        now.Unix(),
		LastReferencedAt: now.Unix(),
	})
	if err != nil {
		l.ReportError(span, err, "Failed to increment blob ref count")
		return 0, fmt.Errorf("failed to increment blob ref count for %s: %w", blob.StorageKey, err)
	}

	l.Trace("Incremented blob ref count",
		logger_domain.Int("newRefCount", newRefCount),
		logger_domain.String(logKeyStorageKey, blob.StorageKey))

	return newRefCount, nil
}

// DecrementBlobRefCount atomically decrements the reference count for a blob and
// indicates whether the blob should be deleted.
//
// Takes storageKey (string) which identifies the blob in storage.
//
// Returns int which is the new reference count after decrementing.
// Returns bool which is true when the blob should be deleted (ref count is 0).
// Returns error when the blob does not exist.
func (c *core) DecrementBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.DecrementBlobRefCount",
		logger_domain.String(logKeyStorageKey, storageKey),
	)
	defer span.End()

	now := c.clock.Now().UTC()
	newRefCount, err := c.driver.DecrementBlobRefCount(ctx, storageKey, now.Unix())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn("Attempted to decrement ref count for non-existent blob",
				logger_domain.String(logKeyStorageKey, storageKey))
			return 0, false, registry_domain.ErrBlobReferenceNotFound
		}
		l.ReportError(span, err, "Failed to decrement blob ref count")
		return 0, false, fmt.Errorf("failed to decrement blob ref count for %s: %w", storageKey, err)
	}

	shouldDelete := newRefCount == 0
	l.Trace("Decremented blob ref count",
		logger_domain.Int("newRefCount", newRefCount),
		logger_domain.Bool("shouldDelete", shouldDelete),
		logger_domain.String(logKeyStorageKey, storageKey))

	if shouldDelete {
		if err := c.driver.DeleteBlobReferenceIfZero(ctx, storageKey); err != nil {
			l.Warn("Failed to delete blob reference record with zero ref count",
				logger_domain.Error(err),
				logger_domain.String(logKeyStorageKey, storageKey))
		}
	}

	return newRefCount, shouldDelete, nil
}

// GetBlobRefCount returns the current reference count for a blob.
// Returns 0 if the blob does not exist (not an error).
//
// Takes storageKey (string) which identifies the blob to look up.
//
// Returns int which is the reference count for the blob.
// Returns error when the database query fails.
func (c *core) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.GetBlobRefCount",
		logger_domain.String(logKeyStorageKey, storageKey),
	)
	defer span.End()

	refCount, err := c.driver.GetBlobRefCount(ctx, storageKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Trace("Blob reference not found, returning 0",
				logger_domain.String(logKeyStorageKey, storageKey))
			return 0, nil
		}
		l.ReportError(span, err, "Failed to get blob ref count")
		return 0, fmt.Errorf("failed to get blob ref count for %s: %w", storageKey, err)
	}

	l.Trace("Retrieved blob ref count",
		logger_domain.Int("refCount", refCount),
		logger_domain.String(logKeyStorageKey, storageKey))

	return refCount, nil
}

// ListArtefactSummary returns artefact counts grouped by status. Status is stored inside
// the FlatBuffer payload so all artefacts are fetched and aggregated in Go.
//
// Returns []registry_domain.ArtefactSummary which contains one entry per status with its
// count.
// Returns error when the database query fails or a FlatBuffer is corrupt.
func (c *core) ListArtefactSummary(ctx context.Context) ([]registry_domain.ArtefactSummary, error) {
	blobs, err := c.driver.ListAllArtefactsData(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing artefacts for summary: %w", err)
	}

	statusCounts := make(map[string]int64)
	for i := range blobs {
		artefact := registry_schema.ParseArtefactMeta(blobs[i])
		if artefact == nil {
			continue
		}
		statusCounts[string(artefact.Status)]++
	}

	results := make([]registry_domain.ArtefactSummary, 0, len(statusCounts))
	for status, count := range statusCounts {
		results = append(results, registry_domain.ArtefactSummary{
			Status: status,
			Count:  count,
		})
	}

	return results, nil
}

// ListVariantSummary returns variant counts grouped by status.
//
// Returns []registry_domain.VariantSummary which contains one entry per variant status
// with its count.
// Returns error when the database query fails.
func (c *core) ListVariantSummary(ctx context.Context) ([]registry_domain.VariantSummary, error) {
	rows, err := c.driver.ListVariantStatusCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing variant status counts: %w", err)
	}

	results := make([]registry_domain.VariantSummary, len(rows))
	for i, row := range rows {
		results[i] = registry_domain.VariantSummary{
			Status: row.Status,
			Count:  row.Count,
		}
	}

	return results, nil
}

// ListRecentArtefacts returns the most recently updated artefacts with variant counts and
// total sizes.
//
// Takes limit (int) which specifies the maximum number of artefacts to return.
//
// Returns []registry_domain.ArtefactListItem which contains artefacts ordered by update
// time descending.
// Returns error when the database query fails or a FlatBuffer is corrupt.
func (c *core) ListRecentArtefacts(ctx context.Context, limit int) ([]registry_domain.ArtefactListItem, error) {
	blobs, err := c.driver.ListRecentArtefactsData(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing recent artefacts: %w", err)
	}

	results := make([]registry_domain.ArtefactListItem, 0, len(blobs))
	for i := range blobs {
		artefact := registry_schema.ParseArtefactMeta(blobs[i])
		if artefact == nil {
			continue
		}

		var totalSize int64
		for variantIndex := range artefact.ActualVariants {
			totalSize += artefact.ActualVariants[variantIndex].SizeBytes
		}

		results = append(results, registry_domain.ArtefactListItem{
			ID:           artefact.ID,
			SourcePath:   artefact.SourcePath,
			Status:       string(artefact.Status),
			VariantCount: int64(len(artefact.ActualVariants)),
			TotalSize:    totalSize,
			CreatedAt:    artefact.CreatedAt.Unix(),
			UpdatedAt:    artefact.UpdatedAt.Unix(),
		})
	}

	return results, nil
}

// runInTransaction executes fn within a transaction.
//
// If the core is already inside a transaction (inTransaction == true), it reuses the
// existing driver to avoid deadlocking on SQLite's single-writer lock.
//
// Takes fn (func(ctx context.Context, driver Driver) error) which is the function to
// execute within the transaction scope.
//
// Returns error when the transaction cannot be started, fn returns an error, or the
// commit fails.
func (c *core) runInTransaction(ctx context.Context, fn func(ctx context.Context, driver Driver) error) error {
	if c.inTransaction {
		return fn(ctx, c.driver)
	}

	if c.sqlDB == nil {
		return errDALNotInitialised
	}

	tx, err := c.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("transaction rollback failed", logger_domain.Error(rollbackErr))
		}
	}()

	if err := fn(ctx, c.driver.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// withTransaction is an internal helper that executes an operation within a database
// transaction, creating a transaction-scoped core clone.
//
// Takes operation (func(ctx context.Context, dal registry_dal.RegistryDAL) error) which
// is the callback to execute within the transaction scope.
//
// Returns error when the core is not initialised, the transaction cannot be started, the
// callback returns an error, or the commit fails.
//
// Panics if operation panics. The transaction is rolled back before re-panicking.
func (c *core) withTransaction(ctx context.Context, operation func(ctx context.Context, dal registry_dal.RegistryDAL) error) error {
	ctx, l := logger_domain.From(ctx, log)

	if c.sqlDB == nil {
		return errDALNotInitialised
	}

	tx, err := c.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				l.Warn("Failed to rollback transaction", logger_domain.Error(rollbackErr))
			}
			panic(p)
		}
	}()

	txCore := &core{
		sqlDB:         c.sqlDB,
		driver:        c.driver.WithTx(tx),
		clock:         c.clock,
		inTransaction: true,
	}

	if err := operation(ctx, txCore); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			l.Warn("Failed to rollback transaction", logger_domain.Error(rollbackErr))
		}
		return err
	}

	return tx.Commit()
}

// processTagQueries finds artefact IDs matching all tag queries using intersection.
//
// Takes tagQuery (map[string]string) which maps tag keys to required values.
//
// Returns []string which contains artefact IDs matching all tag criteria, or nil if no
// matches exist.
// Returns error when the database query fails.
func (c *core) processTagQueries(
	ctx context.Context,
	tagQuery map[string]string,
) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	var intersection *map[string]struct{}

	for key, value := range tagQuery {
		l.Trace("Processing tag query",
			logger_domain.String("tagKey", key),
			logger_domain.String("tagValue", value))

		ids, err := c.driver.FindArtefactIDsByTag(ctx, key, value)
		if err != nil {
			return nil, fmt.Errorf("tag search failed for %s=%s: %w", key, value, err)
		}

		l.Trace("Found artefacts with tag",
			logger_domain.String("tagKey", key),
			logger_domain.String("tagValue", value),
			logger_domain.Int("resultCount", len(ids)))

		intersection = intersectIDSets(intersection, ids)
		if len(*intersection) == 0 {
			l.Trace("No matching artefacts after intersection")
			return nil, nil
		}
	}

	if intersection == nil {
		return nil, nil
	}

	finalIDs := make([]string, 0, len(*intersection))
	for id := range *intersection {
		finalIDs = append(finalIDs, id)
	}
	return finalIDs, nil
}

// processAtomicAction executes a single atomic action within a transaction.
//
// Takes driver (Driver) which provides transactional database access.
// Takes action (registry_dto.AtomicAction) which specifies the operation to perform.
// Takes now (time.Time) which is the timestamp recorded on any GC hints the action adds.
//
// Returns error when the action fails or the action type is unrecognised.
func processAtomicAction(
	ctx context.Context,
	driver Driver,
	action registry_dto.AtomicAction,
	now time.Time,
) error {
	switch action.Type {
	case registry_dto.ActionTypeUpsertArtefact:
		if err := upsertArtefact(ctx, driver, action.Artefact); err != nil {
			return fmt.Errorf("atomic upsert for artefact '%s' failed: %w", action.Artefact.ID, err)
		}
	case registry_dto.ActionTypeDeleteArtefact:
		if err := driver.DeleteArtefact(ctx, action.ArtefactID); err != nil {
			return fmt.Errorf("atomic delete for artefact '%s' failed: %w", action.ArtefactID, err)
		}
	case registry_dto.ActionTypeAddGCHints:
		if err := addGCHints(ctx, driver, action.GCHints, now); err != nil {
			return fmt.Errorf("atomic add GC hints failed: %w", err)
		}
	default:
		return fmt.Errorf("unrecognised atomic action type: %s", action.Type)
	}
	return nil
}

// upsertArtefact inserts or updates an artefact and its related data.
//
// Takes driver (Driver) which provides database access.
// Takes artefact (*registry_dto.ArtefactMeta) which contains the artefact metadata to
// store.
//
// Returns error when the database operation fails.
func upsertArtefact(ctx context.Context, driver Driver, artefact *registry_dto.ArtefactMeta) error {
	fbsData := registry_schema.BuildArtefactMeta(artefact)

	if err := driver.UpsertArtefact(ctx, UpsertArtefactParams{
		ID:         artefact.ID,
		SourcePath: artefact.SourcePath,
		CreatedAt:  artefact.CreatedAt.Unix(),
		UpdatedAt:  artefact.UpdatedAt.Unix(),
		DataFbs:    fbsData,
	}); err != nil {
		return fmt.Errorf("failed to upsert artefact: %w", err)
	}

	if err := deleteExistingArtefactData(ctx, driver, artefact); err != nil {
		return fmt.Errorf("deleting existing artefact data for '%s': %w", artefact.ID, err)
	}

	if err := insertVariantsWithData(ctx, driver, artefact); err != nil {
		return fmt.Errorf("inserting variants for artefact '%s': %w", artefact.ID, err)
	}

	return insertDesiredProfiles(ctx, driver, artefact)
}

// addGCHints stores garbage collection hints for the given storage keys.
//
// Takes driver (Driver) which provides database access.
// Takes hints ([]registry_dto.GCHint) which contains the storage keys to mark for
// cleanup.
// Takes now (time.Time) which is the timestamp recorded on each stored hint.
//
// Returns error when a hint cannot be added to the database.
func addGCHints(ctx context.Context, driver Driver, hints []registry_dto.GCHint, now time.Time) error {
	nowSeconds := now.Unix()
	for _, hint := range hints {
		err := driver.AddGCHint(ctx, hint.BackendID, hint.StorageKey, nowSeconds)
		if err != nil {
			return fmt.Errorf("failed to add GC hint for key '%s': %w", hint.StorageKey, err)
		}
	}
	return nil
}

// deleteExistingArtefactData removes all existing data for an artefact before
// re-importing it.
//
// Takes driver (Driver) which provides database access.
// Takes artefact (*registry_dto.ArtefactMeta) which identifies the artefact to clear.
//
// Returns error when any database deletion fails.
func deleteExistingArtefactData(ctx context.Context, driver Driver, artefact *registry_dto.ArtefactMeta) error {
	if err := driver.DeleteVariantTagsForArtefact(ctx, artefact.ID); err != nil {
		return fmt.Errorf("failed to delete old variant tags: %w", err)
	}

	for i := range artefact.ActualVariants {
		variant := &artefact.ActualVariants[i]
		if err := driver.DeleteChunksForVariant(ctx, artefact.ID, variant.VariantID); err != nil {
			return fmt.Errorf("failed to delete old chunks for variant '%s': %w", variant.VariantID, err)
		}
	}

	if err := driver.DeleteVariantsForArtefact(ctx, artefact.ID); err != nil {
		return fmt.Errorf("failed to delete old variants: %w", err)
	}

	if err := driver.DeleteDesiredProfilesForArtefact(ctx, artefact.ID); err != nil {
		return fmt.Errorf("failed to delete old desired profiles: %w", err)
	}

	return nil
}

// insertVariantsWithData inserts all variants for an artefact along with their tags and
// chunks.
//
// Takes driver (Driver) which provides database access.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the variants to insert.
//
// Returns error when inserting a variant, its tags, or its chunks fails.
func insertVariantsWithData(ctx context.Context, driver Driver, artefact *registry_dto.ArtefactMeta) error {
	for i := range artefact.ActualVariants {
		variant := &artefact.ActualVariants[i]
		if err := insertVariant(ctx, driver, artefact.ID, variant); err != nil {
			return fmt.Errorf("inserting variant '%s': %w", variant.VariantID, err)
		}
		if err := insertVariantTags(ctx, driver, artefact.ID, variant); err != nil {
			return fmt.Errorf("inserting tags for variant '%s': %w", variant.VariantID, err)
		}
		if err := insertVariantChunks(ctx, driver, artefact.ID, variant); err != nil {
			return fmt.Errorf("inserting chunks for variant '%s': %w", variant.VariantID, err)
		}
	}
	return nil
}

// insertVariant stores a variant record in the database for the given artefact.
//
// Takes driver (Driver) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes variant (*registry_dto.Variant) which contains the variant data to store.
//
// Returns error when the database insert fails.
func insertVariant(ctx context.Context, driver Driver, artefactID string, variant *registry_dto.Variant) error {
	return driver.InsertVariant(ctx, InsertVariantParams{
		ArtefactID:       artefactID,
		VariantID:        variant.VariantID,
		StorageKey:       variant.StorageKey,
		StorageBackendID: variant.StorageBackendID,
		MimeType:         variant.MimeType,
		SizeBytes:        variant.SizeBytes,
		Status:           string(variant.Status),
		CreatedAt:        variant.CreatedAt.Unix(),
	})
}

// insertVariantTags stores all metadata tags for a variant in the database.
//
// Takes driver (Driver) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes variant (*registry_dto.Variant) which contains the tags to insert.
//
// Returns error when a tag cannot be inserted.
func insertVariantTags(ctx context.Context, driver Driver, artefactID string, variant *registry_dto.Variant) error {
	for key, value := range variant.MetadataTags.All() {
		err := driver.InsertVariantTag(ctx, artefactID, variant.VariantID, key, value)
		if err != nil {
			return fmt.Errorf("failed to insert tag for variant '%s': %w", variant.VariantID, err)
		}
	}
	return nil
}

// insertVariantChunks stores all chunks for a variant in the database.
//
// Takes driver (Driver) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes variant (*registry_dto.Variant) which contains the chunks to insert.
//
// Returns error when a chunk cannot be inserted.
func insertVariantChunks(ctx context.Context, driver Driver, artefactID string, variant *registry_dto.Variant) error {
	for i := range variant.Chunks {
		chunk := &variant.Chunks[i]
		err := driver.InsertVariantChunk(ctx, InsertVariantChunkParams{
			ArtefactID:       artefactID,
			VariantID:        variant.VariantID,
			ChunkID:          chunk.ChunkID,
			StorageKey:       chunk.StorageKey,
			StorageBackendID: chunk.StorageBackendID,
			SizeBytes:        chunk.SizeBytes,
			ContentHash:      chunk.ContentHash,
			SequenceNumber:   int64(chunk.SequenceNumber),
			MimeType:         chunk.MimeType,
			CreatedAt:        chunk.CreatedAt.Unix(),
			DurationSeconds:  chunk.DurationSeconds,
		})
		if err != nil {
			return fmt.Errorf("failed to insert chunk '%s' for variant '%s': %w", chunk.ChunkID, variant.VariantID, err)
		}
	}
	return nil
}

// insertDesiredProfiles stores the desired profiles from an artefact into the database.
//
// Takes driver (Driver) which provides database access.
// Takes artefact (*registry_dto.ArtefactMeta) which contains the profiles to store.
//
// Returns error when a profile cannot be inserted into the database.
func insertDesiredProfiles(ctx context.Context, driver Driver, artefact *registry_dto.ArtefactMeta) error {
	for i := range artefact.DesiredProfiles {
		desiredProfile := &artefact.DesiredProfiles[i]
		paramsJSON, err := json.Marshal(desiredProfile.Profile.Params)
		if err != nil {
			return fmt.Errorf("marshalling params for desired profile '%s': %w", desiredProfile.Name, err)
		}
		tagsJSON, err := json.Marshal(desiredProfile.Profile.ResultingTags)
		if err != nil {
			return fmt.Errorf("marshalling tags for desired profile '%s': %w", desiredProfile.Name, err)
		}
		dependsOnJSON, err := json.Marshal(desiredProfile.Profile.DependsOn)
		if err != nil {
			return fmt.Errorf("marshalling depends-on for desired profile '%s': %w", desiredProfile.Name, err)
		}
		if err := driver.InsertDesiredProfile(ctx, InsertDesiredProfileParams{
			ArtefactID:     artefact.ID,
			Name:           desiredProfile.Name,
			CapabilityName: desiredProfile.Profile.CapabilityName,
			Priority:       string(desiredProfile.Profile.Priority),
			ParamsJSON:     string(paramsJSON),
			TagsJSON:       string(tagsJSON),
			DependsOnJSON:  string(dependsOnJSON),
		}); err != nil {
			return fmt.Errorf("failed to insert desired profile '%s': %w", desiredProfile.Name, err)
		}
	}
	return nil
}

// intersectIDSets finds the common IDs between the current set and a list of IDs. If
// current is nil, it creates a new set with all the given IDs.
//
// Takes current (*map[string]struct{}) which is the existing ID set to check against, or
// nil to create a new set.
// Takes ids ([]string) which contains the IDs to match or add.
//
// Returns *map[string]struct{} which contains only the IDs found in both current and ids,
// or all ids if current was nil.
func intersectIDSets(current *map[string]struct{}, ids []string) *map[string]struct{} {
	if current == nil {
		idSet := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
		return &idSet
	}

	newSet := make(map[string]struct{})
	for _, id := range ids {
		if _, exists := (*current)[id]; exists {
			newSet[id] = struct{}{}
		}
	}
	return &newSet
}

// convertHintsToDTO converts driver GC hint rows to DTOs and returns IDs for deletion.
//
// Takes dbHints ([]GCHintRow) which contains the rows to convert.
//
// Returns []registry_dto.GCHint which contains the converted hint DTOs.
// Returns []int64 which contains the row IDs to delete from the database.
func convertHintsToDTO(dbHints []GCHintRow) ([]registry_dto.GCHint, []int64) {
	hints := make([]registry_dto.GCHint, len(dbHints))
	idsToDelete := make([]int64, len(dbHints))
	for i, hint := range dbHints {
		hints[i] = registry_dto.GCHint{BackendID: hint.BackendID, StorageKey: hint.StorageKey}
		idsToDelete[i] = hint.ID
	}
	return hints, idsToDelete
}
