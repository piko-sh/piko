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

package llm_provider_mistral

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/wdk/logger"
)

const (
	// maxSSELineBytes bounds a single SSE line so a malicious or malfunctioning
	// peer cannot cause unbounded memory growth by streaming a single oversized
	// `data:` line without a newline. Real SSE lines are typically much smaller,
	// so 1 MiB is a generous ceiling.
	maxSSELineBytes = 1 * 1024 * 1024

	// initialSSEScannerBufferBytes is the initial buffer size used by the SSE
	// line scanner; it grows up to maxSSELineBytes as needed.
	initialSSEScannerBufferBytes = 64 * 1024
)

// ErrSSELineTooLarge indicates that a Mistral SSE stream line exceeded the
// per-line size cap, suggesting either a hostile peer or a malfunctioning
// upstream emitter.
var ErrSSELineTooLarge = errors.New("mistral SSE line exceeded maximum size")

// streamState holds the data gathered during stream processing.
type streamState struct {
	// finalUsage stores the accumulated token usage after the stream completes.
	finalUsage *llm_dto.Usage

	// lastFinishReason stores the most recent finish reason from the LLM
	// response; nil when no response has been received.
	lastFinishReason *llm_dto.FinishReason

	// lastID is the most recent stream entry ID that was processed.
	lastID string

	// lastModel is the most recent model identifier used in this stream.
	lastModel string

	// accumulatedToolCalls stores tool calls received during streaming.
	accumulatedToolCalls []llm_dto.ToolCall
}

// Stream sends a streaming completion request to Mistral.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model and messages.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events as they
// arrive from the API.
// Returns error when the request cannot be created or the API returns an error.
//
// Spawns a goroutine to process the stream. The channel is closed when the
// stream ends or the context is cancelled.
func (p *mistralProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	defer goroutine.RecoverPanic(ctx, "llm.mistralProvider.Stream")

	ctx, l := logger.From(ctx, log)
	streamCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.defaultModel
	}

	apiReq := p.buildRequest(request, model, true)

	l.Debug("Starting Mistral streaming completion",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	streamContext := p.streamContext(ctx)

	httpReq, err := http.NewRequestWithContext(streamContext, http.MethodPost, p.config.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	response, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, classifyStreamErrorResponse(response)
	}

	events := make(chan llm_dto.StreamEvent)

	p.streamWaitGroup.Add(1)
	go p.processStream(streamContext, response, events)

	return events, nil
}

// classifyStreamErrorResponse classifies a non-OK Mistral streaming response.
//
// The response body is drained and closed before returning.
//
// Takes response (*http.Response) which is the non-OK upstream response.
//
// Returns error which carries the classified upstream failure as either a
// safeerror-wrapped 4xx (user-rejection) or a transient 5xx-style error;
// the returned error wraps a *llm_domain.ProviderError so the retry
// executor can classify the failure and honour any Retry-After hint.
func classifyStreamErrorResponse(response *http.Response) error {
	respBody, readErr := readBoundedBody(response.Body)
	detail := http.StatusText(response.StatusCode)
	if len(respBody) > 0 {
		detail = string(respBody)
	}
	if readErr != nil && !errors.Is(readErr, errResponseTruncated) {
		detail = fmt.Sprintf("%s (read error: %v)", detail, readErr)
	}
	baseErr := fmt.Errorf("mistral API error (status %d): %s", response.StatusCode, detail)
	providerErr := newProviderError(response, fmt.Sprintf("mistral API error: %s", detail), baseErr)
	drainAndClose(response)
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError {
		return safeerror.NewError("mistral stream rejected", providerErr)
	}
	return providerErr
}

// streamContext returns a context that is cancelled when either the caller's
// context is cancelled or the provider is closed. The returned cancel function
// must be called when the stream goroutine completes to release the watcher.
//
// Takes ctx (context.Context) which is the caller's context.
//
// Returns context.Context which carries the merged cancellation signal.
func (p *mistralProvider) streamContext(ctx context.Context) context.Context {
	if p.closeContext == nil {
		return ctx
	}
	merged, cancel := context.WithCancelCause(ctx)
	stopWatch := context.AfterFunc(p.closeContext, func() {
		cancel(context.Cause(p.closeContext))
	})
	context.AfterFunc(merged, func() {
		stopWatch()
	})
	return merged
}

// mistralStreamChunk represents a chunk from Mistral's streaming API.
type mistralStreamChunk struct {
	// ID is the unique identifier for this stream chunk.
	ID string `json:"id"`

	// Object is the object type identifier returned by the Mistral API.
	Object string `json:"object"`

	// Model is the identifier of the model that generated this response.
	Model string `json:"model"`

	// Usage contains token usage statistics for this chunk; nil when not reported.
	Usage *mistralUsage `json:"usage,omitempty"`

	// Choices contains the list of generated completions for this stream chunk.
	Choices []mistralStreamChunkChoice `json:"choices"`

	// Created is the Unix timestamp when the chunk was generated.
	Created int64 `json:"created"`
}

// mistralStreamChunkChoice represents a single choice within a Mistral
// streaming response chunk.
type mistralStreamChunkChoice struct {
	// FinishReason indicates why the model stopped generating tokens.
	FinishReason string `json:"finish_reason,omitempty"`

	// Delta contains the incremental content for this streaming chunk.
	Delta mistralDelta `json:"delta"`

	// Index is the position of this choice in the response.
	Index int `json:"index"`
}

// mistralDelta holds the incremental content from a Mistral streaming response.
type mistralDelta struct {
	// Role is the sender role for this message chunk.
	Role string `json:"role,omitempty"`

	// Content is the text content of the message delta.
	Content string `json:"content,omitempty"`

	// ToolCalls contains tool invocations requested by the model.
	ToolCalls []mistralToolCall `json:"tool_calls,omitempty"`
}

// processStream reads from the Mistral stream and converts events.
//
// Takes response (*http.Response) which provides the SSE stream to read from.
// Takes events (chan<- llm_dto.StreamEvent) which receives the converted stream
// events.
func (p *mistralProvider) processStream(ctx context.Context, response *http.Response, events chan<- llm_dto.StreamEvent) {
	defer p.streamWaitGroup.Done()
	defer close(events)
	defer drainAndClose(response)
	defer goroutine.RecoverPanic(ctx, "llm.mistralProcessStream")
	start := time.Now()

	state := &streamState{}
	scanner := newBoundedSSEScanner(response.Body)

	for {
		if p.isContextCancelled(ctx, events) {
			return
		}

		line, done := p.readSSELine(ctx, events, scanner)
		if done {
			break
		}
		if line == nil {
			continue
		}

		chunk, ok := p.parseSSEData(line)
		if !ok {
			continue
		}

		state.lastID = chunk.ID
		state.lastModel = chunk.Model

		if !p.processChunkChoices(ctx, events, chunk, state) {
			return
		}
	}

	streamDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	select {
	case events <- llm_dto.NewDoneEvent(p.buildFinalResponse(state)):
	case <-ctx.Done():
	}
}

// newBoundedSSEScanner constructs a bufio.Scanner whose buffer is capped at
// maxSSELineBytes. When a single SSE line exceeds the cap, scanner.Scan
// returns false and scanner.Err returns bufio.ErrTooLong.
//
// Takes body (io.Reader) which is the SSE response body.
//
// Returns *bufio.Scanner configured with the bounded buffer.
func newBoundedSSEScanner(body io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, initialSSEScannerBufferBytes), maxSSELineBytes)
	return scanner
}

// isContextCancelled checks if the context has been cancelled and sends an
// error event.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives an error event when
// the context is cancelled.
//
// Returns bool which is true when the context has been cancelled.
func (*mistralProvider) isContextCancelled(ctx context.Context, events chan<- llm_dto.StreamEvent) bool {
	select {
	case <-ctx.Done():
		select {
		case events <- llm_dto.NewErrorEvent(context.Cause(ctx)):
		default:
		}
		return true
	default:
		return false
	}
}

// readSSELine reads a single line from the SSE stream via the bounded
// scanner.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives error events when
// stream reading fails or when an oversized line is detected.
// Takes scanner (*bufio.Scanner) which provides the bounded SSE stream.
//
// Returns []byte which contains the parsed data payload, or nil when the line
// should be skipped or the stream has ended.
// Returns bool which is true when the stream has ended (EOF, done signal, or
// fatal read error), false when processing should continue.
func (*mistralProvider) readSSELine(ctx context.Context, events chan<- llm_dto.StreamEvent, scanner *bufio.Scanner) ([]byte, bool) {
	if !scanner.Scan() {
		err := scanner.Err()
		if err == nil {
			return nil, true
		}
		streamErrorCount.Add(ctx, 1)
		if errors.Is(err, bufio.ErrTooLong) {
			select {
			case events <- llm_dto.NewErrorEvent(safeerror.NewError(
				"mistral stream rejected",
				fmt.Errorf("SSE line exceeded %d bytes: %w", maxSSELineBytes, ErrSSELineTooLarge),
			)):
			case <-ctx.Done():
			}
			return nil, true
		}
		select {
		case events <- llm_dto.NewErrorEvent(fmt.Errorf("mistral stream read error: %w", err)):
		case <-ctx.Done():
		}
		return nil, true
	}

	line := bytes.TrimSpace(scanner.Bytes())
	if len(line) == 0 {
		return nil, false
	}

	if !bytes.HasPrefix(line, []byte("data: ")) {
		return nil, false
	}

	data := bytes.TrimPrefix(line, []byte("data: "))
	if string(data) == "[DONE]" {
		return nil, true
	}

	out := make([]byte, len(data))
	copy(out, data)
	return out, false
}

// parseSSEData parses the SSE data into a mistralStreamChunk.
//
// Takes data ([]byte) which contains the raw SSE data to parse.
//
// Returns *mistralStreamChunk which contains the parsed chunk data.
// Returns bool which indicates whether parsing succeeded.
func (*mistralProvider) parseSSEData(data []byte) (*mistralStreamChunk, bool) {
	var chunk mistralStreamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, false
	}
	return &chunk, true
}

// processChunkChoices processes all choices in a chunk.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
// Takes chunk (*mistralStreamChunk) which contains the choices to process.
// Takes state (*streamState) which tracks the current stream state.
//
// Returns bool which is false if processing should stop.
func (p *mistralProvider) processChunkChoices(ctx context.Context, events chan<- llm_dto.StreamEvent, chunk *mistralStreamChunk, state *streamState) bool {
	for _, choice := range chunk.Choices {
		delta := p.buildDelta(&choice, state)
		finishReason := p.extractFinishReason(&choice, state)

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

// buildDelta constructs a MessageDelta from a stream chunk choice.
//
// Takes choice (*mistralStreamChunkChoice) which contains the delta data.
// Takes state (*streamState) which tracks the current streaming state.
//
// Returns *llm_dto.MessageDelta which contains the extracted role, content,
// and tool calls from the chunk.
func (p *mistralProvider) buildDelta(choice *mistralStreamChunkChoice, state *streamState) *llm_dto.MessageDelta {
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

// buildToolCallDeltas converts Mistral tool calls to DTO tool call deltas.
//
// Takes toolCalls ([]mistralToolCall) which contains the Mistral tool calls to
// convert.
// Takes state (*streamState) which tracks the current streaming state.
//
// Returns []llm_dto.ToolCallDelta which contains the converted tool call
// deltas.
func (p *mistralProvider) buildToolCallDeltas(toolCalls []mistralToolCall, state *streamState) []llm_dto.ToolCallDelta {
	deltas := make([]llm_dto.ToolCallDelta, len(toolCalls))
	for i, tc := range toolCalls {
		deltas[i] = p.buildSingleToolCallDelta(i, tc, state)
	}
	return deltas
}

// buildSingleToolCallDelta converts a single Mistral tool call to a DTO delta.
//
// Takes index (int) which specifies the position of this tool call in the
// sequence.
// Takes tc (mistralToolCall) which contains the Mistral tool call data to
// convert.
// Takes state (*streamState) which holds the accumulated tool calls for the
// stream.
//
// Returns llm_dto.ToolCallDelta which contains the converted tool call delta.
func (p *mistralProvider) buildSingleToolCallDelta(index int, tc mistralToolCall, state *streamState) llm_dto.ToolCallDelta {
	tcd := llm_dto.ToolCallDelta{Index: index}

	if tc.ID != "" {
		tcd.ID = new(tc.ID)
	}
	if tc.Type != "" {
		tcd.Type = new(tc.Type)
	}
	if tc.Function.Name != "" || tc.Function.Arguments != "" {
		tcd.Function = p.buildFunctionCallDelta(&tc.Function)
	}

	p.accumulateToolCall(&state.accumulatedToolCalls, tcd)
	return tcd
}

// buildFunctionCallDelta builds a FunctionCallDelta from a Mistral function call.
//
// Takes functionCall (*mistralFunctionCall) which provides the
// function name and arguments.
//
// Returns *llm_dto.FunctionCallDelta which contains the converted function call
// data with optional name and arguments fields.
func (*mistralProvider) buildFunctionCallDelta(functionCall *mistralFunctionCall) *llm_dto.FunctionCallDelta {
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
// Takes choice (*mistralStreamChunkChoice) which contains the stream chunk data.
// Takes state (*streamState) which holds the current streaming state.
//
// Returns *llm_dto.FinishReason which is the extracted reason, or nil if not
// present.
func (p *mistralProvider) extractFinishReason(choice *mistralStreamChunkChoice, state *streamState) *llm_dto.FinishReason {
	if choice.FinishReason == "" {
		return nil
	}

	reason := p.convertFinishReason(choice.FinishReason)
	state.lastFinishReason = &reason
	return &reason
}

// extractUsage extracts usage information from a chunk if present.
//
// Takes chunk (*mistralStreamChunk) which contains the source usage data.
// Takes streamChunk (*llm_dto.StreamChunk) which receives the extracted usage.
// Takes state (*streamState) which stores the final usage for later reference.
func (*mistralProvider) extractUsage(chunk *mistralStreamChunk, streamChunk *llm_dto.StreamChunk, state *streamState) {
	if chunk.Usage == nil || chunk.Usage.TotalTokens == 0 {
		return
	}

	streamChunk.Usage = &llm_dto.Usage{
		PromptTokens:     chunk.Usage.PromptTokens,
		CompletionTokens: chunk.Usage.CompletionTokens,
		TotalTokens:      chunk.Usage.TotalTokens,
	}
	state.finalUsage = streamChunk.Usage
}

// sendEvent sends an event to the channel, handling context cancellation.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream event.
// Takes event (llm_dto.StreamEvent) which is the event to send.
//
// Returns bool which is false if the context was cancelled.
func (*mistralProvider) sendEvent(ctx context.Context, events chan<- llm_dto.StreamEvent, event llm_dto.StreamEvent) bool {
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		select {
		case events <- llm_dto.NewErrorEvent(context.Cause(ctx)):
		default:
		}
		return false
	}
}

// buildFinalResponse constructs the final completion response from
// accumulated state.
//
// Takes state (*streamState) which holds the accumulated streaming data.
//
// Returns *llm_dto.CompletionResponse which contains the assembled response.
func (*mistralProvider) buildFinalResponse(state *streamState) *llm_dto.CompletionResponse {
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
// Takes toolCalls (*[]llm_dto.ToolCall) which stores the accumulated calls.
// Takes delta (llm_dto.ToolCallDelta) which contains the partial data to merge.
func (*mistralProvider) accumulateToolCall(toolCalls *[]llm_dto.ToolCall, delta llm_dto.ToolCallDelta) {
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
