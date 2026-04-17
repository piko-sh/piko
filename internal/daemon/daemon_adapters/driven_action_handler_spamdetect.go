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

package daemon_adapters

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"time"

	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_dto"
)

// validateSpamDetect checks spam detection for actions that implement
// SpamProtected.
//
// Takes request (*http.Request) which provides the HTTP request context.
// Takes action (any) which is the action to check for SpamProtected.
// Takes arguments (map[string]any) which contains the form field values.
// Takes actionName (string) which identifies the action for logging.
//
// Returns error when the submission is detected as spam.
func (h *ActionHandler) validateSpamDetect(
	ctx context.Context,
	request *http.Request,
	action any,
	arguments map[string]any,
	actionName string,
) error {
	spamAction, ok := action.(daemon_domain.SpamProtected)
	if !ok {
		return nil
	}

	schema := spamAction.SpamSchema()
	if schema == nil {
		return nil
	}

	if h.spamdetectService == nil || !h.spamdetectService.IsEnabled() {
		return nil
	}

	submission := buildSpamSubmission(request, arguments, schema, actionName)

	return h.evaluateSpamResult(ctx, submission, schema, action, actionName)
}

// buildSpamSubmission constructs a Submission from the HTTP request
// and schema.
//
// Takes request (*http.Request) which provides the HTTP request data.
// Takes arguments (map[string]any) which contains the form field values.
// Takes schema (*spamdetect_dto.Schema) which describes the form fields.
// Takes actionName (string) which identifies the action.
//
// Returns *spamdetect_dto.Submission which is the populated submission.
func buildSpamSubmission(
	request *http.Request,
	arguments map[string]any,
	schema *spamdetect_dto.Schema,
	actionName string,
) *spamdetect_dto.Submission {
	fields := make(map[string]spamdetect_dto.FieldValue, len(schema.Fields()))
	for _, field := range schema.Fields() {
		fields[field.Key] = spamdetect_dto.FieldValue{
			Value: readStringArg(arguments, field.Key),
			Type:  field.Type,
		}
	}

	honeypotValue := ""
	if honeypotKey := schema.HoneypotKey(); honeypotKey != "" {
		honeypotValue = extractStringArg(arguments, honeypotKey)
	}

	var formLoadedAt time.Time
	if timingKey := schema.TimingKey(); timingKey != "" {
		timestampToken := extractStringArg(arguments, timingKey)
		formLoadedAt = parseFormTimestamp(timestampToken)
	}

	remoteIP := security_dto.ClientIPFromRequest(request)
	if remoteIP == "" {
		remoteIP = request.RemoteAddr
	}

	metadata := copyMetadata(schema.GetMetadata())
	headers := captureHeaders(request, schema.CapturedHeaders())

	return &spamdetect_dto.Submission{
		Fields:          fields,
		Metadata:        metadata,
		Headers:         headers,
		RemoteIP:        remoteIP,
		UserAgent:       request.Header.Get("User-Agent"),
		HoneypotValue:   honeypotValue,
		FormSubmittedAt: time.Now(),
		FormLoadedAt:    formLoadedAt,
		PageURL:         request.Referer(),
		ActionName:      actionName,
	}
}

// copyMetadata returns a shallow clone of the metadata map.
//
// Takes source (map[string]string) which is the metadata to clone.
//
// Returns map[string]string which is the cloned map, or nil if empty.
func copyMetadata(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	return maps.Clone(source)
}

// captureHeaders extracts declared header values from the request.
//
// Takes request (*http.Request) which provides the HTTP headers.
// Takes headerNames ([]string) which lists the headers to capture.
//
// Returns map[string]string which contains the captured values, or nil.
func captureHeaders(request *http.Request, headerNames []string) map[string]string {
	if len(headerNames) == 0 {
		return nil
	}
	captured := make(map[string]string, len(headerNames))
	for _, name := range headerNames {
		if value := request.Header.Get(name); value != "" {
			captured[name] = value
		}
	}
	if len(captured) == 0 {
		return nil
	}
	return captured
}

// evaluateSpamResult runs spam analysis, records the score on the
// action's request metadata, and checks if the submission is spam.
//
// Takes submission (*spamdetect_dto.Submission) which is the form data.
// Takes schema (*spamdetect_dto.Schema) which describes the form fields.
// Takes action (any) which may implement SpamConfigurable and provide
// request metadata.
// Takes actionName (string) which identifies the action for logging.
//
// Returns error when the submission is detected as spam.
func (h *ActionHandler) evaluateSpamResult(
	ctx context.Context,
	submission *spamdetect_dto.Submission,
	schema *spamdetect_dto.Schema,
	action any,
	actionName string,
) error {
	response, err := h.spamdetectService.Analyse(ctx, submission, schema)
	if err != nil {
		return handleSpamAnalyseError(ctx, err, actionName)
	}

	recordSpamScore(action, response)

	threshold := response.Threshold
	if configurable, ok := action.(daemon_domain.SpamConfigurable); ok {
		if config := configurable.SpamConfig(); config != nil && config.ScoreThreshold > 0 {
			threshold = config.ScoreThreshold
		}
	}

	if response.Score >= threshold {
		return spamdetect_dto.ErrSpamDetected
	}

	return nil
}

// recordSpamScore stores the spam detection score and reasons on the
// action's request metadata when the action provides it.
//
// Takes action (any) which may expose request metadata.
// Takes result (*spamdetect_dto.AnalysisResult) which contains the
// analysis verdict.
func recordSpamScore(action any, result *spamdetect_dto.AnalysisResult) {
	if result == nil {
		return
	}

	type requestProvider interface {
		Request() *daemon_dto.RequestMetadata
	}

	provider, ok := action.(requestProvider)
	if !ok {
		return
	}

	requestMeta := provider.Request()
	if requestMeta == nil {
		return
	}

	requestMeta.SpamScore = new(result.Score)

	allReasons := collectSpamReasons(result)
	if len(allReasons) > 0 {
		requestMeta.SpamReasons = allReasons
	}

	fieldScores := collectSpamFieldScores(result)
	if len(fieldScores) > 0 {
		requestMeta.SpamFieldScores = fieldScores
	}
}

// collectSpamReasons gathers all reasons from form-level and
// field-level results.
//
// Takes result (*spamdetect_dto.AnalysisResult) which contains the
// analysis verdict.
//
// Returns []string which contains the combined reasons.
func collectSpamReasons(result *spamdetect_dto.AnalysisResult) []string {
	var reasons []string
	reasons = append(reasons, result.FormReasons...)
	for _, fieldResult := range result.FieldResults {
		reasons = append(reasons, fieldResult.Reasons...)
	}
	return reasons
}

// collectSpamFieldScores extracts non-zero field scores from the
// analysis result.
//
// Takes result (*spamdetect_dto.AnalysisResult) which contains the
// analysis verdict.
//
// Returns map[string]float64 which maps field keys to their scores.
func collectSpamFieldScores(result *spamdetect_dto.AnalysisResult) map[string]float64 {
	if len(result.FieldResults) == 0 {
		return nil
	}
	scores := make(map[string]float64, len(result.FieldResults))
	for _, fieldResult := range result.FieldResults {
		if fieldResult.Score > 0 {
			scores[fieldResult.Key] = fieldResult.Score
		}
	}
	if len(scores) == 0 {
		return nil
	}
	return scores
}

// handleSpamAnalyseError converts spam analysis errors into appropriate
// responses.
//
// Takes err (error) which is the analysis error.
// Takes actionName (string) which identifies the action for logging.
//
// Returns error which is nil when the error is non-fatal, or a
// rejection error otherwise.
func handleSpamAnalyseError(ctx context.Context, err error, actionName string) error {
	if errors.Is(err, spamdetect_dto.ErrSpamDetectDisabled) {
		return nil
	}
	if errors.Is(err, spamdetect_dto.ErrAllDetectorsFailed) {
		_, l := logger_domain.From(ctx, log)
		l.Error("All spam detection detectors failed; rejecting submission as a precaution",
			logger_domain.String(attributeKeyAction, actionName),
		)
		return spamdetect_dto.ErrSpamDetected
	}
	if errors.Is(err, spamdetect_dto.ErrNoMatchingDetectors) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("No spam detectors match schema signals; allowing submission",
			logger_domain.String(attributeKeyAction, actionName),
		)
		return nil
	}
	return fmt.Errorf("spam detection analysis failed: %w", err)
}

// readStringArg reads a string value from the arguments map without
// deleting the key.
//
// Takes arguments (map[string]any) which contains the form arguments.
// Takes key (string) which identifies the argument to read.
//
// Returns string which is the value, or empty if not present.
func readStringArg(arguments map[string]any, key string) string {
	raw, ok := arguments[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

// extractStringArg reads a string value from the arguments map and
// deletes the key.
//
// Takes arguments (map[string]any) which contains the form arguments.
// Takes key (string) which identifies the argument to extract.
//
// Returns string which is the value, or empty if not present.
func extractStringArg(arguments map[string]any, key string) string {
	raw, ok := arguments[key]
	delete(arguments, key)
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

// writeSpamDetectError writes a spam detection rejection response and
// logs the underlying error for debugging.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes err (error) which is the underlying spam detection error.
func (h *ActionHandler) writeSpamDetectError(ctx context.Context, w http.ResponseWriter, err error) {
	_, l := logger_domain.From(ctx, log)
	l.Debug("Writing spam detection error response", logger_domain.Error(err))

	h.writeJSON(w, http.StatusForbidden, map[string]any{
		"status":  http.StatusForbidden,
		"code":    "SPAM_DETECTED",
		"message": "Your submission was flagged by our spam filter. Please try again or contact support.",
	})
}

// parseFormTimestamp parses a millisecond Unix timestamp string into
// time.Time.
//
// Takes token (string) which is the millisecond timestamp.
//
// Returns time.Time which is the parsed time, or zero if invalid.
func parseFormTimestamp(token string) time.Time {
	if token == "" {
		return time.Time{}
	}

	millis, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.UnixMilli(millis)
}
