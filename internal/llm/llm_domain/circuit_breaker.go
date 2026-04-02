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
	"time"

	"github.com/sony/gobreaker/v2"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// circuitBreakerBucketPeriod is the duration of each measurement bucket
// for tracking failure counts.
const circuitBreakerBucketPeriod = 10 * time.Second

// circuitBreakerProvider wraps an LLMProviderPort with a Sony circuit breaker
// so that repeated failures cause the circuit to open and fast-fail requests
// rather than sending them to a failing provider.
type circuitBreakerProvider struct {
	// inner is the wrapped LLM provider that requests are forwarded to.
	inner LLMProviderPort

	// completeBreaker guards non-streaming completion requests.
	completeBreaker *gobreaker.CircuitBreaker[*llm_dto.CompletionResponse]

	// streamBreaker guards streaming completion requests.
	streamBreaker *gobreaker.CircuitBreaker[<-chan llm_dto.StreamEvent]
}

// Complete sends a completion request through the circuit breaker.
//
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion parameters.
//
// Returns *llm_dto.CompletionResponse which is the provider response.
// Returns error when the circuit is open or the request fails.
func (p *circuitBreakerProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	response, err := p.completeBreaker.Execute(func() (*llm_dto.CompletionResponse, error) {
		return p.inner.Complete(ctx, request)
	})
	if err != nil {
		return nil, mapCircuitBreakerError(err)
	}
	return response, nil
}

// Stream sends a streaming request through the circuit breaker.
//
// Takes request (*llm_dto.CompletionRequest) which contains the
// completion parameters.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events.
// Returns error when the circuit is open or the request fails.
func (p *circuitBreakerProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	eventChannel, err := p.streamBreaker.Execute(func() (<-chan llm_dto.StreamEvent, error) {
		return p.inner.Stream(ctx, request)
	})
	if err != nil {
		return nil, mapCircuitBreakerError(err)
	}
	return eventChannel, nil
}

// SupportsStreaming delegates to the inner provider.
//
// Returns bool which is true if the inner provider supports
// streaming.
func (p *circuitBreakerProvider) SupportsStreaming() bool {
	return p.inner.SupportsStreaming()
}

// SupportsStructuredOutput delegates to the inner provider.
//
// Returns bool which is true if structured output is supported.
func (p *circuitBreakerProvider) SupportsStructuredOutput() bool {
	return p.inner.SupportsStructuredOutput()
}

// SupportsTools delegates to the inner provider.
//
// Returns bool which is true if tool calling is supported.
func (p *circuitBreakerProvider) SupportsTools() bool {
	return p.inner.SupportsTools()
}

// SupportsPenalties delegates to the inner provider.
//
// Returns bool which is true if penalties are supported.
func (p *circuitBreakerProvider) SupportsPenalties() bool { return p.inner.SupportsPenalties() }

// SupportsSeed delegates to the inner provider.
//
// Returns bool which is true if seed is supported.
func (p *circuitBreakerProvider) SupportsSeed() bool { return p.inner.SupportsSeed() }

// SupportsParallelToolCalls delegates to the inner provider.
//
// Returns bool which is true if parallel tool calls are supported.
func (p *circuitBreakerProvider) SupportsParallelToolCalls() bool {
	return p.inner.SupportsParallelToolCalls()
}

// SupportsMessageName delegates to the inner provider.
//
// Returns bool which is true if message names are supported.
func (p *circuitBreakerProvider) SupportsMessageName() bool { return p.inner.SupportsMessageName() }

// ListModels delegates to the inner provider.
//
// Returns []ModelInfo which lists the available models.
// Returns error when the inner provider fails.
func (p *circuitBreakerProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	return p.inner.ListModels(ctx)
}

// Close delegates to the inner provider.
//
// Returns error when the inner provider fails to close.
func (p *circuitBreakerProvider) Close(ctx context.Context) error {
	return p.inner.Close(ctx)
}

// DefaultModel delegates to the inner provider.
//
// Returns string which is the default model name.
func (p *circuitBreakerProvider) DefaultModel() string {
	return p.inner.DefaultModel()
}

// newCircuitBreakerProvider wraps a provider with circuit breakers for both
// Complete and Stream operations.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation in circuit breaker callbacks.
// Takes name (string) which identifies the provider in log messages.
// Takes inner (LLMProviderPort) which is the provider to wrap.
// Takes maxFailures (int) which sets the consecutive failure threshold.
// Takes timeout (time.Duration) which specifies how long the circuit stays
// open before attempting recovery.
//
// Returns LLMProviderPort which is the wrapped provider.
func newCircuitBreakerProvider(ctx context.Context, name string, inner LLMProviderPort, maxFailures int, timeout time.Duration) LLMProviderPort {
	makeSettings := func(opName string) gobreaker.Settings {
		return gobreaker.Settings{
			Name:         "llm-" + name + "-" + opName,
			MaxRequests:  1,
			Interval:     0,
			Timeout:      timeout,
			BucketPeriod: circuitBreakerBucketPeriod,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= safeconv.IntToUint32(maxFailures)
			},
			OnStateChange: func(cbName string, from gobreaker.State, to gobreaker.State) {
				_, l := logger_domain.From(ctx, log)
				l.Warn("LLM provider circuit breaker state changed",
					logger_domain.String("breaker", cbName),
					logger_domain.String("from_state", from.String()),
					logger_domain.String("to_state", to.String()),
				)
			},
			IsExcluded: isCircuitBreakerExcluded,
		}
	}

	return &circuitBreakerProvider{
		inner:           inner,
		completeBreaker: gobreaker.NewCircuitBreaker[*llm_dto.CompletionResponse](makeSettings("complete")),
		streamBreaker:   gobreaker.NewCircuitBreaker[<-chan llm_dto.StreamEvent](makeSettings("stream")),
	}
}

// isCircuitBreakerExcluded returns true for errors that should not
// count as circuit breaker failures (client-side errors, not provider
// faults).
//
// Takes err (error) which is the error to evaluate.
//
// Returns bool which is true if the error should be excluded from
// failure counting.
func isCircuitBreakerExcluded(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrBudgetExceeded) ||
		errors.Is(err, ErrMaxCostExceeded) ||
		errors.Is(err, ErrRateLimited)
}

// mapCircuitBreakerError maps gobreaker-specific errors to LLM
// domain errors that are retryable and trigger fallback.
//
// Takes err (error) which is the gobreaker error to map.
//
// Returns error which is the mapped domain error, or the original
// error if no mapping applies.
func mapCircuitBreakerError(err error) error {
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrProviderOverloaded
	}
	return err
}
