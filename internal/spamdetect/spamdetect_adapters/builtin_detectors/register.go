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

package builtin_detectors

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
)

// RegisterDefaults creates and registers all six built-in detectors with
// the service.
//
// Takes service (spamdetect_domain.SpamDetectServicePort) which receives
// the detectors.
// Takes config (Config) which configures the built-in detectors.
//
// Returns error when a detector fails to create or register.
func RegisterDefaults(ctx context.Context, service spamdetect_domain.SpamDetectServicePort, config Config) error {
	config.applyDefaults()

	blocklistDetector, err := NewBlocklistDetector(config.BlocklistPatterns)
	if err != nil {
		return fmt.Errorf("creating blocklist detector: %w", err)
	}

	detectors := []struct {
		detector spamdetect_domain.Detector
		name     string
	}{
		{NewHoneypotDetector(), "honeypot"},
		{NewGibberishDetector(config.GibberishThreshold, config.BigramAnalysers), "gibberish"},
		{NewLinkDensityDetector(config.LinkDensityMaxLinks), "link_density"},
		{blocklistDetector, "blocklist"},
		{NewTimingDetector(config.TimingMinDuration), "timing"},
		{NewRepetitionDetector(config.RepetitionCache, config.RepetitionTTL, resolveIPScoped(config.RepetitionIPScoped)), "repetition"},
	}

	for _, entry := range detectors {
		if err := service.RegisterDetector(ctx, entry.name, entry.detector); err != nil {
			return fmt.Errorf("registering built-in detector %q: %w", entry.name, err)
		}
	}

	return nil
}

// resolveIPScoped returns the bool value or true as default.
//
// Takes value (*bool) which is the optional override.
//
// Returns bool which is the resolved value, defaulting to true.
func resolveIPScoped(value *bool) bool {
	if value != nil {
		return *value
	}
	return true
}
