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

package ratelimiter_dto

import "testing"

func TestTokenBucketConfig_MaxTokens(t *testing.T) {

	c := TokenBucketConfig{Rate: 10, Burst: 20}
	if got := c.MaxTokens(); got != 20 {
		t.Errorf("MaxTokens() = %f, want 20", got)
	}

	c2 := TokenBucketConfig{Rate: 15}
	if got := c2.MaxTokens(); got != 15 {
		t.Errorf("MaxTokens() = %f, want 15", got)
	}
}
