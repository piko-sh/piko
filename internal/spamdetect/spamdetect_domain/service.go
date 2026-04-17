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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

const (
	// circuitBreakerTimeout is how long the circuit stays open before retrying.
	circuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the measurement bucket duration.
	circuitBreakerBucketPeriod = 10 * time.Second

	// circuitBreakerConsecutiveFailures is the failure count to trip the breaker.
	circuitBreakerConsecutiveFailures = 5

	// safeCallDetectorAnalyse is the goroutine.SafeCall label for detector analysis.
	safeCallDetectorAnalyse = "spamdetect.Analyse"

	// defaultFieldWeight is the default scoring weight for fields and detectors.
	defaultFieldWeight = 1.0
)

var (
	// errDetectorNameEmpty is returned when an empty name is passed to RegisterDetector.
	errDetectorNameEmpty = errors.New("detector name cannot be empty")

	// errDetectorNil is returned when a nil detector is passed to RegisterDetector.
	errDetectorNil = errors.New("detector cannot be nil")

	// errTooManyDetectors is returned when the detector limit is exceeded.
	errTooManyDetectors = errors.New("maximum number of spam detection detectors reached")
)

// spamDetectService is the concrete implementation of SpamDetectServicePort.
type spamDetectService struct {
	// registry stores and looks up detectors by name.
	registry *provider_domain.StandardRegistry[Detector]

	// breakers holds per-detector circuit breakers.
	breakers map[string]*gobreaker.CircuitBreaker[any]

	// feedbackStore persists spam/ham feedback reports.
	feedbackStore FeedbackStore

	// detectorWeights maps detector names to their scoring weight.
	detectorWeights map[string]float64

	// cacheEntries maps submission IDs to cached analysis records.
	cacheEntries map[string]*cachedRecord

	// cacheKeys tracks insertion order for ring-buffer eviction.
	cacheKeys []string

	// inflight tracks in-flight analysis operations for graceful shutdown.
	inflight sync.WaitGroup

	// breakerMu guards concurrent access to the breakers map.
	breakerMu sync.RWMutex

	// cacheMu guards concurrent access to cacheEntries and cacheKeys.
	cacheMu sync.Mutex

	// feedbackMu guards concurrent access to feedbackStore.
	feedbackMu sync.RWMutex

	// closed ensures Close runs exactly once.
	closed sync.Once

	// scoreThreshold is the default composite score above which a submission is spam.
	scoreThreshold float64

	// timeout is the maximum duration to wait for all detectors.
	timeout time.Duration

	// cacheSize is the maximum number of cached analysis records.
	cacheSize int

	// cacheIndex is the ring-buffer write position.
	cacheIndex int
}

// cachedRecord pairs a submission with its analysis result for feedback correlation.
type cachedRecord struct {
	// submission is the original submission.
	submission *spamdetect_dto.Submission

	// result is the analysis result for this submission.
	result *spamdetect_dto.AnalysisResult
}

// detectorInfo pairs a detector with its registered name.
type detectorInfo struct {
	// detector is the Detector instance.
	detector Detector

	// name is the registered name of the detector.
	name string
}

// aggregationResult holds the aggregated analysis result and failure state.
type aggregationResult struct {
	// analysisResult is the composite analysis result.
	analysisResult *spamdetect_dto.AnalysisResult

	// allFailed is true when every detector returned an error.
	allFailed bool
}

// fieldAccumulator accumulates weighted scores and reasons for a single field.
type fieldAccumulator struct {
	// reasons collects per-detector reason strings for this field.
	reasons []string

	// totalScore is the weighted sum of detector scores for this field.
	totalScore float64

	// totalWeight is the sum of detector weights that contributed scores.
	totalWeight float64
}

// ServiceOption configures the spam detection service.
type ServiceOption func(*spamDetectService)

// WithScoreThreshold sets the default composite score threshold.
//
// Takes threshold (float64) which is the minimum score for spam classification.
//
// Returns ServiceOption which configures the threshold.
func WithScoreThreshold(threshold float64) ServiceOption {
	return func(s *spamDetectService) {
		s.scoreThreshold = threshold
	}
}

// WithTimeout sets the maximum duration to wait for all detectors.
//
// Takes timeout (time.Duration) which is the maximum analysis duration.
//
// Returns ServiceOption which configures the timeout.
func WithTimeout(timeout time.Duration) ServiceOption {
	return func(s *spamDetectService) {
		s.timeout = timeout
	}
}

// NewSpamDetectService creates a new spam detection service.
//
// Takes config (*spamdetect_dto.ServiceConfig) which provides service settings.
// Takes opts (...ServiceOption) which are optional configuration functions.
//
// Returns SpamDetectServicePort which is the configured service.
// Returns error when the service cannot be created.
func NewSpamDetectService(config *spamdetect_dto.ServiceConfig, opts ...ServiceOption) (SpamDetectServicePort, error) {
	if config == nil {
		config = spamdetect_dto.DefaultServiceConfig()
	}

	cacheSize := config.FeedbackCacheSize
	if cacheSize <= 0 {
		cacheSize = spamdetect_dto.DefaultFeedbackCacheSize
	}

	service := &spamDetectService{
		registry:        provider_domain.NewStandardRegistry[Detector]("spamdetect"),
		breakers:        make(map[string]*gobreaker.CircuitBreaker[any]),
		detectorWeights: config.DetectorWeights,
		cacheEntries:    make(map[string]*cachedRecord, cacheSize),
		cacheKeys:       make([]string, 0, cacheSize),
		scoreThreshold:  config.ScoreThreshold,
		timeout:         config.Timeout,
		cacheSize:       cacheSize,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service, nil
}

// Analyse runs all matching detectors in parallel and returns a composite
// verdict with per-field breakdowns.
//
// Takes submission (*spamdetect_dto.Submission) which contains the form data.
// Takes schema (*spamdetect_dto.Schema) which describes the form fields.
//
// Returns *spamdetect_dto.AnalysisResult which contains the composite verdict.
// Returns error when analysis fails.
func (s *spamDetectService) Analyse(
	ctx context.Context,
	submission *spamdetect_dto.Submission,
	schema *spamdetect_dto.Schema,
) (*spamdetect_dto.AnalysisResult, error) {
	ctx, l := logger_domain.From(ctx, log)

	if submission == nil {
		return nil, errors.New("submission cannot be nil")
	}
	if schema == nil {
		return nil, errors.New("schema cannot be nil")
	}

	if submission.ID == "" {
		submission.ID = generateSubmissionID()
	}

	submission.Sanitise(schema)
	if submission.WasTruncated() {
		l.Trace("Submission fields were truncated during sanitisation")
	}

	matchingDetectors := s.findMatchingDetectors(ctx, schema)
	if len(matchingDetectors) == 0 {
		return nil, spamdetect_dto.ErrNoMatchingDetectors
	}

	s.inflight.Add(1)
	defer s.inflight.Done()

	startTime := time.Now()
	analyseCtx, analyseCancel := context.WithTimeoutCause(
		ctx,
		s.timeout,
		fmt.Errorf("spam detection analysis exceeded %s timeout", s.timeout),
	)
	defer analyseCancel()

	detectorResults := s.runDetectors(analyseCtx, matchingDetectors, submission, schema)
	result := s.aggregateResults(detectorResults, schema, time.Since(startTime))

	if result.allFailed {
		recordAnalyseMetric(ctx, statusError, false, 0)
		return result.analysisResult, spamdetect_dto.ErrAllDetectorsFailed
	}

	recordAnalyseMetric(ctx, statusSuccess, result.analysisResult.IsSpam, result.analysisResult.Duration)

	l.Trace("Spam detection analysis completed",
		logger_domain.Float64("score", result.analysisResult.Score),
		logger_domain.Float64("threshold", result.analysisResult.Threshold),
		logger_domain.Bool("is_spam", result.analysisResult.IsSpam),
		logger_domain.Int64(attributeKeyDurationMS, result.analysisResult.Duration.Milliseconds()),
	)

	result.analysisResult.SubmissionID = submission.ID
	s.cacheRecord(submission, result.analysisResult)

	return result.analysisResult, nil
}

// IsEnabled reports whether at least one detector is registered.
//
// Returns bool which is true when detectors are available.
func (s *spamDetectService) IsEnabled() bool {
	return len(s.registry.ListProviders(context.Background())) > 0
}

// RegisterDetector adds a new detector with the given name.
//
// Takes name (string) which identifies the detector.
// Takes detector (Detector) which handles spam analysis.
//
// Returns error when the detector cannot be registered.
//
// Concurrency: Safe for concurrent use; acquires breakerMu internally.
func (s *spamDetectService) RegisterDetector(ctx context.Context, name string, detector Detector) error {
	if name == "" {
		return errDetectorNameEmpty
	}
	if detector == nil {
		return errDetectorNil
	}

	existing := s.registry.ListProviders(ctx)
	if len(existing) >= spamdetect_dto.MaxDetectorCount() {
		return fmt.Errorf("%w: limit is %d", errTooManyDetectors, spamdetect_dto.MaxDetectorCount())
	}

	if err := s.registry.RegisterProvider(ctx, name, detector); err != nil {
		return err
	}

	s.breakerMu.Lock()
	s.breakers[name] = newDetectorCircuitBreaker(name)
	s.breakerMu.Unlock()

	return nil
}

// GetDetectors returns a sorted list of all registered detector names.
//
// Returns []string which contains the sorted detector names.
func (s *spamDetectService) GetDetectors(ctx context.Context) []string {
	detectors := s.registry.ListProviders(ctx)
	names := make([]string, 0, len(detectors))
	for _, detector := range detectors {
		names = append(names, detector.Name)
	}
	slices.Sort(names)
	return names
}

// HasDetector checks if a detector with the given name has been registered.
//
// Takes name (string) which is the detector name to look up.
//
// Returns bool which is true if the detector exists.
func (s *spamDetectService) HasDetector(name string) bool {
	_, err := s.registry.GetProvider(context.Background(), name)
	return err == nil
}

// ListDetectors returns details about all registered detectors.
//
// Returns []provider_domain.ProviderInfo which contains detector information.
func (s *spamDetectService) ListDetectors(ctx context.Context) []provider_domain.ProviderInfo {
	return s.registry.ListProviders(ctx)
}

// HealthCheck verifies all registered detectors are operational.
//
// Returns error when any detector health check fails.
func (s *spamDetectService) HealthCheck(ctx context.Context) error {
	detectors := s.registry.ListProviders(ctx)
	var errs []error
	for _, info := range detectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("detector %s: %w", info.Name, err))
			continue
		}
		if err := detector.HealthCheck(ctx); err != nil {
			errs = append(errs, fmt.Errorf("detector %s health check: %w", info.Name, err))
		}
	}
	return errors.Join(errs...)
}

// Close waits for in-flight analyses to complete, then shuts down all
// detectors.
//
// Returns error when shutdown fails or the context is cancelled.
//
// Concurrency: Safe to call multiple times; guarded by sync.Once.
func (s *spamDetectService) Close(ctx context.Context) error {
	var closeErr error
	s.closed.Do(func() {
		done := make(chan struct{})
		go func() {
			defer func() { recover() }()
			s.inflight.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			closeErr = fmt.Errorf("timed out waiting for in-flight analyses: %w", ctx.Err())
			return
		}

		closeErr = s.registry.CloseAll(ctx)
	})
	return closeErr
}

// SetFeedbackStore configures the feedback persistence backend.
//
// Takes store (FeedbackStore) which persists spam/ham feedback.
//
// Concurrency: Safe for concurrent use; acquires feedbackMu internally.
func (s *spamDetectService) SetFeedbackStore(store FeedbackStore) {
	s.feedbackMu.Lock()
	defer s.feedbackMu.Unlock()
	s.feedbackStore = store
}

// ReportSpam records that a submission was confirmed as spam and notifies
// feedback-aware detectors.
//
// Takes submissionID (string) which identifies the submission.
//
// Returns error when persistence or notification fails.
func (s *spamDetectService) ReportSpam(ctx context.Context, submissionID string) error {
	return s.reportFeedback(ctx, submissionID, true)
}

// ReportHam records that a submission was confirmed as legitimate and
// notifies feedback-aware detectors.
//
// Takes submissionID (string) which identifies the submission.
//
// Returns error when persistence or notification fails.
func (s *spamDetectService) ReportHam(ctx context.Context, submissionID string) error {
	return s.reportFeedback(ctx, submissionID, false)
}

// reportFeedback persists a feedback report and notifies
// feedback-aware detectors.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns error when persistence or notification fails.
//
// Concurrency: Acquires feedbackMu for store access and cacheMu
// transitively via buildFeedbackRecord.
func (s *spamDetectService) reportFeedback(ctx context.Context, submissionID string, isSpam bool) error {
	ctx, l := logger_domain.From(ctx, log)

	record := s.buildFeedbackRecord(submissionID, isSpam)

	s.feedbackMu.RLock()
	store := s.feedbackStore
	s.feedbackMu.RUnlock()

	if store != nil {
		var storeErr error
		if isSpam {
			storeErr = store.ReportSpam(ctx, record)
		} else {
			storeErr = store.ReportHam(ctx, record)
		}
		if storeErr != nil {
			l.Error("Failed to store feedback", logger_domain.Error(storeErr), logger_domain.Bool("is_spam", isSpam))
			return fmt.Errorf("storing feedback: %w", storeErr)
		}
	}

	return s.notifyFeedbackDetectors(ctx, submissionID, isSpam)
}

// buildFeedbackRecord constructs a SubmissionRecord from the cache.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns *spamdetect_dto.SubmissionRecord which is the feedback
// record.
//
// Concurrency: Acquires cacheMu to read from the cache.
func (s *spamDetectService) buildFeedbackRecord(submissionID string, isSpam bool) *spamdetect_dto.SubmissionRecord {
	record := &spamdetect_dto.SubmissionRecord{
		SubmissionID: submissionID,
		ReportedAt:   time.Now(),
		IsSpam:       isSpam,
	}

	s.cacheMu.Lock()
	entry, found := s.cacheEntries[submissionID]
	s.cacheMu.Unlock()

	if found {
		record.Submission = entry.submission
		record.Result = entry.result
	} else {
		_, l := logger_domain.From(context.Background(), log)
		l.Trace("No cached analysis result for feedback submission",
			logger_domain.String("submission_id", submissionID),
		)
	}

	return record
}

// cacheRecord stores a submission and result in the ring-buffer cache.
//
// Takes submission (*spamdetect_dto.Submission) which is the form data.
// Takes result (*spamdetect_dto.AnalysisResult) which is the verdict.
//
// Concurrency: Acquires cacheMu to write to the cache.
func (s *spamDetectService) cacheRecord(submission *spamdetect_dto.Submission, result *spamdetect_dto.AnalysisResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	entry := &cachedRecord{
		submission: submission,
		result:     result,
	}

	if _, exists := s.cacheEntries[submission.ID]; exists {
		s.cacheEntries[submission.ID] = entry
		return
	}

	if len(s.cacheKeys) >= s.cacheSize {
		oldestKey := s.cacheKeys[s.cacheIndex]
		delete(s.cacheEntries, oldestKey)
		s.cacheKeys[s.cacheIndex] = submission.ID
		s.cacheIndex = (s.cacheIndex + 1) % s.cacheSize
	} else {
		s.cacheKeys = append(s.cacheKeys, submission.ID)
	}

	s.cacheEntries[submission.ID] = entry
}

// notifyFeedbackDetectors forwards feedback to detectors that
// implement FeedbackAwareDetector.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns error when any detector notification fails.
func (s *spamDetectService) notifyFeedbackDetectors(ctx context.Context, submissionID string, isSpam bool) error {
	detectors := s.registry.ListProviders(ctx)
	var errs []error
	for _, info := range detectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			continue
		}
		feedbackDetector, ok := detector.(FeedbackAwareDetector)
		if !ok {
			continue
		}
		if err := feedbackDetector.ReportFeedback(ctx, submissionID, isSpam); err != nil {
			errs = append(errs, fmt.Errorf("detector %s feedback: %w", info.Name, err))
		}
	}
	return errors.Join(errs...)
}

// findMatchingDetectors returns detectors whose signals overlap with the
// schema's declared signals.
//
// Takes schema (*spamdetect_dto.Schema) which declares the required signals.
//
// Returns []detectorInfo which contains the matching detectors.
func (s *spamDetectService) findMatchingDetectors(ctx context.Context, schema *spamdetect_dto.Schema) []detectorInfo {
	schemaSignals := schema.AllSignals()
	signalSet := make(map[spamdetect_dto.Signal]struct{}, len(schemaSignals))
	for _, signal := range schemaSignals {
		signalSet[signal] = struct{}{}
	}

	allDetectors := s.registry.ListProviders(ctx)
	var matched []detectorInfo

	for _, info := range allDetectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			continue
		}

		for _, detectorSignal := range detector.Signals() {
			if _, exists := signalSet[detectorSignal]; exists {
				matched = append(matched, detectorInfo{name: info.Name, detector: detector})
				break
			}
		}
	}

	return matched
}

// runDetectors groups detectors by priority tier and executes tiers
// sequentially, with parallel execution within each tier.
//
// Takes detectors ([]detectorInfo) which are the matching detectors to run.
// Takes submission (*spamdetect_dto.Submission) which contains the form data.
// Takes schema (*spamdetect_dto.Schema) which describes the form fields.
//
// Returns []spamdetect_dto.DetectorResult which contains all detector results.
func (s *spamDetectService) runDetectors(
	ctx context.Context,
	detectors []detectorInfo,
	submission *spamdetect_dto.Submission,
	schema *spamdetect_dto.Schema,
) []spamdetect_dto.DetectorResult {
	tiers := groupByPriority(detectors)
	threshold := schema.ScoreThreshold()
	if threshold <= 0 {
		threshold = s.scoreThreshold
	}

	var allResults []spamdetect_dto.DetectorResult

	for _, tier := range tiers {
		tierResults := s.runTier(ctx, tier, submission, schema)
		allResults = append(allResults, tierResults...)

		interimResult := s.aggregateResults(allResults, schema, 0)
		if interimResult.analysisResult.Score >= threshold {
			break
		}
	}

	return allResults
}

// groupByPriority groups detectors into tiers ordered by priority.
//
// Takes detectors ([]detectorInfo) which are the detectors to group.
//
// Returns [][]detectorInfo which contains the priority-ordered tiers.
func groupByPriority(detectors []detectorInfo) [][]detectorInfo {
	tierMap := make(map[spamdetect_dto.DetectorPriority][]detectorInfo)
	for _, info := range detectors {
		priority := info.detector.Priority()
		tierMap[priority] = append(tierMap[priority], info)
	}

	priorities := []spamdetect_dto.DetectorPriority{
		spamdetect_dto.PriorityCritical,
		spamdetect_dto.PriorityHigh,
		spamdetect_dto.PriorityNormal,
	}

	var tiers [][]detectorInfo
	for _, priority := range priorities {
		if tier, exists := tierMap[priority]; exists {
			tiers = append(tiers, tier)
		}
	}

	return tiers
}

// runTier executes all detectors in a single priority tier in
// parallel.
//
// Takes detectors ([]detectorInfo) which are the tier's detectors.
// Takes submission (*spamdetect_dto.Submission) which is the form data.
// Takes schema (*spamdetect_dto.Schema) which describes the fields.
//
// Returns []spamdetect_dto.DetectorResult which contains the results.
func (s *spamDetectService) runTier(
	ctx context.Context,
	detectors []detectorInfo,
	submission *spamdetect_dto.Submission,
	schema *spamdetect_dto.Schema,
) []spamdetect_dto.DetectorResult {
	results := make([]spamdetect_dto.DetectorResult, len(detectors))
	var waitGroup sync.WaitGroup

	for index, info := range detectors {
		detectorName := info.name
		det := info.detector
		resultIndex := index
		waitGroup.Go(func() {
			results[resultIndex] = s.runSingleDetector(ctx, detectorName, det, submission, schema)
		})
	}

	waitGroup.Wait()
	return results
}

// runSingleDetector executes one detector with circuit breaker and panic
// protection.
//
// Takes detectorName (string) which identifies the detector.
// Takes detector (Detector) which handles the analysis.
// Takes submission (*spamdetect_dto.Submission) which contains the form data.
// Takes schema (*spamdetect_dto.Schema) which describes the form fields.
//
// Returns spamdetect_dto.DetectorResult which is the detector's verdict.
func (s *spamDetectService) runSingleDetector(
	ctx context.Context,
	detectorName string,
	detector Detector,
	submission *spamdetect_dto.Submission,
	schema *spamdetect_dto.Schema,
) spamdetect_dto.DetectorResult {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	breaker := s.getBreakerForDetector(detectorName)
	rawResult, callErr := goroutine.SafeCall1(ctx, safeCallDetectorAnalyse, func() (any, error) {
		return breaker.Execute(func() (any, error) {
			return detector.Analyse(ctx, submission, schema)
		})
	})
	duration := time.Since(startTime)

	if callErr != nil {
		l.Warn("Spam detection detector failed",
			logger_domain.String(attributeKeyDetector, detectorName),
			logger_domain.Error(callErr),
			logger_domain.Int64(attributeKeyDurationMS, duration.Milliseconds()),
		)
		recordDetectorMetric(ctx, detectorName, statusError)
		return detectorErrorResult(detectorName, callErr, duration)
	}

	result, ok := rawResult.(*spamdetect_dto.DetectorResult)
	if !ok || result == nil {
		recordDetectorMetric(ctx, detectorName, statusError)
		return detectorErrorResult(detectorName, errors.New("unexpected response type from detector"), duration)
	}

	result.Detector = detectorName
	result.Duration = duration
	recordDetectorMetric(ctx, detectorName, statusSuccess)

	return *result
}

// aggregateResults computes per-field scores and a weighted composite score.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the
// individual detector verdicts.
// Takes schema (*spamdetect_dto.Schema) which provides field weights.
// Takes totalDuration (time.Duration) which is the elapsed analysis time.
//
// Returns aggregationResult which contains the composite verdict.
func (s *spamDetectService) aggregateResults(
	detectorResults []spamdetect_dto.DetectorResult,
	schema *spamdetect_dto.Schema,
	totalDuration time.Duration,
) aggregationResult {
	threshold := schema.ScoreThreshold()
	if threshold <= 0 {
		threshold = s.scoreThreshold
	}

	fieldScores := s.computeFieldScores(detectorResults, schema)
	compositeScore, allFailed := s.computeCompositeScore(detectorResults, fieldScores, schema)

	result := &spamdetect_dto.AnalysisResult{
		DetectorResults: detectorResults,
		FieldResults:    fieldScores,
		FormReasons:     collectFormReasons(detectorResults),
		Duration:        totalDuration,
		Score:           compositeScore,
		Threshold:       threshold,
		IsSpam:          compositeScore >= threshold,
	}

	return aggregationResult{analysisResult: result, allFailed: allFailed}
}

// computeFieldScores calculates per-field scores using detector weights
// and precise per-field reason attribution.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the
// individual detector verdicts.
// Takes schema (*spamdetect_dto.Schema) which provides field definitions.
//
// Returns []spamdetect_dto.FieldResult which contains the per-field scores.
func (s *spamDetectService) computeFieldScores(
	detectorResults []spamdetect_dto.DetectorResult,
	schema *spamdetect_dto.Schema,
) []spamdetect_dto.FieldResult {
	fieldTypes := make(map[string]spamdetect_dto.FieldType, len(schema.Fields()))
	accumulators := make(map[string]*fieldAccumulator, len(schema.Fields()))
	for _, field := range schema.Fields() {
		accumulators[field.Key] = &fieldAccumulator{}
		fieldTypes[field.Key] = field.Type
	}

	for index := range detectorResults {
		if detectorResults[index].Error != nil {
			continue
		}
		s.accumulateDetectorResult(&detectorResults[index], schema, accumulators)
	}

	fieldResults := make([]spamdetect_dto.FieldResult, 0, len(accumulators))
	for key, accumulator := range accumulators {
		score := 0.0
		if accumulator.totalWeight > 0 {
			score = accumulator.totalScore / accumulator.totalWeight
		}
		fieldResults = append(fieldResults, spamdetect_dto.FieldResult{
			Key:     key,
			Type:    fieldTypes[key],
			Score:   score,
			Reasons: accumulator.reasons,
		})
	}

	return fieldResults
}

// resolveDetectorWeight returns the weight for a detector from the
// schema or service config.
//
// Takes detectorName (string) which identifies the detector.
// Takes schema (*spamdetect_dto.Schema) which may override the weight.
//
// Returns float64 which is the resolved weight.
func (s *spamDetectService) resolveDetectorWeight(detectorName string, schema *spamdetect_dto.Schema) float64 {
	if weight := schema.GetDetectorWeight(detectorName); weight > 0 {
		return weight
	}
	if s.detectorWeights != nil {
		if weight, exists := s.detectorWeights[detectorName]; exists && weight > 0 {
			return weight
		}
	}
	return defaultFieldWeight
}

// accumulateDetectorResult adds a single detector's scores to the
// field accumulators.
//
// Takes result (*spamdetect_dto.DetectorResult) which is the verdict.
// Takes schema (*spamdetect_dto.Schema) which provides field info.
// Takes accumulators (map[string]*fieldAccumulator) which receives
// the scores.
func (s *spamDetectService) accumulateDetectorResult(
	result *spamdetect_dto.DetectorResult,
	schema *spamdetect_dto.Schema,
	accumulators map[string]*fieldAccumulator,
) {
	detectorWeight := s.resolveDetectorWeight(result.Detector, schema)

	for fieldKey, fieldScore := range result.FieldScores {
		accumulator, exists := accumulators[fieldKey]
		if !exists {
			continue
		}
		accumulator.totalScore += fieldScore * detectorWeight
		accumulator.totalWeight += detectorWeight
	}

	for fieldKey, accumulator := range accumulators {
		if fieldReasons, hasFieldReasons := result.FieldReasons[fieldKey]; hasFieldReasons {
			accumulator.reasons = append(accumulator.reasons, fieldReasons...)
		}
	}
}

// computeCompositeScore calculates the weighted composite score from
// field-level scores.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the
// individual detector verdicts.
// Takes fieldScores ([]spamdetect_dto.FieldResult) which are the per-field scores.
// Takes schema (*spamdetect_dto.Schema) which provides field weights.
//
// Returns float64 which is the composite score.
// Returns bool which is true when all detectors failed.
func (*spamDetectService) computeCompositeScore(
	detectorResults []spamdetect_dto.DetectorResult,
	fieldScores []spamdetect_dto.FieldResult,
	schema *spamdetect_dto.Schema,
) (float64, bool) {
	allFailed := true
	for index := range detectorResults {
		if detectorResults[index].Error == nil {
			allFailed = false
			break
		}
	}

	if allFailed {
		return 0, true
	}

	fieldWeights := make(map[string]float64)
	for _, field := range schema.Fields() {
		weight := field.Weight
		if weight <= 0 {
			weight = defaultFieldWeight
		}
		fieldWeights[field.Key] = weight
	}

	var weightedSum float64
	var totalWeight float64

	for _, fieldResult := range fieldScores {
		weight, exists := fieldWeights[fieldResult.Key]
		if !exists {
			weight = defaultFieldWeight
		}
		weightedSum += fieldResult.Score * weight
		totalWeight += weight
	}

	formLevelScore, formLevelWeight := accumulateFormLevelScores(detectorResults)
	weightedSum += formLevelScore
	totalWeight += formLevelWeight

	if totalWeight == 0 {
		return 0, false
	}

	return weightedSum / totalWeight, false
}

// accumulateFormLevelScores collects scores from detectors that have no
// per-field breakdown (e.g. honeypot, timing).
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the
// individual detector verdicts.
//
// Returns weightedSum (float64) which is the weighted score total.
// Returns totalWeight (float64) which is the sum of weights.
func accumulateFormLevelScores(detectorResults []spamdetect_dto.DetectorResult) (weightedSum float64, totalWeight float64) {
	for index := range detectorResults {
		if detectorResults[index].Error != nil {
			continue
		}
		if len(detectorResults[index].FieldScores) > 0 || detectorResults[index].Score <= 0 {
			continue
		}
		weightedSum += detectorResults[index].Score * defaultFieldWeight
		totalWeight += defaultFieldWeight
	}
	return weightedSum, totalWeight
}

// getBreakerForDetector returns or creates a circuit breaker for the
// named detector.
//
// Takes name (string) which identifies the detector.
//
// Returns *gobreaker.CircuitBreaker[any] which is the breaker.
//
// Concurrency: Acquires breakerMu for read and write access.
func (s *spamDetectService) getBreakerForDetector(name string) *gobreaker.CircuitBreaker[any] {
	s.breakerMu.RLock()
	breaker, exists := s.breakers[name]
	s.breakerMu.RUnlock()

	if exists {
		return breaker
	}

	s.breakerMu.Lock()
	defer s.breakerMu.Unlock()

	if breaker, exists = s.breakers[name]; exists {
		return breaker
	}

	breaker = newDetectorCircuitBreaker(name)
	s.breakers[name] = breaker
	return breaker
}

// detectorErrorResult creates a DetectorResult representing a detector
// failure.
//
// Takes name (string) which identifies the detector.
// Takes err (error) which is the failure cause.
// Takes duration (time.Duration) which is the elapsed time.
//
// Returns spamdetect_dto.DetectorResult which represents the failure.
func detectorErrorResult(name string, err error, duration time.Duration) spamdetect_dto.DetectorResult {
	return spamdetect_dto.DetectorResult{
		Detector: name,
		Error:    err,
		Duration: duration,
	}
}

// recordDetectorMetric records an OTel counter for a single detector
// invocation.
//
// Takes detectorName (string) which identifies the detector.
// Takes status (string) which is the outcome status.
func recordDetectorMetric(ctx context.Context, detectorName string, status string) {
	spamDetectCheckCount.Add(ctx, 1,
		metricAttributes(attributeKeyOperation, opAnalyse, attributeKeyDetector, detectorName, attributeKeyStatus, status),
	)
}

// newDetectorCircuitBreaker creates a circuit breaker for a detector.
//
// Takes detectorName (string) which identifies the detector.
//
// Returns *gobreaker.CircuitBreaker[any] which is the breaker.
func newDetectorCircuitBreaker(detectorName string) *gobreaker.CircuitBreaker[any] {
	settings := gobreaker.Settings{
		Name:         "spamdetect-" + detectorName,
		MaxRequests:  1,
		Interval:     0,
		Timeout:      circuitBreakerTimeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= circuitBreakerConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[any](settings)
}

// recordAnalyseMetric records OTel metrics for a completed analysis
// operation.
//
// Takes status (string) which is the outcome status.
// Takes isSpam (bool) which indicates the spam verdict.
// Takes duration (time.Duration) which is the elapsed analysis time.
func recordAnalyseMetric(ctx context.Context, status string, isSpam bool, duration time.Duration) {
	if duration > 0 {
		spamDetectCheckDuration.Record(ctx, float64(duration.Milliseconds()),
			metricAttributes(attributeKeyOperation, opAnalyse),
		)
	}
	spamDetectCheckCount.Add(ctx, 1,
		metricAttributes(attributeKeyOperation, opAnalyse, attributeKeyStatus, status, attributeKeyIsSpam, strconv.FormatBool(isSpam)),
	)
}

// generateSubmissionID creates a random base64-encoded submission
// identifier.
//
// Returns string which is the base64-encoded ID.
func generateSubmissionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// collectFormReasons gathers form-level reason strings from all
// detector results.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are
// the detector verdicts.
//
// Returns []string which contains the collected reasons.
func collectFormReasons(detectorResults []spamdetect_dto.DetectorResult) []string {
	var formReasons []string
	for index := range detectorResults {
		if detectorResults[index].Error != nil {
			continue
		}
		formReasons = append(formReasons, detectorResults[index].Reasons...)
	}
	return formReasons
}
