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

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/spamdetect/spamdetect_adapters/builtin_detectors"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
)

// Config holds configuration for the built-in detectors.
type Config = builtin_detectors.Config

// RepetitionOption configures the repetition detector at construction
// time.
type RepetitionOption = builtin_detectors.RepetitionOption

// HoneypotDetector is the built-in honeypot detector.
type HoneypotDetector = builtin_detectors.HoneypotDetector

// GibberishDetector is the built-in gibberish detector.
type GibberishDetector = builtin_detectors.GibberishDetector

// LinkDensityDetector is the built-in link density detector.
type LinkDensityDetector = builtin_detectors.LinkDensityDetector

// BlocklistDetector is the built-in blocklist detector.
type BlocklistDetector = builtin_detectors.BlocklistDetector

// TimingDetector is the built-in timing detector.
type TimingDetector = builtin_detectors.TimingDetector

// RepetitionDetector is the built-in repetition detector.
type RepetitionDetector = builtin_detectors.RepetitionDetector

// RegisterDefaults creates and registers all six built-in detectors
// with the service.
//
// Takes ctx (context.Context) which is the caller context.
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
// Returns *HoneypotDetector which is the configured detector.
func NewHoneypotDetector() *HoneypotDetector {
	return builtin_detectors.NewHoneypotDetector()
}

// NewGibberishDetector creates a gibberish detector.
//
// Takes threshold (float64) which is the gibberish ratio threshold.
// Takes bigramAnalysers ([]linguistics_domain.BigramAnalyserPort) which
// provide language-aware analysis. Pass nil for the built-in English
// fallback only.
//
// Returns *GibberishDetector which is the configured detector.
func NewGibberishDetector(threshold float64, bigramAnalysers []linguistics_domain.BigramAnalyserPort) *GibberishDetector {
	return builtin_detectors.NewGibberishDetector(threshold, bigramAnalysers)
}

// NewLinkDensityDetector creates a link density detector.
//
// Takes maxLinks (int) which is the maximum allowed link count.
//
// Returns *LinkDensityDetector which is the configured detector.
func NewLinkDensityDetector(maxLinks int) *LinkDensityDetector {
	return builtin_detectors.NewLinkDensityDetector(maxLinks)
}

// NewBlocklistDetector creates a blocklist detector.
//
// Takes patterns ([]string) which are the regex patterns to match against.
//
// Returns *BlocklistDetector which is the configured detector.
// Returns error when a pattern fails to compile.
func NewBlocklistDetector(patterns []string) (*BlocklistDetector, error) {
	return builtin_detectors.NewBlocklistDetector(patterns)
}

// NewTimingDetector creates a timing detector.
//
// Takes minDuration (time.Duration) which is the minimum expected submission
// duration.
//
// Returns *TimingDetector which is the configured detector.
func NewTimingDetector(minDuration time.Duration) *TimingDetector {
	return builtin_detectors.NewTimingDetector(minDuration)
}

// RepetitionEntry is the cache value type for repetition tracking.
type RepetitionEntry = builtin_detectors.RepetitionEntry

// NewRepetitionDetector creates a repetition detector.
//
// Takes cache (cache_domain.Cache[string, RepetitionEntry]) which
// stores content hashes. Pass nil to disable repetition detection.
// Takes ttl (time.Duration) which is the tracking window.
// Takes ipScoped (bool) which scopes tracking per client IP when true.
// Takes opts (...RepetitionOption) which override defaults (e.g. clock).
//
// Returns *RepetitionDetector which is the configured detector.
func NewRepetitionDetector(
	cache cache_domain.Cache[string, RepetitionEntry],
	ttl time.Duration,
	ipScoped bool,
	opts ...RepetitionOption,
) *RepetitionDetector {
	return builtin_detectors.NewRepetitionDetector(cache, ttl, ipScoped, opts...)
}

// WithRepetitionClock sets the clock source used for repetition
// FirstSeen timestamps. Tests inject a mock clock for deterministic
// behaviour.
var WithRepetitionClock = builtin_detectors.WithRepetitionClock
