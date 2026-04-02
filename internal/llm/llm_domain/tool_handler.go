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
	"fmt"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// ToolHandlerFunc handles tool calls from the model by processing JSON
// arguments and returning result text. Errors are sent back to the model
// as the tool result, allowing it to recover or try a different approach.
type ToolHandlerFunc func(ctx context.Context, arguments string) (string, error)

// DefaultMaxToolRounds is the maximum number of tool dispatch rounds before
// the loop terminates. Each round consists of dispatching all tool calls in a
// response and re-calling the LLM.
const DefaultMaxToolRounds = 10

// ToolFunc adds a function tool definition and registers a handler for it.
// When Do() is called and the model returns a tool call matching name, the
// handler is invoked automatically and the result is fed back to the model.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes params (*llm_dto.JSONSchema) which describes the function parameters.
// Takes handler (ToolHandlerFunc) which is called when the model invokes this
// tool.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) ToolFunc(name, description string, params *llm_dto.JSONSchema, handler ToolHandlerFunc) *CompletionBuilder {
	b.request.Tools = append(b.request.Tools, llm_dto.NewFunctionTool(name, description, params))
	if b.toolHandlers == nil {
		b.toolHandlers = make(map[string]ToolHandlerFunc)
	}
	b.toolHandlers[name] = handler
	return b
}

// StrictToolFunc adds a function tool with strict schema enforcement and
// registers a handler for it.
//
// Takes name (string) which is the function name.
// Takes description (string) which explains what the function does.
// Takes params (*llm_dto.JSONSchema) which describes the function parameters.
// Takes handler (ToolHandlerFunc) which is called when the model invokes this
// tool.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) StrictToolFunc(name, description string, params *llm_dto.JSONSchema, handler ToolHandlerFunc) *CompletionBuilder {
	b.request.Tools = append(b.request.Tools, llm_dto.NewStrictFunctionTool(name, description, params))
	if b.toolHandlers == nil {
		b.toolHandlers = make(map[string]ToolHandlerFunc)
	}
	b.toolHandlers[name] = handler
	return b
}

// MaxToolRounds sets the maximum number of tool dispatch rounds.
//
// When n is 0, DefaultMaxToolRounds is used. When n is negative, unlimited
// rounds are allowed.
//
// Takes n (int) which is the maximum number of rounds.
//
// Returns *CompletionBuilder for method chaining.
func (b *CompletionBuilder) MaxToolRounds(n int) *CompletionBuilder {
	b.maxToolRounds = n
	return b
}

// hasToolHandlers reports whether any tool handlers have been registered.
//
// Returns bool which is true when at least one tool handler exists.
func (b *CompletionBuilder) hasToolHandlers() bool {
	return len(b.toolHandlers) > 0
}

// resolvedMaxToolRounds returns the effective max tool rounds, applying the
// default when the configured value is 0.
//
// Returns int which is the configured value or DefaultMaxToolRounds if zero.
func (b *CompletionBuilder) resolvedMaxToolRounds() int {
	if b.maxToolRounds == 0 {
		return DefaultMaxToolRounds
	}
	return b.maxToolRounds
}

// executeToolLoop runs the tool dispatch loop. It checks whether the response
// contains tool calls and, if handlers are registered, dispatches them and
// re-calls the LLM until the model stops requesting tools or the maximum
// number of rounds is reached.
//
// Takes providerName (string) which identifies the LLM provider.
// Takes response (*llm_dto.CompletionResponse) which is the initial response.
//
// Returns *llm_dto.CompletionResponse which is the final response after all
// tool rounds.
// Returns error when a re-call to the LLM fails.
func (b *CompletionBuilder) executeToolLoop(ctx context.Context, providerName string, response *llm_dto.CompletionResponse) (*llm_dto.CompletionResponse, error) {
	if !b.hasToolHandlers() {
		return response, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	maxRounds := b.resolvedMaxToolRounds()
	round := 0

	for response.HasToolCalls() {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("tool loop cancelled: %w", err)
		}

		if maxRounds >= 0 && round >= maxRounds {
			l.Debug("Tool loop reached maximum rounds",
				logger_domain.Int("max_rounds", maxRounds),
			)
			toolLoopMaxRoundsCount.Add(ctx, 1)
			break
		}
		round++

		assistantMessage := llm_dto.Message{
			Role:      llm_dto.RoleAssistant,
			Content:   response.Content(),
			ToolCalls: response.ToolCalls(),
		}
		b.request.Messages = append(b.request.Messages, assistantMessage)
		b.toolLoopMessages = append(b.toolLoopMessages, assistantMessage)

		for _, tc := range response.ToolCalls() {
			result := b.dispatchToolCall(ctx, tc)
			toolMessage := llm_dto.NewToolResultMessage(tc.ID, result)
			b.request.Messages = append(b.request.Messages, toolMessage)
			b.toolLoopMessages = append(b.toolLoopMessages, toolMessage)
		}

		toolLoopRoundsCount.Add(ctx, 1)

		var err error
		response, err = b.executeWithCaching(ctx, providerName)
		if err != nil {
			return nil, fmt.Errorf("tool loop round %d: %w", round, err)
		}
	}

	return response, nil
}

// dispatchToolCall looks up the handler for a tool call and invokes it.
// If no handler is registered or the handler fails, an error message is
// returned instead.
//
// Takes tc (llm_dto.ToolCall) which is the tool call to dispatch.
//
// Returns string which is the tool result text or an error message.
func (b *CompletionBuilder) dispatchToolCall(ctx context.Context, tc llm_dto.ToolCall) string {
	ctx, l := logger_domain.From(ctx, log)
	toolLoopDispatchesCount.Add(ctx, 1)

	handler, exists := b.toolHandlers[tc.Function.Name]
	if !exists {
		toolLoopErrorsCount.Add(ctx, 1)
		return fmt.Sprintf("error: no handler registered for tool %q", tc.Function.Name)
	}

	result, err := handler(ctx, tc.Function.Arguments)
	if err != nil {
		toolLoopErrorsCount.Add(ctx, 1)
		l.Error("Tool handler failed",
			logger_domain.String("tool", tc.Function.Name),
			logger_domain.Error(err),
		)
		return "error: the tool encountered an internal error"
	}

	return result
}
