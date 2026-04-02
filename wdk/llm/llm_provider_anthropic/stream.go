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

package llm_provider_anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
)

// streamState holds the data gathered while processing a stream.
type streamState struct {
	// finalUsage stores the accumulated token usage after the stream completes.
	finalUsage *llm_dto.Usage

	// lastFinishReason stores the most recent finish reason from the LLM response;
	// nil if no response has been received yet.
	lastFinishReason *llm_dto.FinishReason

	// messageID is the unique identifier of the current message being streamed.
	messageID string

	// model is the name of the AI model to use for generation.
	model string

	// accumulatedToolCalls collects tool calls received during streaming.
	accumulatedToolCalls []llm_dto.ToolCall

	// inputTokens is the number of prompt tokens from the message_start event.
	inputTokens int

	// cachedTokens is the number of cached input tokens from the message_start event.
	cachedTokens int

	// currentToolIndex is the position of the tool currently being processed.
	currentToolIndex int
}

// Stream sends a streaming completion request to Anthropic.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model and messages.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events as they
// arrive from the Anthropic API.
// Returns error when the request cannot be initiated.
//
// Spawns a goroutine to process the stream and send events on the returned
// channel. The channel is closed when the stream completes or errors.
func (p *anthropicProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	ctx, l := logger.From(ctx, log)
	streamCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	params := p.buildMessageParams(request, model)

	l.Debug("Starting Anthropic streaming completion",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	stream := p.client.Messages.NewStreaming(ctx, params)

	events := make(chan llm_dto.StreamEvent)

	go p.processStream(ctx, stream, events, model)

	return events, nil
}

// processStream reads from the Anthropic stream and converts events.
//
// Takes stream (*ssestream.Stream) which provides the Anthropic message events.
// Takes events (chan<- llm_dto.StreamEvent) which receives the converted stream
// events.
// Takes model (string) which identifies the model name for response metadata.
func (p *anthropicProvider) processStream(ctx context.Context, stream *ssestream.Stream[anthropic.MessageStreamEventUnion], events chan<- llm_dto.StreamEvent, model string) {
	defer close(events)
	defer goroutine.RecoverPanic(ctx, "llm.anthropicProcessStream")
	start := time.Now()

	state := newStreamState(model)

	for stream.Next() {
		if ctx.Err() != nil {
			break
		}

		delta, shouldSend := p.handleStreamEvent(new(stream.Current()), state)
		if !shouldSend {
			continue
		}

		chunk := &llm_dto.StreamChunk{
			ID:    state.messageID,
			Model: model,
			Delta: delta,
		}

		if !p.sendEvent(ctx, events, llm_dto.NewChunkEvent(chunk)) {
			return
		}
	}

	streamDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	if err := stream.Err(); err != nil {
		streamErrorCount.Add(ctx, 1)
		events <- llm_dto.NewErrorEvent(fmt.Errorf("anthropic stream error: %w", wrapError(err)))
		return
	}

	events <- llm_dto.NewDoneEvent(p.buildFinalResponse(state))
}

// handleStreamEvent processes a single stream event and returns the delta if
// one should be sent.
//
// Takes event (*anthropic.MessageStreamEventUnion) which is the stream event to
// process.
// Takes state (*streamState) which tracks the current streaming state.
//
// Returns *llm_dto.MessageDelta which contains the delta to send, or nil if
// no delta should be sent.
// Returns bool which indicates whether a delta should be sent.
func (p *anthropicProvider) handleStreamEvent(event *anthropic.MessageStreamEventUnion, state *streamState) (*llm_dto.MessageDelta, bool) {
	switch e := event.AsAny().(type) {
	case anthropic.MessageStartEvent:
		state.messageID = e.Message.ID
		state.inputTokens = int(e.Message.Usage.InputTokens)
		state.cachedTokens = int(e.Message.Usage.CacheReadInputTokens)
		return nil, false

	case anthropic.ContentBlockStartEvent:
		p.handleContentBlockStart(&e, state)
		return nil, false

	case anthropic.ContentBlockDeltaEvent:
		return p.handleContentBlockDelta(&e, state), true

	case anthropic.MessageDeltaEvent:
		p.handleMessageDelta(&e, state)
		return nil, false

	case anthropic.MessageStopEvent:
		return nil, false

	default:
		return nil, false
	}
}

// handleContentBlockStart processes content block start events for tool use.
//
// Takes e (*anthropic.ContentBlockStartEvent) which contains the event data.
// Takes state (*streamState) which tracks accumulated tool calls.
func (*anthropicProvider) handleContentBlockStart(e *anthropic.ContentBlockStartEvent, state *streamState) {
	toolUse, ok := e.ContentBlock.AsAny().(anthropic.ToolUseBlock)
	if !ok {
		return
	}

	state.currentToolIndex++
	for len(state.accumulatedToolCalls) <= state.currentToolIndex {
		state.accumulatedToolCalls = append(state.accumulatedToolCalls, llm_dto.ToolCall{Type: "function"})
	}
	state.accumulatedToolCalls[state.currentToolIndex].ID = toolUse.ID
	state.accumulatedToolCalls[state.currentToolIndex].Function.Name = toolUse.Name
}

// handleContentBlockDelta processes content block delta events (text or tool
// input).
//
// Takes e (*anthropic.ContentBlockDeltaEvent) which contains the delta event to
// process.
// Takes state (*streamState) which holds the accumulated tool calls and current
// index.
//
// Returns *llm_dto.MessageDelta which contains the processed content or tool
// call delta.
func (*anthropicProvider) handleContentBlockDelta(e *anthropic.ContentBlockDeltaEvent, state *streamState) *llm_dto.MessageDelta {
	delta := &llm_dto.MessageDelta{}

	switch d := e.Delta.AsAny().(type) {
	case anthropic.TextDelta:
		delta.Content = new(d.Text)

	case anthropic.InputJSONDelta:
		if state.currentToolIndex >= 0 && state.currentToolIndex < len(state.accumulatedToolCalls) {
			state.accumulatedToolCalls[state.currentToolIndex].Function.Arguments += d.PartialJSON
			delta.ToolCalls = []llm_dto.ToolCallDelta{
				{
					Index: state.currentToolIndex,
					Function: &llm_dto.FunctionCallDelta{
						Arguments: &d.PartialJSON,
					},
				},
			}
		}
	}

	return delta
}

// handleMessageDelta processes message delta events for usage and finish reason.
//
// Takes e (*anthropic.MessageDeltaEvent) which contains the delta event data.
// Takes state (*streamState) which holds the streaming state to update.
func (p *anthropicProvider) handleMessageDelta(e *anthropic.MessageDeltaEvent, state *streamState) {
	if e.Usage.OutputTokens > 0 {
		completionTokens := int(e.Usage.OutputTokens)
		state.finalUsage = &llm_dto.Usage{
			PromptTokens:     state.inputTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      state.inputTokens + completionTokens,
			CachedTokens:     state.cachedTokens,
		}
	}
	if e.Delta.StopReason != "" {
		state.lastFinishReason = new(p.convertStopReason(anthropic.StopReason(e.Delta.StopReason)))
	}
}

// sendEvent sends an event to the channel, handling context cancellation.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream event.
// Takes event (llm_dto.StreamEvent) which is the event to send.
//
// Returns bool which is false if context was cancelled, true otherwise.
func (*anthropicProvider) sendEvent(ctx context.Context, events chan<- llm_dto.StreamEvent, event llm_dto.StreamEvent) bool {
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
// Takes state (*streamState) which holds the accumulated streaming data.
//
// Returns *llm_dto.CompletionResponse which contains the assembled response
// with choices, usage data, and any tool calls.
func (p *anthropicProvider) buildFinalResponse(state *streamState) *llm_dto.CompletionResponse {
	finalResponse := &llm_dto.CompletionResponse{
		ID:    state.messageID,
		Model: state.model,
		Usage: state.finalUsage,
	}

	finishReason := llm_dto.FinishReasonStop
	if state.lastFinishReason != nil {
		finishReason = *state.lastFinishReason
	}

	finalMessage := llm_dto.Message{
		Role: llm_dto.RoleAssistant,
	}

	if len(state.accumulatedToolCalls) > 0 {
		p.validateToolCallArguments(state.accumulatedToolCalls)
		finalMessage.ToolCalls = state.accumulatedToolCalls
	}

	finalResponse.Choices = []llm_dto.Choice{
		{
			Index:        0,
			Message:      finalMessage,
			FinishReason: finishReason,
		},
	}

	return finalResponse
}

// validateToolCallArguments verifies accumulated JSON arguments are valid.
//
// Takes toolCalls ([]llm_dto.ToolCall) which contains the tool calls to
// validate.
func (*anthropicProvider) validateToolCallArguments(toolCalls []llm_dto.ToolCall) {
	for i := range toolCalls {
		var parsed any
		if err := json.Unmarshal([]byte(toolCalls[i].Function.Arguments), &parsed); err != nil {
			continue
		}
	}
}

// newStreamState creates a new stream state with initial values.
//
// Takes model (string) which specifies the model identifier.
//
// Returns *streamState which is the initialised state ready for streaming.
func newStreamState(model string) *streamState {
	return &streamState{
		currentToolIndex: -1,
		model:            model,
	}
}
