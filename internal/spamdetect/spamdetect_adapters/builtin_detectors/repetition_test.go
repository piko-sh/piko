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
	"errors"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
	"piko.sh/piko/wdk/clock"
)

type mockRepetitionCache struct {
	entries map[string]RepetitionEntry
	mu      sync.Mutex
}

func newMockRepetitionCache() *mockRepetitionCache {
	return &mockRepetitionCache{entries: make(map[string]RepetitionEntry)}
}

func (m *mockRepetitionCache) GetIfPresent(_ context.Context, key string) (RepetitionEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, found := m.entries[key]
	return value, found, nil
}

func (m *mockRepetitionCache) ComputeWithTTL(_ context.Context, key string, computeFunction func(oldValue RepetitionEntry, found bool) cache_dto.ComputeResult[RepetitionEntry]) (RepetitionEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldValue, found := m.entries[key]
	result := computeFunction(oldValue, found)
	switch result.Action {
	case cache_dto.ComputeActionSet:
		m.entries[key] = result.Value
	case cache_dto.ComputeActionDelete:
		delete(m.entries, key)
	}
	return result.Value, true, nil
}

func (m *mockRepetitionCache) Get(_ context.Context, _ string, _ cache_dto.Loader[string, RepetitionEntry]) (RepetitionEntry, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) Set(_ context.Context, _ string, _ RepetitionEntry, _ ...string) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) SetWithTTL(_ context.Context, _ string, _ RepetitionEntry, _ time.Duration, _ ...string) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) Invalidate(_ context.Context, _ string) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) Compute(_ context.Context, _ string, _ func(RepetitionEntry, bool) (RepetitionEntry, cache_dto.ComputeAction)) (RepetitionEntry, bool, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) ComputeIfAbsent(_ context.Context, _ string, _ func() RepetitionEntry) (RepetitionEntry, bool, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) ComputeIfPresent(_ context.Context, _ string, _ func(RepetitionEntry) (RepetitionEntry, cache_dto.ComputeAction)) (RepetitionEntry, bool, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) BulkGet(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, RepetitionEntry]) (map[string]RepetitionEntry, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) BulkSet(_ context.Context, _ map[string]RepetitionEntry, _ ...string) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) InvalidateByTags(_ context.Context, _ ...string) (int, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) InvalidateAll(_ context.Context) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) BulkRefresh(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, RepetitionEntry]) {
	panic("not implemented")
}

func (m *mockRepetitionCache) Refresh(_ context.Context, _ string, _ cache_dto.Loader[string, RepetitionEntry]) <-chan cache_dto.LoadResult[RepetitionEntry] {
	panic("not implemented")
}

func (m *mockRepetitionCache) All() iter.Seq2[string, RepetitionEntry] {
	panic("not implemented")
}

func (m *mockRepetitionCache) Keys() iter.Seq[string] {
	panic("not implemented")
}

func (m *mockRepetitionCache) Values() iter.Seq[RepetitionEntry] {
	panic("not implemented")
}

func (m *mockRepetitionCache) GetEntry(_ context.Context, _ string) (cache_dto.Entry[string, RepetitionEntry], bool, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) ProbeEntry(_ context.Context, _ string) (cache_dto.Entry[string, RepetitionEntry], bool, error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) EstimatedSize() int { panic("not implemented") }

func (m *mockRepetitionCache) Stats() cache_dto.Stats { panic("not implemented") }

func (m *mockRepetitionCache) Close(_ context.Context) error { panic("not implemented") }

func (m *mockRepetitionCache) GetMaximum() uint64 { panic("not implemented") }

func (m *mockRepetitionCache) SetMaximum(_ uint64) { panic("not implemented") }

func (m *mockRepetitionCache) WeightedSize() uint64 { panic("not implemented") }

func (m *mockRepetitionCache) SetExpiresAfter(_ context.Context, _ string, _ time.Duration) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) SetRefreshableAfter(_ context.Context, _ string, _ time.Duration) error {
	panic("not implemented")
}

func (m *mockRepetitionCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, RepetitionEntry], error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, RepetitionEntry], error) {
	panic("not implemented")
}

func (m *mockRepetitionCache) SupportsSearch() bool { panic("not implemented") }

func (m *mockRepetitionCache) GetSchema() *cache_dto.SearchSchema { panic("not implemented") }

func TestRepetitionDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	assert.Equal(t, "repetition", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalRepetition}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityHigh, detector.Priority())
}

func TestRepetitionDetector_HealthCheck_NilCache(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	assert.NoError(t, detector.HealthCheck(t.Context()))
}

func TestRepetitionDetector_HealthCheck_WithCache(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	assert.NoError(t, detector.HealthCheck(t.Context()))
}

func TestRepetitionDetector_NilCache_NoOp(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_FirstSubmission_ScoreZero(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
	assert.False(t, result.IsSpam)
}

func TestRepetitionDetector_SecondSubmission_ScoreHalf(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01)
	assert.NotEmpty(t, result.Reasons)
}

func TestRepetitionDetector_ThirdSubmission_ScoreOne(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	for range 2 {
		_, err := detector.Analyse(t.Context(), submission, schema)
		require.NoError(t, err)
	}

	result, err := detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 1.0, result.Score)
	assert.True(t, result.IsSpam)
	assert.NotEmpty(t, result.Reasons)
}

func TestRepetitionDetector_DifferentContent_ScoreZero(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)

	submissionA := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}
	submissionB := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("completely different content"),
		},
	}

	result, err := detector.Analyse(t.Context(), submissionA, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(t.Context(), submissionB, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_IPScoped(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, true)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)

	submissionIP1 := &spamdetect_dto.Submission{
		Fields:   map[string]spamdetect_dto.FieldValue{"message": field("hello world")},
		RemoteIP: "10.0.0.1",
	}
	submissionIP2 := &spamdetect_dto.Submission{
		Fields:   map[string]spamdetect_dto.FieldValue{"message": field("hello world")},
		RemoteIP: "10.0.0.2",
	}

	result, err := detector.Analyse(t.Context(), submissionIP1, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(t.Context(), submissionIP2, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(t.Context(), submissionIP1, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01)
}

func TestRepetitionDetector_NilSubmission(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)

	result, err := detector.Analyse(t.Context(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_NilSchema(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(t.Context(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_ContextCancellation(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := detector.Analyse(ctx, submission, schema)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestRepetitionDetector_NoMatchingFields(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)

	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(t.Context(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_Global_NoIPScope(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)

	submissionA := &spamdetect_dto.Submission{
		Fields:   map[string]spamdetect_dto.FieldValue{"message": field("identical content")},
		RemoteIP: "10.0.0.1",
	}
	submissionB := &spamdetect_dto.Submission{
		Fields:   map[string]spamdetect_dto.FieldValue{"message": field("identical content")},
		RemoteIP: "10.0.0.2",
	}

	result, err := detector.Analyse(t.Context(), submissionA, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(t.Context(), submissionB, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01)
}

func TestRepetitionDetector_DefaultTTL(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	assert.Equal(t, "repetition", detector.Name())
}

func TestRepetitionDetector_NegativeTTL(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, -5*time.Minute, false)
	assert.Equal(t, "repetition", detector.Name())
}

func TestRepetitionDetector_CountResetsAfterQuietWindow(t *testing.T) {
	t.Parallel()

	const window = 10 * time.Minute

	testCases := []struct {
		name      string
		gaps      []time.Duration
		wantCount int
	}{
		{
			name:      "repeats inside the window keep counting",
			gaps:      []time.Duration{time.Minute, time.Minute},
			wantCount: 3,
		},
		{
			name:      "a repeat a full window after the last one starts a new count",
			gaps:      []time.Duration{time.Minute, window},
			wantCount: 1,
		},
		{
			name:      "a repeat long after the last one starts a new count",
			gaps:      []time.Duration{time.Minute, time.Minute, 24 * time.Hour},
			wantCount: 1,
		},
		{
			name:      "steady repeats keep counting past a window since the first",
			gaps:      []time.Duration{window - time.Second, window - time.Second, window - time.Second},
			wantCount: 4,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			detector := NewRepetitionDetector(newMockRepetitionCache(), window, false, WithRepetitionClock(mockClock))

			count, err := detector.recordAndCount(ctx, "sd:rep:content")
			require.NoError(t, err)
			for _, gap := range testCase.gaps {
				mockClock.Advance(gap)
				count, err = detector.recordAndCount(ctx, "sd:rep:content")
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.wantCount, count)
		})
	}
}

func TestRepetitionDetector_NormalisedVariantsCountAsRepeats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		first   string
		variant string
	}{
		{name: "case differs", first: "Buy Cheap Watches", variant: "bUY cHEAP wATCHES"},
		{name: "surrounding whitespace", first: "buy cheap watches", variant: "  buy cheap watches \n"},
		{name: "internal whitespace runs", first: "buy cheap watches", variant: "buy \t cheap\n\nwatches"},
		{name: "decomposed accents", first: "café crème", variant: "café crème"},
		{name: "full case folding", first: "STRASSE", variant: "straße"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			detector := NewRepetitionDetector(newMockRepetitionCache(), 10*time.Minute, false)
			schema := spamdetect_dto.MustNewSchema(
				spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
			)

			first := &spamdetect_dto.Submission{
				Fields: map[string]spamdetect_dto.FieldValue{"message": field(testCase.first)},
			}
			variant := &spamdetect_dto.Submission{
				Fields: map[string]spamdetect_dto.FieldValue{"message": field(testCase.variant)},
			}

			_, err := detector.Analyse(t.Context(), first, schema)
			require.NoError(t, err)

			result, err := detector.Analyse(t.Context(), variant, schema)
			require.NoError(t, err)
			assert.InDelta(t, 0.5, result.Score, 0.01, "a cosmetic variant is the same content")
		})
	}
}

func TestRepetitionDetector_NormalisationKeepsDistinctContentApart(t *testing.T) {
	t.Parallel()

	detector := NewRepetitionDetector(newMockRepetitionCache(), 10*time.Minute, false)
	fields := []spamdetect_dto.Field{{Key: "message"}}

	hashOf := func(value string) string {
		return detector.hashFieldContent(&spamdetect_dto.Submission{
			Fields: map[string]spamdetect_dto.FieldValue{"message": field(value)},
		}, fields)
	}

	assert.NotEqual(t, hashOf("buy cheap watches"), hashOf("buycheapwatches"),
		"removing word breaks changes the content")
	assert.NotEqual(t, hashOf("cafe"), hashOf("café"), "a real accent changes the content")
	assert.Len(t, hashOf("anything"), repetitionHashLength)
}

func TestRepetitionDetector_HashInputIsBoundedBySanitise(t *testing.T) {
	t.Parallel()

	const maxLength = 32

	detector := NewRepetitionDetector(newMockRepetitionCache(), 10*time.Minute, false)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition).MaxLength(maxLength),
	)
	prefix := strings.Repeat("a", maxLength)

	first := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field(prefix + strings.Repeat("x", 1<<20))},
	}
	second := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field(prefix + strings.Repeat("y", 1<<20))},
	}
	first.Sanitise(schema)
	second.Sanitise(schema)

	require.Len(t, first.FieldString("message"), maxLength, "the hashed value is capped before hashing")
	require.True(t, first.WasTruncated())

	_, err := detector.Analyse(t.Context(), first, schema)
	require.NoError(t, err)

	result, err := detector.Analyse(t.Context(), second, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01, "content beyond the field limit is never hashed")
}

func TestRepetitionDetector_BuildCacheKeySanitisesRemoteIP(t *testing.T) {
	t.Parallel()

	detector := NewRepetitionDetector(nil, 0, true)

	testCases := []struct {
		name     string
		remoteIP string
		expected string
	}{
		{name: "IPv4", remoteIP: "203.0.113.7", expected: repetitionCachePrefix + "203.0.113.7:hash"},
		{name: "IPv4Mapped", remoteIP: "::ffff:203.0.113.7", expected: repetitionCachePrefix + "203.0.113.7:hash"},
		{name: "IPv6GroupedByPrefix", remoteIP: "2001:db8:1:2:aaaa::7", expected: repetitionCachePrefix + "2001:db8:1:2:::hash"},
		{name: "Empty", remoteIP: "", expected: repetitionCachePrefix + "hash"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, detector.buildCacheKey("hash", testCase.remoteIP))
		})
	}

	malformed := detector.buildCacheKey("hash", "evil\n:injected")
	assert.NotContains(t, malformed, "\n")
	assert.Len(t, malformed, len(repetitionCachePrefix)+security_dto.RateLimitKeyMaxLength+len(":hash"))
}

func TestRepetitionDetector_IPv6SiblingsShareAKey(t *testing.T) {
	t.Parallel()

	detector := NewRepetitionDetector(newMockRepetitionCache(), 10*time.Minute, true)
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)

	newSubmission := func(remoteIP string) *spamdetect_dto.Submission {
		return &spamdetect_dto.Submission{
			Fields:   map[string]spamdetect_dto.FieldValue{"message": field("identical content")},
			RemoteIP: remoteIP,
		}
	}

	_, err := detector.Analyse(t.Context(), newSubmission("2001:db8:1:2::1"), schema)
	require.NoError(t, err)

	result, err := detector.Analyse(t.Context(), newSubmission("2001:db8:1:2:ffff::2"), schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01, "a rotated address in the same /64 is the same client")

	result, err = detector.Analyse(t.Context(), newSubmission("2001:db8:1:3::1"), schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score, "a different /64 is a different client")
}

func TestRepetitionDetector_HonoursConfiguredIPv6Prefix(t *testing.T) {
	t.Parallel()

	detector := NewRepetitionDetector(newMockRepetitionCache(), 10*time.Minute, true,
		WithRepetitionIPv6PrefixBits(48))
	schema := spamdetect_dto.MustNewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)

	newSubmission := func(remoteIP string) *spamdetect_dto.Submission {
		return &spamdetect_dto.Submission{
			Fields:   map[string]spamdetect_dto.FieldValue{"message": field("identical content")},
			RemoteIP: remoteIP,
		}
	}

	_, err := detector.Analyse(t.Context(), newSubmission("2001:db8:1:2::1"), schema)
	require.NoError(t, err)

	result, err := detector.Analyse(t.Context(), newSubmission("2001:db8:1:3::1"), schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01, "a different /64 inside the same /48 is the same client")

	result, err = detector.Analyse(t.Context(), newSubmission("2001:db8:2::1"), schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score, "a different /48 is a different client")
}

func TestNormaliseRepetitionValue_MatchesCanonicalCaselessForms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		variant string
		plain   string
	}{
		{name: "fullwidth letters", variant: "\uff28\uff45\uff4c\uff4c\uff4f", plain: "hello"},
		{name: "ligature", variant: "\ufb01ne offer", plain: "fine offer"},
		{name: "zero width space", variant: "buy\u200bnow", plain: "buynow"},
		{name: "soft hyphen", variant: "free\u00admoney", plain: "freemoney"},
		{name: "word joiner", variant: "cheap\u2060pills", plain: "cheappills"},
		{name: "zero width space between words", variant: "buy \u200b now", plain: "buy now"},
		{name: "variation selector 16", variant: "spam\ufe0foffer", plain: "spamoffer"},
		{name: "combining grapheme joiner", variant: "free\u034fmoney", plain: "freemoney"},
		{name: "hangul choseong filler", variant: "win\u115fnow", plain: "winnow"},
		{name: "combining accent", variant: "cafe\u0301", plain: "caf\u00e9"},
		{name: "sharp s", variant: "STRASSE", plain: "stra\u00dfe"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				normaliseRepetitionValue(cases.Fold(), testCase.plain),
				normaliseRepetitionValue(cases.Fold(), testCase.variant))
		})
	}
}
