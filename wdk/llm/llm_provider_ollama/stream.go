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

package llm_provider_ollama

import (
	"context"
	"fmt"
	"time"

	"github.com/ollama/ollama/api"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
)

// Stream sends a streaming completion request to Ollama.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters including model and messages.
//
// Returns <-chan llm_dto.StreamEvent which yields streaming events as they
// arrive from the Ollama API.
// Returns error when the stream cannot be started.
//
// Spawns a goroutine that processes the stream and sends events to the
// returned channel until the stream completes or the context is cancelled.
func (p *ollamaProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	ctx, l := logger.From(ctx, log)
	streamCount.Add(ctx, 1)

	model, ref := p.resolveModel(request.Model, p.defaultModel)

	if err := p.ensureModel(ctx, model, ref); err != nil {
		streamErrorCount.Add(ctx, 1)
		return nil, err
	}

	l.Debug("Starting Ollama streaming completion",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	chatRequest := p.buildChatRequest(ctx, request, model)

	events := make(chan llm_dto.StreamEvent)

	go p.processStream(ctx, chatRequest, model, events)

	return events, nil
}

// processStream runs the Ollama chat with streaming and feeds events into
// the channel.
//
// Takes chatRequest (*api.ChatRequest) which is the Ollama request.
// Takes model (string) which is the model being used.
// Takes events (chan<- llm_dto.StreamEvent) which receives converted events.
func (p *ollamaProvider) processStream(ctx context.Context, chatRequest *api.ChatRequest, model string, events chan<- llm_dto.StreamEvent) {
	defer close(events)
	defer goroutine.RecoverPanic(ctx, "llm.ollamaProcessStream")
	start := time.Now()

	var totalPromptTokens int
	var totalEvalTokens int
	var doneToolCalls []api.ToolCall

	err := p.client.Chat(ctx, chatRequest, func(response api.ChatResponse) error {
		if response.Done {
			totalPromptTokens = response.PromptEvalCount
			totalEvalTokens = response.EvalCount
			doneToolCalls = response.Message.ToolCalls
			return nil
		}

		chunk := &llm_dto.StreamChunk{
			Model: model,
			Delta: &llm_dto.MessageDelta{
				Content: new(response.Message.Content),
			},
		}

		select {
		case events <- llm_dto.NewChunkEvent(chunk):
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	})

	streamDuration.Record(ctx, float64(time.Since(start).Milliseconds()))

	if err != nil {
		streamErrorCount.Add(ctx, 1)
		events <- llm_dto.NewErrorEvent(fmt.Errorf("ollama stream error: %w", wrapError(err)))
		return
	}

	events <- buildStreamDoneEvent(model, doneToolCalls, totalPromptTokens, totalEvalTokens)
}

// buildStreamDoneEvent constructs the final StreamEvent from
// Ollama stream completion data.
//
// Takes model (string) which is the model name for the response.
// Takes toolCalls ([]api.ToolCall) which holds any tool calls
// made during the stream.
// Takes promptTokens (int) which is the prompt token count.
// Takes evalTokens (int) which is the completion token count.
//
// Returns llm_dto.StreamEvent which is the done event carrying
// the final completion response.
func buildStreamDoneEvent(
	model string, toolCalls []api.ToolCall, promptTokens, evalTokens int,
) llm_dto.StreamEvent {
	finishReason := llm_dto.FinishReasonStop
	doneMessage := llm_dto.Message{
		Role: llm_dto.RoleAssistant,
	}

	if len(toolCalls) > 0 {
		doneMessage.ToolCalls = convertOllamaToolCalls(toolCalls)
		finishReason = llm_dto.FinishReasonToolCalls
	}

	return llm_dto.NewDoneEvent(&llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index:        0,
				Message:      doneMessage,
				FinishReason: finishReason,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: evalTokens,
			TotalTokens:      promptTokens + evalTokens,
		},
	})
}
