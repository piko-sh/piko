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
	"errors"

	"github.com/openai/openai-go/v3"

	"piko.sh/piko/internal/llm/llm_domain"
)

// wrapError wraps an OpenAI SDK error as a *llm_domain.ProviderError when the
// underlying error is an *openai.Error, preserving the HTTP status code. For
// non-API errors the original error is returned unchanged.
//
// Takes err (error) which is the error to inspect and potentially wrap.
//
// Returns error which is either a wrapped *llm_domain.ProviderError or the
// original error.
func wrapError(err error) error {
	if apiErr, ok := errors.AsType[*openai.Error](err); ok {
		return &llm_domain.ProviderError{
			Provider:   "openai",
			StatusCode: apiErr.StatusCode,
			Message:    apiErr.Message,
			Err:        err,
		}
	}
	return err
}
