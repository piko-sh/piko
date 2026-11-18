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
	"slices"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultRepetitionTTL is the default time window for tracking repeated submissions.
	defaultRepetitionTTL = 10 * time.Minute

	// repetitionCachePrefix is the key prefix for exact content hashes.
	repetitionCachePrefix = "sd:rep:"

	// repetitionHashLength is the number of hex characters to use from the SHA-256 hash. 16
	// hex chars = 64 bits of collision resistance.
	repetitionHashLength = 16
)

// RepetitionEntry tracks how many times a content hash has been seen.
type RepetitionEntry struct {
	// FirstSeen is when this content hash was first recorded.
	FirstSeen time.Time

	// LastSeen is when this content hash was most recently recorded.
	LastSeen time.Time

	// Count is the number of times this content has been submitted since FirstSeen, with no
	// gap between submissions as long as the tracking window.
	Count int
}

// RepetitionDetector checks for repeated submission content using a distributed cache.
// When no cache is injected, the detector is a no-op.
type RepetitionDetector struct {
	// cache stores content hashes with hit counts.
	cache cache_domain.Cache[string, RepetitionEntry]

	// clock supplies the time source for FirstSeen timestamps.
	clock clock.Clock

	// ttl is the time window for tracking repetitions.
	ttl time.Duration

	// ipv6PrefixBits is how many leading bits of an IPv6 client address form the IP scope.
	// Zero keeps the shared default.
	ipv6PrefixBits int

	// ipScoped scopes cache keys by IP for per-client tracking.
	ipScoped bool
}

// RepetitionOption configures a RepetitionDetector.
type RepetitionOption func(*RepetitionDetector)

// WithRepetitionIPv6PrefixBits sets how many leading bits of an IPv6 client address form
// the IP scope, so every address in one allocation shares a count. Values outside 1 to
// 128 keep the default of security_dto.DefaultIPv6RateLimitPrefixBits.
//
// Takes bits (int) which is the prefix length to keep.
//
// Returns RepetitionOption which applies the prefix length.
func WithRepetitionIPv6PrefixBits(bits int) RepetitionOption {
	return func(d *RepetitionDetector) {
		d.ipv6PrefixBits = bits
	}
}

// WithRepetitionClock sets the clock source used for FirstSeen timestamps. Tests inject a
// mock clock for deterministic behaviour.
//
// Takes c (clock.Clock) which provides the time source.
//
// Returns RepetitionOption which configures the clock.
func WithRepetitionClock(c clock.Clock) RepetitionOption {
	return func(d *RepetitionDetector) {
		if c != nil {
			d.clock = c
		}
	}
}

// NewRepetitionDetector creates a repetition detector.
//
// Takes cache (cache_domain.Cache[string, RepetitionEntry]) which stores content hashes.
// Pass nil to disable repetition detection. Build it with PerEntryExpiration so hashes
// that stop recurring are dropped; counts reset after a quiet window either way.
// Takes ttl (time.Duration) which is the tracking window. Zero uses the default of 10
// minutes.
// Takes ipScoped (bool) which scopes tracking per client IP when true.
// Takes opts (...RepetitionOption) which override the detector defaults (notably the
// clock source for tests).
//
// Returns *RepetitionDetector which is the configured detector.
func NewRepetitionDetector(
	cache cache_domain.Cache[string, RepetitionEntry],
	ttl time.Duration,
	ipScoped bool,
	opts ...RepetitionOption,
) *RepetitionDetector {
	if ttl <= 0 {
		ttl = defaultRepetitionTTL
	}
	detector := &RepetitionDetector{
		cache:          cache,
		clock:          clock.RealClock(),
		ttl:            ttl,
		ipv6PrefixBits: 0,
		ipScoped:       ipScoped,
	}
	for _, opt := range opts {
		opt(detector)
	}
	return detector
}

// Name returns the detector identifier.
//
// Returns string which is "repetition".
func (*RepetitionDetector) Name() string { return "repetition" }

// Signals returns the spam detection signals handled.
//
// Returns []spamdetect_dto.Signal containing SignalRepetition.
func (*RepetitionDetector) Signals() []spamdetect_dto.Signal {
	return []spamdetect_dto.Signal{spamdetect_dto.SignalRepetition}
}

// Priority returns the execution tier. Repetition detection is a cache lookup, cheaper
// than third-party APIs but not free.
//
// Returns spamdetect_dto.DetectorPriority which is PriorityHigh.
func (*RepetitionDetector) Priority() spamdetect_dto.DetectorPriority {
	return spamdetect_dto.PriorityHigh
}

// Mode returns the execution mode.
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
		return spamdetect_dto.NewCleanDetectorResult(), nil
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("repetition analysis: %w", err)
	}

	if d.cache == nil {
		return spamdetect_dto.NewCleanDetectorResult(), nil
	}

	fields := schema.FieldsWithSignal(spamdetect_dto.SignalRepetition)
	if len(fields) == 0 {
		return spamdetect_dto.NewCleanDetectorResult(), nil
	}

	contentHash := d.hashFieldContent(submission, fields)
	cacheKey := d.buildCacheKey(contentHash, submission.RemoteIP)

	count, err := d.recordAndCount(ctx, cacheKey)
	if err != nil {
		return nil, fmt.Errorf("repetition cache error: %w", err)
	}

	score, reason := d.scoreFromCount(count)

	result := spamdetect_dto.NewReasonDetectorResult(score, score >= detectorSpamThreshold, nil)

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

// hashFieldContent returns a truncated SHA-256 hex digest of the sorted field key-value
// pairs.
//
// Each value is normalised first, so changes in case, spacing or Unicode form do not hide
// a repeat. Values are already capped to their schema MaxLength by Submission.Sanitise,
// so the work done here is bounded.
//
// Takes submission (*spamdetect_dto.Submission) which provides values.
// Takes fields ([]spamdetect_dto.Field) which lists the fields to hash.
//
// Returns string which is the truncated hex digest.
func (*RepetitionDetector) hashFieldContent(submission *spamdetect_dto.Submission, fields []spamdetect_dto.Field) string {
	keys := make([]string, len(fields))
	for index, field := range fields {
		keys[index] = field.Key
	}
	slices.Sort(keys)

	caser := cases.Fold()
	hasher := sha256.New()
	var separator = []byte{0}
	for _, key := range keys {
		value := normaliseRepetitionValue(caser, submission.FieldString(key))
		_, _ = hasher.Write([]byte(key))
		_, _ = hasher.Write(separator)
		_, _ = hasher.Write([]byte(value))
		_, _ = hasher.Write(separator)
	}

	fullHash := hex.EncodeToString(hasher.Sum(nil))
	if len(fullHash) > repetitionHashLength {
		return fullHash[:repetitionHashLength]
	}
	return fullHash
}

// buildCacheKey assembles the cache key from the prefix, optional IP scope, and content
// hash. The IP is sanitised so a malformed value cannot shape the key and every address
// in one IPv6 /64 shares a scope.
//
// Takes contentHash (string) which is the content digest.
// Takes remoteIP (string) which is the client IP for scoping.
//
// Returns string which is the assembled cache key.
func (d *RepetitionDetector) buildCacheKey(contentHash string, remoteIP string) string {
	var builder strings.Builder
	builder.WriteString(repetitionCachePrefix)
	if d.ipScoped && remoteIP != "" {
		builder.WriteString(security_dto.SanitiseRateLimitKey(remoteIP, security_dto.WithIPv6PrefixBits(d.ipv6PrefixBits)))
		builder.WriteByte(':')
	}
	builder.WriteString(contentHash)
	return builder.String()
}

// recordAndCount atomically increments the hit count for cacheKey.
//
// A submission at or after the tracking window starts a new count. The detector resets
// the count itself because its cache may keep entries after expiry.
//
// Takes cacheKey (string) which is the cache key to increment.
//
// Returns int which is the updated hit count.
// Returns error when the cache operation fails.
func (d *RepetitionDetector) recordAndCount(ctx context.Context, cacheKey string) (int, error) {
	now := d.clock.Now()

	newEntry, _, err := d.cache.ComputeWithTTL(ctx, cacheKey, func(oldValue RepetitionEntry, found bool) cache_dto.ComputeResult[RepetitionEntry] {
		entry := RepetitionEntry{FirstSeen: now, LastSeen: now, Count: 1}
		if found && d.seenWithinWindow(oldValue, now) {
			entry = RepetitionEntry{FirstSeen: oldValue.FirstSeen, LastSeen: now, Count: oldValue.Count + 1}
		}
		return cache_dto.ComputeResult[RepetitionEntry]{
			Value:  entry,
			Action: cache_dto.ComputeActionSet,
			TTL:    d.ttl,
		}
	})
	if err != nil {
		return 0, err
	}

	return newEntry.Count, nil
}

// seenWithinWindow reports whether the content was last recorded less than a tracking
// window before now.
//
// Takes entry (RepetitionEntry) which is the stored record for the content.
// Takes now (time.Time) which is the time of the current submission.
//
// Returns bool which is true when the current submission continues the count.
func (d *RepetitionDetector) seenWithinWindow(entry RepetitionEntry, now time.Time) bool {
	return now.Sub(entry.LastSeen) < d.ttl
}

// scoreFromCount maps a hit count to a spam score and optional human-readable reason.
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

// normaliseRepetitionValue reduces a field value to the form compared for repeats.
//
// The value is matched caselessly in the canonical way, NFKC(fold(NFD(value))), so
// fullwidth letters, ligatures, accents written as combining marks and case variants all
// reduce to one form. Characters that render as nothing, such as U+200B ZERO WIDTH SPACE,
// U+00AD SOFT HYPHEN and U+2060 WORD JOINER, are then removed so they cannot be slipped
// in to make a repeat look new. Last, leading and trailing space is removed and every run
// of whitespace becomes one space.
//
// Takes caser (cases.Caser) which folds case; it holds state, so each caller passes its
// own.
// Takes value (string) which is the raw field value.
//
// Returns string which is the normalised value.
func normaliseRepetitionValue(caser cases.Caser, value string) string {
	folded := norm.NFKC.String(caser.String(norm.NFD.String(value)))
	visible := strings.Map(dropInvisibleRune, folded)

	return strings.Join(strings.Fields(visible), " ")
}

// dropInvisibleRune removes a rune that renders as nothing, for use with strings.Map.
//
// Takes r (rune) which is the rune to check.
//
// Returns rune which is r, or -1 to drop a format character, a variation selector or any
// other default ignorable code point.
func dropInvisibleRune(r rune) rune {
	if unicode.In(r, unicode.Cf, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector) {
		return -1
	}
	return r
}
