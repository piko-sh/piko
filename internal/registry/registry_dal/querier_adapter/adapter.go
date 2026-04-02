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

package querier_adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/registry/registry_dal"
	registry_db "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxTransactionTimeout is the maximum duration a RunAtomic transaction
	// may hold before being cancelled.
	maxTransactionTimeout = 30 * time.Second

	// defaultGCHintLimit is the default number of GC hints to fetch at once.
	defaultGCHintLimit = 100

	// logKeyDurationMs is the log field key for operation duration in milliseconds.
	logKeyDurationMs = "durationMs"

	// logKeyStorageKey is the logging field name for blob storage keys.
	logKeyStorageKey = "storageKey"
)

var (
	log = logger.GetLogger("piko/internal/registry/registry_dal/querier_adapter")

	errDALNotInitialised = errors.New("cannot create transaction: DAL not initialised with a sql.DB connection")

	errSearchQueryEmpty = errors.New("search query is empty")
)

// DAL wraps the querier-generated Queries struct to satisfy the
// RegistryDALWithTx interface. It delegates simple queries directly
// to the generated code and handles IN-clause expansion, FlatBuffer
// serialisation, and transaction management.
type DAL struct {
	// db is the underlying database connection for health checks and
	// transaction creation.
	db *sql.DB

	// dbtx is the active database or transaction handle used for dynamic
	// queries that require IN-clause expansion. The generated Queries
	// struct does not expose its internal reader/writer, so we keep a
	// parallel reference.
	dbtx registry_db.DBTX

	// queries provides access to the generated query methods.
	queries *registry_db.Queries

	// inTransaction is true when this DAL is a transaction-scoped clone
	// created by withTransaction. It prevents nested transactions.
	inTransaction bool
}

// NewDAL creates a new DAL backed by the given database connection.
//
// Takes database (*sql.DB) which provides the database connection.
//
// Returns *DAL which is ready for use.
func NewDAL(database *sql.DB) *DAL {
	return &DAL{
		db:      database,
		dbtx:    database,
		queries: registry_db.New(database),
	}
}

// NewDALWithTx creates a transaction-scoped DAL clone. The clone uses the
// provided transaction for all queries but retains the parent database
// connection for health checks.
//
// Takes tx (*sql.Tx) which provides the transactional database connection.
// Takes parentDB (*sql.DB) which is retained for health checks.
//
// Returns *DAL which is scoped to the transaction.
func NewDALWithTx(tx *sql.Tx, parentDB *sql.DB) *DAL {
	return &DAL{
		db:            parentDB,
		dbtx:          tx,
		queries:       registry_db.New(tx),
		inTransaction: true,
	}
}

// HealthCheck performs a health check on the database connection.
//
// Returns error when the database ping fails.
func (d *DAL) HealthCheck(ctx context.Context) error {
	if d.db != nil {
		return d.db.PingContext(ctx)
	}
	return nil
}

// Close is a no-op because the caller owns the database connection.
//
// Returns error which is always nil.
func (*DAL) Close() error {
	return nil
}

// RunAtomic executes fn within a transaction.
//
// The provided MetadataStore is scoped to the transaction, so all
// reads and writes through it are atomic. If fn returns an error
// (or panics), all mutations are rolled back.
//
// Takes fn (func(ctx context.Context,
// transactionStore MetadataStore) error) which receives a
// transactional MetadataStore.
//
// Returns error when fn returns an error or the transaction fails
// to commit.
func (d *DAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
	if d.inTransaction {
		return cache_domain.ErrNestedTransactionUnsupported
	}

	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	return d.withTransaction(ctx, func(ctx context.Context, transactionDAL registry_dal.RegistryDAL) error {
		store, ok := transactionDAL.(registry_domain.MetadataStore)
		if !ok {
			return errors.New("transaction DAL does not implement MetadataStore")
		}
		return fn(ctx, store)
	})
}

// GetArtefact retrieves a single artefact by ID with all its variants and
// profiles. Uses FlatBuffer blob for optimised reads.
//
// Takes artefactID (string) which specifies the unique identifier of the
// artefact to retrieve.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact with its
// variants and profiles.
// Returns error when the artefact is not found or the database query fails.
func (d *DAL) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.GetArtefact",
		logger.String("artefactID", artefactID),
	)
	defer span.End()

	startTime := time.Now()

	dbRow, err := d.queries.GetArtefact(ctx, artefactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Trace("Artefact not found", logger.String("artefactID", artefactID))
			return nil, registry_domain.ErrArtefactNotFound
		}
		l.ReportError(span, err, "Failed to get artefact")
		return nil, fmt.Errorf("failed to get artefact '%s': %w", artefactID, err)
	}

	art := registry_schema.ParseArtefactMeta(dbRow.DataFbs)
	if art == nil {
		return nil, fmt.Errorf("failed to parse artefact '%s': corrupted or empty data", artefactID)
	}

	duration := time.Since(startTime)
	l.Trace("GetArtefact completed",
		logger.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger.Int("variantCount", len(art.ActualVariants)),
		logger.Int("profileCount", len(art.DesiredProfiles)))

	return art, nil
}

// GetMultipleArtefacts retrieves multiple artefacts by their IDs.
// Uses dynamic IN-clause expansion because the generated query does not
// handle slices correctly.
//
// Takes artefactIDs ([]string) which specifies the artefact IDs to retrieve.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts
// in the same order as the input IDs.
// Returns error when the database query fails.
func (d *DAL) GetMultipleArtefacts(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, error) {
	if len(artefactIDs) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	dbRows, err := d.queries.GetMultipleArtefacts(ctx, registry_db.GetMultipleArtefactsParams{
		IDs: artefactIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple artefacts: %w", err)
	}

	artefactMap := make(map[string]*registry_dto.ArtefactMeta, len(dbRows))
	for i := range dbRows {
		row := &dbRows[i]
		art := registry_schema.ParseArtefactMeta(row.DataFbs)
		if art != nil {
			artefactMap[art.ID] = art
		}
	}

	results := make([]*registry_dto.ArtefactMeta, 0, len(artefactIDs))
	for _, id := range artefactIDs {
		if art, ok := artefactMap[id]; ok {
			results = append(results, art)
		}
	}

	return results, nil
}

// ListAllArtefactIDs returns all artefact IDs in the store.
//
// Returns []string which contains all artefact IDs currently stored.
// Returns error when the database query fails.
func (d *DAL) ListAllArtefactIDs(ctx context.Context) ([]string, error) {
	rows, err := d.queries.ListAllArtefactIDs(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, nil
}

// SearchArtefacts searches for artefacts matching the given tag query.
//
// Takes query (registry_domain.SearchQuery) which specifies the search
// criteria including simple tag queries.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the query is empty, uses unsupported RediSearch syntax,
// or when retrieval fails.
func (d *DAL) SearchArtefacts(ctx context.Context, query registry_domain.SearchQuery) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.SearchArtefacts",
		logger.Int("tagQueryCount", len(query.SimpleTagQuery)),
		logger.Bool("hasRediSearch", query.RawRediSearchQuery != ""),
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

	finalIDs, err := d.processTagQueries(ctx, query.SimpleTagQuery)
	if err != nil {
		l.ReportError(span, err, "Failed to process tag queries")
		return nil, fmt.Errorf("processing tag queries: %w", err)
	}

	if len(finalIDs) == 0 {
		l.Trace("No matching artefacts found")
		return []*registry_dto.ArtefactMeta{}, nil
	}

	l.Trace("Found matching artefacts", logger.Int("matchCount", len(finalIDs)))
	artefacts, err := d.GetMultipleArtefacts(ctx, finalIDs)

	duration := time.Since(startTime)
	if err != nil {
		l.ReportError(span, err, "Failed to get multiple artefacts")
		return nil, fmt.Errorf("retrieving matched artefacts: %w", err)
	}

	l.Trace("SearchArtefacts completed",
		logger.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger.Int("resultCount", len(artefacts)))

	return artefacts, nil
}

// SearchArtefactsByTagValues searches for artefacts that have a specific tag
// key with any of the given values. Uses dynamic IN-clause expansion for the
// tag values.
//
// Takes tagKey (string) which specifies the tag key to search for.
// Takes tagValues ([]string) which contains the tag values to match against.
//
// Returns []*registry_dto.ArtefactMeta which contains the matching artefacts.
// Returns error when the database query fails.
func (d *DAL) SearchArtefactsByTagValues(ctx context.Context, tagKey string, tagValues []string) ([]*registry_dto.ArtefactMeta, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.SearchArtefactsByTagValues",
		logger.String("tagKey", tagKey),
		logger.Int("valueCount", len(tagValues)),
	)
	defer span.End()

	if len(tagValues) == 0 {
		return []*registry_dto.ArtefactMeta{}, nil
	}

	dbTagRows, err := d.queries.FindArtefactIDsByTagValues(ctx, registry_db.FindArtefactIDsByTagValuesParams{
		P1:        tagKey,
		TagValues: tagValues,
	})
	if err != nil {
		l.ReportError(span, err, "Failed to find artefact IDs by tag values")
		return nil, fmt.Errorf("failed to find artefact IDs for tag %s: %w", tagKey, err)
	}
	artefactIDs := make([]string, len(dbTagRows))
	for i, row := range dbTagRows {
		artefactIDs[i] = row.ArtefactID
	}

	if len(artefactIDs) == 0 {
		l.Trace("No artefacts found for the given tag values")
		return []*registry_dto.ArtefactMeta{}, nil
	}

	l.Trace("Found matching artefact IDs, fetching full data", logger.Int("idCount", len(artefactIDs)))

	return d.GetMultipleArtefacts(ctx, artefactIDs)
}

// FindArtefactByVariantStorageKey finds an artefact by the storage key of one
// of its variants.
//
// Takes storageKey (string) which identifies the variant's storage location.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
// Returns error when the artefact is not found or the query fails.
func (d *DAL) FindArtefactByVariantStorageKey(ctx context.Context, storageKey string) (*registry_dto.ArtefactMeta, error) {
	row, err := d.queries.FindArtefactByVariantStorageKey(ctx, storageKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, registry_domain.ErrArtefactNotFound
		}
		return nil, fmt.Errorf("failed to find artefact by storage key '%s': %w", storageKey, err)
	}
	return d.GetArtefact(ctx, row.ArtefactID)
}

// PopGCHints retrieves and removes garbage collection hints from the store.
//
// Takes limit (int) which specifies the maximum number of hints to retrieve.
// Uses a default limit when limit is zero or negative.
//
// Returns []registry_dto.GCHint which contains the retrieved hints.
// Returns error when the database transaction fails.
func (d *DAL) PopGCHints(ctx context.Context, limit int) ([]registry_dto.GCHint, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.PopGCHints",
		logger.Int("limit", limit),
	)
	defer span.End()

	startTime := time.Now()
	if limit <= 0 {
		limit = defaultGCHintLimit
		l.Trace("Using default limit", logger.Int("defaultLimit", limit))
	}

	var hints []registry_dto.GCHint
	err := l.RunInSpan(ctx, "PopGCHintsTransaction", func(ctx context.Context, _ logger.Logger) error {
		return d.runInTransaction(ctx, func(ctx context.Context, _ registry_db.DBTX, qtx *registry_db.Queries) error {
			dbHints, err := qtx.PopGCHints(ctx, safeconv.IntToInt32(limit))
			if err != nil {
				return fmt.Errorf("failed to pop GC hints from DB: %w", err)
			}

			if len(dbHints) == 0 {
				l.Trace("No GC hints found")
				hints = []registry_dto.GCHint{}
				return nil
			}

			l.Trace("Processing GC hints", logger.Int("hintCount", len(dbHints)))
			var idsToDelete []int32
			hints, idsToDelete = convertDBHintsToDTO(dbHints)
			if err := qtx.DeleteGCHints(ctx, registry_db.DeleteGCHintsParams{IDs: idsToDelete}); err != nil {
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
		logger.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger.Int("hintCount", len(hints)))

	return hints, nil
}

// AtomicUpdate performs a batch of atomic operations within a single
// transaction.
//
// Takes actions ([]registry_dto.AtomicAction) which specifies the operations
// to execute atomically.
//
// Returns error when the transaction fails to begin, an action fails, or the
// commit fails.
func (d *DAL) AtomicUpdate(ctx context.Context, actions []registry_dto.AtomicAction) error {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.AtomicUpdate",
		logger.Int("actionCount", len(actions)),
	)
	defer span.End()

	startTime := time.Now()

	err := l.RunInSpan(ctx, "AtomicUpdateTransaction", func(ctx context.Context, _ logger.Logger) error {
		return d.runInTransaction(ctx, func(ctx context.Context, _ registry_db.DBTX, qtx *registry_db.Queries) error {
			for i, action := range actions {
				l.Trace("Processing atomic action",
					logger.Int("actionIndex", i),
					logger.String("actionType", string(action.Type)))

				if err := d.processAtomicAction(ctx, qtx, action); err != nil {
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
		logger.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger.Int("actionCount", len(actions)))

	return nil
}

// IncrementBlobRefCount atomically increments the reference count for a blob.
// If the blob does not exist, it creates it with a reference count of one.
//
// Takes blob (registry_domain.BlobReference) which identifies the blob to
// increment.
//
// Returns int which is the new reference count after the increment.
// Returns error when the database operation fails.
func (d *DAL) IncrementBlobRefCount(ctx context.Context, blob registry_domain.BlobReference) (int, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.IncrementBlobRefCount",
		logger.String(logKeyStorageKey, blob.StorageKey),
	)
	defer span.End()

	now := time.Now().UTC()
	row, err := d.queries.IncrementBlobRefCount(ctx, registry_db.IncrementBlobRefCountParams{
		P1: blob.StorageKey,
		P2: blob.StorageBackendID,
		P3: blob.ContentHash,
		P4: safeconv.Int64ToInt32(blob.SizeBytes),
		P5: blob.MimeType,
		P6: safeconv.Int64ToInt32(now.Unix()),
		P7: safeconv.Int64ToInt32(now.Unix()),
	})
	if err != nil {
		l.ReportError(span, err, "Failed to increment blob ref count")
		return 0, fmt.Errorf("failed to increment blob ref count for %s: %w", blob.StorageKey, err)
	}

	l.Trace("Incremented blob ref count",
		logger.Int("newRefCount", int(row.RefCount)),
		logger.String(logKeyStorageKey, blob.StorageKey))

	return int(row.RefCount), nil
}

// DecrementBlobRefCount atomically decrements the reference count for a blob
// and indicates whether the blob should be deleted.
//
// Takes storageKey (string) which identifies the blob in storage.
//
// Returns int which is the new reference count after decrementing.
// Returns bool which is true when the blob should be deleted (ref count is 0).
// Returns error when the blob does not exist.
func (d *DAL) DecrementBlobRefCount(ctx context.Context, storageKey string) (int, bool, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.DecrementBlobRefCount",
		logger.String(logKeyStorageKey, storageKey),
	)
	defer span.End()

	now := time.Now().UTC()
	row, err := d.queries.DecrementBlobRefCount(ctx, registry_db.DecrementBlobRefCountParams{
		P1: safeconv.Int64ToInt32(now.Unix()),
		P2: storageKey,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Warn("Attempted to decrement ref count for non-existent blob",
				logger.String(logKeyStorageKey, storageKey))
			return 0, false, registry_domain.ErrBlobReferenceNotFound
		}
		l.ReportError(span, err, "Failed to decrement blob ref count")
		return 0, false, fmt.Errorf("failed to decrement blob ref count for %s: %w", storageKey, err)
	}

	shouldDelete := row.RefCount == 0
	l.Trace("Decremented blob ref count",
		logger.Int("newRefCount", int(row.RefCount)),
		logger.Bool("shouldDelete", shouldDelete),
		logger.String(logKeyStorageKey, storageKey))

	if shouldDelete {
		if err := d.queries.DeleteBlobReferenceIfZero(ctx, storageKey); err != nil {
			l.Warn("Failed to delete blob reference record with zero ref count",
				logger.Error(err),
				logger.String(logKeyStorageKey, storageKey))
		}
	}

	return int(row.RefCount), shouldDelete, nil
}

// GetBlobRefCount returns the current reference count for a blob.
// Returns 0 if the blob does not exist (not an error).
//
// Takes storageKey (string) which identifies the blob to look up.
//
// Returns int which is the reference count for the blob.
// Returns error when the database query fails.
func (d *DAL) GetBlobRefCount(ctx context.Context, storageKey string) (int, error) {
	ctx, l := logger.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DAL.GetBlobRefCount",
		logger.String(logKeyStorageKey, storageKey),
	)
	defer span.End()

	row, err := d.queries.GetBlobRefCount(ctx, storageKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.Trace("Blob reference not found, returning 0",
				logger.String(logKeyStorageKey, storageKey))
			return 0, nil
		}
		l.ReportError(span, err, "Failed to get blob ref count")
		return 0, fmt.Errorf("failed to get blob ref count for %s: %w", storageKey, err)
	}

	l.Trace("Retrieved blob ref count",
		logger.Int("refCount", int(row.RefCount)),
		logger.String(logKeyStorageKey, storageKey))

	return int(row.RefCount), nil
}

// runInTransaction executes fn within a transaction.
//
// If the DAL is already inside a transaction (inTransaction == true), it
// reuses the existing queries and DBTX to avoid deadlocking on SQLite's
// single-writer lock.
//
// Takes fn (func(ctx context.Context, dbtx DBTX, qtx *Queries) error) which
// is the function to execute within the transaction scope.
//
// Returns error when the transaction cannot be started, fn returns an error,
// or the commit fails.
func (d *DAL) runInTransaction(ctx context.Context, fn func(ctx context.Context, dbtx registry_db.DBTX, qtx *registry_db.Queries) error) error {
	if d.inTransaction {
		return fn(ctx, d.dbtx, d.queries)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, tx, d.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// withTransaction is an internal helper that executes a function within a
// database transaction, creating a transaction-scoped DAL clone.
//
// Takes operation (func(ctx context.Context, dal RegistryDAL) error) which is
// the function to execute within the transaction scope.
//
// Returns error when the DAL is not initialised, the transaction cannot be
// started, the function returns an error, or the commit fails.
//
// Panics if operation panics. The transaction is rolled back before
// re-panicking.
func (d *DAL) withTransaction(ctx context.Context, operation func(ctx context.Context, dal registry_dal.RegistryDAL) error) error {
	ctx, l := logger.From(ctx, log)

	if d.db == nil {
		return errDALNotInitialised
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txDAL := NewDALWithTx(tx, d.db)

	if err := operation(ctx, txDAL); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			l.Warn("Failed to rollback transaction", logger.Error(rbErr))
		}
		return err
	}

	return tx.Commit()
}

// processTagQueries finds artefact IDs matching all tag queries using
// intersection.
//
// Takes tagQuery (map[string]string) which maps tag keys to required values.
//
// Returns []string which contains artefact IDs matching all tag criteria, or
// nil if no matches exist.
// Returns error when the database query fails.
func (d *DAL) processTagQueries(
	ctx context.Context,
	tagQuery map[string]string,
) ([]string, error) {
	ctx, l := logger.From(ctx, log)
	var intersection *map[string]struct{}

	for key, value := range tagQuery {
		l.Trace("Processing tag query",
			logger.String("tagKey", key),
			logger.String("tagValue", value))

		rows, err := d.queries.FindArtefactIDsByTag(ctx, registry_db.FindArtefactIDsByTagParams{
			P1: key,
			P2: value,
		})
		if err != nil {
			return nil, fmt.Errorf("tag search failed for %s=%s: %w", key, value, err)
		}

		ids := make([]string, len(rows))
		for i, row := range rows {
			ids[i] = row.ArtefactID
		}

		l.Trace("Found artefacts with tag",
			logger.String("tagKey", key),
			logger.String("tagValue", value),
			logger.Int("resultCount", len(ids)))

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
// Takes qtx (*registry_db.Queries) which provides transactional database
// access.
// Takes action (registry_dto.AtomicAction) which specifies the operation to
// perform.
//
// Returns error when the action fails or the action type is unrecognised.
func (*DAL) processAtomicAction(
	ctx context.Context,
	qtx *registry_db.Queries,
	action registry_dto.AtomicAction,
) error {
	switch action.Type {
	case registry_dto.ActionTypeUpsertArtefact:
		if err := upsertArtefact(ctx, qtx, action.Artefact); err != nil {
			return fmt.Errorf("atomic upsert for artefact '%s' failed: %w", action.Artefact.ID, err)
		}
	case registry_dto.ActionTypeDeleteArtefact:
		if err := qtx.DeleteArtefact(ctx, action.ArtefactID); err != nil {
			return fmt.Errorf("atomic delete for artefact '%s' failed: %w", action.ArtefactID, err)
		}
	case registry_dto.ActionTypeAddGCHints:
		if err := addGCHints(ctx, qtx, action.GCHints); err != nil {
			return fmt.Errorf("atomic add GC hints failed: %w", err)
		}
	default:
		return fmt.Errorf("unrecognised atomic action type: %s", action.Type)
	}
	return nil
}

// upsertArtefact inserts or updates an artefact and its related data.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes art (*registry_dto.ArtefactMeta) which contains the artefact metadata
// to store.
//
// Returns error when the database operation fails.
func upsertArtefact(ctx context.Context, qtx *registry_db.Queries, art *registry_dto.ArtefactMeta) error {
	fbsData := registry_schema.BuildArtefactMeta(art)

	if err := qtx.UpsertArtefact(ctx, registry_db.UpsertArtefactParams{
		P1: art.ID,
		P2: art.SourcePath,
		P3: safeconv.Int64ToInt32(art.CreatedAt.Unix()),
		P4: safeconv.Int64ToInt32(art.UpdatedAt.Unix()),
		P5: fbsData,
	}); err != nil {
		return fmt.Errorf("failed to upsert artefact: %w", err)
	}

	if err := deleteExistingArtefactData(ctx, qtx, art); err != nil {
		return fmt.Errorf("deleting existing artefact data for '%s': %w", art.ID, err)
	}

	if err := insertVariantsWithData(ctx, qtx, art); err != nil {
		return fmt.Errorf("inserting variants for artefact '%s': %w", art.ID, err)
	}

	return insertDesiredProfiles(ctx, qtx, art)
}

// addGCHints stores garbage collection hints for the given storage keys.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes hints ([]registry_dto.GCHint) which contains the storage keys to mark
// for cleanup.
//
// Returns error when a hint cannot be added to the database.
func addGCHints(ctx context.Context, qtx *registry_db.Queries, hints []registry_dto.GCHint) error {
	nowSeconds := safeconv.Int64ToInt32(time.Now().Unix())
	for _, hint := range hints {
		err := qtx.AddGCHint(ctx, registry_db.AddGCHintParams{
			P1: hint.BackendID,
			P2: hint.StorageKey,
			P3: nowSeconds,
		})
		if err != nil {
			return fmt.Errorf("failed to add GC hint for key '%s': %w", hint.StorageKey, err)
		}
	}
	return nil
}

// deleteExistingArtefactData removes all existing data for an artefact before
// re-importing it.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes art (*registry_dto.ArtefactMeta) which identifies the artefact to
// clear.
//
// Returns error when any database deletion fails.
func deleteExistingArtefactData(ctx context.Context, qtx *registry_db.Queries, art *registry_dto.ArtefactMeta) error {
	if err := qtx.DeleteVariantTagsForArtefact(ctx, art.ID); err != nil {
		return fmt.Errorf("failed to delete old variant tags: %w", err)
	}

	for i := range art.ActualVariants {
		v := &art.ActualVariants[i]
		if err := qtx.DeleteChunksForVariant(ctx, registry_db.DeleteChunksForVariantParams{
			P1: art.ID,
			P2: v.VariantID,
		}); err != nil {
			return fmt.Errorf("failed to delete old chunks for variant '%s': %w", v.VariantID, err)
		}
	}

	if err := qtx.DeleteVariantsForArtefact(ctx, art.ID); err != nil {
		return fmt.Errorf("failed to delete old variants: %w", err)
	}

	if err := qtx.DeleteDesiredProfilesForArtefact(ctx, art.ID); err != nil {
		return fmt.Errorf("failed to delete old desired profiles: %w", err)
	}

	return nil
}

// insertVariantsWithData inserts all variants for an artefact along with their
// tags and chunks.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes art (*registry_dto.ArtefactMeta) which holds the variants to insert.
//
// Returns error when inserting a variant, its tags, or its chunks fails.
func insertVariantsWithData(ctx context.Context, qtx *registry_db.Queries, art *registry_dto.ArtefactMeta) error {
	for i := range art.ActualVariants {
		v := &art.ActualVariants[i]
		if err := insertVariant(ctx, qtx, art.ID, v); err != nil {
			return fmt.Errorf("inserting variant '%s': %w", v.VariantID, err)
		}
		if err := insertVariantTags(ctx, qtx, art.ID, v); err != nil {
			return fmt.Errorf("inserting tags for variant '%s': %w", v.VariantID, err)
		}
		if err := insertVariantChunks(ctx, qtx, art.ID, v); err != nil {
			return fmt.Errorf("inserting chunks for variant '%s': %w", v.VariantID, err)
		}
	}
	return nil
}

// insertVariant stores a variant record in the database for the given
// artefact.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes v (*registry_dto.Variant) which contains the variant data to store.
//
// Returns error when the database insert fails.
func insertVariant(ctx context.Context, qtx *registry_db.Queries, artefactID string, v *registry_dto.Variant) error {
	return qtx.InsertVariant(ctx, registry_db.InsertVariantParams{
		P1: artefactID,
		P2: v.VariantID,
		P3: v.StorageKey,
		P4: v.StorageBackendID,
		P5: v.MimeType,
		P6: safeconv.Int64ToInt32(v.SizeBytes),
		P7: string(v.Status),
		P8: safeconv.Int64ToInt32(v.CreatedAt.Unix()),
	})
}

// insertVariantTags stores all metadata tags for a variant in the database.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes v (*registry_dto.Variant) which contains the tags to insert.
//
// Returns error when a tag cannot be inserted.
func insertVariantTags(ctx context.Context, qtx *registry_db.Queries, artefactID string, v *registry_dto.Variant) error {
	for key, value := range v.MetadataTags.All() {
		err := qtx.InsertVariantTag(ctx, registry_db.InsertVariantTagParams{
			P1: artefactID,
			P2: v.VariantID,
			P3: key,
			P4: value,
		})
		if err != nil {
			return fmt.Errorf("failed to insert tag for variant '%s': %w", v.VariantID, err)
		}
	}
	return nil
}

// insertVariantChunks stores all chunks for a variant in the database.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes artefactID (string) which identifies the parent artefact.
// Takes v (*registry_dto.Variant) which contains the chunks to insert.
//
// Returns error when a chunk cannot be inserted.
func insertVariantChunks(ctx context.Context, qtx *registry_db.Queries, artefactID string, v *registry_dto.Variant) error {
	for i := range v.Chunks {
		chunk := &v.Chunks[i]
		err := qtx.InsertVariantChunk(ctx, registry_db.InsertVariantChunkParams{
			P1:  artefactID,
			P2:  v.VariantID,
			P3:  chunk.ChunkID,
			P4:  chunk.StorageKey,
			P5:  chunk.StorageBackendID,
			P6:  safeconv.Int64ToInt32(chunk.SizeBytes),
			P7:  chunk.ContentHash,
			P8:  safeconv.IntToInt32(chunk.SequenceNumber),
			P9:  chunk.MimeType,
			P10: safeconv.Int64ToInt32(chunk.CreatedAt.Unix()),
			P11: chunk.DurationSeconds,
		})
		if err != nil {
			return fmt.Errorf("failed to insert chunk '%s' for variant '%s': %w", chunk.ChunkID, v.VariantID, err)
		}
	}
	return nil
}

// insertDesiredProfiles stores the desired profiles from an artefact into the
// database.
//
// Takes qtx (*registry_db.Queries) which provides database access.
// Takes art (*registry_dto.ArtefactMeta) which contains the profiles to store.
//
// Returns error when a profile cannot be inserted into the database.
func insertDesiredProfiles(ctx context.Context, qtx *registry_db.Queries, art *registry_dto.ArtefactMeta) error {
	for i := range art.DesiredProfiles {
		np := &art.DesiredProfiles[i]
		paramsJSON, _ := json.Marshal(np.Profile.Params)
		tagsJSON, _ := json.Marshal(np.Profile.ResultingTags)
		dependsOnJSON, _ := json.Marshal(np.Profile.DependsOn)
		err := qtx.InsertDesiredProfile(ctx, registry_db.InsertDesiredProfileParams{
			P1: art.ID,
			P2: np.Name,
			P3: np.Profile.CapabilityName,
			P4: string(np.Profile.Priority),
			P5: string(paramsJSON),
			P6: string(tagsJSON),
			P7: string(dependsOnJSON),
		})
		if err != nil {
			return fmt.Errorf("failed to insert desired profile '%s': %w", np.Name, err)
		}
	}
	return nil
}

// intersectIDSets finds the common IDs between the current set and a list of
// IDs. If current is nil, it creates a new set with all the given IDs.
//
// Takes current (*map[string]struct{}) which is the existing ID set to check
// against, or nil to create a new set.
// Takes ids ([]string) which contains the IDs to match or add.
//
// Returns *map[string]struct{} which contains only the IDs found in both
// current and ids, or all ids if current was nil.
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

// convertDBHintsToDTO converts database GC hint rows to DTOs and returns IDs
// for deletion.
//
// Takes dbHints ([]registry_db.PopGCHintsRow) which contains the database rows
// to convert.
//
// Returns []registry_dto.GCHint which contains the converted hint DTOs.
// Returns []int32 which contains the row IDs to delete from the database.
func convertDBHintsToDTO(dbHints []registry_db.PopGCHintsRow) ([]registry_dto.GCHint, []int32) {
	hints := make([]registry_dto.GCHint, len(dbHints))
	idsToDelete := make([]int32, len(dbHints))
	for i, h := range dbHints {
		hints[i] = registry_dto.GCHint{BackendID: h.BackendID, StorageKey: h.StorageKey}
		idsToDelete[i] = h.ID
	}
	return hints, idsToDelete
}

var (
	_ registry_dal.RegistryDALWithTx = (*DAL)(nil)

	_ registry_domain.MetadataStore = (*DAL)(nil)
)
