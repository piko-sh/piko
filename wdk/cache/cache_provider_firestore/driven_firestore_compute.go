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

package cache_provider_firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/logger"
)

// fetchValueInTransaction reads and decodes a value within a Firestore
// transaction.
//
// Takes tx (*firestore.Transaction) which provides the transaction context.
// Takes ref (*firestore.DocumentRef) which is the document to read.
//
// Returns V which is the decoded value if found.
// Returns bool which is true if the document exists and is not expired.
// Returns error when the Firestore get or decoding fails.
func (a *FirestoreAdapter[K, V]) fetchValueInTransaction(tx *firestore.Transaction, ref *firestore.DocumentRef) (V, bool, error) {
	snap, err := tx.Get(ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("firestore transaction get failed: %w", err)
	}

	return a.extractValue(snap)
}

// executeComputeAction executes the compute action within a Firestore
// transaction.
//
// Takes tx (*firestore.Transaction) which is the transaction to execute in.
// Takes ref (*firestore.DocumentRef) which is the document to operate on.
// Takes newValue (V) which is the value to set when the action is Set.
// Takes action (cache.ComputeAction) which specifies the operation to perform.
// Takes found (bool) which indicates whether the key exists in the cache.
// Takes ttl (time.Duration) which is the TTL to use; zero uses the default.
//
// Returns error when encoding the value fails.
func (a *FirestoreAdapter[K, V]) executeComputeAction(tx *firestore.Transaction, ref *firestore.DocumentRef, newValue V, action cache.ComputeAction, found bool, ttl time.Duration) error {
	switch action {
	case cache.ComputeActionSet:
		effectiveTTL := a.ttl
		if ttl > 0 {
			effectiveTTL = ttl
		}
		docData, err := a.buildDocumentData(newValue, nil, effectiveTTL)
		if err != nil {
			return err
		}
		return tx.Set(ref, docData)

	case cache.ComputeActionDelete:
		if found {
			return tx.Delete(ref)
		}

	case cache.ComputeActionNoop:
	}
	return nil
}

// Compute atomically updates a cache entry using a compute function with
// Firestore transactions for optimistic concurrency.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which calculates the new value based on
// the current value and whether it exists.
//
// Returns V which is the computed value, or zero value if the operation fails.
// Returns bool which indicates whether the operation succeeded.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) Compute(ctx context.Context, key K, computeFunction func(oldValue V, found bool) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("firestore Compute exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	var resultValue V
	var resultFound bool

	err = a.client.RunTransaction(timeoutCtx, func(_ context.Context, tx *firestore.Transaction) error {
		oldValue, found, getErr := a.fetchValueInTransaction(tx, ref)
		if getErr != nil {
			return getErr
		}

		newValue, action := computeFunction(oldValue, found)

		if err := a.executeComputeAction(tx, ref, newValue, action, found, 0); err != nil {
			return err
		}

		switch action {
		case cache.ComputeActionSet:
			resultValue = newValue
			resultFound = true
		case cache.ComputeActionDelete:
			resultValue = *new(V)
			resultFound = false
		case cache.ComputeActionNoop:
			resultValue = oldValue
			resultFound = found
		}

		return nil
	}, firestore.MaxAttempts(a.maxComputeRetries))

	if err != nil {
		return *new(V), false, fmt.Errorf("firestore Compute transaction failed: %w", err)
	}

	return resultValue, resultFound, nil
}

// tryGetExistingValue attempts to read a non-expired value from a transaction
// snapshot.
//
// Returns V which is the existing value if found and not expired.
// Returns bool which is true if a valid value was found.
// Returns error when the get or decode fails.
func (a *FirestoreAdapter[K, V]) tryGetExistingValue(tx *firestore.Transaction, ref *firestore.DocumentRef) (V, bool, error) {
	snap, getErr := tx.Get(ref)
	if getErr != nil && status.Code(getErr) != codes.NotFound {
		return *new(V), false, fmt.Errorf("firestore transaction get failed: %w", getErr)
	}

	if snap != nil && snap.Exists() {
		value, ok, decodeErr := a.extractValue(snap)
		if decodeErr != nil {
			return *new(V), false, decodeErr
		}
		if ok {
			return value, true, nil
		}
	}

	return *new(V), false, nil
}

// ComputeIfAbsent atomically computes and stores a value only if the key is
// not present.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to check or create.
// Takes computeFunction (func() V) which generates the value if the key is
// absent.
//
// Returns V which is the existing or newly computed value.
// Returns bool which indicates whether computation occurred.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) ComputeIfAbsent(ctx context.Context, key K, computeFunction func() V) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("firestore ComputeIfAbsent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	var resultValue V
	var didCompute bool

	err = a.client.RunTransaction(timeoutCtx, func(_ context.Context, tx *firestore.Transaction) error {
		existing, found, getErr := a.tryGetExistingValue(tx, ref)
		if getErr != nil {
			return getErr
		}
		if found {
			resultValue = existing
			didCompute = false
			return nil
		}

		newValue := computeFunction()
		didCompute = true
		docData, buildErr := a.buildDocumentData(newValue, nil, a.ttl)
		if buildErr != nil {
			return buildErr
		}
		resultValue = newValue
		return tx.Set(ref, docData)
	}, firestore.MaxAttempts(a.maxComputeRetries))

	if err != nil {
		return *new(V), false, fmt.Errorf("firestore ComputeIfAbsent transaction failed: %w", err)
	}

	return resultValue, didCompute, nil
}

// applyComputeActionInTransaction applies a compute action within a Firestore
// transaction and returns the resulting value and presence flag.
//
// Returns V which is the value after the action is applied.
// Returns bool which is true if a value is present after the action.
// Returns error when the transaction operation fails.
func (a *FirestoreAdapter[K, V]) applyComputeActionInTransaction(
	tx *firestore.Transaction,
	ref *firestore.DocumentRef,
	oldValue V,
	newValue V,
	action cache.ComputeAction,
) (V, bool, error) {
	switch action {
	case cache.ComputeActionSet:
		docData, buildErr := a.buildDocumentData(newValue, nil, a.ttl)
		if buildErr != nil {
			return *new(V), false, buildErr
		}
		if err := tx.Set(ref, docData); err != nil {
			return *new(V), false, err
		}
		return newValue, true, nil

	case cache.ComputeActionDelete:
		if err := tx.Delete(ref); err != nil {
			return *new(V), false, err
		}
		return *new(V), false, nil

	default:
		return oldValue, true, nil
	}
}

// ComputeIfPresent atomically updates a value only if the key exists in the
// cache.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the current value and
// returns the new value along with an action indicating whether to update or
// remove.
//
// Returns V which is the resulting value after computation, or the zero value
// if the key was not found or the operation failed.
// Returns bool which is true if the key existed and the computation succeeded.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) ComputeIfPresent(ctx context.Context, key K, computeFunction func(oldValue V) (newValue V, action cache.ComputeAction)) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("firestore ComputeIfPresent exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	var resultValue V
	var resultFound bool

	err = a.client.RunTransaction(timeoutCtx, func(_ context.Context, tx *firestore.Transaction) error {
		oldValue, found, getErr := a.fetchValueInTransaction(tx, ref)
		if getErr != nil {
			return getErr
		}
		if !found {
			resultValue = *new(V)
			resultFound = false
			return nil
		}

		newValue, action := computeFunction(oldValue)
		val, present, actionErr := a.applyComputeActionInTransaction(tx, ref, oldValue, newValue, action)
		if actionErr != nil {
			return actionErr
		}
		resultValue = val
		resultFound = present
		return nil
	}, firestore.MaxAttempts(a.maxComputeRetries))

	if err != nil {
		return *new(V), false, fmt.Errorf("firestore ComputeIfPresent transaction failed: %w", err)
	}

	return resultValue, resultFound, nil
}

// ComputeWithTTL atomically computes a new value with per-call TTL control.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes key (K) which identifies the cache entry to update.
// Takes computeFunction (func(...)) which receives the old value and found
// flag, returning a ComputeResult containing the new value, action, and
// optional TTL.
//
// Returns V which is the resulting value after the operation.
// Returns bool which indicates whether a value is now present.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) ComputeWithTTL(ctx context.Context, key K, computeFunction func(oldValue V, found bool) cache.ComputeResult[V]) (V, bool, error) {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.atomicOperationTimeout, fmt.Errorf("firestore ComputeWithTTL exceeded %s timeout", a.atomicOperationTimeout))
	defer cancel()

	ref, err := a.docRef(key)
	if err != nil {
		return *new(V), false, fmt.Errorf(errFmtEncodeKey, err)
	}

	var resultValue V
	var resultFound bool

	err = a.client.RunTransaction(timeoutCtx, func(_ context.Context, tx *firestore.Transaction) error {
		oldValue, found, getErr := a.fetchValueInTransaction(tx, ref)
		if getErr != nil {
			return getErr
		}

		result := computeFunction(oldValue, found)

		if err := a.executeComputeAction(tx, ref, result.Value, result.Action, found, result.TTL); err != nil {
			return err
		}

		switch result.Action {
		case cache.ComputeActionSet:
			resultValue = result.Value
			resultFound = true
		case cache.ComputeActionDelete:
			resultValue = *new(V)
			resultFound = false
		case cache.ComputeActionNoop:
			resultValue = oldValue
			resultFound = found
		}

		return nil
	}, firestore.MaxAttempts(a.maxComputeRetries))

	if err != nil {
		return *new(V), false, fmt.Errorf("firestore ComputeWithTTL transaction failed: %w", err)
	}

	return resultValue, resultFound, nil
}

// buildDocRefs builds document references for the given keys and returns a
// reverse map from document ID to original key.
//
// Returns the document references and the ID-to-key mapping.
func (a *FirestoreAdapter[K, V]) buildDocRefs(keys []K, l logger.Logger) ([]*firestore.DocumentRef, map[string]K) {
	refs := make([]*firestore.DocumentRef, 0, len(keys))
	refKeyMap := make(map[string]K, len(keys))
	for _, key := range keys {
		ref, err := a.docRef(key)
		if err != nil {
			l.Warn(errMessageEncodeKey, logger.Error(err))
			continue
		}
		refs = append(refs, ref)
		refKeyMap[ref.ID] = key
	}
	return refs, refKeyMap
}

// classifySnapshot classifies a document snapshot as a hit or a miss.
// When the snapshot contains a valid, non-expired value the key-value pair is
// added to results and true is returned. Otherwise the original key is returned
// as a miss with false.
func (a *FirestoreAdapter[K, V]) classifySnapshot(
	snap *firestore.DocumentSnapshot,
	refKeyMap map[string]K,
	results map[K]V,
	l logger.Logger,
) (missKey K, isMiss bool) {
	originalKey, ok := refKeyMap[snap.Ref.ID]
	if !ok {
		return *new(K), false
	}
	if !snap.Exists() {
		return originalKey, true
	}
	value, valueOK, decodeErr := a.extractValue(snap)
	if decodeErr != nil {
		l.Warn("Failed to decode value in BulkGet",
			logger.String(logKeyField, snap.Ref.ID), logger.Error(decodeErr))
		return originalKey, true
	}
	if !valueOK {
		return originalKey, true
	}
	results[originalKey] = value
	return *new(K), false
}

// fetchChunkedSnapshots retrieves documents in chunks and classifies each as a
// hit (added to results) or miss (appended to misses).
//
// Returns []K containing keys that were not found or expired.
// Returns error when a Firestore GetAll call fails.
func (a *FirestoreAdapter[K, V]) fetchChunkedSnapshots(
	ctx context.Context,
	refs []*firestore.DocumentRef,
	refKeyMap map[string]K,
	results map[K]V,
	l logger.Logger,
) ([]K, error) {
	var misses []K
	for chunkStart := 0; chunkStart < len(refs); chunkStart += a.batchSize {
		chunkEnd := min(chunkStart+a.batchSize, len(refs))
		snaps, err := a.client.GetAll(ctx, refs[chunkStart:chunkEnd])
		if err != nil {
			return misses, fmt.Errorf("firestore GetAll failed: %w", err)
		}
		for _, snap := range snaps {
			if missKey, isMiss := a.classifySnapshot(snap, refKeyMap, results, l); isMiss {
				misses = append(misses, missKey)
			}
		}
	}
	return misses, nil
}

// BulkGet retrieves multiple values from the cache, loading missing ones via
// the bulk loader.
//
// Takes keys ([]K) which specifies the cache keys to retrieve.
// Takes bulkLoader (BulkLoader[K, V]) which loads values for any cache misses.
//
// Returns map[K]V which contains the retrieved and loaded values.
// Returns error when the Firestore GetAll operation or bulk loader fails.
func (a *FirestoreAdapter[K, V]) BulkGet(ctx context.Context, keys []K, bulkLoader cache.BulkLoader[K, V]) (map[K]V, error) {
	ctx, l := logger.From(ctx, log)
	if len(keys) == 0 {
		return make(map[K]V), nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout,
		fmt.Errorf("firestore BulkGet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	results := make(map[K]V, len(keys))
	refs, refKeyMap := a.buildDocRefs(keys, l)
	if len(refs) == 0 {
		return results, nil
	}

	misses, err := a.fetchChunkedSnapshots(timeoutCtx, refs, refKeyMap, results, l)
	if err != nil {
		return results, err
	}

	if len(misses) > 0 {
		loaded, loadErr := bulkLoader.BulkLoad(ctx, misses)
		if loadErr != nil {
			return results, fmt.Errorf("bulk loader failed: %w", loadErr)
		}
		if len(loaded) > 0 {
			a.storeLoadedValues(timeoutCtx, loaded, results)
		}
	}

	return results, nil
}

// storeLoadedValues stores loaded values to Firestore using a bulk writer and
// updates the results map.
//
// Takes loaded (map[K]V) which holds the values fetched from the loader.
// Takes results (map[K]V) which is updated with entries that were stored.
func (a *FirestoreAdapter[K, V]) storeLoadedValues(ctx context.Context, loaded map[K]V, results map[K]V) {
	_, l := logger.From(ctx, log)

	bulkWriter := a.client.BulkWriter(ctx)
	defer bulkWriter.End()

	var jobs []*firestore.BulkWriterJob

	for k, v := range loaded {
		ref, err := a.docRef(k)
		if err != nil {
			l.Warn("Failed to encode key for loaded value, skipping", logger.Error(err))
			continue
		}

		docData, err := a.buildDocumentData(v, nil, a.ttl)
		if err != nil {
			l.Warn("Failed to build document data for loaded value",
				logger.String(logKeyField, ref.ID),
				logger.Error(err))
			continue
		}

		job, err := bulkWriter.Set(ref, docData)
		if err != nil {
			l.Warn("Failed to enqueue set in bulk writer after bulk load",
				logger.String(logKeyField, ref.ID),
				logger.Error(err))
			continue
		}
		jobs = append(jobs, job)
		results[k] = v
	}

	bulkWriter.Flush()

	if err := checkBulkWriterJobs(jobs); err != nil {
		l.Warn("Some bulk writer operations failed during load", logger.Error(err))
	}
}

// BulkSet stores multiple key-value pairs in the cache using a bulk writer for
// efficiency.
//
// Takes items (map[K]V) which contains the key-value pairs to store.
// Takes tags (...string) which specifies optional tags to associate with the
// keys.
//
// Returns error when the bulk write fails.
func (a *FirestoreAdapter[K, V]) BulkSet(ctx context.Context, items map[K]V, tags ...string) error {
	_, l := logger.From(ctx, log)
	if len(items) == 0 {
		return nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout, fmt.Errorf("firestore BulkSet exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	bulkWriter := a.client.BulkWriter(timeoutCtx)
	defer bulkWriter.End()

	var jobs []*firestore.BulkWriterJob

	for key, value := range items {
		if timeoutCtx.Err() != nil {
			return timeoutCtx.Err()
		}

		ref, err := a.docRef(key)
		if err != nil {
			l.Warn(errMessageEncodeKey, logger.Error(err))
			continue
		}

		ttl := a.ttl
		if a.expiryCalculator != nil {
			entry := cache.Entry[K, V]{
				Key: key, Value: value, SnapshotAtNano: time.Now().UnixNano(),
			}
			ttl = a.expiryCalculator.ExpireAfterCreate(entry)
		}

		docData, err := a.buildDocumentData(value, tags, ttl)
		if err != nil {
			l.Warn("Failed to build document data in BulkSet",
				logger.String(logKeyField, ref.ID),
				logger.Error(err))
			continue
		}

		job, err := bulkWriter.Set(ref, docData)
		if err != nil {
			l.Warn("Failed to enqueue set in bulk writer for BulkSet",
				logger.String(logKeyField, ref.ID),
				logger.Error(err))
			continue
		}
		jobs = append(jobs, job)
	}

	bulkWriter.Flush()

	return checkBulkWriterJobs(jobs)
}

// InvalidateAll removes all entries from the cache namespace by iterating all
// documents and deleting them via a bulk writer.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) InvalidateAll(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.flushTimeout, fmt.Errorf("firestore InvalidateAll exceeded %s timeout", a.flushTimeout))
	defer cancel()

	_, l := logger.From(timeoutCtx, log)
	l.Internal("Invalidating all entries in Firestore namespace",
		logger.String("namespace", a.namespace))

	docIter := a.collection.Documents(timeoutCtx)
	defer docIter.Stop()

	bulkWriter := a.client.BulkWriter(timeoutCtx)
	defer bulkWriter.End()

	deletedCount := 0
	var jobs []*firestore.BulkWriterJob

	for {
		snap, err := docIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("error iterating documents in InvalidateAll: %w", err)
		}

		job, deleteErr := bulkWriter.Delete(snap.Ref)
		if deleteErr != nil {
			return fmt.Errorf("firestore bulk delete failed for document %q: %w", snap.Ref.ID, deleteErr)
		}
		jobs = append(jobs, job)
		deletedCount++
	}

	bulkWriter.Flush()

	if err := checkBulkWriterJobs(jobs); err != nil {
		return err
	}

	l.Internal("InvalidateAll completed",
		logger.String("namespace", a.namespace),
		logger.Int("documents_deleted", deletedCount))

	return nil
}

// checkBulkWriterJobs inspects the results of all enqueued BulkWriter
// operations after Flush. Returns the first error encountered, if any.
func checkBulkWriterJobs(jobs []*firestore.BulkWriterJob) error {
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return fmt.Errorf("firestore bulk write operation failed: %w", err)
		}
	}
	return nil
}
