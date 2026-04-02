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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderError_Unwrap(t *testing.T) {
	sentinel := errors.New("sentinel")
	provErr := &ProviderError{
		Err:        sentinel,
		Provider:   "openai",
		Message:    "something broke",
		StatusCode: 500,
	}

	assert.True(t, errors.Is(provErr, sentinel))
}

func TestProviderError_Unwrap_Nil(t *testing.T) {
	provErr := &ProviderError{
		Err:        nil,
		Provider:   "openai",
		Message:    "no underlying cause",
		StatusCode: 400,
	}

	assert.Nil(t, provErr.Unwrap())
}

func TestProviderError_ErrorsAs(t *testing.T) {
	original := &ProviderError{
		Err:        errors.New("root cause"),
		Provider:   "anthropic",
		Message:    "overloaded",
		StatusCode: 503,
	}
	wrapped := fmt.Errorf("outer: %w", original)

	target, ok := errors.AsType[*ProviderError](wrapped)
	require.True(t, ok)
	assert.Equal(t, "anthropic", target.Provider)
	assert.Equal(t, "overloaded", target.Message)
	assert.Equal(t, 503, target.StatusCode)
}

func TestProviderError_Error_Format(t *testing.T) {
	provErr := &ProviderError{
		Provider:   "openai",
		Message:    "rate limited",
		StatusCode: 429,
	}

	assert.Equal(t, "provider openai: 429 rate limited", provErr.Error())
}

func TestProviderError_IsRetryable_AllCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "408 Request Timeout", statusCode: 408, want: true},
		{name: "409 Conflict", statusCode: 409, want: true},
		{name: "425 Too Early", statusCode: 425, want: true},
		{name: "429 Too Many Requests", statusCode: 429, want: true},
		{name: "500 Internal Server Error", statusCode: 500, want: true},
		{name: "502 Bad Gateway", statusCode: 502, want: true},
		{name: "503 Service Unavailable", statusCode: 503, want: true},
		{name: "504 Gateway Timeout", statusCode: 504, want: true},
		{name: "400 Bad Request", statusCode: 400, want: false},
		{name: "401 Unauthorised", statusCode: 401, want: false},
		{name: "403 Forbidden", statusCode: 403, want: false},
		{name: "404 Not Found", statusCode: 404, want: false},
		{name: "422 Unprocessable Entity", statusCode: 422, want: false},
		{name: "0 zero value", statusCode: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provErr := &ProviderError{
				Provider:   "test",
				Message:    "test",
				StatusCode: tt.statusCode,
			}

			assert.Equal(t, tt.want, provErr.IsRetryable())
		})
	}
}
