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

package builtin_detectors

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

const (
	// defaultRepetitionTTL is the default time window for tracking repeated
	// submissions.
	defaultRepetitionTTL = 10 * time.Minute

	// repetitionCachePrefix is the key prefix for exact content hashes.
	repetitionCachePrefix = "sd:rep:"

	// repetitionHashLength is the number of hex characters to use from the
	// SHA-256 hash. 16 hex chars = 64 bits of collision resistance.
	repetitionHashLength = 16
)

// repetitionEntry tracks how many times a content hash has been seen.
type repetitionEntry struct {
	// FirstSeen is when this content hash was first recorded.
	FirstSeen time.Time

	// Count is the number of times this content has been submitted.
	Count int
}

// RepetitionDetector checks for repeated submission content using a
// distributed cache. When no cache is injected, the detector is a no-op.
type RepetitionDetector struct {
	// cache stores content hashes with hit counts.
	cache cache_domain.Cache[string, repetitionEntry]

	// ttl is the time window for tracking repetitions.
	ttl time.Duration

	// ipScoped scopes cache keys by IP for per-client tracking.
	ipScoped bool
}

// NewRepetitionDetector creates a repetition detector.
//
// Takes cache (cache_domain.Cache[string, repetitionEntry]) which stores
// content hashes. Pass nil to disable repetition detection.
// Takes ttl (time.Duration) which is the tracking window. Zero uses the
// default of 10 minutes.
// Takes ipScoped (bool) which scopes tracking per client IP when true.
//
// Returns *RepetitionDetector which is the configured detector.
func NewRepetitionDetector(
	cache cache_domain.Cache[string, repetitionEntry],
	ttl time.Duration,
	ipScoped bool,
) *RepetitionDetector {
	if ttl <= 0 {
		ttl = defaultRepetitionTTL
	}
	return &RepetitionDetector{
		cache:    cache,
		ttl:      ttl,
		ipScoped: ipScoped,
	}
}

// Name returns the detector identifier.
//
// Returns string which is "repetition".
func (*RepetitionDetector) Name() string { return "repetition" }

// Signals returns the signals this detector handles.
//
// Returns []spamdetect_dto.Signal which contains SignalRepetition.
func (*RepetitionDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalRepetition}
}

// Priority returns PriorityHigh. Repetition detection is a cache lookup,
// cheaper than third-party APIs but not free.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*RepetitionDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns DetectorModeSync.
//
// Returns spamdetect_dto.DetectorMode which is DetectorModeSync.
func (*RepetitionDetector) Mode() spamdetect_dto.DetectorMode {
	return spamdetect_dto.DetectorModeSync
}

// Analyse checks whether the submission content has been seen recently.
//
// Takes submission (*spamdetect_dto.Submission) which contains the field values.
// Takes schema (*spamdetect_dto.Schema) which identifies the fields to check.
//
// Returns *spamdetect_dto.DetectorResult which contains the detection result.
// Returns error when the context is cancelled or the cache fails.
func (d *RepetitionDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if submission == nil || schema == nil {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if d.cache == nil {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	fields := schema.FieldsWithSignal(spamdetect_dto.SignalRepetition)
	if len(fields) == 0 {
		return &spamdetect_dto.DetectorResult{Score: 0}, nil
	}

	contentHash := d.hashFieldContent(submission, fields)
	cacheKey := d.buildCacheKey(contentHash, submission.RemoteIP)

	count, err := d.recordAndCount(ctx, cacheKey)
	if err != nil {
		return nil, fmt.Errorf("repetition cache error: %w", err)
	}

	score, reason := d.scoreFromCount(count)

	result := &spamdetect_dto.DetectorResult{
		Score:  score,
		IsSpam: score >= detectorSpamThreshold,
	}

	if reason != "" {
		result.Reasons = []string{reason}
	}

	return result, nil
}

// HealthCheck verifies the cache is accessible.
//
// Returns error when the cache is unavailable.
func (d *RepetitionDetector) HealthCheck(ctx context.Context) error {
	if d.cache == nil {
		return nil
	}

	_, _, err := d.cache.GetIfPresent(ctx, repetitionCachePrefix+"health")
	if err != nil {
		return fmt.Errorf("repetition cache health check failed: %w", err)
	}

	return nil
}

// hashFieldContent returns a truncated SHA-256 hex digest of the
// sorted field key-value pairs.
//
// Takes submission (*spamdetect_dto.Submission) which provides values.
// Takes fields ([]spamdetect_dto.Field) which lists the fields to hash.
//
// Returns string which is the truncated hex digest.
func (d *RepetitionDetector) hashFieldContent(submission *spamdetect_dto.Submission, fields []spamdetect_dto.Field) string {
	keys := make([]string, len(fields))
	for index, field := range fields {
		keys[index] = field.Key
	}
	sort.Strings(keys)

	hasher := sha256.New()
	for _, key := range keys {
		value := submission.FieldString(key)
		hasher.Write([]byte(key))
		hasher.Write([]byte{0})
		hasher.Write([]byte(value))
		hasher.Write([]byte{0})
	}

	fullHash := hex.EncodeToString(hasher.Sum(nil))
	if len(fullHash) > repetitionHashLength {
		return fullHash[:repetitionHashLength]
	}
	return fullHash
}

// buildCacheKey assembles the cache key from the prefix, optional IP
// scope, and content hash.
//
// Takes contentHash (string) which is the content digest.
// Takes remoteIP (string) which is the client IP for scoping.
//
// Returns string which is the assembled cache key.
func (d *RepetitionDetector) buildCacheKey(contentHash string, remoteIP string) string {
	var builder strings.Builder
	builder.WriteString(repetitionCachePrefix)
	if d.ipScoped && remoteIP != "" {
		builder.WriteString(remoteIP)
		builder.WriteByte(':')
	}
	builder.WriteString(contentHash)
	return builder.String()
}

// recordAndCount atomically increments the hit count for cacheKey and
// returns the new total.
//
// Takes cacheKey (string) which is the cache key to increment.
//
// Returns int which is the updated hit count.
// Returns error when the cache operation fails.
func (d *RepetitionDetector) recordAndCount(ctx context.Context, cacheKey string) (int, error) {
	now := time.Now()

	newEntry, _, err := d.cache.ComputeWithTTL(ctx, cacheKey, func(oldValue repetitionEntry, found bool) cache_dto.ComputeResult[repetitionEntry] {
		if found {
			return cache_dto.ComputeResult[repetitionEntry]{
				Value:  repetitionEntry{FirstSeen: oldValue.FirstSeen, Count: oldValue.Count + 1},
				Action: cache_dto.ComputeActionSet,
				TTL:    d.ttl,
			}
		}
		return cache_dto.ComputeResult[repetitionEntry]{
			Value:  repetitionEntry{FirstSeen: now, Count: 1},
			Action: cache_dto.ComputeActionSet,
			TTL:    d.ttl,
		}
	})
	if err != nil {
		return 0, err
	}

	return newEntry.Count, nil
}

// scoreFromCount maps a hit count to a spam score and optional
// human-readable reason.
//
// Takes count (int) which is the number of times the content was seen.
//
// Returns float64 which is the spam score.
// Returns string which is the reason, or empty if benign.
func (*RepetitionDetector) scoreFromCount(count int) (float64, string) {
	switch {
	case count <= 1:
		return 0.0, ""
	case count == 2:
		return detectorSpamThreshold, fmt.Sprintf("identical content submitted %d times", count)
	default:
		return 1.0, fmt.Sprintf("identical content submitted %d times", count)
	}
}
