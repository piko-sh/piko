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

// Package llm_domain orchestrates provider-agnostic interaction with
// large language models.
//
// It handles completions, streaming, embeddings, batch processing,
// cost tracking, budget enforcement, rate limiting, response caching,
// conversation memory, retry with exponential backoff, and
// multi-provider fallback routing. It owns the driving port
// ([Service]) and defines driven ports for providers, caches, memory
// stores, budget stores, and vector stores.
//
// # Usage
//
//	// Create a completion using the fluent builder
//	response, err := service.NewCompletion().
//	    Model("gpt-5").
//	    System("You are helpful.").
//	    User("Hello!").
//	    Do(ctx)
//
//	// Or stream responses
//	events, err := service.NewCompletion().
//	    Model("claude-sonnet-4-5-20250929").
//	    User("Write a poem.").
//	    Stream(ctx)
//
//	for event := range events {
//	    if event.Chunk != nil && event.Chunk.Delta.Content != nil {
//	        fmt.Print(*event.Chunk.Delta.Content)
//	    }
//	}
//
// # Context handling
//
// All terminal operations ([CompletionBuilder.Do],
// [CompletionBuilder.Stream], [IngestBuilder.Do]) honour context
// cancellation and deadlines. Multi-stage operations check context
// between phases to avoid unnecessary work if the context has been
// cancelled.
//
// # Thread safety
//
// [Service], [CostCalculator], [BudgetManager], [RateLimiter], and
// all [Memory] implementations are safe for concurrent use. Methods
// document their locking behaviour individually.
// [CompletionBuilder] and [EmbeddingBuilder] instances are not safe
// for concurrent use; create one per goroutine.
package llm_domain
