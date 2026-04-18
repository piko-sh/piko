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

package system_memlimit_automemlimit

import (
	"fmt"

	"github.com/KimMachineGun/automemlimit/memlimit"
)

const (
	// defaultRatio is the fraction of the cgroup memory limit used as
	// GOMEMLIMIT when no custom ratio is specified.
	defaultRatio = 0.9
)

// ProviderOption configures the auto memory limit provider.
type ProviderOption func(*providerConfig)

// providerConfig holds the configuration for the auto memory limit provider.
type providerConfig struct {
	// ratio is the fraction of the cgroup memory limit to set as GOMEMLIMIT.
	ratio float64
}

// WithRatio sets the fraction of the cgroup memory limit to use. The
// default is 0.9 (90%).
//
// Takes ratio (float64) which is the fraction between 0.0 and 1.0.
//
// Returns ProviderOption which configures the ratio.
func WithRatio(ratio float64) ProviderOption {
	return func(config *providerConfig) {
		config.ratio = ratio
	}
}

// Provider returns a function that detects the container's cgroup memory
// limit and sets GOMEMLIMIT to a fraction of it. Pass the result to
// piko.WithAutoMemoryLimit.
//
// Takes opts (...ProviderOption) which configure the ratio and behaviour.
//
// Returns func() (int64, error) which detects and applies the memory limit.
func Provider(opts ...ProviderOption) func() (int64, error) {
	config := providerConfig{
		ratio: defaultRatio,
	}
	for _, opt := range opts {
		opt(&config)
	}

	ratio := config.ratio

	return func() (int64, error) {
		limit, err := memlimit.SetGoMemLimitWithOpts(
			memlimit.WithRatio(ratio),
			memlimit.WithProvider(
				memlimit.ApplyFallback(
					memlimit.FromCgroup,
					memlimit.FromSystem,
				),
			),
		)
		if err != nil {
			return 0, fmt.Errorf("auto memory limit detection failed: %w", err)
		}

		return limit, nil
	}
}
