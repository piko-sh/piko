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

package spamdetect_domain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

type mockDetector struct {
	name     string
	signals  []spamdetect_dto.Signal
	err      error
	result   *spamdetect_dto.DetectorResult
	delay    time.Duration
	priority spamdetect_dto.DetectorPriority
}

func (m *mockDetector) Name() string                              { return m.name }
func (m *mockDetector) Signals() []spamdetect_dto.Signal          { return m.signals }
func (m *mockDetector) Priority() spamdetect_dto.DetectorPriority { return m.priority }
func (m *mockDetector) Mode() spamdetect_dto.DetectorMode         { return spamdetect_dto.DetectorModeSync }
func (m *mockDetector) HealthCheck(_ context.Context) error       { return nil }

func (m *mockDetector) Analyse(ctx context.Context, _ *spamdetect_dto.Submission, _ *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.result, m.err
}

func testSchema(signals ...spamdetect_dto.Signal) *spamdetect_dto.Schema {
	return spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", signals...),
		spamdetect_dto.Honeypot("_hp"),
		spamdetect_dto.Timing("_ts"),
	)
}

func testSubmission() *spamdetect_dto.Submission {
	return &spamdetect_dto.Submission{
		Fields: map[string]spamdetect_dto.FieldValue{
			"message": {Value: "test content", Type: spamdetect_dto.FieldTypeText},
		},
	}
}

func TestNewSpamDetectService_NilConfig(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestSpamDetectService_IsEnabled(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)
	assert.False(t, service.IsEnabled())

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
	}))
	assert.True(t, service.IsEnabled())
}

func TestSpamDetectService_RegisterDetector_Validation(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	assert.Error(t, service.RegisterDetector(context.Background(), "", &mockDetector{}))
	assert.Error(t, service.RegisterDetector(context.Background(), "test", nil))
}

func TestSpamDetectService_RegisterDetector_MaxLimit(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	for index := range spamdetect_dto.MaxDetectorCount() {
		name := "detector_" + string(rune('a'+index%26)) + string(rune('0'+index/26))
		require.NoError(t, service.RegisterDetector(context.Background(), name, &mockDetector{name: name}))
	}

	err = service.RegisterDetector(context.Background(), "one_too_many", &mockDetector{})
	assert.Error(t, err)
}

func TestSpamDetectService_Analyse_NoDetectors(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{"unrelated_signal"},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	_, err = service.Analyse(context.Background(), testSubmission(), schema)
	assert.ErrorIs(t, err, spamdetect_dto.ErrNoMatchingDetectors)
}

func TestSpamDetectService_Analyse_NilInputs(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	_, err = service.Analyse(context.Background(), nil, testSchema())
	assert.Error(t, err)

	_, err = service.Analyse(context.Background(), testSubmission(), nil)
	assert.Error(t, err)
}

func TestSpamDetectService_Analyse_SingleDetector_Clean(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.False(t, result.IsSpam)
	assert.Len(t, result.DetectorResults, 1)
}

func TestSpamDetectService_Analyse_SingleDetector_Spam(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.9, IsSpam: true, FieldScores: map[string]float64{"message": 0.9}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.True(t, result.IsSpam)
}

func TestSpamDetectService_Analyse_AllDetectorsFailed(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "bad", &mockDetector{
		name: "bad", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		err: errors.New("detector failure"),
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	_, err = service.Analyse(context.Background(), testSubmission(), schema)
	assert.ErrorIs(t, err, spamdetect_dto.ErrAllDetectorsFailed)
}

func TestSpamDetectService_Analyse_PartialFailure(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "good", &mockDetector{
		name: "good", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.3, FieldScores: map[string]float64{"message": 0.3}},
	}))
	require.NoError(t, service.RegisterDetector(context.Background(), "bad", &mockDetector{
		name: "bad", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		err: errors.New("detector failure"),
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.Len(t, result.DetectorResults, 2)
}

func TestSpamDetectService_Analyse_SignalMatching(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "gibberish", &mockDetector{
		name: "gibberish", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))
	require.NoError(t, service.RegisterDetector(context.Background(), "links", &mockDetector{
		name: "links", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalLinkDensity},
		result: &spamdetect_dto.DetectorResult{Score: 0.9},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.Len(t, result.DetectorResults, 1)
	assert.Equal(t, "gibberish", result.DetectorResults[0].Detector)
}

func TestSpamDetectService_Analyse_WithTimeout(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil, WithTimeout(50*time.Millisecond))
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "slow", &mockDetector{
		name: "slow", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		delay: 200 * time.Millisecond, result: &spamdetect_dto.DetectorResult{Score: 0.9},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	_, err = service.Analyse(context.Background(), testSubmission(), schema)
	assert.ErrorIs(t, err, spamdetect_dto.ErrAllDetectorsFailed)
}

func TestSpamDetectService_Analyse_WithScoreThreshold(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil, WithScoreThreshold(0.3))
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.4, FieldScores: map[string]float64{"message": 0.4}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.True(t, result.IsSpam)
}

func TestSpamDetectService_Analyse_SchemaThresholdOverridesService(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil, WithScoreThreshold(0.9))
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.5, FieldScores: map[string]float64{"message": 0.5}},
	}))

	schema := spamdetect_dto.NewSchema(
		spamdetect_dto.TextField("message", spamdetect_dto.SignalGibberish),
		spamdetect_dto.Threshold(0.3),
	)
	result, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)
	assert.True(t, result.IsSpam)
	assert.InDelta(t, 0.3, result.Threshold, 0.01)
}

func TestSpamDetectService_GetDetectors(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "bravo", &mockDetector{name: "bravo"}))
	require.NoError(t, service.RegisterDetector(context.Background(), "alpha", &mockDetector{name: "alpha"}))

	names := service.GetDetectors(context.Background())
	assert.Equal(t, []string{"alpha", "bravo"}, names)
}

func TestSpamDetectService_HasDetector(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{name: "test"}))
	assert.True(t, service.HasDetector("test"))
	assert.False(t, service.HasDetector("nonexistent"))
}

func TestSpamDetectService_ListDetectors(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{name: "test"}))
	detectors := service.ListDetectors(context.Background())
	assert.Len(t, detectors, 1)
	assert.Equal(t, "test", detectors[0].Name)
}

func TestSpamDetectService_HealthCheck(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{name: "test"}))
	assert.NoError(t, service.HealthCheck(context.Background()))
}

func TestSpamDetectService_Close(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)
	assert.NoError(t, service.Close(context.Background()))
}

func TestDisabledSpamDetectService_AllMethods(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	assert.False(t, service.IsEnabled())

	_, err := service.Analyse(context.Background(), &spamdetect_dto.Submission{}, &spamdetect_dto.Schema{})
	assert.ErrorIs(t, err, spamdetect_dto.ErrSpamDetectDisabled)

	assert.ErrorIs(t, service.RegisterDetector(context.Background(), "test", &mockDetector{}), spamdetect_dto.ErrSpamDetectDisabled)
	assert.Empty(t, service.GetDetectors(context.Background()))
	assert.False(t, service.HasDetector("test"))
	assert.Empty(t, service.ListDetectors(context.Background()))
	assert.NoError(t, service.HealthCheck(context.Background()))
	assert.NoError(t, service.Close(context.Background()))
}

func TestSpamDetectService_Close_Idempotent(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	assert.NoError(t, service.Close(context.Background()))
	assert.NoError(t, service.Close(context.Background()))
	assert.NoError(t, service.Close(context.Background()))
}

type mockFeedbackStore struct {
	mu          sync.Mutex
	spamRecords []*spamdetect_dto.SubmissionRecord
	hamRecords  []*spamdetect_dto.SubmissionRecord
	err         error
}

func (m *mockFeedbackStore) ReportSpam(_ context.Context, record *spamdetect_dto.SubmissionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spamRecords = append(m.spamRecords, record)
	return m.err
}

func (m *mockFeedbackStore) ReportHam(_ context.Context, record *spamdetect_dto.SubmissionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hamRecords = append(m.hamRecords, record)
	return m.err
}

func TestSpamDetectService_SetFeedbackStore_ReportSpam(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	store := &mockFeedbackStore{}
	service.SetFeedbackStore(store)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.9, IsSpam: true, FieldScores: map[string]float64{"message": 0.9}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	analysisResult, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)

	reportErr := service.ReportSpam(context.Background(), analysisResult.SubmissionID)
	require.NoError(t, reportErr)

	store.mu.Lock()
	assert.Len(t, store.spamRecords, 1)
	assert.True(t, store.spamRecords[0].IsSpam)
	assert.NotNil(t, store.spamRecords[0].Submission)
	assert.NotNil(t, store.spamRecords[0].Result)
	store.mu.Unlock()
}

func TestSpamDetectService_SetFeedbackStore_ReportHam(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	store := &mockFeedbackStore{}
	service.SetFeedbackStore(store)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	analysisResult, err := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, err)

	reportErr := service.ReportHam(context.Background(), analysisResult.SubmissionID)
	require.NoError(t, reportErr)

	store.mu.Lock()
	assert.Len(t, store.hamRecords, 1)
	assert.False(t, store.hamRecords[0].IsSpam)
	store.mu.Unlock()
}

func TestSpamDetectService_ReportSpam_NoStore(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	reportErr := service.ReportSpam(context.Background(), "unknown-id")
	require.NoError(t, reportErr)
}

func TestSpamDetectService_ReportHam_NoStore(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	reportErr := service.ReportHam(context.Background(), "unknown-id")
	require.NoError(t, reportErr)
}

func TestSpamDetectService_FeedbackStore_CacheMiss(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	store := &mockFeedbackStore{}
	service.SetFeedbackStore(store)

	reportErr := service.ReportSpam(context.Background(), "nonexistent-id")
	require.NoError(t, reportErr)

	store.mu.Lock()
	assert.Len(t, store.spamRecords, 1)

	assert.Nil(t, store.spamRecords[0].Submission)
	assert.Nil(t, store.spamRecords[0].Result)
	store.mu.Unlock()
}

func TestSpamDetectService_FeedbackStore_ErrorPropagates(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	store := &mockFeedbackStore{err: errors.New("database unreachable")}
	service.SetFeedbackStore(store)

	reportErr := service.ReportSpam(context.Background(), "any-id")
	assert.Error(t, reportErr)
	assert.Contains(t, reportErr.Error(), "database unreachable")
}

type mockFeedbackAwareDetector struct {
	mockDetector
	mu            sync.Mutex
	feedbackCalls []feedbackCall
	feedbackErr   error
}

type feedbackCall struct {
	submissionID string
	isSpam       bool
}

func (m *mockFeedbackAwareDetector) ReportFeedback(_ context.Context, submissionID string, isSpam bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.feedbackCalls = append(m.feedbackCalls, feedbackCall{submissionID: submissionID, isSpam: isSpam})
	return m.feedbackErr
}

func TestSpamDetectService_ReportSpam_NotifiesFeedbackAwareDetectors(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	feedbackDetector := &mockFeedbackAwareDetector{
		mockDetector: mockDetector{
			name: "feedback_detector", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
			result: &spamdetect_dto.DetectorResult{Score: 0.8, FieldScores: map[string]float64{"message": 0.8}},
		},
	}

	require.NoError(t, service.RegisterDetector(context.Background(), "feedback_detector", feedbackDetector))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	analysisResult, analyseErr := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, analyseErr)

	reportErr := service.ReportSpam(context.Background(), analysisResult.SubmissionID)
	require.NoError(t, reportErr)

	feedbackDetector.mu.Lock()
	assert.Len(t, feedbackDetector.feedbackCalls, 1)
	assert.True(t, feedbackDetector.feedbackCalls[0].isSpam)
	feedbackDetector.mu.Unlock()
}

func TestSpamDetectService_PriorityTierShortCircuit(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil, WithScoreThreshold(0.5))
	require.NoError(t, err)

	criticalDetector := &mockDetector{
		name:     "critical_blocker",
		signals:  []spamdetect_dto.Signal{spamdetect_dto.SignalHoneypot},
		priority: spamdetect_dto.PriorityCritical,
		result: &spamdetect_dto.DetectorResult{
			Score:  1.0,
			IsSpam: true,
		},
	}

	normalDetectorRan := false
	normalDetector := &trackingDetector{
		mockDetector: mockDetector{
			name:     "normal_tracker",
			signals:  []spamdetect_dto.Signal{spamdetect_dto.SignalHoneypot},
			priority: spamdetect_dto.PriorityNormal,
			result:   &spamdetect_dto.DetectorResult{Score: 0.1},
		},
		ran: &normalDetectorRan,
	}

	require.NoError(t, service.RegisterDetector(context.Background(), "critical_blocker", criticalDetector))
	require.NoError(t, service.RegisterDetector(context.Background(), "normal_tracker", normalDetector))

	schema := spamdetect_dto.NewSchema(spamdetect_dto.Honeypot("_hp"))
	submission := &spamdetect_dto.Submission{HoneypotValue: "bot"}

	result, analyseErr := service.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.True(t, result.IsSpam)
	assert.Equal(t, 1.0, result.Score)

	assert.False(t, normalDetectorRan, "Normal-priority detector should not run when critical tier exceeds threshold")
}

type trackingDetector struct {
	mockDetector
	ran *bool
}

func (d *trackingDetector) Analyse(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) (*spamdetect_dto.DetectorResult, error) {
	*d.ran = true
	return d.mockDetector.Analyse(ctx, submission, schema)
}

func TestSpamDetectService_DetectorWeight_FromServiceConfig(t *testing.T) {
	t.Parallel()
	config := &spamdetect_dto.ServiceConfig{
		ScoreThreshold:    0.7,
		Timeout:           3 * time.Second,
		FeedbackCacheSize: 100,
		DetectorWeights:   map[string]float64{"weighted": 3.0},
	}
	service, err := NewSpamDetectService(config)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "weighted", &mockDetector{
		name: "weighted", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.9, IsSpam: true, FieldScores: map[string]float64{"message": 0.9}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, analyseErr := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, analyseErr)
	assert.True(t, result.IsSpam)
}

func TestDisabledSpamDetectService_FeedbackMethods(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	service.SetFeedbackStore(nil)
	assert.ErrorIs(t, service.ReportSpam(context.Background(), "id"), spamdetect_dto.ErrSpamDetectDisabled)
	assert.ErrorIs(t, service.ReportHam(context.Background(), "id"), spamdetect_dto.ErrSpamDetectDisabled)
}

func TestSpamDetectService_Analyse_AssignsSubmissionID(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	submission := testSubmission()
	assert.Empty(t, submission.ID)

	result, analyseErr := service.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.NotEmpty(t, result.SubmissionID)
	assert.Equal(t, submission.ID, result.SubmissionID)
}

func TestSpamDetectService_Analyse_PreservesExistingSubmissionID(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	submission := testSubmission()
	submission.ID = "my-custom-id"

	result, analyseErr := service.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.Equal(t, "my-custom-id", result.SubmissionID)
}

func TestSpamDetectService_Analyse_MultipleDetectors_CompositeScore(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "detector_a", &mockDetector{
		name: "detector_a", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.4, FieldScores: map[string]float64{"message": 0.4}},
	}))
	require.NoError(t, service.RegisterDetector(context.Background(), "detector_b", &mockDetector{
		name: "detector_b", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.6, FieldScores: map[string]float64{"message": 0.6}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	result, analyseErr := service.Analyse(context.Background(), testSubmission(), schema)
	require.NoError(t, analyseErr)
	assert.Len(t, result.DetectorResults, 2)

	assert.Greater(t, result.Score, 0.0)
}

func concreteService(t *testing.T) *spamDetectService {
	t.Helper()
	service, err := NewSpamDetectService(nil)
	require.NoError(t, err)
	concrete, ok := service.(*spamDetectService)
	require.True(t, ok)
	return concrete
}

func TestSpamDetectService_HealthProbe_Name(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	assert.Equal(t, "SpamDetectService", service.Name())
}

func TestSpamDetectService_HealthProbe_Liveness_NoDetectors(t *testing.T) {
	t.Parallel()
	service := concreteService(t)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
	assert.Contains(t, status.Message, "no spam detection detectors configured")
}

func TestSpamDetectService_HealthProbe_Liveness_WithDetector(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{name: "test"}))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Contains(t, status.Message, "operational")
}

func TestSpamDetectService_HealthProbe_Readiness_NoDetectors(t *testing.T) {
	t.Parallel()
	service := concreteService(t)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
}

func TestSpamDetectService_HealthProbe_Readiness_HealthyDetector(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "healthy", &mockDetector{name: "healthy"}))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
	assert.Len(t, status.Dependencies, 1)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.Dependencies[0].State)
}

type mockUnhealthyDetector struct {
	mockDetector
}

func (m *mockUnhealthyDetector) HealthCheck(_ context.Context) error {
	return errors.New("detector unavailable")
}

func TestSpamDetectService_HealthProbe_Readiness_UnhealthyDetector(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "unhealthy", &mockUnhealthyDetector{
		mockDetector: mockDetector{name: "unhealthy"},
	}))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
	assert.Contains(t, status.Message, "unhealthy")
	assert.Len(t, status.Dependencies, 1)
	assert.Equal(t, healthprobe_dto.StateUnhealthy, status.Dependencies[0].State)
}

func TestSpamDetectService_HealthCheck_UnhealthyDetector(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "unhealthy", &mockUnhealthyDetector{
		mockDetector: mockDetector{name: "unhealthy"},
	}))

	err := service.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detector unavailable")
}

func TestSpamDetectService_ResourceType(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	assert.Equal(t, "spamdetect", service.ResourceType())
}

func TestSpamDetectService_ResourceListColumns(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	columns := service.ResourceListColumns()
	assert.Len(t, columns, 3)
	assert.Equal(t, "NAME", columns[0].Header)
	assert.Equal(t, "SIGNALS", columns[1].Header)
	assert.Equal(t, "REGISTERED", columns[2].Header)
}

func TestSpamDetectService_ResourceListProviders(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "test_det", &mockDetector{
		name:    "test_det",
		signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
	}))

	entries := service.ResourceListProviders(context.Background())
	assert.Len(t, entries, 1)
	assert.Equal(t, "test_det", entries[0].Name)
	assert.Contains(t, entries[0].Values["signals"], "gibberish")
}

func TestSpamDetectService_ResourceDescribeProvider(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "test_det", &mockDetector{
		name:    "test_det",
		signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish, spamdetect_dto.SignalLinkDensity},
	}))

	detail, err := service.ResourceDescribeProvider(context.Background(), "test_det")
	require.NoError(t, err)
	assert.Equal(t, "test_det", detail.Name)
	assert.NotEmpty(t, detail.Sections)
}

func TestSpamDetectService_ResourceDescribeProvider_NotFound(t *testing.T) {
	t.Parallel()
	service := concreteService(t)

	_, err := service.ResourceDescribeProvider(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

type mockMetadataDetector struct {
	mockDetector
}

func (m *mockMetadataDetector) GetProviderType() string {
	return "mock"
}

func (m *mockMetadataDetector) GetProviderMetadata() map[string]any {
	return map[string]any{
		"threshold": 0.6,
		"version":   "1.0",
	}
}

func TestSpamDetectService_ResourceDescribeProvider_WithMetadata(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	require.NoError(t, service.RegisterDetector(context.Background(), "meta_det", &mockMetadataDetector{
		mockDetector: mockDetector{
			name:    "meta_det",
			signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		},
	}))

	detail, err := service.ResourceDescribeProvider(context.Background(), "meta_det")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(detail.Sections), 2)
}

func TestSpamDetectService_ResourceListProviders_Empty(t *testing.T) {
	t.Parallel()
	service := concreteService(t)
	entries := service.ResourceListProviders(context.Background())
	assert.Empty(t, entries)
}

func TestFormatRegisteredAge_Zero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", formatRegisteredAge(time.Time{}))
}

func TestFormatRegisteredAge_Seconds(t *testing.T) {
	t.Parallel()
	result := formatRegisteredAge(time.Now().Add(-30 * time.Second))
	assert.Contains(t, result, "s ago")
}

func TestFormatRegisteredAge_Minutes(t *testing.T) {
	t.Parallel()
	result := formatRegisteredAge(time.Now().Add(-5 * time.Minute))
	assert.Contains(t, result, "m ago")
}

func TestFormatRegisteredAge_Hours(t *testing.T) {
	t.Parallel()
	result := formatRegisteredAge(time.Now().Add(-5 * time.Hour))
	assert.Contains(t, result, "h ago")
}

func TestFormatRegisteredAge_Days(t *testing.T) {
	t.Parallel()
	result := formatRegisteredAge(time.Now().Add(-48 * time.Hour))
	assert.Contains(t, result, "d ago")
}

func TestFindDetectorInfo_Found(t *testing.T) {
	t.Parallel()
	infos := []provider_domain.ProviderInfo{
		{Name: "alpha"},
		{Name: "bravo"},
	}
	result := findDetectorInfo(infos, "bravo")
	assert.Equal(t, "bravo", result.Name)
}

func TestFindDetectorInfo_NotFound(t *testing.T) {
	t.Parallel()
	infos := []provider_domain.ProviderInfo{
		{Name: "alpha"},
	}
	result := findDetectorInfo(infos, "missing")
	assert.Equal(t, "missing", result.Name)
}

func TestSpamDetectService_CacheEviction(t *testing.T) {
	t.Parallel()
	config := &spamdetect_dto.ServiceConfig{
		ScoreThreshold:    0.7,
		Timeout:           3 * time.Second,
		FeedbackCacheSize: 3,
	}
	service, err := NewSpamDetectService(config)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	var submissionIDs []string

	for range 5 {
		submission := testSubmission()
		result, analyseErr := service.Analyse(context.Background(), submission, schema)
		require.NoError(t, analyseErr)
		submissionIDs = append(submissionIDs, result.SubmissionID)
	}

	store := &mockFeedbackStore{}
	service.SetFeedbackStore(store)

	reportErr := service.ReportSpam(context.Background(), submissionIDs[0])
	require.NoError(t, reportErr)

	store.mu.Lock()

	assert.Nil(t, store.spamRecords[0].Submission)
	store.mu.Unlock()

	reportErr = service.ReportSpam(context.Background(), submissionIDs[4])
	require.NoError(t, reportErr)

	store.mu.Lock()
	assert.NotNil(t, store.spamRecords[1].Submission)
	store.mu.Unlock()
}

func TestSpamDetectService_CacheDuplicateID(t *testing.T) {
	t.Parallel()
	config := &spamdetect_dto.ServiceConfig{
		ScoreThreshold:    0.7,
		Timeout:           3 * time.Second,
		FeedbackCacheSize: 10,
	}
	service, err := NewSpamDetectService(config)
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "test", &mockDetector{
		name: "test", signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result: &spamdetect_dto.DetectorResult{Score: 0.1, FieldScores: map[string]float64{"message": 0.1}},
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)

	submission1 := testSubmission()
	submission1.ID = "fixed-id"
	_, err = service.Analyse(context.Background(), submission1, schema)
	require.NoError(t, err)

	submission2 := testSubmission()
	submission2.ID = "fixed-id"
	_, err = service.Analyse(context.Background(), submission2, schema)
	require.NoError(t, err)

	store := &mockFeedbackStore{}
	service.SetFeedbackStore(store)
	require.NoError(t, service.ReportSpam(context.Background(), "fixed-id"))

	store.mu.Lock()
	assert.NotNil(t, store.spamRecords[0].Submission)
	store.mu.Unlock()
}

func TestDisabledSpamDetectService_SetFeedbackStore(t *testing.T) {
	t.Parallel()
	service := NewDisabledSpamDetectService()

	service.SetFeedbackStore(nil)
	service.SetFeedbackStore(&mockFeedbackStore{})
}

func TestSpamDetectService_Analyse_DetectorReturnsNilResult(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(&spamdetect_dto.ServiceConfig{
		ScoreThreshold: 0.7, Timeout: 3 * time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "nil_result", &mockDetector{
		name:    "nil_result",
		signals: []spamdetect_dto.Signal{spamdetect_dto.SignalGibberish},
		result:  nil,
	}))

	schema := testSchema(spamdetect_dto.SignalGibberish)
	_, analyseErr := service.Analyse(context.Background(), testSubmission(), schema)

	assert.ErrorIs(t, analyseErr, spamdetect_dto.ErrAllDetectorsFailed)
}

func TestSpamDetectService_Analyse_FormLevelScore(t *testing.T) {
	t.Parallel()
	service, err := NewSpamDetectService(nil, WithScoreThreshold(0.5))
	require.NoError(t, err)

	require.NoError(t, service.RegisterDetector(context.Background(), "form_level", &mockDetector{
		name:     "form_level",
		signals:  []spamdetect_dto.Signal{spamdetect_dto.SignalHoneypot},
		priority: spamdetect_dto.PriorityCritical,
		result: &spamdetect_dto.DetectorResult{
			Score:   1.0,
			IsSpam:  true,
			Reasons: []string{"honeypot was filled"},
		},
	}))

	schema := spamdetect_dto.NewSchema(spamdetect_dto.Honeypot("_hp"))
	submission := &spamdetect_dto.Submission{HoneypotValue: "bot"}

	result, analyseErr := service.Analyse(context.Background(), submission, schema)
	require.NoError(t, analyseErr)
	assert.True(t, result.IsSpam)
	assert.Contains(t, result.FormReasons, "honeypot was filled")
}
