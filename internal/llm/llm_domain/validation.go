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
	"fmt"

	"piko.sh/piko/internal/llm/llm_dto"
)

// ValidateRequest checks that a completion request is valid.
//
// Takes request (*llm_dto.CompletionRequest) which is the request to check.
//
// Returns error when the request fails validation.
func ValidateRequest(request *llm_dto.CompletionRequest) error {
	if request.Model == "" {
		return ErrEmptyModel
	}
	if len(request.Messages) == 0 {
		return ErrEmptyMessages
	}
	if request.Temperature != nil && (*request.Temperature < 0 || *request.Temperature > 2) {
		return ErrInvalidTemperature
	}
	if request.TopP != nil && (*request.TopP < 0 || *request.TopP > 1) {
		return ErrInvalidTopP
	}
	if request.MaxTokens != nil && *request.MaxTokens <= 0 {
		return ErrInvalidMaxTokens
	}
	return nil
}

// ValidateRequestForProvider checks that a request is valid for a provider.
//
// Takes request (*llm_dto.CompletionRequest) which is the request to validate.
// Takes provider (LLMProviderPort) which is the target provider.
//
// Returns error when the request is incompatible with the provider.
func ValidateRequestForProvider(request *llm_dto.CompletionRequest, provider LLMProviderPort) error {
	if err := ValidateRequest(request); err != nil {
		return fmt.Errorf("validating request: %w", err)
	}
	if err := validateProviderCapabilities(request, provider); err != nil {
		return err
	}
	return validateMessageNames(request, provider)
}

// validateProviderCapabilities checks that the request's features are
// supported by the provider.
//
// Takes request (*llm_dto.CompletionRequest) which is the request to check.
// Takes provider (LLMProviderPort) which is the provider to check against.
//
// Returns error when the request uses a feature the provider does not support.
func validateProviderCapabilities(request *llm_dto.CompletionRequest, provider LLMProviderPort) error {
	if request.Stream && !provider.SupportsStreaming() {
		return ErrStreamingNotSupported
	}
	if len(request.Tools) > 0 && !provider.SupportsTools() {
		return ErrToolsNotSupported
	}
	if request.ResponseFormat != nil && request.ResponseFormat.Type == llm_dto.ResponseFormatJSONSchema && !provider.SupportsStructuredOutput() {
		return ErrStructuredOutputNotSupported
	}
	if (request.FrequencyPenalty != nil || request.PresencePenalty != nil) && !provider.SupportsPenalties() {
		return ErrPenaltiesNotSupported
	}
	if request.Seed != nil && !provider.SupportsSeed() {
		return ErrSeedNotSupported
	}
	if request.ParallelToolCalls != nil && !provider.SupportsParallelToolCalls() {
		return ErrParallelToolCallsNotSupported
	}
	return nil
}

// validateMessageNames checks that no messages use the Name field when the
// provider does not support it.
//
// Takes request (*llm_dto.CompletionRequest) which contains the messages to check.
// Takes provider (LLMProviderPort) which is the provider to check against.
//
// Returns error when a message has a Name but the provider does not support it.
func validateMessageNames(request *llm_dto.CompletionRequest, provider LLMProviderPort) error {
	if provider.SupportsMessageName() {
		return nil
	}
	for _, message := range request.Messages {
		if message.Name != nil {
			return ErrMessageNameNotSupported
		}
	}
	return nil
}
