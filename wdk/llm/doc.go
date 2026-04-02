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

// Package llm provides a unified, provider-agnostic interface for
// interacting with large language models.
//
// The fluent builder API covers completions, streaming,
// tool/function calling, structured JSON output, embeddings, and
// conversation memory. Provider adapters live in separate
// sub-packages so you only import the ones you need.
//
// # Basic usage
//
//	response, err := llm.NewCompletion().
//	    Model("gpt-5").
//	    System("You are helpful.").
//	    User("Hello!").
//	    Do(ctx)
//
//	fmt.Println(response.Content())
//
// # Streaming
//
//	events, err := llm.NewCompletion().
//	    Model("claude-sonnet-4-5-20250929").
//	    User("Write a poem.").
//	    WithProvider("anthropic").
//	    Stream(ctx)
//
//	for event := range events {
//	    if event.Chunk != nil && event.Chunk.Delta.Content != nil {
//	        fmt.Print(*event.Chunk.Delta.Content)
//	    }
//	}
//
// # Structured output
//
//	response, err := llm.NewCompletion().
//	    Model("gpt-5").
//	    User("Extract entities from: 'John works at Acme'").
//	    StructuredResponse("entities", llm.ObjectSchema(
//	        map[string]*llm.JSONSchema{
//	            "people":    llm.Ptr(llm.ArraySchema(llm.StringSchema())),
//	            "companies": llm.Ptr(llm.ArraySchema(llm.StringSchema())),
//	        },
//	        []string{"people", "companies"},
//	    )).
//	    Do(ctx)
//
// # Tool calling
//
//	response, err := llm.NewCompletion().
//	    Model("gpt-5").
//	    User("What's the weather in Paris?").
//	    WithTool("get_weather", "Get current weather",
//	        &llm.JSONSchema{
//	            Type: "object",
//	            Properties: map[string]*llm.JSONSchema{
//	                "city": llm.Ptr(llm.StringSchema()),
//	            },
//	            Required: []string{"city"},
//	        }).
//	    Do(ctx)
//
//	if response.HasToolCalls() {
//	    for _, tc := range response.ToolCalls() {
//	        fmt.Printf("Call %s with %s\n",
//	            tc.Function.Name, tc.Function.Arguments)
//	    }
//	}
//
// # Thread safety
//
// [Service], [CostCalculator], [BudgetManager], [RateLimiter], and
// [CacheManager] are safe for concurrent use. Builder instances
// should not be shared between goroutines.
package llm
