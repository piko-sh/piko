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

package llm_provider_openai

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
)

// streamState holds data gathered during stream processing.
type streamState struct {
	// finalUsage stores the accumulated token usage after the stream completes.
	finalUsage *llm_dto.Usage

	// lastFinishReason is the most recent reason why generation stopped;
	// nil until the first chunk completes.
	lastFinishReason *llm_dto.FinishReason

	// lastID is the most recent stream entry ID that was processed.
	lastID string

	// lastModel is the most recent model name used in the stream.
	lastModel string

	// accumulatedToolCalls collects tool calls received during streaming.
	accumulatedToolCalls []llm_dto.ToolCall
}

// Stream sends a streaming completion request to OpenAI.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model and messages.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events as they
// arrive from the OpenAI API.
// Returns error when the stream cannot be started.
//
// Spawns a goroutine to process incoming stream events. The channel is closed
// when the stream completes or encounters an error.
func (p *openaiProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	ctx, l := logger.From(ctx, log)
	streamCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	params := p.buildChatParams(request, model)

	if request.StreamOptions != nil && request.StreamOptions.IncludeUsage {
		params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}
	}

	l.Debug("Starting OpenAI streaming completion",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	events := make(chan llm_dto.StreamEvent)

	go p.processStream(ctx, stream, events)

	return events, nil
}

// processStream reads from the OpenAI stream and converts events.
//
// Takes stream (*ssestream.Stream) which provides the SSE stream to read from.
// Takes events (chan<- llm_dto.StreamEvent) which receives the converted events.
func (p *openaiProvider) processStream(ctx context.Context, stream *ssestream.Stream[openai.ChatCompletionChunk], events chan<- llm_dto.StreamEvent) {
	defer close(events)
	defer goroutine.RecoverPanic(ctx, "llm.openaiProcessStream")
	start := time.Now()

	state := &streamState{}

	for stream.Next() {
		if ctx.Err() != nil {
			break
		}

		chunk := stream.Current()

		state.lastID = chunk.ID
		state.lastModel = chunk.Model

		if !p.processChunkChoices(ctx, events, &chunk, state) {
			return
		}
	}

	streamDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	if err := stream.Err(); err != nil {
		streamErrorCount.Add(ctx, 1)
		events <- llm_dto.NewErrorEvent(fmt.Errorf("openai stream error: %w", wrapError(err)))
		return
	}

	events <- llm_dto.NewDoneEvent(p.buildFinalResponse(state))
}

// processChunkChoices processes all choices in a chunk.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives stream events.
// Takes chunk (*openai.ChatCompletionChunk) which contains the choices to process.
// Takes state (*streamState) which tracks the current stream state.
//
// Returns bool which is false if processing should stop.
func (p *openaiProvider) processChunkChoices(ctx context.Context, events chan<- llm_dto.StreamEvent, chunk *openai.ChatCompletionChunk, state *streamState) bool {
	for i := range chunk.Choices {
		choice := &chunk.Choices[i]
		delta := p.buildDelta(choice, state)
		finishReason := p.extractFinishReason(choice, state)

		streamChunk := &llm_dto.StreamChunk{
			ID:           chunk.ID,
			Model:        chunk.Model,
			Delta:        delta,
			FinishReason: finishReason,
		}

		p.extractUsage(chunk, streamChunk, state)

		if !p.sendEvent(ctx, events, llm_dto.NewChunkEvent(streamChunk)) {
			return false
		}
	}
	return true
}

// buildDelta constructs a MessageDelta from a streaming response choice.
//
// Takes choice (*openai.ChatCompletionChunkChoice) which contains the streamed
// content fragment.
// Takes state (*streamState) which tracks tool call assembly across chunks.
//
// Returns *llm_dto.MessageDelta which contains the extracted role, content,
// and tool calls from the chunk.
func (p *openaiProvider) buildDelta(choice *openai.ChatCompletionChunkChoice, state *streamState) *llm_dto.MessageDelta {
	delta := &llm_dto.MessageDelta{}

	if choice.Delta.Role != "" {
		delta.Role = new(llm_dto.Role(choice.Delta.Role))
	}

	if choice.Delta.Content != "" {
		delta.Content = new(choice.Delta.Content)
	}

	if len(choice.Delta.ToolCalls) > 0 {
		delta.ToolCalls = p.buildToolCallDeltas(choice.Delta.ToolCalls, state)
	}

	return delta
}

// buildToolCallDeltas converts OpenAI tool calls to DTO tool call deltas.
//
// Takes toolCalls ([]openai.ChatCompletionChunkChoiceDeltaToolCall) which
// contains the raw tool call data from the OpenAI streaming response.
// Takes state (*streamState) which tracks the current streaming state.
//
// Returns []llm_dto.ToolCallDelta which contains the converted tool call
// deltas ready for use by the LLM DTO layer.
func (p *openaiProvider) buildToolCallDeltas(toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall, state *streamState) []llm_dto.ToolCallDelta {
	deltas := make([]llm_dto.ToolCallDelta, len(toolCalls))
	for i := range toolCalls {
		deltas[i] = p.buildSingleToolCallDelta(&toolCalls[i], state)
	}
	return deltas
}

// buildSingleToolCallDelta converts a single OpenAI tool call to a DTO delta.
//
// Takes tc (*openai.ChatCompletionChunkChoiceDeltaToolCall) which is the OpenAI
// tool call chunk to convert.
// Takes state (*streamState) which holds accumulated tool calls for the stream.
//
// Returns llm_dto.ToolCallDelta which is the converted tool call delta.
func (p *openaiProvider) buildSingleToolCallDelta(tc *openai.ChatCompletionChunkChoiceDeltaToolCall, state *streamState) llm_dto.ToolCallDelta {
	tcd := llm_dto.ToolCallDelta{Index: int(tc.Index)}

	if tc.ID != "" {
		tcd.ID = new(tc.ID)
	}
	if tc.Type != "" {
		tcd.Type = new(string(tc.Type))
	}
	if tc.Function.Name != "" || tc.Function.Arguments != "" {
		tcd.Function = p.buildFunctionCallDelta(&tc.Function)
	}

	p.accumulateToolCall(&state.accumulatedToolCalls, tcd)
	return tcd
}

// buildFunctionCallDelta builds a FunctionCallDelta from an OpenAI function.
//
// Takes functionCall
// (*openai.ChatCompletionChunkChoiceDeltaToolCallFunction) which
// provides the OpenAI function call data to convert.
//
// Returns *llm_dto.FunctionCallDelta which contains the converted function
// call delta with name and arguments set when present.
func (*openaiProvider) buildFunctionCallDelta(functionCall *openai.ChatCompletionChunkChoiceDeltaToolCallFunction) *llm_dto.FunctionCallDelta {
	fcd := &llm_dto.FunctionCallDelta{}
	if functionCall.Name != "" {
		fcd.Name = new(functionCall.Name)
	}
	if functionCall.Arguments != "" {
		fcd.Arguments = new(functionCall.Arguments)
	}
	return fcd
}

// extractFinishReason extracts the finish reason from a choice if present.
//
// Takes choice (*openai.ChatCompletionChunkChoice) which contains the streamed
// response data.
// Takes state (*streamState) which tracks the streaming session state.
//
// Returns *llm_dto.FinishReason which is the converted finish reason, or nil
// if no finish reason is present in the choice.
func (p *openaiProvider) extractFinishReason(choice *openai.ChatCompletionChunkChoice, state *streamState) *llm_dto.FinishReason {
	if choice.FinishReason == "" {
		return nil
	}

	reason := p.convertFinishReason(choice.FinishReason)
	state.lastFinishReason = &reason
	return &reason
}

// extractUsage extracts usage information from a chunk if present.
//
// Takes chunk (*openai.ChatCompletionChunk) which contains the streaming
// response data to extract usage from.
// Takes streamChunk (*llm_dto.StreamChunk) which receives the extracted usage
// data.
// Takes state (*streamState) which stores the final usage for later reference.
func (*openaiProvider) extractUsage(chunk *openai.ChatCompletionChunk, streamChunk *llm_dto.StreamChunk, state *streamState) {
	if chunk.Usage.TotalTokens == 0 {
		return
	}

	streamChunk.Usage = &llm_dto.Usage{
		PromptTokens:     int(chunk.Usage.PromptTokens),
		CompletionTokens: int(chunk.Usage.CompletionTokens),
		TotalTokens:      int(chunk.Usage.TotalTokens),
		CachedTokens:     int(chunk.Usage.PromptTokensDetails.CachedTokens),
	}
	state.finalUsage = streamChunk.Usage
}

// sendEvent sends an event to the channel, handling context cancellation.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
// Takes event (llm_dto.StreamEvent) which is the event to send.
//
// Returns bool which is false if the context was cancelled.
func (*openaiProvider) sendEvent(ctx context.Context, events chan<- llm_dto.StreamEvent, event llm_dto.StreamEvent) bool {
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		events <- llm_dto.NewErrorEvent(context.Cause(ctx))
		return false
	}
}

// buildFinalResponse constructs the final completion response from
// accumulated state.
//
// Takes state (*streamState) which holds the accumulated stream data.
//
// Returns *llm_dto.CompletionResponse which contains the assembled response.
func (*openaiProvider) buildFinalResponse(state *streamState) *llm_dto.CompletionResponse {
	finalResponse := &llm_dto.CompletionResponse{
		ID:    state.lastID,
		Model: state.lastModel,
		Usage: state.finalUsage,
	}

	if state.lastFinishReason != nil {
		finalResponse.Choices = []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:      llm_dto.RoleAssistant,
					ToolCalls: state.accumulatedToolCalls,
				},
				FinishReason: *state.lastFinishReason,
			},
		}
	}

	return finalResponse
}

// accumulateToolCall accumulates tool call deltas into complete tool calls.
//
// Takes toolCalls (*[]llm_dto.ToolCall) which holds the accumulated calls.
// Takes delta (llm_dto.ToolCallDelta) which contains the partial data to merge.
func (*openaiProvider) accumulateToolCall(toolCalls *[]llm_dto.ToolCall, delta llm_dto.ToolCallDelta) {
	for len(*toolCalls) <= delta.Index {
		*toolCalls = append(*toolCalls, llm_dto.ToolCall{})
	}

	tc := &(*toolCalls)[delta.Index]

	if delta.ID != nil {
		tc.ID = *delta.ID
	}
	if delta.Type != nil {
		tc.Type = *delta.Type
	}
	if delta.Function != nil {
		if delta.Function.Name != nil {
			tc.Function.Name = *delta.Function.Name
		}
		if delta.Function.Arguments != nil {
			tc.Function.Arguments += *delta.Function.Arguments
		}
	}
}
