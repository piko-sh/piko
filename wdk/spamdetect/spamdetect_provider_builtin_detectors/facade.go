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

package spamdetect_provider_builtin_detectors

import (
	"context"
	"time"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_adapters/builtin_detectors"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
)

// Config holds configuration for the built-in detectors.
type Config = builtin_detectors.Config

// RegisterDefaults creates and registers all five built-in detectors with
// the service.
//
// Takes service (spamdetect_domain.SpamDetectServicePort) which receives
// the detectors.
// Takes config (Config) which configures the built-in detectors.
//
// Returns error when a detector fails to create or register.
func RegisterDefaults(ctx context.Context, service spamdetect_domain.SpamDetectServicePort, config Config) error {
	return builtin_detectors.RegisterDefaults(ctx, service, config)
}

// NewHoneypotDetector creates a honeypot detector.
//
// Returns *builtin_detectors.HoneypotDetector which is the configured detector.
func NewHoneypotDetector() *builtin_detectors.HoneypotDetector {
	return builtin_detectors.NewHoneypotDetector()
}

// NewGibberishDetector creates a gibberish detector.
//
// Takes threshold (float64) which is the gibberish ratio threshold.
// Takes bigramAnalysers ([]linguistics_domain.BigramAnalyserPort) which
// provide language-aware analysis. Pass nil for the built-in English
// fallback only.
//
// Returns *builtin_detectors.GibberishDetector which is the configured detector.
func NewGibberishDetector(threshold float64, bigramAnalysers []linguistics_domain.BigramAnalyserPort) *builtin_detectors.GibberishDetector {
	return builtin_detectors.NewGibberishDetector(threshold, bigramAnalysers)
}

// NewLinkDensityDetector creates a link density detector.
//
// Takes maxLinks (int) which is the maximum allowed link count.
//
// Returns *builtin_detectors.LinkDensityDetector which is the configured
// detector.
func NewLinkDensityDetector(maxLinks int) *builtin_detectors.LinkDensityDetector {
	return builtin_detectors.NewLinkDensityDetector(maxLinks)
}

// NewBlocklistDetector creates a blocklist detector.
//
// Takes patterns ([]string) which are the regex patterns to match against.
//
// Returns *builtin_detectors.BlocklistDetector which is the configured detector.
// Returns error when a pattern fails to compile.
func NewBlocklistDetector(patterns []string) (*builtin_detectors.BlocklistDetector, error) {
	return builtin_detectors.NewBlocklistDetector(patterns)
}

// NewTimingDetector creates a timing detector.
//
// Takes minDuration (time.Duration) which is the minimum expected submission
// duration.
//
// Returns *builtin_detectors.TimingDetector which is the configured detector.
func NewTimingDetector(minDuration time.Duration) *builtin_detectors.TimingDetector {
	return builtin_detectors.NewTimingDetector(minDuration)
}
