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
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/logger"
)

// structuredOutputToolName is the name of the synthetic tool used for
// structured output.
const structuredOutputToolName = "__piko_structured_output__"

// completeWithStructuredOutput handles structured output requests by
// translating them to tool use. Claude does not support
// response_format.json_schema natively, so this creates a synthetic tool with
// the schema and forces the model to call it.
//
// Takes request (*llm_dto.CompletionRequest) which contains the structured output
// schema and request parameters.
// Takes model (string) which specifies the Anthropic model to use.
//
// Returns *llm_dto.CompletionResponse which contains the structured output
// extracted from the tool call result.
// Returns error when the API request fails or response conversion fails.
func (p *anthropicProvider) completeWithStructuredOutput(ctx context.Context, request *llm_dto.CompletionRequest, model string) (*llm_dto.CompletionResponse, error) {
	defer goroutine.RecoverPanic(ctx, "llm.anthropicProvider.completeWithStructuredOutput")

	ctx, l := logger.From(ctx, log)

	schema := request.ResponseFormat.JSONSchema
	syntheticTool := llm_dto.ToolDefinition{
		Type: "function",
		Function: llm_dto.FunctionDefinition{
			Name:        structuredOutputToolName,
			Description: schema.Description,
			Parameters:  &schema.Schema,
		},
	}

	modifiedReq := *request
	modifiedReq.Tools = append([]llm_dto.ToolDefinition{syntheticTool}, request.Tools...)
	modifiedReq.ToolChoice = llm_dto.ToolChoiceSpecific(structuredOutputToolName)
	modifiedReq.ResponseFormat = nil

	params := p.buildMessageParams(&modifiedReq, model)

	l.Debug("Sending Anthropic structured output request via tool use",
		logger.String("model", model),
		logger.String("schema_name", schema.Name),
	)

	message, err := p.client.Messages.New(ctx, params)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("anthropic structured output completion failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "anthropic structured output rejected")
	}

	return p.convertStructuredOutputResponse(message, model)
}

// convertStructuredOutputResponse converts a tool-use response back to a
// structured output response format.
//
// Takes message (*anthropic.Message) which contains the raw Anthropic response.
// Takes model (string) which identifies the model used for the completion.
//
// Returns *llm_dto.CompletionResponse which contains the converted response
// with JSON content extracted from the synthetic tool use block.
// Returns error when the structured output cannot be marshalled to JSON.
func (p *anthropicProvider) convertStructuredOutputResponse(message *anthropic.Message, model string) (*llm_dto.CompletionResponse, error) {
	for i := range message.Content {
		block := &message.Content[i]
		if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			if toolUse.Name == structuredOutputToolName {
				jsonContent, err := json.Marshal(toolUse.Input)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal structured output: %w", err)
				}

				response := &llm_dto.CompletionResponse{
					ID:    message.ID,
					Model: model,
					Choices: []llm_dto.Choice{
						{
							Index: 0,
							Message: llm_dto.Message{
								Role:    llm_dto.RoleAssistant,
								Content: string(jsonContent),
							},
							FinishReason: llm_dto.FinishReasonStop,
						},
					},
				}

				if message.Usage.InputTokens > 0 || message.Usage.OutputTokens > 0 {
					response.Usage = &llm_dto.Usage{
						PromptTokens:     int(message.Usage.InputTokens),
						CompletionTokens: int(message.Usage.OutputTokens),
						TotalTokens:      int(message.Usage.InputTokens + message.Usage.OutputTokens),
						CachedTokens:     int(message.Usage.CacheReadInputTokens),
					}
				}

				return response, nil
			}
		}
	}

	return p.convertResponse(message, model), nil
}
