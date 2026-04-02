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

package llm_provider_zoltai

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/llm/llm_dto"
)

// Stream sends a streaming completion that delivers a fortune word by word.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes request (*llm_dto.CompletionRequest) which is used for the model name.
//
// Returns <-chan llm_dto.StreamEvent which yields word-by-word chunks.
// Returns error which is always nil.
//
// Safe for concurrent use. Spawns a background goroutine that closes the
// returned channel when streaming completes.
func (p *zoltaiProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	streamCount.Add(ctx, 1)

	model := request.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	fortune := p.config.Fortunes[p.randomSource.IntN(len(p.config.Fortunes))]
	full := p.config.FormatFortune(fortune)

	events := make(chan llm_dto.StreamEvent)

	if len(request.Tools) > 0 {
		go processToolStream(ctx, model, request.Tools[0], len(request.Messages), events)
	} else {
		content := full
		if request.ResponseFormat != nil {
			content = (*zoltaiProvider).formatStructuredOutput(nil, full, request.ResponseFormat)
		}
		go p.processStream(ctx, model, content, len(request.Messages), events)
	}

	return events, nil
}

// processStream delivers a fortune word by word into the events channel,
// then sends a done event.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes model (string) which is the model name for responses.
// Takes full (string) which is the full text to stream.
// Takes msgCount (int) which is the number of input messages for usage stats.
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
func (*zoltaiProvider) processStream(ctx context.Context, model, full string, msgCount int, events chan<- llm_dto.StreamEvent) {
	defer close(events)
	defer goroutine.RecoverPanic(ctx, "llm.zoltaiProcessStream")

	totalWords := len(strings.Fields(full))

	if err := streamLines(ctx, model, full, events); err != nil {
		return
	}

	events <- llm_dto.NewDoneEvent(&llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role: llm_dto.RoleAssistant,
				},
				FinishReason: llm_dto.FinishReasonStop,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     msgCount * estimatedTokensPerMessage,
			CompletionTokens: totalWords,
			TotalTokens:      msgCount*estimatedTokensPerMessage + totalWords,
		},
	})
}

// streamLines splits text into lines and words and sends each
// token as a stream chunk.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes model (string) which is the model name for chunk
// metadata.
// Takes full (string) which is the full text to split and
// stream.
//
// Returns error when the context is cancelled.
func streamLines(ctx context.Context, model, full string, events chan<- llm_dto.StreamEvent) error {
	lines := strings.Split(full, "\n")

	for li, line := range lines {
		if line == "" {
			nl := "\n"
			if err := sendChunk(ctx, model, nl, events); err != nil {
				return err
			}
			continue
		}

		if err := streamWords(ctx, model, line, events); err != nil {
			return err
		}

		if li < len(lines)-1 {
			nl := "\n"
			if err := sendChunk(ctx, model, nl, events); err != nil {
				return err
			}
		}
	}

	return nil
}

// streamWords sends each word in a line as a separate stream
// chunk, with trailing spaces between words.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes model (string) which is the model name for chunk
// metadata.
// Takes line (string) which is the line to split into words.
//
// Returns error when the context is cancelled.
func streamWords(ctx context.Context, model, line string, events chan<- llm_dto.StreamEvent) error {
	words := strings.Fields(line)
	for wi, word := range words {
		token := word
		if wi < len(words)-1 {
			token += " "
		}
		if err := sendChunk(ctx, model, token, events); err != nil {
			return err
		}
	}
	return nil
}

// sendChunk sends a single token as a stream chunk event.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes model (string) which is the model name for the chunk
// metadata.
// Takes token (string) which is the text token to send.
// Takes events (chan<- llm_dto.StreamEvent) which receives the
// chunk event.
//
// Returns error when the context is cancelled.
func sendChunk(ctx context.Context, model, token string, events chan<- llm_dto.StreamEvent) error {
	chunk := &llm_dto.StreamChunk{
		Model: model,
		Delta: &llm_dto.MessageDelta{
			Content: &token,
		},
	}
	select {
	case events <- llm_dto.NewChunkEvent(chunk):
		return nil
	case <-ctx.Done():
		events <- llm_dto.NewErrorEvent(context.Cause(ctx))
		return context.Cause(ctx)
	}
}

// processToolStream sends a tool call via streaming deltas: one chunk with
// the tool call header (ID, name) and one with the arguments, then a done
// event with FinishReasonToolCalls.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes model (string) which is the model name for responses.
// Takes tool (llm_dto.ToolDefinition) which is the tool to call.
// Takes msgCount (int) which is the number of input messages for usage stats.
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
func processToolStream(ctx context.Context, model string, tool llm_dto.ToolDefinition, msgCount int, events chan<- llm_dto.StreamEvent) {
	defer close(events)
	defer goroutine.RecoverPanic(ctx, "llm.zoltaiProcessToolStream")

	callID := fmt.Sprintf("zoltai-call-%s", tool.Function.Name)
	headerChunk := &llm_dto.StreamChunk{
		Model: model,
		Delta: &llm_dto.MessageDelta{
			ToolCalls: []llm_dto.ToolCallDelta{
				{
					Index: 0,
					ID:    &callID,
					Type:  new("function"),
					Function: &llm_dto.FunctionCallDelta{
						Name:      &tool.Function.Name,
						Arguments: new("{}"),
					},
				},
			},
		},
	}

	select {
	case events <- llm_dto.NewChunkEvent(headerChunk):
	case <-ctx.Done():
		events <- llm_dto.NewErrorEvent(context.Cause(ctx))
		return
	}

	finishReason := llm_dto.FinishReasonToolCalls
	events <- llm_dto.NewDoneEvent(&llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role: llm_dto.RoleAssistant,
					ToolCalls: []llm_dto.ToolCall{
						{
							ID:   callID,
							Type: "function",
							Function: llm_dto.FunctionCall{
								Name:      tool.Function.Name,
								Arguments: "{}",
							},
						},
					},
				},
				FinishReason: finishReason,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     msgCount * estimatedTokensPerMessage,
			CompletionTokens: 5,
			TotalTokens:      msgCount*estimatedTokensPerMessage + 5,
		},
	})
}
