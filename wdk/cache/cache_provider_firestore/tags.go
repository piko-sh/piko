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

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"piko.sh/piko/wdk/logger"
)

// batchTags splits a slice of tags into groups of at most maxArrayContainsAny
// (30) elements. This is required because Firestore limits array-contains-any
// queries to 30 values per query.
//
// Takes tags ([]string) which is the full list of tags to batch.
//
// Returns [][]string where each inner slice contains at most 30 tags.
func batchTags(tags []string) [][]string {
	if len(tags) == 0 {
		return nil
	}

	batches := make([][]string, 0, (len(tags)+maxArrayContainsAny-1)/maxArrayContainsAny)
	for chunkStart := 0; chunkStart < len(tags); chunkStart += maxArrayContainsAny {
		chunkEnd := min(chunkStart+maxArrayContainsAny, len(tags))
		batches = append(batches, tags[chunkStart:chunkEnd])
	}
	return batches
}

// collectDocIDsByTags queries Firestore for documents matching any of the given
// tags and returns their unique document IDs.
//
// Tags are processed in groups of maxArrayContainsAny (30) to respect the
// Firestore array-contains-any limit.
//
// Returns []string containing the unique document IDs to delete.
// Returns error when a Firestore query fails.
func (a *FirestoreAdapter[K, V]) collectDocIDsByTags(ctx context.Context, tags []string) ([]string, error) {
	seen := make(map[string]bool)
	var docIDs []string

	for _, tagChunk := range batchTags(tags) {
		tagValues := make([]any, len(tagChunk))
		for i, tag := range tagChunk {
			tagValues[i] = tag
		}

		docIter := a.collection.Where(fieldTags, "array-contains-any", tagValues).Documents(ctx)
		for {
			snap, err := docIter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				docIter.Stop()
				return nil, fmt.Errorf("error querying documents by tags: %w", err)
			}
			if !seen[snap.Ref.ID] {
				seen[snap.Ref.ID] = true
				docIDs = append(docIDs, snap.Ref.ID)
			}
		}
		docIter.Stop()
	}

	return docIDs, nil
}

// InvalidateByTags removes all cache entries associated with any of the given
// tags. Tags are stored inline as an array field on each document, so this
// method uses Firestore's array-contains-any query to find matching documents
// and then deletes them via a bulk writer.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes tags (...string) which specifies the tags whose entries should be
// removed.
//
// Returns int which is the number of entries removed.
// Returns error when the operation fails.
func (a *FirestoreAdapter[K, V]) InvalidateByTags(ctx context.Context, tags ...string) (int, error) {
	if len(tags) == 0 {
		return 0, nil
	}

	timeoutCtx, cancel := context.WithTimeoutCause(ctx, a.bulkOperationTimeout,
		fmt.Errorf("firestore InvalidateByTags exceeded %s timeout", a.bulkOperationTimeout))
	defer cancel()

	_, l := logger.From(timeoutCtx, log)

	deleteRefs, err := a.collectDocIDsByTags(timeoutCtx, tags)
	if err != nil {
		return 0, err
	}
	if len(deleteRefs) == 0 {
		return 0, nil
	}

	bulkWriter := a.client.BulkWriter(timeoutCtx)
	defer bulkWriter.End()

	deletedCount := 0
	var jobs []*firestore.BulkWriterJob
	for _, docID := range deleteRefs {
		ref := a.collection.Doc(docID)
		job, writeErr := bulkWriter.Delete(ref)
		if writeErr != nil {
			l.Warn("Failed to enqueue delete in bulk writer for InvalidateByTags",
				logger.String(logKeyField, docID),
				logger.Error(writeErr))
			continue
		}
		jobs = append(jobs, job)
		deletedCount++
	}

	bulkWriter.Flush()

	if err := checkBulkWriterJobs(jobs); err != nil {
		return 0, err
	}

	return deletedCount, nil
}
