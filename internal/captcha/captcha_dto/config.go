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

package captcha_dto

const (
	// defaultScoreThreshold is the default minimum score for score-based
	// captcha providers. A score of 0.5 balances false positives and false
	// negatives for most applications.
	defaultScoreThreshold = 0.5

	// defaultVerifyRateLimit is the maximum number of captcha verification
	// calls per IP per minute. Protects provider APIs from token-flooding.
	defaultVerifyRateLimit = 20

	// defaultChallengeRateLimit is the maximum number of HMAC challenge
	// token generation calls per IP per minute.
	defaultChallengeRateLimit = 30
)

// ServiceConfig holds configuration for the captcha service.
type ServiceConfig struct {
	// DefaultScoreThreshold is the minimum score (0.0-1.0) required for
	// score-based providers like reCAPTCHA v3. Actions can override this
	// with their own threshold via CaptchaConfig.ScoreThreshold.
	DefaultScoreThreshold float64

	// VerifyRateLimit is the maximum number of verification calls per IP
	// per minute. Zero disables rate limiting.
	VerifyRateLimit int

	// ChallengeRateLimit is the maximum number of challenge token
	// generation calls per IP per minute. Zero disables rate limiting.
	ChallengeRateLimit int
}

// DefaultServiceConfig returns a service config with sensible defaults.
//
// Returns *ServiceConfig which is set up with a 0.5 score threshold and
// rate limits of 20 verifications and 30 challenges per IP per minute.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultScoreThreshold: defaultScoreThreshold,
		VerifyRateLimit:       defaultVerifyRateLimit,
		ChallengeRateLimit:    defaultChallengeRateLimit,
	}
}
