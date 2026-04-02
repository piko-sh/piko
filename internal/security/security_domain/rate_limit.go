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

package security_domain

import (
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

// RateLimitService provides rate limiting functionality for controlling
// request rates.
type RateLimitService interface {
	// CheckLimit checks whether the given key has exceeded its rate limit.
	//
	// Takes key (string) which identifies the client or resource being limited.
	// Takes limit (int) which specifies the maximum number of requests allowed.
	// Takes window (time.Duration) which defines the time period for the limit.
	//
	// Returns ratelimiter_dto.Result which contains allowed status, remaining
	// count, and reset time.
	// Returns error when the limit check fails.
	CheckLimit(key string, limit int, window time.Duration) (ratelimiter_dto.Result, error)
}
