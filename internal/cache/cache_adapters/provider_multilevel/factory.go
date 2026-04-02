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

package provider_multilevel

import (
	"time"
)

// Config holds the settings for the multi-level cache provider.
type Config struct {
	// L1ProviderName is the name of the level 1 (fastest) cache provider.
	L1ProviderName string `mapstructure:"l1_provider"`

	// L2ProviderName is the name of the level 2 cache provider.
	L2ProviderName string `mapstructure:"l2_provider"`

	// MaxConsecutiveFailures is the number of failures before the circuit breaker
	// opens for the L2 provider.
	MaxConsecutiveFailures int `mapstructure:"l2_max_failures"`

	// OpenStateTimeout is how long the circuit breaker stays open before
	// trying again. A value of 0 uses the default timeout.
	OpenStateTimeout time.Duration `mapstructure:"l2_open_timeout"`
}
