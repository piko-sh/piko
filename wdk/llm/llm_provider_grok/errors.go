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

package llm_provider_grok

import (
	"errors"

	"piko.sh/piko/internal/llm/llm_domain"
)

// rewrapError converts ProviderError instances from "openai" to "grok".
// Non-ProviderError errors pass through unchanged.
//
// Takes err (error) which is the error to inspect and potentially rewrap.
//
// Returns error which is either a rewrapped *llm_domain.ProviderError with
// Provider set to "grok", or the original error.
func rewrapError(err error) error {
	if err == nil {
		return nil
	}

	if pe, ok := errors.AsType[*llm_domain.ProviderError](err); ok {
		return &llm_domain.ProviderError{
			Provider:   "grok",
			StatusCode: pe.StatusCode,
			Message:    pe.Message,
			Err:        pe.Err,
		}
	}

	return err
}
