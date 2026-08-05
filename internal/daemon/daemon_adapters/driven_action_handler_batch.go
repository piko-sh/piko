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
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/binder"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/safeerror"
)

// handleBatch processes a batch action request (continue all, report failures).
//
// Takes w (http.ResponseWriter) which receives the JSON response.
// Takes request (*http.Request) which contains the batch action request body.
func (h *ActionHandler) handleBatch(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, h.maxBodyBytes)

	ctx := extractOTelContext(request)
	ctx, span := tracer.Start(ctx, "handleBatchActionRequest")
	defer span.End()

	ctx, l := logger_domain.From(ctx, log)
	h.trackBatchMetrics(ctx, request)
	l.Trace("Handling batch action request")

	batchReq, ok := h.parseBatchRequest(w, request, span, l)
	if !ok {
		return
	}

	if len(batchReq.Actions) == 0 {
		h.writeJSON(w, http.StatusOK, daemon_dto.BatchActionResponse{Results: []daemon_dto.BatchActionResult{}, Success: true})
		return
	}

	if len(batchReq.Actions) > maxBatchActions {
		h.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("batch exceeds maximum of %d actions", maxBatchActions), nil, isDevelopmentModeFromContext(ctx))
		return
	}

	span.SetAttributes(attribute.Int("batch.action_count", len(batchReq.Actions)))
	results, allSuccess := h.executeBatchActions(ctx, request, batchReq.Actions)

	l.Trace("Batch action request completed",
		logger_domain.Int("total_actions", len(batchReq.Actions)),
		logger_domain.Bool("all_success", allSuccess),
	)
	span.SetStatus(codes.Ok, "Batch request completed")
	h.writeJSON(w, http.StatusOK, daemon_dto.BatchActionResponse{Results: results, Success: allSuccess})
}

// parseBatchRequest reads, depth-validates, decodes, and CSRF-validates a batch action
// request.
//
// On any failure it writes the appropriate HTTP error and returns ok=false.
//
// Takes w (http.ResponseWriter) which receives any HTTP error responses.
// Takes request (*http.Request) which provides the batch request body and headers.
// Takes span (trace.Span) which records errors encountered during parsing.
// Takes l (logger_domain.Logger) which provides structured logging for failures.
//
// Returns daemon_dto.BatchActionRequest which contains the decoded batch request when
// parsing succeeds, or a zero value on failure.
// Returns bool which is true when the request was parsed successfully and false when an
// HTTP error response has already been written.
func (h *ActionHandler) parseBatchRequest(w http.ResponseWriter, request *http.Request, span trace.Span, l logger_domain.Logger) (daemon_dto.BatchActionRequest, bool) {
	var batchReq daemon_dto.BatchActionRequest

	developmentMode := isDevelopmentModeFromContext(request.Context())

	body, err := io.ReadAll(request.Body)
	if err != nil {
		l.ReportError(span, err, "Failed to read batch request body")
		h.writeError(w, http.StatusBadRequest, "Invalid batch request body", err, developmentMode)
		return batchReq, false
	}

	if err := validateJSONStructuralDepth(body, defaultMaxJSONBodyDepth); err != nil {
		l.ReportError(span, err, "Batch request body exceeds nesting limit")
		h.writeError(w, http.StatusBadRequest, "Invalid batch request body", err, developmentMode)
		return batchReq, false
	}

	decoder := stdjson.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&batchReq); err != nil {
		l.ReportError(span, err, "Failed to parse batch request body")
		h.writeError(w, http.StatusBadRequest, "Invalid batch request body", err, developmentMode)
		return batchReq, false
	}

	if csrfErr := h.validateCSRFWithToken(request, batchReq.CSRFEphemeralToken); csrfErr != nil {
		l.Warn("CSRF validation failed for batch request",
			logger_domain.Error(csrfErr),
		)
		h.writeCSRFError(w, csrfErr)
		return batchReq, false
	}

	return batchReq, true
}

// trackBatchMetrics records metrics for a batch action request.
//
// Takes request (*http.Request) which provides the HTTP method for metric labels.
func (*ActionHandler) trackBatchMetrics(ctx context.Context, request *http.Request) {
	actionRequestCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, "_batch"), attribute.String("method", request.Method)),
	)
}

// executeBatchActions executes all actions in the batch and returns results.
//
// Takes request (*http.Request) which provides the original request context.
// Takes actions ([]daemon_dto.BatchActionItem) which contains the actions to execute.
//
// Returns []daemon_dto.BatchActionResult which contains the result for each action in the
// same order as the input.
// Returns bool which indicates whether all actions succeeded.
func (h *ActionHandler) executeBatchActions(ctx context.Context, request *http.Request, actions []daemon_dto.BatchActionItem) ([]daemon_dto.BatchActionResult, bool) {
	results := make([]daemon_dto.BatchActionResult, len(actions))
	allSuccess := true
	for i, item := range actions {
		results[i] = h.executeSingleAction(ctx, request, item)
		if results[i].Status >= httpErrorStatusThreshold {
			allSuccess = false
		}
	}
	return results, allSuccess
}

// executeSingleAction executes a single action within a batch request.
//
// Takes request (*http.Request) which provides the original HTTP request context.
// Takes item (daemon_dto.BatchActionItem) which specifies the action to run.
//
// Returns daemon_dto.BatchActionResult which contains the action outcome.
func (h *ActionHandler) executeSingleAction(
	ctx context.Context,
	request *http.Request,
	item daemon_dto.BatchActionItem,
) daemon_dto.BatchActionResult {
	ctx, l := logger_domain.From(ctx, log)

	entry, ok := h.registry[item.Name]
	if !ok {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusNotFound,
			Error:  fmt.Sprintf("action %q not found", item.Name),
			Code:   "NOT_FOUND",
		}
	}

	action := entry.Create()

	h.injectMetadata(request, action)

	if !h.checkBatchActionRateLimit(ctx, request, action, entry) {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusTooManyRequests,
			Error:  "Rate limit exceeded",
			Code:   "RATE_LIMITED",
		}
	}

	arguments := item.Args
	if arguments == nil {
		arguments = make(map[string]any)
	}

	if captchaErr := h.validateCaptcha(ctx, request, action, arguments, item.Name); captchaErr != nil {
		return buildBatchCaptchaResult(l, item.Name, captchaErr)
	}

	if spamErr := h.validateSpamDetect(ctx, request, action, arguments, item.Name); spamErr != nil {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusForbidden,
			Error:  "Submission flagged by spam filter",
			Code:   "SPAM_DETECTED",
		}
	}

	result, err := entry.Invoke(ctx, action, arguments)
	if err != nil {
		return h.buildBatchErrorResult(item.Name, err, isDevelopmentModeFromContext(request.Context()))
	}

	return daemon_dto.BatchActionResult{
		Name:   item.Name,
		Status: http.StatusOK,
		Data:   result,
	}
}

// buildBatchCaptchaResult converts a captcha validation error into the matching batch
// result, mirroring the single-action 429/403 split. It also emits the structured warning
// that the single-action path emits.
//
// Takes l (logger_domain.Logger) which receives the structured warning.
// Takes name (string) which identifies the action that failed validation.
// Takes captchaErr (error) which carries the underlying captcha failure.
//
// Returns daemon_dto.BatchActionResult populated with the right status code and code
// string (RATE_LIMITED for ErrRateLimited, otherwise CAPTCHA_FAILED).
func buildBatchCaptchaResult(l logger_domain.Logger, name string, captchaErr error) daemon_dto.BatchActionResult {
	l.Warn("Captcha validation failed",
		logger_domain.String(attributeKeyAction, name),
		logger_domain.Error(captchaErr),
	)
	if errors.Is(captchaErr, captcha_dto.ErrRateLimited) {
		return daemon_dto.BatchActionResult{
			Name:   name,
			Status: http.StatusTooManyRequests,
			Error:  "Too many captcha attempts",
			Code:   "RATE_LIMITED",
		}
	}
	return daemon_dto.BatchActionResult{
		Name:   name,
		Status: http.StatusForbidden,
		Error:  "Captcha validation failed",
		Code:   "CAPTCHA_FAILED",
	}
}

// isRequestFault reports whether err describes the request rather than the server.
//
// A payload rejected by its `validate:"..."` tags, or one the binder could not convert,
// is the input layer doing its job. Reporting it as a server error would attach a stack
// trace and mark the span failed, which lets anyone drive the error rate and the log
// volume by posting malformed forms. The generated wrapper has already logged the failure
// at its source, with the parameter that caused it, so it is not reported again here
// either.
//
// Takes err (error) which is the failure returned by an action invocation.
//
// Returns bool which is true when the request, not the server, is at fault.
func isRequestFault(err error) bool {
	if errors.Is(err, binder.ErrValidationFailed) {
		return true
	}

	var bindErrors binder.MultiError
	return errors.As(err, &bindErrors)
}

// fieldErrorsFor extracts per-field validation messages from err, whichever layer
// produced them, so a failure raised by an action and one raised by the binder answer
// with the same response body.
//
// The error itself is never replaced. Both sources already carry their own status and
// safe message, and rewriting the error would override a status the action chose
// deliberately and discard the context a developer needs.
//
// Takes err (error) which is the error returned by the action or its bind step.
//
// Returns map[string]string keyed by field name, or nil when err carries no field detail.
func fieldErrorsFor(err error) map[string]string {
	if validationErr, ok := errors.AsType[*daemon_dto.ValidationError](err); ok && len(validationErr.Fields) > 0 {
		return validationErr.Fields
	}

	bindErr, ok := errors.AsType[*binder.ValidationFailedError](err)
	if !ok || len(bindErr.Fields) == 0 {
		return nil
	}

	fields := make(map[string]string, len(bindErr.Fields))
	for name, messages := range bindErr.Fields {
		fields[name] = strings.Join(messages, ", ")
	}
	return fields
}

// buildBatchErrorResult creates a BatchActionResult from an error.
//
// Takes name (string) which identifies the action that failed.
// Takes err (error) which is the error to convert.
// Takes developmentMode (bool) which controls whether internal error details are exposed
// in the response message.
//
// Returns daemon_dto.BatchActionResult which contains the error details with appropriate
// status code and error code extracted from ActionError, or a generic internal error if
// the error is not an ActionError.
func (*ActionHandler) buildBatchErrorResult(name string, err error, developmentMode bool) daemon_dto.BatchActionResult {
	if actionErr, ok := errors.AsType[daemon_dto.ActionError](err); ok {
		return daemon_dto.BatchActionResult{
			Name:   name,
			Status: actionErr.StatusCode(),
			Error:  safeerror.ExtractSafeMessage(err, developmentMode),
			Code:   actionErr.ErrorCode(),
			Errors: fieldErrorsFor(err),
		}
	}

	return daemon_dto.BatchActionResult{
		Name:   name,
		Status: http.StatusInternalServerError,
		Error:  safeerror.ExtractSafeMessage(err, developmentMode),
		Code:   "INTERNAL_ERROR",
	}
}

// writeJSON writes a JSON response.
//
// Takes w (http.ResponseWriter) which receives the JSON output.
// Takes status (int) which sets the HTTP status code.
// Takes data (any) which is the value to encode as JSON, or nil to skip.
func (*ActionHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.ConfigDefault.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response. When err is non-nil the underlying error message
// is exposed under "details" only when developmentMode is true; in production the
// user-safe message extracted via safeerror.ExtractSafeMessage is surfaced instead so
// internal details never reach the client.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes status (int) which specifies the HTTP status code.
// Takes message (string) which provides the error message for clients.
// Takes err (error) which contains the underlying error details, or nil.
// Takes developmentMode (bool) which controls whether internal error details are exposed
// under the "details" key.
func (h *ActionHandler) writeError(w http.ResponseWriter, status int, message string, err error, developmentMode bool) {
	response := map[string]any{"error": message}
	if err != nil {
		response["details"] = safeerror.ExtractSafeMessage(err, developmentMode)
	}
	h.writeJSON(w, status, response)
}

// handleActionError processes an action error and writes the appropriate response.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes request (*http.Request) which provides the request context for development mode
// detection.
// Takes action (any) which provides response metadata and optional helpers.
// Takes err (error) which is the error to process.
//
// It discriminates between structured ActionError types and generic errors. If the action
// set helpers before returning an error, they are included in the response.
func (h *ActionHandler) handleActionError(w http.ResponseWriter, request *http.Request, action any, err error) {
	h.applyResponseMetadata(w, action)

	var helpers []daemon_dto.HelperCall
	if getter, ok := action.(interface {
		Response() *daemon_dto.ResponseWriter
	}); ok {
		if response := getter.Response(); response != nil {
			helpers = response.GetHelpers()
		}
	}

	developmentMode := isDevelopmentModeFromContext(request.Context())

	if actionErr, ok := errors.AsType[daemon_dto.ActionError](err); ok {
		response := map[string]any{
			"status":  actionErr.StatusCode(),
			"code":    actionErr.ErrorCode(),
			"message": safeerror.ExtractSafeMessage(err, developmentMode),
		}

		if fields := fieldErrorsFor(err); fields != nil {
			response["errors"] = fields
		}

		if len(helpers) > 0 {
			response["_helpers"] = helpers
		}

		h.writeJSON(w, actionErr.StatusCode(), response)
		return
	}

	message := safeerror.ExtractSafeMessage(err, developmentMode)

	response := map[string]any{
		"status":  http.StatusInternalServerError,
		"code":    "INTERNAL_ERROR",
		"message": message,
	}
	if len(helpers) > 0 {
		response["_helpers"] = helpers
	}
	h.writeJSON(w, http.StatusInternalServerError, response)
}

// buildCacheKey constructs a cache key for a cacheable action response.
//
// Takes request (*http.Request) which provides headers for VaryHeaders.
// Takes arguments (map[string]any) which holds the parsed request arguments.
// Takes actionName (string) which identifies the action.
// Takes cc (*daemon_domain.CacheConfig) which provides the cache configuration.
//
// Returns string which is the computed cache key.
func (*ActionHandler) buildCacheKey(
	request *http.Request,
	arguments map[string]any,
	actionName string,
	cc *daemon_domain.CacheConfig,
) string {
	argsJSON, _ := json.Marshal(arguments)

	var b strings.Builder
	b.WriteString(actionName)
	b.WriteByte(':')
	b.Write(argsJSON)

	for _, header := range cc.VaryHeaders {
		b.WriteByte(':')
		b.WriteString(header)
		b.WriteByte('=')
		b.WriteString(request.Header.Get(header))
	}
	return b.String()
}

// recordSlowAction logs a warning and increments the slow action metric when the action
// execution exceeds the configured slow threshold.
//
// Takes actionName (string) which identifies the action.
// Takes startTime (time.Time) which marks when execution began.
// Takes threshold (time.Duration) which is the slow threshold; zero disables.
func (*ActionHandler) recordSlowAction(
	ctx context.Context,
	actionName string,
	startTime time.Time,
	threshold time.Duration,
) {
	ctx, l := logger_domain.From(ctx, log)

	if threshold <= 0 {
		return
	}

	elapsed := time.Since(startTime)
	if elapsed <= threshold {
		return
	}

	actionSlowCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, actionName)))
	l.Warn("Slow action execution detected",
		logger_domain.String(attributeKeyAction, actionName),
		logger_domain.String("elapsed", elapsed.String()),
		logger_domain.String("threshold", threshold.String()),
	)
}
