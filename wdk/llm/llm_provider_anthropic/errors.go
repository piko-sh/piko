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
	"errors"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"piko.sh/piko/internal/llm/llm_domain"
)

// wrapError wraps an Anthropic SDK error as a *llm_domain.ProviderError.
//
// Preserves the HTTP status code for retry classification. When the
// upstream response carries a Retry-After header, its parsed value is
// propagated so the retry executor can honour the server hint. Non-API
// errors are returned unchanged.
//
// Takes err (error) which is the error to wrap.
//
// Returns error which is a *llm_domain.ProviderError if the underlying error
// is an *anthropic.Error, or the original error otherwise.
func wrapError(err error) error {
	if apiErr, ok := errors.AsType[*anthropic.Error](err); ok {
		providerErr := &llm_domain.ProviderError{
			Provider:   providerNameAnthropic,
			StatusCode: apiErr.StatusCode,
			Message:    err.Error(),
			Err:        err,
		}
		if apiErr.Response != nil {
			providerErr.RetryAfter = llm_domain.ParseRetryAfter(
				apiErr.Response.Header.Get("Retry-After"), time.Now(),
			)
		}
		return providerErr
	}
	return err
}
