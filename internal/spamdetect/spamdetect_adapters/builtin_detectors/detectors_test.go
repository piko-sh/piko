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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

func field(value string) spamdetect_dto.FieldValue {
	return spamdetect_dto.FieldValue{Value: value, Type: spamdetect_dto.FieldTypeText}
}

func TestHoneypotDetector_Filled(t *testing.T) {
	t.Parallel()
	detector := NewHoneypotDetector()
	schema := spamdetect_dto.NewSchema(spamdetect_dto.Honeypot("_hp"))
	submission := &spamdetect_dto.Submission{HoneypotValue: "bot-filled"}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 1.0, result.Score)
	assert.True(t, result.IsSpam)
	assert.Equal(t, spamdetect_dto.PriorityCritical, detector.Priority())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
}

func TestHoneypotDetector_Empty(t *testing.T) {
	t.Parallel()
	detector := NewHoneypotDetector()
	submission := &spamdetect_dto.Submission{HoneypotValue: ""}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestGibberishDetector_RandomString(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0.6, nil)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.NameField("name", spamdetect_dto.SignalGibberish))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"name": {Value: "eIHZuXpkzPeauGviuFQnsgjx hglGLKJzsBPiNAUXzkTr", Type: spamdetect_dto.FieldTypeName},
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.3)
	assert.NotEmpty(t, result.FieldReasons["name"])
}

func TestGibberishDetector_NormalText(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0.6, nil)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.NameField("name", spamdetect_dto.SignalGibberish))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"name": {Value: "John Smith", Type: spamdetect_dto.FieldTypeName}},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Less(t, result.Score, 0.5)
}

func TestGibberishDetector_PerSchemaThreshold(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0.6, nil)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish),
		spamdetect_dto.DetectorConfig("gibberish", map[string]any{"threshold": 0.9}),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("eIHZuXpkzPeauGviuFQnsgjx")},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)

	assert.Less(t, result.Score, 0.8)
}

func TestGibberishDetector_EmptyFields(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0.6, nil)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("name", spamdetect_dto.SignalGibberish))
	submission := &spamdetect_dto.Submission{Fields: map[string]spamdetect_dto.FieldValue{}}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestLinkDensityDetector_ManyLinks(t *testing.T) {
	t.Parallel()
	detector := NewLinkDensityDetector(3)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalLinkDensity))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("Visit https://spam.com and https://spam2.com and https://spam3.com and https://spam4.com"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.5)
	assert.NotEmpty(t, result.FieldReasons["message"])
}

func TestLinkDensityDetector_NoLinks(t *testing.T) {
	t.Parallel()
	detector := NewLinkDensityDetector(3)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalLinkDensity))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("No links here")},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestBlocklistDetector_Match(t *testing.T) {
	t.Parallel()
	detector, err := NewBlocklistDetector([]string{`(?i)buy\s+now`})
	require.NoError(t, err)

	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalBlocklist))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("Please Buy Now")},
	}

	result, analyseErr := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.Equal(t, 1.0, result.Score)
	assert.True(t, result.IsSpam)
	assert.NotEmpty(t, result.FieldReasons["message"])
}

func TestBlocklistDetector_NoMatch(t *testing.T) {
	t.Parallel()
	detector, err := NewBlocklistDetector([]string{`(?i)buy\s+now`})
	require.NoError(t, err)

	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalBlocklist))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("Normal message")},
	}

	result, analyseErr := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.Equal(t, 0.0, result.Score)
}

func TestBlocklistDetector_InvalidPattern(t *testing.T) {
	t.Parallel()
	_, err := NewBlocklistDetector([]string{"[invalid"})
	assert.Error(t, err)
}

func TestBlocklistDetector_TooManyPatterns(t *testing.T) {
	t.Parallel()
	patterns := make([]string, maxBlocklistPatterns+1)
	for index := range patterns {
		patterns[index] = "test"
	}
	_, err := NewBlocklistDetector(patterns)
	assert.Error(t, err)
}

func TestTimingDetector_InstantSubmission(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	now := time.Now()
	submission := &spamdetect_dto.Submission{
		FormLoadedAt:    now,
		FormSubmittedAt: now.Add(100 * time.Millisecond),
	}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 1.0, result.Score)
	assert.True(t, result.IsSpam)
	assert.Equal(t, spamdetect_dto.PriorityCritical, detector.Priority())
}

func TestTimingDetector_NormalTiming(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	now := time.Now()
	submission := &spamdetect_dto.Submission{
		FormLoadedAt:    now.Add(-10 * time.Second),
		FormSubmittedAt: now,
	}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestTimingDetector_MissingTimestamps(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	submission := &spamdetect_dto.Submission{}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRealSpamExample_EndToEnd(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.NameField("firstname", spamdetect_dto.SignalGibberish),
		spamdetect_dto.NameField("lastname", spamdetect_dto.SignalGibberish),
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish, spamdetect_dto.SignalLinkDensity, spamdetect_dto.SignalBlocklist),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"firstname": {Value: "eIHZuXpkzPeauGviuFQnsgjx", Type: spamdetect_dto.FieldTypeName},
			"lastname":  {Value: "hglGLKJzsBPiNAUXzkTr", Type: spamdetect_dto.FieldTypeName},
			"message":   {Value: "MtIjkwxgOZvImDuJbzxmn", Type: spamdetect_dto.FieldTypeText},
		},
	}

	gibberishDetector := NewGibberishDetector(0.6, nil)
	result, err := gibberishDetector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.3, "real spam should score above 0.3 on gibberish")
}

func TestGibberishRatio_RandomString(t *testing.T) {
	t.Parallel()
	ratio, analysed := fallbackGibberishRatio("eIHZuXpkzPeauGviuFQnsgjx")
	assert.True(t, analysed)
	assert.Greater(t, ratio, 0.5)
}

func TestGibberishRatio_NormalText(t *testing.T) {
	t.Parallel()
	ratio, analysed := fallbackGibberishRatio("Hello there I would like to arrange a viewing")
	assert.True(t, analysed)
	assert.Less(t, ratio, 0.5)
}

func TestGibberishRatio_ShortText(t *testing.T) {
	t.Parallel()
	_, analysed := fallbackGibberishRatio("Hi")
	assert.False(t, analysed)
}

func TestFieldTypedBuilders(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.EmailField("email", spamdetect_dto.SignalBlocklist),
		spamdetect_dto.PhoneField("phone"),
		spamdetect_dto.NameField("name", spamdetect_dto.SignalGibberish),
		spamdetect_dto.URLField("website", spamdetect_dto.SignalBlocklist),
		spamdetect_dto.TypedField("postcode", "postcode", spamdetect_dto.SignalBlocklist),
	)

	emailFields := schema.FieldsWithType(spamdetect_dto.FieldTypeEmail)
	assert.Len(t, emailFields, 1)
	assert.Equal(t, "email", emailFields[0].Key)

	phoneFields := schema.FieldsWithType(spamdetect_dto.FieldTypePhone)
	assert.Len(t, phoneFields, 1)

	nameFields := schema.FieldsWithType(spamdetect_dto.FieldTypeName)
	assert.Len(t, nameFields, 1)

	customFields := schema.FieldsWithType("postcode")
	assert.Len(t, customFields, 1)
}

func TestDetectorWeightOnSchema(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.DetectorWeight("akismet", 3.0),
		spamdetect_dto.DetectorWeight("blocklist", 0.5),
	)

	assert.InDelta(t, 3.0, schema.GetDetectorWeight("akismet"), 0.01)
	assert.InDelta(t, 0.5, schema.GetDetectorWeight("blocklist"), 0.01)
	assert.InDelta(t, 0.0, schema.GetDetectorWeight("unknown"), 0.01)
}

func TestDetectorConfigOnSchema(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.DetectorConfig("gibberish", map[string]any{"threshold": 0.8}),
	)

	opts := schema.DetectorOptions("gibberish")
	assert.NotNil(t, opts)
	assert.Equal(t, 0.8, opts["threshold"])

	assert.Nil(t, schema.DetectorOptions("unknown"))
}

func TestLanguageOnSchema_Single(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(spamdetect_dto.Language("en"))
	assert.Equal(t, []string{"en"}, schema.GetLanguages())
}

func TestLanguageOnSchema_Multiple(t *testing.T) {
	t.Parallel()
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.Language("en"),
		spamdetect_dto.Language("fr"),
	)
	assert.Equal(t, []string{"en", "fr"}, schema.GetLanguages())
}

func TestShortCircuit_HoneypotSkipsLaterTiers(t *testing.T) {
	t.Parallel()

	honeypot := NewHoneypotDetector()
	timing := NewTimingDetector(2 * time.Second)
	gibberish := NewGibberishDetector(0.6, nil)
	linkDensity := NewLinkDensityDetector(3)
	blocklist, _ := NewBlocklistDetector(nil)

	assert.Equal(t, spamdetect_dto.PriorityCritical, honeypot.Priority())
	assert.Equal(t, spamdetect_dto.PriorityCritical, timing.Priority())
	assert.Equal(t, spamdetect_dto.PriorityHigh, gibberish.Priority())
	assert.Equal(t, spamdetect_dto.PriorityHigh, linkDensity.Priority())
	assert.Equal(t, spamdetect_dto.PriorityHigh, blocklist.Priority())
}

type mockRepetitionCache struct {
	mu      sync.Mutex
	entries map[string]repetitionEntry
}

func newMockRepetitionCache() *mockRepetitionCache {
	return &mockRepetitionCache{entries: make(map[string]repetitionEntry)}
}

func (m *mockRepetitionCache) GetIfPresent(_ context.Context, key string) (repetitionEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, found := m.entries[key]
	return value, found, nil
}

func (m *mockRepetitionCache) ComputeWithTTL(_ context.Context, key string, computeFunction func(oldValue repetitionEntry, found bool) cache_dto.ComputeResult[repetitionEntry]) (repetitionEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldValue, found := m.entries[key]
	result := computeFunction(oldValue, found)
	if result.Action == cache_dto.ComputeActionSet {
		m.entries[key] = result.Value
	} else if result.Action == cache_dto.ComputeActionDelete {
		delete(m.entries, key)
	}
	return result.Value, true, nil
}

func (m *mockRepetitionCache) Get(_ context.Context, _ string, _ cache_dto.Loader[string, repetitionEntry]) (repetitionEntry, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) Set(_ context.Context, _ string, _ repetitionEntry, _ ...string) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) SetWithTTL(_ context.Context, _ string, _ repetitionEntry, _ time.Duration, _ ...string) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) Invalidate(_ context.Context, _ string) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) Compute(_ context.Context, _ string, _ func(repetitionEntry, bool) (repetitionEntry, cache_dto.ComputeAction)) (repetitionEntry, bool, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) ComputeIfAbsent(_ context.Context, _ string, _ func() repetitionEntry) (repetitionEntry, bool, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) ComputeIfPresent(_ context.Context, _ string, _ func(repetitionEntry) (repetitionEntry, cache_dto.ComputeAction)) (repetitionEntry, bool, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) BulkGet(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, repetitionEntry]) (map[string]repetitionEntry, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) BulkSet(_ context.Context, _ map[string]repetitionEntry, _ ...string) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) InvalidateByTags(_ context.Context, _ ...string) (int, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) InvalidateAll(_ context.Context) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) BulkRefresh(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, repetitionEntry]) {
	panic("not implemented")
}
func (m *mockRepetitionCache) Refresh(_ context.Context, _ string, _ cache_dto.Loader[string, repetitionEntry]) <-chan cache_dto.LoadResult[repetitionEntry] {
	panic("not implemented")
}
func (m *mockRepetitionCache) All() iter.Seq2[string, repetitionEntry] {
	panic("not implemented")
}
func (m *mockRepetitionCache) Keys() iter.Seq[string] {
	panic("not implemented")
}
func (m *mockRepetitionCache) Values() iter.Seq[repetitionEntry] {
	panic("not implemented")
}
func (m *mockRepetitionCache) GetEntry(_ context.Context, _ string) (cache_dto.Entry[string, repetitionEntry], bool, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) ProbeEntry(_ context.Context, _ string) (cache_dto.Entry[string, repetitionEntry], bool, error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) EstimatedSize() int            { panic("not implemented") }
func (m *mockRepetitionCache) Stats() cache_dto.Stats        { panic("not implemented") }
func (m *mockRepetitionCache) Close(_ context.Context) error { panic("not implemented") }
func (m *mockRepetitionCache) GetMaximum() uint64            { panic("not implemented") }
func (m *mockRepetitionCache) SetMaximum(_ uint64)           { panic("not implemented") }
func (m *mockRepetitionCache) WeightedSize() uint64          { panic("not implemented") }
func (m *mockRepetitionCache) SetExpiresAfter(_ context.Context, _ string, _ time.Duration) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) SetRefreshableAfter(_ context.Context, _ string, _ time.Duration) error {
	panic("not implemented")
}
func (m *mockRepetitionCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, repetitionEntry], error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, repetitionEntry], error) {
	panic("not implemented")
}
func (m *mockRepetitionCache) SupportsSearch() bool               { panic("not implemented") }
func (m *mockRepetitionCache) GetSchema() *cache_dto.SearchSchema { panic("not implemented") }

func TestHoneypotDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewHoneypotDetector()
	assert.Equal(t, "honeypot", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalHoneypot}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityCritical, detector.Priority())
}

func TestTimingDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	assert.Equal(t, "timing", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalTiming}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityCritical, detector.Priority())
}

func TestGibberishDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0.6, nil)
	assert.Equal(t, "gibberish", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityHigh, detector.Priority())
}

func TestLinkDensityDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewLinkDensityDetector(3)
	assert.Equal(t, "link_density", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalLinkDensity}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityHigh, detector.Priority())
}

func TestBlocklistDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector, err := NewBlocklistDetector(nil)
	require.NoError(t, err)
	assert.Equal(t, "blocklist", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalBlocklist}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityHigh, detector.Priority())
}

func TestRepetitionDetector_NameSignalsMode(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	assert.Equal(t, "repetition", detector.Name())
	assert.Equal(t, []spamdetect_dto.Signal{spamdetect_dto.SignalRepetition}, detector.Signals())
	assert.Equal(t, spamdetect_dto.DetectorModeSync, detector.Mode())
	assert.Equal(t, spamdetect_dto.PriorityHigh, detector.Priority())
}

func TestHoneypotDetector_HealthCheck(t *testing.T) {
	t.Parallel()
	assert.NoError(t, NewHoneypotDetector().HealthCheck(context.Background()))
}

func TestTimingDetector_HealthCheck(t *testing.T) {
	t.Parallel()
	assert.NoError(t, NewTimingDetector(2*time.Second).HealthCheck(context.Background()))
}

func TestGibberishDetector_HealthCheck(t *testing.T) {
	t.Parallel()
	assert.NoError(t, NewGibberishDetector(0.6, nil).HealthCheck(context.Background()))
}

func TestLinkDensityDetector_HealthCheck(t *testing.T) {
	t.Parallel()
	assert.NoError(t, NewLinkDensityDetector(3).HealthCheck(context.Background()))
}

func TestBlocklistDetector_HealthCheck(t *testing.T) {
	t.Parallel()
	detector, err := NewBlocklistDetector(nil)
	require.NoError(t, err)
	assert.NoError(t, detector.HealthCheck(context.Background()))
}

func TestRepetitionDetector_HealthCheck_NilCache(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	assert.NoError(t, detector.HealthCheck(context.Background()))
}

func TestRepetitionDetector_HealthCheck_WithCache(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	assert.NoError(t, detector.HealthCheck(context.Background()))
}

func TestRepetitionDetector_NilCache_NoOp(t *testing.T) {
	t.Parallel()
	detector := NewRepetitionDetector(nil, 0, false)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_FirstSubmission_ScoreZero(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
	assert.False(t, result.IsSpam)
}

func TestRepetitionDetector_SecondSubmission_ScoreHalf(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01)
	assert.NotEmpty(t, result.Reasons)
}

func TestRepetitionDetector_ThirdSubmission_ScoreOne(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	for range 2 {
		_, err := detector.Analyse(context.Background(), submission, schema)
		require.NoError(t, err)
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 1.0, result.Score)
	assert.True(t, result.IsSpam)
	assert.NotEmpty(t, result.Reasons)
}

func TestRepetitionDetector_DifferentContent_ScoreZero(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
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

	result, err := detector.Analyse(context.Background(), submissionA, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(context.Background(), submissionB, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_IPScoped(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, true)
	schema := spamdetect_dto.NewSchema(
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

	result, err := detector.Analyse(context.Background(), submissionIP1, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(context.Background(), submissionIP2, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(context.Background(), submissionIP1, schema)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, result.Score, 0.01)
}

func TestRepetitionDetector_NilSubmission(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)

	result, err := detector.Analyse(context.Background(), nil, nil)
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

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestRepetitionDetector_ContextCancellation(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalRepetition),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.Analyse(ctx, submission, schema)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestRepetitionDetector_NoMatchingFields(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)

	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("hello world"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestConfig_ApplyDefaults_ZeroValues(t *testing.T) {
	t.Parallel()
	config := Config{}
	config.applyDefaults()

	assert.InDelta(t, defaultGibberishThreshold, config.GibberishThreshold, 0.001)
	assert.Equal(t, defaultLinkDensityMaxLinks, config.LinkDensityMaxLinks)
	assert.Equal(t, defaultTimingMinDuration, config.TimingMinDuration)
}

func TestConfig_ApplyDefaults_PreservesExplicitValues(t *testing.T) {
	t.Parallel()
	config := Config{
		GibberishThreshold:  0.9,
		LinkDensityMaxLinks: 10,
		TimingMinDuration:   5 * time.Second,
	}
	config.applyDefaults()

	assert.InDelta(t, 0.9, config.GibberishThreshold, 0.001)
	assert.Equal(t, 10, config.LinkDensityMaxLinks)
	assert.Equal(t, 5*time.Second, config.TimingMinDuration)
}

func TestResolveIPScoped_Nil(t *testing.T) {
	t.Parallel()
	assert.True(t, resolveIPScoped(nil))
}

func TestResolveIPScoped_True(t *testing.T) {
	t.Parallel()
	value := true
	assert.True(t, resolveIPScoped(&value))
}

func TestResolveIPScoped_False(t *testing.T) {
	t.Parallel()
	value := false
	assert.False(t, resolveIPScoped(&value))
}

type mockRegistrationService struct {
	registered []string
}

func (m *mockRegistrationService) Analyse(_ context.Context, _ *spamdetect_dto.Submission, _ *spamdetect_dto.Schema) (*spamdetect_dto.AnalysisResult, error) {
	return nil, nil
}

func (m *mockRegistrationService) RegisterDetector(_ context.Context, name string, _ spamdetect_domain.Detector) error {
	m.registered = append(m.registered, name)
	return nil
}

func (m *mockRegistrationService) IsEnabled() bool                         { return false }
func (m *mockRegistrationService) GetDetectors(_ context.Context) []string { return nil }
func (m *mockRegistrationService) HasDetector(_ string) bool               { return false }

func (m *mockRegistrationService) ListDetectors(_ context.Context) []provider_domain.ProviderInfo {
	return nil
}

func (m *mockRegistrationService) SetFeedbackStore(_ spamdetect_domain.FeedbackStore) {}
func (m *mockRegistrationService) ReportSpam(_ context.Context, _ string) error       { return nil }
func (m *mockRegistrationService) ReportHam(_ context.Context, _ string) error        { return nil }
func (m *mockRegistrationService) HealthCheck(_ context.Context) error                { return nil }
func (m *mockRegistrationService) Close(_ context.Context) error                      { return nil }

func TestRegisterDefaults_RegistersAllSixDetectors(t *testing.T) {
	t.Parallel()
	mockService := &mockRegistrationService{}
	config := Config{}

	err := RegisterDefaults(context.Background(), mockService, config)
	require.NoError(t, err)

	assert.Len(t, mockService.registered, 6)
	assert.Contains(t, mockService.registered, "honeypot")
	assert.Contains(t, mockService.registered, "gibberish")
	assert.Contains(t, mockService.registered, "link_density")
	assert.Contains(t, mockService.registered, "blocklist")
	assert.Contains(t, mockService.registered, "timing")
	assert.Contains(t, mockService.registered, "repetition")
}

func TestRegisterDefaults_InvalidBlocklistPatternFails(t *testing.T) {
	t.Parallel()
	mockService := &mockRegistrationService{}
	config := Config{
		BlocklistPatterns: []string{"[invalid"},
	}

	err := RegisterDefaults(context.Background(), mockService, config)
	assert.Error(t, err)
}

func TestHoneypotDetector_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector := NewHoneypotDetector()
	_, err := detector.Analyse(ctx, &spamdetect_dto.Submission{HoneypotValue: "bot"}, nil)
	assert.Error(t, err)
}

func TestTimingDetector_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector := NewTimingDetector(2 * time.Second)
	_, err := detector.Analyse(ctx, &spamdetect_dto.Submission{}, nil)
	assert.Error(t, err)
}

func TestGibberishDetector_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector := NewGibberishDetector(0.6, nil)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("test")},
	}
	_, err := detector.Analyse(ctx, submission, schema)
	assert.Error(t, err)
}

func TestLinkDensityDetector_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector := NewLinkDensityDetector(3)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalLinkDensity))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("test")},
	}
	_, err := detector.Analyse(ctx, submission, schema)
	assert.Error(t, err)
}

func TestBlocklistDetector_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	detector, err := NewBlocklistDetector([]string{`spam`})
	require.NoError(t, err)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalBlocklist))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("spam text")},
	}
	_, analyseErr := detector.Analyse(ctx, submission, schema)
	assert.Error(t, analyseErr)
}

func TestTimingDetector_NegativeDuration(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	now := time.Now()
	submission := &spamdetect_dto.Submission{
		FormLoadedAt:    now,
		FormSubmittedAt: now.Add(-1 * time.Second),
	}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestTimingDetector_BetweenInstantAndMin(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(2 * time.Second)
	now := time.Now()
	submission := &spamdetect_dto.Submission{
		FormLoadedAt:    now,
		FormSubmittedAt: now.Add(1 * time.Second),
	}

	result, err := detector.Analyse(context.Background(), submission, nil)
	require.NoError(t, err)
	assert.Greater(t, result.Score, 0.0)
	assert.Less(t, result.Score, 1.0)
}

func TestTimingDetector_DefaultMinDuration(t *testing.T) {
	t.Parallel()
	detector := NewTimingDetector(0)
	assert.Equal(t, "timing", detector.Name())
}

func TestLinkDensityDetector_NoFieldsWithSignal(t *testing.T) {
	t.Parallel()
	detector := NewLinkDensityDetector(3)
	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("https://link.com")},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)
}

func TestLinkDensityDetector_PerSchemaMaxLinks(t *testing.T) {
	t.Parallel()
	detector := NewLinkDensityDetector(3)
	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalLinkDensity),
		spamdetect_dto.DetectorConfig("link_density", map[string]any{"max_links": 10}),
	)
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": field("Visit https://a.com and https://b.com and https://c.com and https://d.com"),
		},
	}

	result, err := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, err)

	assert.Less(t, result.Score, 0.5)
}

func TestBlocklistDetector_EmptyPatterns(t *testing.T) {
	t.Parallel()
	detector, err := NewBlocklistDetector(nil)
	require.NoError(t, err)

	schema := spamdetect_dto.NewSchema(spamdetect_dto.TextField("message", spamdetect_dto.SignalBlocklist))
	submission := &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{"message": field("anything")},
	}

	result, analyseErr := detector.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.Equal(t, 0.0, result.Score)
}

func TestGibberishDetector_DefaultThreshold(t *testing.T) {
	t.Parallel()
	detector := NewGibberishDetector(0, nil)
	assert.Equal(t, "gibberish", detector.Name())
}

func TestRepetitionDetector_Global_NoIPScope(t *testing.T) {
	t.Parallel()
	cache := newMockRepetitionCache()
	detector := NewRepetitionDetector(cache, 10*time.Minute, false)
	schema := spamdetect_dto.NewSchema(
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

	result, err := detector.Analyse(context.Background(), submissionA, schema)
	require.NoError(t, err)
	assert.Equal(t, 0.0, result.Score)

	result, err = detector.Analyse(context.Background(), submissionB, schema)
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
