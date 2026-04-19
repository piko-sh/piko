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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicProvider_Close_Idempotent(t *testing.T) {
	p := newTestProvider(t)

	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
}

func TestAnthropicProvider_Close_HasCloseContext(t *testing.T) {
	p := newTestProvider(t)

	require.NotNil(t, p.closeContext, "close context should be initialised")
	require.NotNil(t, p.closeCancel, "close cancel should be initialised")

	select {
	case <-p.closeContext.Done():
		t.Fatal("close context should not be cancelled before Close")
	default:
	}

	require.NoError(t, p.Close(context.Background()))

	select {
	case <-p.closeContext.Done():
	default:
		t.Fatal("close context should be cancelled after Close")
	}
}

func TestAnthropicProvider_ClientHasTimeout(t *testing.T) {
	p := newTestProvider(t)

	require.NotNil(t, p.httpClient, "underlying HTTP client should be set")
	assert.Greater(t, p.httpClient.Timeout, time.Duration(0),
		"anthropic HTTP client should carry a top-level timeout")
	assert.Equal(t, httpClientTimeout, p.httpClient.Timeout,
		"timeout should match the configured constant")
}
