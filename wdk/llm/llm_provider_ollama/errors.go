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
	"errors"

	"github.com/ollama/ollama/api"

	"piko.sh/piko/internal/llm/llm_domain"
)

// wrapError converts known Ollama SDK error types into *llm_domain.ProviderError
// so that callers can inspect the HTTP status code and trigger retry logic.
//
// Takes err (error) which is the error returned by the Ollama client.
//
// Returns error which is a *llm_domain.ProviderError when the underlying error
// is a recognised Ollama type, or the original error otherwise.
func wrapError(err error) error {
	if statusErr, ok := errors.AsType[api.StatusError](err); ok {
		return &llm_domain.ProviderError{
			Provider:   "ollama",
			StatusCode: statusErr.StatusCode,
			Message:    statusErr.ErrorMessage,
			Err:        err,
		}
	}

	if authErr, ok := errors.AsType[api.AuthorizationError](err); ok {
		return &llm_domain.ProviderError{
			Provider:   "ollama",
			StatusCode: authErr.StatusCode,
			Message:    err.Error(),
			Err:        err,
		}
	}

	return err
}
