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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// AttrKeyProvider is the logging attribute key for the provider name.
	AttrKeyProvider = "provider"

	// AttrKeyKey is the logging attribute key for cache keys.
	AttrKeyKey = "key"

	// attributeKeyModel is the logging attribute key for the model name.
	attributeKeyModel = "model"

	// streamEventBufferSize is the channel buffer size for stream event wrapping.
	streamEventBufferSize = 16
)

var (
	// log is the package-level logger for the llm_domain package.
	log = logger_domain.GetLogger("piko/internal/llm/llm_domain")

	// meter is the OpenTelemetry meter for the LLM domain package.
	meter = otel.Meter("piko/internal/llm/llm_domain")

	// completionCount tracks the number of completion requests.
	completionCount metric.Int64Counter

	// completionDuration records the duration of completion requests.
	completionDuration metric.Float64Histogram

	// completionErrorCount tracks completion request failures.
	completionErrorCount metric.Int64Counter

	// streamCount tracks the number of streaming requests.
	streamCount metric.Int64Counter

	// streamDuration records the duration of streaming requests.
	streamDuration metric.Float64Histogram

	// streamErrorCount tracks streaming request failures.
	streamErrorCount metric.Int64Counter

	// promptTokenCount tracks prompt tokens used across requests.
	promptTokenCount metric.Int64Counter

	// completionTokenCount tracks completion tokens generated across requests.
	completionTokenCount metric.Int64Counter

	// totalTokenCount tracks total tokens used across requests.
	totalTokenCount metric.Int64Counter

	// builderCompleteCount tracks CompletionBuilder.Complete invocations.
	builderCompleteCount metric.Int64Counter

	// builderCompleteDuration records CompletionBuilder.Complete duration.
	builderCompleteDuration metric.Float64Histogram

	// builderCompleteErrorCount tracks CompletionBuilder.Complete errors.
	builderCompleteErrorCount metric.Int64Counter

	// requestCostHistogram records the cost of individual
	// requests in USD.
	requestCostHistogram metric.Float64Histogram

	// totalSpendCounter tracks cumulative spend across all requests in USD.
	totalSpendCounter metric.Float64Counter

	// rateLimitedCount tracks requests rejected due to
	// rate limiting.
	rateLimitedCount metric.Int64Counter

	// budgetExceededCount tracks requests that were turned down due to budget
	// limits.
	budgetExceededCount metric.Int64Counter

	// retryAttemptCount tracks the number of retry attempts.
	retryAttemptCount metric.Int64Counter

	// retrySuccessCount tracks successful retries. These are requests that
	// succeeded after an initial failure.
	retrySuccessCount metric.Int64Counter

	// retryExhaustedCount tracks requests that exhausted all retry attempts.
	retryExhaustedCount metric.Int64Counter

	// fallbackAttemptCount tracks the number of provider
	// attempts in fallback chains.
	fallbackAttemptCount metric.Int64Counter

	// fallbackSuccessCount tracks successful fallbacks (requests that succeeded
	// after primary failure).
	fallbackSuccessCount metric.Int64Counter

	// fallbackExhaustedCount tracks requests that exhausted all fallback
	// providers.
	fallbackExhaustedCount metric.Int64Counter

	// cacheHitCount tracks the number of cache hits.
	cacheHitCount metric.Int64Counter

	// cacheMissCount records how many times a cache lookup fails to find an entry.
	cacheMissCount metric.Int64Counter

	// toolLoopRoundsCount tracks the number of completed
	// tool dispatch rounds.
	toolLoopRoundsCount metric.Int64Counter

	// toolLoopDispatchesCount tracks individual tool handler dispatches.
	toolLoopDispatchesCount metric.Int64Counter

	// toolLoopErrorsCount tracks tool dispatch errors (unregistered tools or
	// handler errors).
	toolLoopErrorsCount metric.Int64Counter

	// toolLoopMaxRoundsCount tracks when the tool loop terminates due to
	// reaching the maximum number of rounds.
	toolLoopMaxRoundsCount metric.Int64Counter

	// embeddingCount tracks the number of embedding requests.
	embeddingCount metric.Int64Counter

	// embeddingDuration records the duration of embedding requests.
	embeddingDuration metric.Float64Histogram

	// embeddingErrorCount tracks embedding request failures.
	embeddingErrorCount metric.Int64Counter

	// embeddingTokenCount tracks tokens used for embeddings.
	embeddingTokenCount metric.Int64Counter
)

func init() {
	var err error

	completionCount, err = meter.Int64Counter(
		"piko.llm.completion.count",
		metric.WithDescription("Number of LLM completion requests."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completionDuration, err = meter.Float64Histogram(
		"piko.llm.completion.duration",
		metric.WithDescription("Duration of LLM completion requests."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completionErrorCount, err = meter.Int64Counter(
		"piko.llm.completion.errors",
		metric.WithDescription("Number of LLM completion errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamCount, err = meter.Int64Counter(
		"piko.llm.stream.count",
		metric.WithDescription("Number of LLM streaming requests."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamDuration, err = meter.Float64Histogram(
		"piko.llm.stream.duration",
		metric.WithDescription("Duration of LLM streaming requests."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	streamErrorCount, err = meter.Int64Counter(
		"piko.llm.stream.errors",
		metric.WithDescription("Number of LLM streaming errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	promptTokenCount, err = meter.Int64Counter(
		"piko.llm.tokens.prompt",
		metric.WithDescription("Total prompt tokens used."),
		metric.WithUnit("{tok}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	completionTokenCount, err = meter.Int64Counter(
		"piko.llm.tokens.completion",
		metric.WithDescription("Total completion tokens generated."),
		metric.WithUnit("{tok}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	totalTokenCount, err = meter.Int64Counter(
		"piko.llm.tokens.total",
		metric.WithDescription("Total tokens used."),
		metric.WithUnit("{tok}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderCompleteCount, err = meter.Int64Counter(
		"piko.llm.builder.complete.count",
		metric.WithDescription("Number of CompletionBuilder.Complete invocations."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderCompleteDuration, err = meter.Float64Histogram(
		"piko.llm.builder.complete.duration",
		metric.WithDescription("Duration of CompletionBuilder.Complete operations."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	builderCompleteErrorCount, err = meter.Int64Counter(
		"piko.llm.builder.complete.errors",
		metric.WithDescription("Number of CompletionBuilder.Complete errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	requestCostHistogram, err = meter.Float64Histogram(
		"piko.llm.request.cost",
		metric.WithDescription("Cost of LLM requests in USD."),
		metric.WithUnit("USD"),
	)
	if err != nil {
		otel.Handle(err)
	}

	totalSpendCounter, err = meter.Float64Counter(
		"piko.llm.spend.total",
		metric.WithDescription("Cumulative LLM spend in USD."),
		metric.WithUnit("USD"),
	)
	if err != nil {
		otel.Handle(err)
	}

	rateLimitedCount, err = meter.Int64Counter(
		"piko.llm.rate_limited.count",
		metric.WithDescription("Number of requests rejected due to rate limiting."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	budgetExceededCount, err = meter.Int64Counter(
		"piko.llm.budget_exceeded.count",
		metric.WithDescription("Number of requests rejected due to budget limits."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryAttemptCount, err = meter.Int64Counter(
		"piko.llm.retry.attempts",
		metric.WithDescription("Number of retry attempts for LLM requests."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retrySuccessCount, err = meter.Int64Counter(
		"piko.llm.retry.success",
		metric.WithDescription("Number of successful retries (requests that succeeded after initial failure)."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	retryExhaustedCount, err = meter.Int64Counter(
		"piko.llm.retry.exhausted",
		metric.WithDescription("Number of requests that exhausted all retry attempts."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fallbackAttemptCount, err = meter.Int64Counter(
		"piko.llm.fallback.attempts",
		metric.WithDescription("Number of provider attempts in fallback chains."),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fallbackSuccessCount, err = meter.Int64Counter(
		"piko.llm.fallback.success",
		metric.WithDescription("Number of successful fallbacks (requests that succeeded after primary failure)."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	fallbackExhaustedCount, err = meter.Int64Counter(
		"piko.llm.fallback.exhausted",
		metric.WithDescription("Number of requests that exhausted all fallback providers."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheHitCount, err = meter.Int64Counter(
		"piko.llm.cache.hits",
		metric.WithDescription("Number of cache hits."),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	cacheMissCount, err = meter.Int64Counter(
		"piko.llm.cache.misses",
		metric.WithDescription("Number of cache misses."),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	toolLoopRoundsCount, err = meter.Int64Counter(
		"piko.llm.tool_loop.rounds",
		metric.WithDescription("Number of completed tool dispatch rounds."),
		metric.WithUnit("{round}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	toolLoopDispatchesCount, err = meter.Int64Counter(
		"piko.llm.tool_loop.dispatches",
		metric.WithDescription("Number of individual tool handler dispatches."),
		metric.WithUnit("{dispatch}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	toolLoopErrorsCount, err = meter.Int64Counter(
		"piko.llm.tool_loop.errors",
		metric.WithDescription("Number of tool dispatch errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	toolLoopMaxRoundsCount, err = meter.Int64Counter(
		"piko.llm.tool_loop.max_rounds",
		metric.WithDescription("Number of times the tool loop terminated due to reaching max rounds."),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embeddingCount, err = meter.Int64Counter(
		"piko.llm.embedding.count",
		metric.WithDescription("Number of embedding requests."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embeddingDuration, err = meter.Float64Histogram(
		"piko.llm.embedding.duration",
		metric.WithDescription("Duration of embedding requests."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embeddingErrorCount, err = meter.Int64Counter(
		"piko.llm.embedding.errors",
		metric.WithDescription("Number of embedding errors."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		otel.Handle(err)
	}

	embeddingTokenCount, err = meter.Int64Counter(
		"piko.llm.embedding.tokens",
		metric.WithDescription("Total tokens used for embeddings."),
		metric.WithUnit("{tok}"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
