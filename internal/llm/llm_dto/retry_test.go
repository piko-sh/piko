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

package llm_dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryPolicy(t *testing.T) {
	t.Parallel()

	p := DefaultRetryPolicy()

	assert.Equal(t, DefaultRetryMaxRetries, p.MaxRetries)
	assert.Equal(t, DefaultRetryInitialBackoff, p.InitialBackoff)
	assert.Equal(t, DefaultRetryMaxBackoff, p.MaxBackoff)
	assert.Equal(t, DefaultRetryBackoffMultiplier, p.BackoffMultiplier)
	assert.Equal(t, DefaultRetryJitterFraction, p.JitterFraction)
	assert.Nil(t, p.OnRetry)

	assert.Equal(t, 3, p.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, p.InitialBackoff)
	assert.Equal(t, 8*time.Second, p.MaxBackoff)
	assert.InDelta(t, 2.0, p.BackoffMultiplier, 0.001)
	assert.InDelta(t, 0.1, p.JitterFraction, 0.001)
}

func TestNoRetryPolicy(t *testing.T) {
	t.Parallel()

	p := NoRetryPolicy()

	assert.Equal(t, 0, p.MaxRetries)
	assert.Nil(t, p.OnRetry)
}
