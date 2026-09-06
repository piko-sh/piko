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
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
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

	// safeCallDetectorFeedback is the goroutine.SafeCall label for feedback dispatch.
	safeCallDetectorFeedback = "spamdetect.ReportFeedback"

	// safeCallDetectorHealth is the goroutine.SafeCall label for health checks.
	safeCallDetectorHealth = "spamdetect.HealthCheck"

	// safeCallDrain is the goroutine.SafeCall label for the Close drain goroutine.
	safeCallDrain = "spamdetect.Close.drain"

	// defaultFieldWeight is the default scoring weight for fields and detectors.
	defaultFieldWeight = 1.0

	// defaultHealthCheckTimeout is the per-detector readiness deadline.
	defaultHealthCheckTimeout = 2 * time.Second

	// operationAnalyse is the operation label embedded in SpamDetectError wrappers produced
	// by the service.
	operationAnalyse = "analyse"

	// submissionIDByteLength is the number of random bytes used to seed a submission
	// identifier. 16 bytes produces a 22-character base64-encoded ID with 128 bits of
	// entropy.
	submissionIDByteLength = 16
)

// spamDetectService is the concrete implementation of SpamDetectServicePort.
type spamDetectService struct {
	// feedbackStore persists spam/ham feedback reports.
	feedbackStore FeedbackStore

	// clock is the time source used for analysis duration and cache timestamps. Defaults to
	// wall-clock time.
	clock clock.Clock

	// shutdown signals in-flight analyses to abort during Close.
	shutdown chan struct{}

	// detectorWeights maps detector names to their scoring weight.
	detectorWeights map[string]float64

	// cacheEntries maps submission IDs to cached analysis records.
	cacheEntries map[string]*cachedRecord

	// matchCache caches detector matching by schema identity to avoid rebuilding the signal
	// set per request.
	matchCache map[*spamdetect_dto.Schema][]string

	// breakers holds per-detector circuit breakers.
	breakers map[string]*gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult]

	// registry stores and looks up detectors by name.
	registry *provider_domain.StandardRegistry[Detector]

	// cacheKeys tracks insertion order for ring-buffer eviction.
	cacheKeys []string

	// inflight tracks in-flight analysis operations for graceful shutdown.
	inflight sync.WaitGroup

	// cacheIndex is the ring-buffer write position.
	cacheIndex int

	// cacheSize is the maximum number of cached analysis records.
	cacheSize int

	// timeout is the maximum duration to wait for all detectors.
	timeout time.Duration

	// healthCheckTimeout caps per-detector readiness probes.
	healthCheckTimeout time.Duration

	// scoreThreshold is the default composite score above which a submission is spam.
	scoreThreshold float64

	// breakerMu guards concurrent access to the breakers map.
	breakerMu sync.RWMutex

	// feedbackMu guards concurrent access to feedbackStore.
	feedbackMu sync.RWMutex

	// matchCacheMu guards concurrent access to matchCache.
	matchCacheMu sync.RWMutex

	// closed ensures Close runs exactly once.
	closed sync.Once

	// cacheMu guards concurrent access to cacheEntries and cacheKeys.
	cacheMu sync.Mutex

	// registerMu serialises detector registration with the cap check so that the goroutine
	// fan-out limit is not breached by racing callers.
	registerMu sync.Mutex
}

// cachedRecord pairs a submission with its analysis result for feedback correlation.
type cachedRecord struct {
	// submission is the original submission, deep-copied at insert time so callers cannot
	// mutate the cached value.
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
	// reasons collects per-detector reason strings for the field.
	reasons []string

	// totalScore is the weighted sum of detector scores for the field.
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

// WithFeedbackStore sets the feedback persistence backend at construction time.
// Equivalent to calling SetFeedbackStore after NewSpamDetectService.
//
// Takes store (FeedbackStore) which persists spam/ham feedback.
//
// Returns ServiceOption which configures the store.
func WithFeedbackStore(store FeedbackStore) ServiceOption {
	return func(s *spamDetectService) {
		s.feedbackStore = store
	}
}

// WithClock sets the time source used for duration measurements and cache timestamps.
// Tests inject a mock clock for deterministic behaviour.
//
// Takes c (clock.Clock) which provides the time source.
//
// Returns ServiceOption which configures the clock.
func WithClock(c clock.Clock) ServiceOption {
	return func(s *spamDetectService) {
		if c != nil {
			s.clock = c
		}
	}
}

// WithHealthCheckTimeout sets the per-detector readiness probe deadline. Zero or negative
// values reset to the default.
//
// Takes timeout (time.Duration) which is the maximum per-detector health probe duration.
//
// Returns ServiceOption which configures the readiness timeout.
func WithHealthCheckTimeout(timeout time.Duration) ServiceOption {
	return func(s *spamDetectService) {
		if timeout > 0 {
			s.healthCheckTimeout = timeout
		}
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

	cacheSize := resolveFeedbackCacheSize(config.FeedbackCacheSize)

	service := &spamDetectService{
		registry:           provider_domain.NewStandardRegistry[Detector]("spamdetect"),
		breakers:           make(map[string]*gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult]),
		detectorWeights:    config.DetectorWeights,
		cacheEntries:       make(map[string]*cachedRecord, cacheSize),
		cacheKeys:          make([]string, 0, cacheSize),
		matchCache:         make(map[*spamdetect_dto.Schema][]string),
		shutdown:           make(chan struct{}),
		clock:              clock.RealClock(),
		scoreThreshold:     config.ScoreThreshold,
		timeout:            config.Timeout,
		cacheSize:          cacheSize,
		healthCheckTimeout: defaultHealthCheckTimeout,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service, nil
}

// Analyse runs all matching detectors in parallel and returns a composite verdict with
// per-field breakdowns.
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

	if err := s.prepareSubmission(ctx, submission, schema); err != nil {
		return nil, err
	}

	matchingDetectors := s.findMatchingDetectors(ctx, schema)
	if len(matchingDetectors) == 0 {
		return nil, spamdetect_dto.ErrNoMatchingDetectors
	}

	s.inflight.Add(1)
	defer s.inflight.Done()

	startTime := s.clock.Now()
	analyseCtx, analyseCancel := context.WithTimeoutCause(
		ctx,
		s.timeout,
		fmt.Errorf("spam detection analysis exceeded %s timeout", s.timeout),
	)
	defer analyseCancel()

	detectorResults := s.runDetectors(analyseCtx, matchingDetectors, submission, schema)
	result := s.aggregateResults(detectorResults, schema, s.clock.Now().Sub(startTime))

	if submission.WasTruncated() {
		result.analysisResult.Truncated = true
		result.analysisResult.TruncatedFields = submission.TruncatedFields()
	}

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
// Takes ctx (context.Context) which is the caller context.
//
// Returns bool which is true when detectors are available.
func (s *spamDetectService) IsEnabled(ctx context.Context) bool {
	return len(s.registry.ListProviders(ctx)) > 0
}

// RegisterDetector adds a new detector with the given name.
//
// Takes name (string) which identifies the detector.
// Takes detector (Detector) which handles spam analysis.
//
// Returns error when the detector cannot be registered.
//
// Concurrency: Safe for concurrent use; the cap check and registry write are guarded by
// registerMu so the goroutine fan-out limit is enforced even under racing callers.
func (s *spamDetectService) RegisterDetector(ctx context.Context, name string, detector Detector) error {
	if name == "" {
		return spamdetect_dto.ErrDetectorNameEmpty
	}
	if detector == nil {
		return spamdetect_dto.ErrDetectorNil
	}

	s.registerMu.Lock()
	defer s.registerMu.Unlock()

	existing := s.registry.ListProviders(ctx)
	if len(existing) >= spamdetect_dto.MaxDetectorCount() {
		return fmt.Errorf("%w: limit is %d", spamdetect_dto.ErrTooManyDetectors, spamdetect_dto.MaxDetectorCount())
	}

	if err := s.registry.RegisterProvider(ctx, name, detector); err != nil {
		return fmt.Errorf("spamdetect: registering detector %q: %w", name, err)
	}

	s.breakerMu.Lock()
	s.breakers[name] = newDetectorCircuitBreaker(name)
	s.breakerMu.Unlock()

	s.invalidateMatchCache()

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
// Takes ctx (context.Context) which is the caller context.
// Takes name (string) which is the detector name to look up.
//
// Returns bool which is true if the detector exists.
func (s *spamDetectService) HasDetector(ctx context.Context, name string) bool {
	_, err := s.registry.GetProvider(ctx, name)
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
	ctx, l := logger_domain.From(ctx, log)
	detectors := s.registry.ListProviders(ctx)
	var errs []error
	for _, info := range detectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			l.Warn("Failed to resolve detector for health check",
				logger_domain.String(attributeKeyDetector, info.Name),
				logger_domain.Error(err),
			)
			errs = append(errs, fmt.Errorf("detector %s: %w", info.Name, errors.Join(spamdetect_dto.ErrDetectorUnavailable, err)))
			continue
		}
		if err := s.invokeDetectorHealthCheck(ctx, info.Name, detector); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close waits for in-flight analyses to complete, then shuts down all detectors. Signals
// shutdown so analyses can abort early.
//
// Returns error when shutdown fails or the context is cancelled.
//
// Concurrency: Safe to call multiple times; guarded by sync.Once.
func (s *spamDetectService) Close(ctx context.Context) error {
	var closeErr error
	s.closed.Do(func() {
		close(s.shutdown)

		done := make(chan struct{})
		go func() {
			defer close(done)
			defer goroutine.RecoverPanic(ctx, safeCallDrain)
			s.inflight.Wait()
		}()

		select {
		case <-done:
		case <-ctx.Done():
			closeErr = fmt.Errorf("spamdetect: timed out waiting for in-flight analyses: %w", ctx.Err())
			return
		}

		if err := s.registry.CloseAll(ctx); err != nil {
			closeErr = fmt.Errorf("spamdetect: closing detectors: %w", err)
		}
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

// ReportSpam records that a submission was confirmed as spam and notifies feedback-aware
// detectors.
//
// Takes submissionID (string) which identifies the submission.
//
// Returns error when persistence or notification fails.
func (s *spamDetectService) ReportSpam(ctx context.Context, submissionID string) error {
	return s.reportFeedback(ctx, submissionID, true)
}

// ReportHam records that a submission was confirmed as legitimate and notifies
// feedback-aware detectors.
//
// Takes submissionID (string) which identifies the submission.
//
// Returns error when persistence or notification fails.
func (s *spamDetectService) ReportHam(ctx context.Context, submissionID string) error {
	return s.reportFeedback(ctx, submissionID, false)
}

// prepareSubmission validates inputs, assigns a submission identifier if absent, and runs
// the schema-aware sanitisation step.
//
// Takes ctx (context.Context) which is the caller context.
// Takes submission (*spamdetect_dto.Submission) which is mutated in place.
// Takes schema (*spamdetect_dto.Schema) which provides per-field limits.
//
// Returns error when validation or ID generation fails.
func (*spamDetectService) prepareSubmission(ctx context.Context, submission *spamdetect_dto.Submission, schema *spamdetect_dto.Schema) error {
	_, l := logger_domain.From(ctx, log)

	if submission == nil {
		return fmt.Errorf("spamdetect.Analyse: %w", spamdetect_dto.ErrSubmissionNil)
	}
	if schema == nil {
		return fmt.Errorf("spamdetect.Analyse: %w", spamdetect_dto.ErrSchemaNil)
	}

	if submission.ID == "" {
		id, err := generateSubmissionID()
		if err != nil {
			l.Error("Failed to generate submission identifier", logger_domain.Error(err))
			return fmt.Errorf("spamdetect.Analyse: %w", err)
		}
		submission.ID = id
	}

	submission.Sanitise(schema)
	if submission.WasTruncated() {
		l.Trace("Submission fields were truncated during sanitisation",
			logger_domain.Int("truncated_field_count", len(submission.TruncatedFields())),
		)
	}
	return nil
}

// reportFeedback persists a feedback report and notifies feedback-aware detectors. Errors
// from the store are wrapped and returned without duplicate logging at this layer; the
// top-level handler logs them alongside its own operational context.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns error when persistence or notification fails.
//
// Concurrency: Acquires feedbackMu for store access and cacheMu transitively via
// buildFeedbackRecord.
func (s *spamDetectService) reportFeedback(ctx context.Context, submissionID string, isSpam bool) error {
	record := s.buildFeedbackRecord(ctx, submissionID, isSpam)

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
			return fmt.Errorf("spamdetect: storing feedback (is_spam=%t): %w", isSpam, storeErr)
		}
	}

	return s.notifyFeedbackDetectors(ctx, submissionID, isSpam)
}

// buildFeedbackRecord constructs a SubmissionRecord from the cache.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns *spamdetect_dto.SubmissionRecord which is the feedback record.
//
// Concurrency: Acquires cacheMu to read from the cache.
func (s *spamDetectService) buildFeedbackRecord(ctx context.Context, submissionID string, isSpam bool) *spamdetect_dto.SubmissionRecord {
	record := &spamdetect_dto.SubmissionRecord{
		SubmissionID: submissionID,
		ReportedAt:   s.clock.Now(),
		IsSpam:       isSpam,
	}

	s.cacheMu.Lock()
	entry, found := s.cacheEntries[submissionID]
	s.cacheMu.Unlock()

	if found {
		record.Submission = entry.submission
		record.Result = entry.result
		return record
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("No cached analysis result for feedback submission",
		logger_domain.String("submission_id", submissionID),
	)
	return record
}

// cacheRecord stores a submission and result in the ring-buffer cache. The submission is
// cloned so subsequent caller mutations do not change the cached value.
//
// Takes submission (*spamdetect_dto.Submission) which is the form data.
// Takes result (*spamdetect_dto.AnalysisResult) which is the verdict.
//
// Concurrency: Acquires cacheMu to write to the cache.
func (s *spamDetectService) cacheRecord(submission *spamdetect_dto.Submission, result *spamdetect_dto.AnalysisResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	entry := &cachedRecord{
		submission: submission.Clone(),
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

// notifyFeedbackDetectors forwards feedback to detectors that implement
// FeedbackAwareDetector. Each invocation is panic-isolated via goroutine.SafeCall.
//
// Takes submissionID (string) which identifies the submission.
// Takes isSpam (bool) which is true for spam, false for ham.
//
// Returns error when any detector notification fails.
func (s *spamDetectService) notifyFeedbackDetectors(ctx context.Context, submissionID string, isSpam bool) error {
	ctx, l := logger_domain.From(ctx, log)
	detectors := s.registry.ListProviders(ctx)
	var errs []error
	for _, info := range detectors {
		detector, err := s.registry.GetProvider(ctx, info.Name)
		if err != nil {
			l.Warn("Failed to resolve detector for feedback dispatch",
				logger_domain.String(attributeKeyDetector, info.Name),
				logger_domain.Error(err),
			)
			errs = append(errs, fmt.Errorf("detector %s: %w", info.Name, errors.Join(spamdetect_dto.ErrDetectorUnavailable, err)))
			continue
		}
		feedbackDetector, ok := detector.(FeedbackAwareDetector)
		if !ok {
			continue
		}

		detectorName := info.Name
		callErr := goroutine.SafeCall(ctx, safeCallDetectorFeedback, func() error {
			return feedbackDetector.ReportFeedback(ctx, submissionID, isSpam)
		})
		if callErr != nil {
			errs = append(errs, spamdetect_dto.NewSpamDetectError("feedback", detectorName, callErr))
		}
	}
	return errors.Join(errs...)
}

// findMatchingDetectors returns detectors whose signals overlap with the schema's
// declared signals. The matching set is cached by schema identity since schemas are
// immutable post-construction.
//
// Takes ctx (context.Context) which is the caller context.
// Takes schema (*spamdetect_dto.Schema) which declares the required signals.
//
// Returns []detectorInfo which contains the matching detectors.
func (s *spamDetectService) findMatchingDetectors(ctx context.Context, schema *spamdetect_dto.Schema) []detectorInfo {
	ctx, l := logger_domain.From(ctx, log)
	matchedNames := s.matchedDetectorNames(schema)

	matched := make([]detectorInfo, 0, len(matchedNames))
	for _, name := range matchedNames {
		detector, err := s.registry.GetProvider(ctx, name)
		if err != nil {
			l.Warn("Failed to resolve matching detector",
				logger_domain.String(attributeKeyDetector, name),
				logger_domain.Error(err),
			)
			continue
		}
		matched = append(matched, detectorInfo{name: name, detector: detector})
	}
	return matched
}

// matchedDetectorNames returns the cached or freshly computed list of detector names
// whose signals overlap the schema.
//
// Takes schema (*spamdetect_dto.Schema) which declares signals.
//
// Returns []string which contains the matching detector names.
//
// Concurrency: Acquires matchCacheMu for read and write access; the computation outside
// the cache uses the registry's own locking.
func (s *spamDetectService) matchedDetectorNames(schema *spamdetect_dto.Schema) []string {
	s.matchCacheMu.RLock()
	cached, ok := s.matchCache[schema]
	s.matchCacheMu.RUnlock()
	if ok {
		return cached
	}

	names := s.computeMatchedDetectorNames(schema)

	s.matchCacheMu.Lock()
	if existing, ok := s.matchCache[schema]; ok {
		s.matchCacheMu.Unlock()
		return existing
	}
	s.matchCache[schema] = names
	s.matchCacheMu.Unlock()
	return names
}

// computeMatchedDetectorNames computes the detector names matching the schema's signal
// set.
//
// Takes schema (*spamdetect_dto.Schema) which declares signals.
//
// Returns []string which contains the matching detector names.
func (s *spamDetectService) computeMatchedDetectorNames(schema *spamdetect_dto.Schema) []string {
	schemaSignals := schema.AllSignals()
	signalSet := make(map[spamdetect_dto.Signal]struct{}, len(schemaSignals))
	for _, signal := range schemaSignals {
		signalSet[signal] = struct{}{}
	}

	allDetectors := s.registry.ListProviders(context.Background())
	var matched []string

	for _, info := range allDetectors {
		detector, err := s.registry.GetProvider(context.Background(), info.Name)
		if err != nil {
			continue
		}
		for _, detectorSignal := range detector.Signals() {
			if _, exists := signalSet[detectorSignal]; exists {
				matched = append(matched, info.Name)
				break
			}
		}
	}
	return matched
}

// invalidateMatchCache clears the detector-matching cache when the registered detector
// set changes.
//
// Concurrency: Acquires matchCacheMu for write access.
func (s *spamDetectService) invalidateMatchCache() {
	s.matchCacheMu.Lock()
	if len(s.matchCache) > 0 {
		s.matchCache = make(map[*spamdetect_dto.Schema][]string)
	}
	s.matchCacheMu.Unlock()
}

// runDetectors groups detectors by priority tier and executes tiers sequentially, with
// parallel execution within each tier.
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
		if ctx.Err() != nil {
			break
		}
		tierResults := s.runTier(ctx, tier, submission, schema)
		allResults = append(allResults, tierResults...)

		if compositeScore(allResults, schema, s.scoreThreshold) >= threshold {
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

// runTier executes all detectors in a single priority tier in parallel.
//
// Each goroutine is panic-isolated; an unexpected panic outside SafeCall propagates as an
// error in the corresponding result slot.
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
			defer goroutine.RecoverPanic(ctx, "spamdetect.runTier")
			results[resultIndex] = s.runSingleDetector(ctx, detectorName, det, submission, schema)
		})
	}

	waitGroup.Wait()
	return results
}

// runSingleDetector executes one detector with circuit breaker and panic protection.
// SafeCall sits inside the breaker so that panics count as failures and a flapping
// detector trips the breaker after the configured failure threshold.
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
	startTime := s.clock.Now()

	breaker := s.getBreakerForDetector(detectorName)
	result, callErr := breaker.Execute(func() (*spamdetect_dto.DetectorResult, error) {
		return goroutine.SafeCall1(ctx, safeCallDetectorAnalyse, func() (*spamdetect_dto.DetectorResult, error) {
			return detector.Analyse(ctx, submission, schema)
		})
	})
	duration := s.clock.Now().Sub(startTime)

	if callErr != nil {
		recordDetectorMetric(ctx, detectorName, statusError)
		wrapped := spamdetect_dto.NewSpamDetectError(operationAnalyse, detectorName, callErr)
		return detectorErrorResult(detectorName, wrapped, duration)
	}

	if result == nil {
		recordDetectorMetric(ctx, detectorName, statusError)
		wrapped := spamdetect_dto.NewSpamDetectError(operationAnalyse, detectorName, spamdetect_dto.ErrUnexpectedDetectorResponse)
		return detectorErrorResult(detectorName, wrapped, duration)
	}

	result.Detector = detectorName
	result.Duration = duration
	recordDetectorMetric(ctx, detectorName, statusSuccess)

	return *result
}

// invokeDetectorHealthCheck runs a detector health check under a short timeout with panic
// protection.
//
// Takes ctx (context.Context) which is the caller context.
// Takes name (string) which identifies the detector.
// Takes detector (Detector) which is the detector to probe.
//
// Returns error wrapped with operation context when the check fails or times out.
func (s *spamDetectService) invokeDetectorHealthCheck(ctx context.Context, name string, detector Detector) error {
	probeCtx, cancel := context.WithTimeoutCause(
		ctx,
		s.healthCheckTimeout,
		fmt.Errorf("spam detection health probe exceeded %s timeout", s.healthCheckTimeout),
	)
	defer cancel()

	err := goroutine.SafeCall(probeCtx, safeCallDetectorHealth, func() error {
		return detector.HealthCheck(probeCtx)
	})
	if err != nil {
		return fmt.Errorf("detector %s health check: %w", name, err)
	}
	return nil
}

// aggregateResults computes per-field scores and a weighted composite score.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
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
	score, allFailed := s.computeCompositeScore(detectorResults, fieldScores, schema)

	result := &spamdetect_dto.AnalysisResult{
		DetectorResults: detectorResults,
		FieldResults:    fieldScores,
		FormReasons:     collectFormReasons(detectorResults),
		Duration:        totalDuration,
		Score:           score,
		Threshold:       threshold,
		IsSpam:          score >= threshold,
	}

	return aggregationResult{analysisResult: result, allFailed: allFailed}
}

// computeFieldScores calculates per-field scores using detector weights and precise
// per-field reason attribution.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
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

// resolveDetectorWeight returns the weight for a detector from the schema or service
// config.
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

// accumulateDetectorResult adds a single detector's scores to the field accumulators.
//
// Takes result (*spamdetect_dto.DetectorResult) which is the verdict.
// Takes schema (*spamdetect_dto.Schema) which provides field info.
// Takes accumulators (map[string]*fieldAccumulator) which receives the scores.
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

// computeCompositeScore calculates the weighted composite score from field-level scores.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
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

// accumulateFormLevelScores collects scores from detectors that have no per-field
// breakdown (e.g. honeypot, timing).
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
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

// compositeScore performs a lightweight composite score calculation used by runDetectors
// for the tier short-circuit. The full aggregateResults path is reserved for the final
// result so the hot-path allocations stay outside the tier loop.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
// Takes schema (*spamdetect_dto.Schema) which provides field weights.
//
// Returns float64 which is the composite score, or zero when no successful detectors are
// present.
func compositeScore(detectorResults []spamdetect_dto.DetectorResult, schema *spamdetect_dto.Schema, _ float64) float64 {
	if !hasSuccessfulDetector(detectorResults) {
		return 0
	}

	fieldWeights := buildFieldWeights(schema)
	fieldTotals, fieldDivisors := aggregateFieldRatios(detectorResults, fieldWeights)

	weightedSum, totalWeight := combineFieldScores(fieldWeights, fieldTotals, fieldDivisors)
	formSum, formWeight := accumulateFormLevelScores(detectorResults)
	weightedSum += formSum
	totalWeight += formWeight

	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// hasSuccessfulDetector reports whether at least one detector result has no error.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
//
// Returns bool which is true when any successful result exists.
func hasSuccessfulDetector(detectorResults []spamdetect_dto.DetectorResult) bool {
	for index := range detectorResults {
		if detectorResults[index].Error == nil {
			return true
		}
	}
	return false
}

// buildFieldWeights returns the per-field weight map for a schema, substituting the
// default weight for any zero or negative entry.
//
// Takes schema (*spamdetect_dto.Schema) which provides field weights.
//
// Returns map[string]float64 which maps field keys to their weights.
func buildFieldWeights(schema *spamdetect_dto.Schema) map[string]float64 {
	fields := schema.Fields()
	weights := make(map[string]float64, len(fields))
	for _, field := range fields {
		weight := field.Weight
		if weight <= 0 {
			weight = defaultFieldWeight
		}
		weights[field.Key] = weight
	}
	return weights
}

// aggregateFieldRatios sums detector field scores into per-key totals and contributor
// counts, restricting accumulation to fields declared in the supplied weights map.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the individual
// detector verdicts.
// Takes fieldWeights (map[string]float64) which is the set of valid keys.
//
// Returns totals (map[string]float64) which maps keys to summed ratios.
// Returns divisors (map[string]float64) which maps keys to contributor counts.
func aggregateFieldRatios(detectorResults []spamdetect_dto.DetectorResult, fieldWeights map[string]float64) (totals map[string]float64, divisors map[string]float64) {
	totals = make(map[string]float64, len(fieldWeights))
	divisors = make(map[string]float64, len(fieldWeights))
	for index := range detectorResults {
		result := &detectorResults[index]
		if result.Error != nil {
			continue
		}
		for fieldKey, fieldScore := range result.FieldScores {
			if _, exists := fieldWeights[fieldKey]; !exists {
				continue
			}
			totals[fieldKey] += fieldScore
			divisors[fieldKey]++
		}
	}
	return totals, divisors
}

// combineFieldScores reduces field totals and divisors into a single weighted sum and
// weight total, using each field's configured weight.
//
// Takes fieldWeights (map[string]float64) which maps keys to weights.
// Takes fieldTotals (map[string]float64) which maps keys to summed ratios.
// Takes fieldDivisors (map[string]float64) which maps keys to contributor counts.
//
// Returns weightedSum (float64) which is the weighted aggregate.
// Returns totalWeight (float64) which is the sum of contributing weights.
func combineFieldScores(fieldWeights map[string]float64, fieldTotals map[string]float64, fieldDivisors map[string]float64) (weightedSum float64, totalWeight float64) {
	for key, weight := range fieldWeights {
		divisor := fieldDivisors[key]
		if divisor == 0 {
			continue
		}
		weightedSum += (fieldTotals[key] / divisor) * weight
		totalWeight += weight
	}
	return weightedSum, totalWeight
}

// getBreakerForDetector returns or creates a circuit breaker for the named detector.
//
// Takes name (string) which identifies the detector.
//
// Returns *gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult] which is the breaker.
//
// Concurrency: Acquires breakerMu for read and write access.
func (s *spamDetectService) getBreakerForDetector(name string) *gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult] {
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

// resolveFeedbackCacheSize clamps the configured size to the supported range, falling
// back to the default for zero or negative inputs.
//
// Takes configured (int) which is the value from ServiceConfig.
//
// Returns int which is the clamped cache size.
func resolveFeedbackCacheSize(configured int) int {
	if configured <= 0 {
		return spamdetect_dto.DefaultFeedbackCacheSize
	}
	if maximum := spamdetect_dto.MaxFeedbackCacheSize(); configured > maximum {
		return maximum
	}
	return configured
}

// detectorErrorResult creates a DetectorResult representing a detector failure.
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

// recordDetectorMetric records an OTel counter for a single detector invocation.
//
// Takes detectorName (string) which identifies the detector.
// Takes status (string) which is the outcome status.
func recordDetectorMetric(ctx context.Context, detectorName string, status string) {
	spamDetectCheckCount.Add(ctx, 1,
		metricAttributes(ctx, attributeKeyOperation, opAnalyse, attributeKeyDetector, detectorName, attributeKeyStatus, status),
	)
}

// newDetectorCircuitBreaker creates a circuit breaker for a detector.
//
// Takes detectorName (string) which identifies the detector.
//
// Returns *gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult] which is the breaker.
func newDetectorCircuitBreaker(detectorName string) *gobreaker.CircuitBreaker[*spamdetect_dto.DetectorResult] {
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
	return gobreaker.NewCircuitBreaker[*spamdetect_dto.DetectorResult](settings)
}

// recordAnalyseMetric records OTel metrics for a completed analysis operation.
//
// Takes status (string) which is the outcome status.
// Takes isSpam (bool) which indicates the spam verdict.
// Takes duration (time.Duration) which is the elapsed analysis time.
func recordAnalyseMetric(ctx context.Context, status string, isSpam bool, duration time.Duration) {
	if duration > 0 {
		spamDetectCheckDuration.Record(ctx, float64(duration.Milliseconds()),
			metricAttributes(ctx, attributeKeyOperation, opAnalyse),
		)
	}
	spamDetectCheckCount.Add(ctx, 1,
		metricAttributes(ctx, attributeKeyOperation, opAnalyse, attributeKeyStatus, status, attributeKeyIsSpam, strconv.FormatBool(isSpam)),
	)
}

// generateSubmissionID creates a random base64-encoded submission identifier. Returns an
// error when the system entropy source fails so callers can decide whether to proceed
// with a deterministic fallback or reject the analysis.
//
// Returns string which is the base64-encoded ID.
// Returns error which wraps ErrSubmissionIDGeneration on entropy failure.
func generateSubmissionID() (string, error) {
	bytes := make([]byte, submissionIDByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("%w: %w", spamdetect_dto.ErrSubmissionIDGeneration, err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// collectFormReasons gathers form-level reason strings from all detector results.
//
// Takes detectorResults ([]spamdetect_dto.DetectorResult) which are the detector
// verdicts.
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
