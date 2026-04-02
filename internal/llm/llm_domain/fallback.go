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

package llm_domain

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

// FallbackRouter runs LLM requests with fallback to other providers.
type FallbackRouter struct {
	// service provides access to the LLM completion functionality.
	service *service
}

// NewFallbackRouter creates a new FallbackRouter for the given service.
//
// Takes service (*service) which provides access to LLM providers.
//
// Returns *FallbackRouter which is ready to run requests with fallback.
func NewFallbackRouter(service *service) *FallbackRouter {
	return &FallbackRouter{
		service: service,
	}
}

// Execute runs a completion request with fallback across providers.
// It tries each provider in the configured order until one succeeds.
//
// Takes config (*llm_dto.FallbackConfig) which configures fallback behaviour.
// Takes request (*llm_dto.CompletionRequest) which is the request to execute.
// Takes budgetScope (string) which is the budget scope for cost tracking.
// Takes maxCost (maths.Money) which is the per-request cost limit.
// Takes retryPolicy (*llm_dto.RetryPolicy) which is the retry policy to use
// (may be nil).
//
// Returns *llm_dto.CompletionResponse from the successful provider.
// Returns *llm_dto.FallbackResult with details about fallback execution.
// Returns error if all providers fail.
func (r *FallbackRouter) Execute(
	ctx context.Context,
	config *llm_dto.FallbackConfig,
	request *llm_dto.CompletionRequest,
	budgetScope string,
	maxCost maths.Money,
	retryPolicy *llm_dto.RetryPolicy,
) (*llm_dto.CompletionResponse, *llm_dto.FallbackResult, error) {
	if config == nil || len(config.Providers) == 0 {
		return nil, nil, errors.New("fallback config must have at least one provider")
	}

	result := &llm_dto.FallbackResult{
		UsedProvider:       "",
		AttemptedProviders: make([]string, 0, len(config.Providers)),
		Errors:             make(map[string]error),
	}

	triggers := config.Triggers
	if triggers == 0 {
		triggers = llm_dto.FallbackOnAll
	}

	params := &fallbackParams{
		config:      config,
		request:     request,
		budgetScope: budgetScope,
		maxCost:     maxCost,
		retryPolicy: retryPolicy,
		triggers:    triggers,
	}

	for _, providerName := range config.Providers {
		if ctx.Err() != nil {
			return nil, result, ctx.Err()
		}

		response, shouldContinue, err := r.attemptProvider(ctx, params, providerName, result)
		if !shouldContinue {
			return response, result, err
		}
	}

	fallbackExhaustedCount.Add(ctx, 1)

	errs := make([]error, 0, len(result.Errors)+1)
	errs = append(errs, fmt.Errorf("all %d fallback providers failed", len(config.Providers)))
	for provider, provErr := range result.Errors {
		errs = append(errs, fmt.Errorf("provider %s: %w", provider, provErr))
	}
	return nil, result, errors.Join(errs...)
}

// fallbackParams holds the parameters for fallback execution to reduce function
// arguments.
type fallbackParams struct {
	// config holds the fallback routing configuration.
	config *llm_dto.FallbackConfig

	// request holds the original completion request to send to providers.
	request *llm_dto.CompletionRequest

	// retryPolicy specifies the retry behaviour; nil means no retries.
	retryPolicy *llm_dto.RetryPolicy

	// budgetScope identifies the budget scope for cost tracking during completion.
	budgetScope string

	// maxCost is the maximum budget allowed for this request.
	maxCost maths.Money

	// triggers specifies which error conditions should cause a fallback.
	triggers llm_dto.FallbackTrigger
}

// attemptProvider tries a single provider in the fallback chain.
//
// Takes ctx (context.Context) which carries request-scoped values, deadlines,
// and cancellation signals.
// Takes params (*fallbackParams) which contains the request config.
// Takes providerName (string) which identifies the provider to attempt.
// Takes result (*llm_dto.FallbackResult) which accumulates attempt results.
//
// Returns *llm_dto.CompletionResponse which is the successful response if any.
// Returns bool which indicates whether to continue to the next provider.
// Returns error when the provider fails and should not continue.
func (r *FallbackRouter) attemptProvider(
	ctx context.Context,
	params *fallbackParams,
	providerName string,
	result *llm_dto.FallbackResult,
) (*llm_dto.CompletionResponse, bool, error) {
	ctx, l := logger_domain.From(ctx, log)

	result.AttemptedProviders = append(result.AttemptedProviders, providerName)

	reqCopy := deepCopyRequest(params.request)
	reqCopy.Model = params.config.GetModel(providerName, params.request.Model)

	fallbackAttemptCount.Add(ctx, 1)

	l.Debug("Attempting provider in fallback chain",
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.String("model", reqCopy.Model),
		logger_domain.Int("attempt", len(result.AttemptedProviders)),
	)

	response, err := r.executeWithRetry(ctx, params, providerName, &reqCopy)
	if err == nil {
		return r.handleProviderSuccess(ctx, providerName, result, response)
	}

	result.Errors[providerName] = err
	return r.handleProviderFailure(ctx, params, providerName, err, result)
}

// executeWithRetry executes a provider request with optional retry.
//
// Takes ctx (context.Context) which carries request-scoped values, deadlines,
// and cancellation signals.
// Takes params (*fallbackParams) which provides the retry policy and config.
// Takes providerName (string) which identifies the provider to use.
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion request.
//
// Returns *llm_dto.CompletionResponse which contains the completion result.
// Returns error when the request fails after all retry attempts.
func (r *FallbackRouter) executeWithRetry(
	ctx context.Context,
	params *fallbackParams,
	providerName string,
	request *llm_dto.CompletionRequest,
) (*llm_dto.CompletionResponse, error) {
	if params.retryPolicy != nil && params.retryPolicy.MaxRetries > 0 {
		executor := NewRetryExecutor(params.retryPolicy, WithRetryExecutorClock(r.service.clock))
		return ExecuteWithResult(ctx, executor, func() (*llm_dto.CompletionResponse, error) {
			return r.service.completeWithScope(ctx, providerName, request, params.budgetScope, params.maxCost)
		})
	}
	return r.service.completeWithScope(ctx, providerName, request, params.budgetScope, params.maxCost)
}

// handleProviderSuccess records success metrics and returns the successful
// response.
//
// Takes providerName (string) which identifies the provider that succeeded.
// Takes result (*llm_dto.FallbackResult) which tracks the fallback attempt.
// Takes response (*llm_dto.CompletionResponse) which contains the provider
// response.
//
// Returns *llm_dto.CompletionResponse which is the successful response.
// Returns bool which indicates whether to continue the fallback chain.
// Returns error when processing fails.
func (*FallbackRouter) handleProviderSuccess(
	ctx context.Context,
	providerName string,
	result *llm_dto.FallbackResult,
	response *llm_dto.CompletionResponse,
) (*llm_dto.CompletionResponse, bool, error) {
	ctx, l := logger_domain.From(ctx, log)

	result.UsedProvider = providerName
	if len(result.AttemptedProviders) > 1 {
		fallbackSuccessCount.Add(ctx, 1)
	}
	l.Debug("Provider succeeded in fallback chain",
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.Int("attempts", len(result.AttemptedProviders)),
	)
	return response, false, nil
}

// handleProviderFailure determines whether to continue to the next provider
// or return the error.
//
// Takes ctx (context.Context) which carries request-scoped values for logging.
// Takes params (*fallbackParams) which contains the fallback triggers.
// Takes providerName (string) which identifies the provider that failed.
// Takes err (error) which is the error from the failed provider.
//
// Returns *llm_dto.CompletionResponse which is always nil on failure.
// Returns bool which indicates whether to continue to the next provider.
// Returns error when the error is not eligible for fallback.
func (r *FallbackRouter) handleProviderFailure(
	ctx context.Context,
	params *fallbackParams,
	providerName string,
	err error,
	_ *llm_dto.FallbackResult,
) (*llm_dto.CompletionResponse, bool, error) {
	_, l := logger_domain.From(ctx, log)

	if !r.shouldFallback(err, params.triggers) {
		l.Debug("Error not eligible for fallback",
			logger_domain.String(AttrKeyProvider, providerName),
			logger_domain.Error(err),
		)
		return nil, false, err
	}

	l.Debug("Provider failed, trying next in fallback chain",
		logger_domain.String(AttrKeyProvider, providerName),
		logger_domain.Error(err),
	)
	return nil, true, nil
}

// shouldFallback checks if an error should trigger fallback to the next
// provider.
//
// Takes err (error) which is the error to check.
// Takes triggers (llm_dto.FallbackTrigger) which defines the fallback
// conditions.
//
// Returns bool which is true if fallback should be attempted.
func (*FallbackRouter) shouldFallback(err error, triggers llm_dto.FallbackTrigger) bool {
	if triggers.HasTrigger(llm_dto.FallbackOnError) {
		return true
	}

	if triggers.HasTrigger(llm_dto.FallbackOnRateLimit) {
		if errors.Is(err, ErrRateLimited) {
			return true
		}
	}

	if triggers.HasTrigger(llm_dto.FallbackOnTimeout) {
		if errors.Is(err, ErrProviderTimeout) {
			return true
		}
	}

	if triggers.HasTrigger(llm_dto.FallbackOnBudgetExceeded) {
		if errors.Is(err, ErrBudgetExceeded) || errors.Is(err, ErrMaxCostExceeded) {
			return true
		}
	}

	return false
}

// deepCopyRequest returns a shallow copy of request with independent
// copies of all slice and map fields so that mutations during one
// fallback attempt cannot affect subsequent attempts.
//
// Takes request (*llm_dto.CompletionRequest) which is the request to
// copy.
//
// Returns llm_dto.CompletionRequest which is the independent copy.
func deepCopyRequest(request *llm_dto.CompletionRequest) llm_dto.CompletionRequest {
	cp := *request

	if len(request.Messages) > 0 {
		cp.Messages = make([]llm_dto.Message, len(request.Messages))
		copy(cp.Messages, request.Messages)
	}

	if len(request.Tools) > 0 {
		cp.Tools = make([]llm_dto.ToolDefinition, len(request.Tools))
		for i, tool := range request.Tools {
			cp.Tools[i] = tool.DeepCopy()
		}
	}

	if len(request.Stop) > 0 {
		cp.Stop = make([]string, len(request.Stop))
		copy(cp.Stop, request.Stop)
	}

	if len(request.ProviderOptions) > 0 {
		cp.ProviderOptions = make(map[string]any, len(request.ProviderOptions))
		maps.Copy(cp.ProviderOptions, request.ProviderOptions)
	}

	if len(request.Metadata) > 0 {
		cp.Metadata = make(map[string]string, len(request.Metadata))
		maps.Copy(cp.Metadata, request.Metadata)
	}

	return cp
}
