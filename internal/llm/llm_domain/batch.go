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
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
)

const (
	// DefaultPollingInterval is the default interval for polling batch status.
	DefaultPollingInterval = 30 * time.Second
)

// BatchProviderPort defines the interface that batch provider adapters must
// implement. It is a driven port in the hexagonal architecture pattern.
type BatchProviderPort interface {
	// CreateBatch submits a batch of completion requests for processing.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.BatchRequest) which contains the batch parameters.
	//
	// Returns *llm_dto.BatchResponse containing the initial batch status.
	// Returns error when the batch cannot be created.
	CreateBatch(ctx context.Context, request *llm_dto.BatchRequest) (*llm_dto.BatchResponse, error)

	// GetBatchStatus retrieves the current status of a batch job.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes batchID (string) which identifies the batch job.
	//
	// Returns *llm_dto.BatchResponse containing the current status.
	// Returns error when the status cannot be retrieved.
	GetBatchStatus(ctx context.Context, batchID string) (*llm_dto.BatchResponse, error)

	// GetBatchResults retrieves the completed results of a batch job.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes batchID (string) which identifies the batch job.
	//
	// Returns *llm_dto.BatchResponse containing the results.
	// Returns error when the results cannot be retrieved.
	GetBatchResults(ctx context.Context, batchID string) (*llm_dto.BatchResponse, error)

	// CancelBatch cancels a pending or processing batch job.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes batchID (string) which identifies the batch job.
	//
	// Returns error when the batch cannot be cancelled.
	CancelBatch(ctx context.Context, batchID string) error

	// SupportsBatch reports whether this provider supports batch processing.
	//
	// Returns bool which is true if batch processing is supported.
	SupportsBatch() bool

	// Close releases any resources held by the provider.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	//
	// Returns error when the provider cannot be closed cleanly.
	Close(ctx context.Context) error
}

// BatchBuilder provides a fluent builder for creating batch requests.
type BatchBuilder struct {
	// metadata stores key-value pairs to attach to the batch request.
	metadata map[string]string

	// service holds the LLM service that processes the batch.
	service *service

	// providerName is the name of the provider for this batch.
	providerName string

	// budgetScope limits usage tracking to a specific scope identifier.
	budgetScope string

	// requests holds the completion requests to include in the batch.
	requests []llm_dto.CompletionRequest

	// window is the time duration for batch completion; 0 means no window.
	window time.Duration
}

// NewBatchBuilder creates a new batch builder for collecting LLM requests.
//
// Takes service (*service) which is the LLM service that will process the batch.
//
// Returns *BatchBuilder which can be used for method chaining.
func NewBatchBuilder(service *service) *BatchBuilder {
	return &BatchBuilder{
		metadata:     nil,
		service:      service,
		providerName: "",
		budgetScope:  "",
		requests:     make([]llm_dto.CompletionRequest, 0),
		window:       0,
	}
}

// Add adds a completion request to the batch.
//
// Takes request (llm_dto.CompletionRequest) which is the request to add.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) Add(request llm_dto.CompletionRequest) *BatchBuilder {
	b.requests = append(b.requests, request)
	return b
}

// AddBuilder adds a completion request from a builder to the batch.
//
// Takes builder (*CompletionBuilder) which provides the request.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) AddBuilder(builder *CompletionBuilder) *BatchBuilder {
	b.requests = append(b.requests, builder.Build())
	return b
}

// Window sets the completion window for the batch.
// Longer windows typically result in lower costs.
//
// Takes d (time.Duration) which is the completion window.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) Window(d time.Duration) *BatchBuilder {
	b.window = d
	return b
}

// WithProvider sets which registered provider to use.
//
// Takes name (string) which identifies the provider.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) WithProvider(name string) *BatchBuilder {
	b.providerName = name
	return b
}

// WithBudgetScope sets the budget scope for tracking batch costs.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) WithBudgetScope(scope string) *BatchBuilder {
	b.budgetScope = scope
	return b
}

// WithMetadata adds metadata for tracking and logging.
//
// Takes key (string) which identifies the metadata.
// Takes value (string) which is the metadata value.
//
// Returns *BatchBuilder for method chaining.
func (b *BatchBuilder) WithMetadata(key, value string) *BatchBuilder {
	if b.metadata == nil {
		b.metadata = make(map[string]string)
	}
	b.metadata[key] = value
	return b
}

// Build returns the configured batch request without executing it.
//
// Returns llm_dto.BatchRequest which contains the configured parameters.
func (b *BatchBuilder) Build() llm_dto.BatchRequest {
	return llm_dto.BatchRequest{
		Requests:         b.requests,
		CompletionWindow: b.window,
		Metadata:         b.metadata,
	}
}

// BatchPoller checks the status of a batch job at regular intervals.
type BatchPoller struct {
	// service provides batch polling operations and timing.
	service *service

	// batchID is the identifier of the current batch being polled.
	batchID string

	// providerName identifies the data provider for this poller.
	providerName string

	// interval is the duration between poll attempts.
	interval time.Duration
}

// NewBatchPoller creates a new batch poller for checking batch job status.
//
// Takes service (*service) which provides the LLM service for API calls.
// Takes batchID (string) which identifies the batch job to monitor.
// Takes providerName (string) which specifies the LLM provider.
// Takes interval (time.Duration) which sets how often to check status.
//
// Returns *BatchPoller which is ready to poll for batch completion.
func NewBatchPoller(service *service, batchID, providerName string, interval time.Duration) *BatchPoller {
	if interval == 0 {
		interval = DefaultPollingInterval
	}
	return &BatchPoller{
		service:      service,
		batchID:      batchID,
		providerName: providerName,
		interval:     interval,
	}
}

// PollChannel returns a channel that is closed when the context
// is cancelled.
//
// Returns <-chan *llm_dto.BatchResponse which is closed on
// cancellation.
//
// Spawns a goroutine that polls at the configured interval until the context
// is cancelled.
func (p *BatchPoller) PollChannel(ctx context.Context) <-chan *llm_dto.BatchResponse {
	responseChannel := make(chan *llm_dto.BatchResponse)

	go func() {
		defer close(responseChannel)
		defer goroutine.RecoverPanic(ctx, "llm.pollChannel")
		ticker := p.service.clock.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
			}
		}
	}()

	return responseChannel
}
