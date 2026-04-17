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
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

const (
	// defaultGibberishThreshold is the default gibberish ratio threshold.
	defaultGibberishThreshold = 0.6

	// defaultLinkDensityMaxLinks is the default maximum allowed link count.
	defaultLinkDensityMaxLinks = 3

	// defaultTimingMinDuration is the default minimum form submission time.
	defaultTimingMinDuration = 2 * time.Second

	// maxBlocklistPatterns is the upper bound on blocklist regex patterns.
	maxBlocklistPatterns = 1024

	// maxAnalyseFieldLength is the maximum field length processed during analysis.
	maxAnalyseFieldLength = 4096

	// detectorSpamThreshold is the per-detector score above which the
	// detector marks a submission as spam.
	detectorSpamThreshold = 0.5
)

// Config configures the built-in detectors registered by RegisterDefaults.
type Config struct {
	// RepetitionCache is the cache instance for tracking repeated
	// submissions. When nil, repetition detection is disabled.
	RepetitionCache cache_domain.Cache[string, repetitionEntry]

	// RepetitionIPScoped scopes repetition tracking per client IP when
	// true. Default true.
	RepetitionIPScoped *bool

	// BigramAnalysers holds language-specific bigram analysers injected
	// via DI from the linguistics module. When empty, the built-in
	// English bigram fallback is used.
	BigramAnalysers []linguistics_domain.BigramAnalyserPort

	// BlocklistPatterns is a list of regex patterns for the blocklist
	// detector. Limited to 1024 patterns.
	BlocklistPatterns []string

	// RepetitionTTL is the time window for tracking repeated submissions.
	// Default 10 minutes.
	RepetitionTTL time.Duration

	// TimingMinDuration is the minimum expected time between form load
	// and submit. Default 2s.
	TimingMinDuration time.Duration

	// GibberishThreshold is the entropy ratio above which text is flagged
	// as gibberish. Default 0.6.
	GibberishThreshold float64

	// LinkDensityMaxLinks is the maximum number of links allowed before
	// scoring as spam. Default 3.
	LinkDensityMaxLinks int
}

// applyDefaults fills in zero-valued fields with sensible defaults.
func (c *Config) applyDefaults() {
	if c.GibberishThreshold <= 0 {
		c.GibberishThreshold = defaultGibberishThreshold
	}
	if c.LinkDensityMaxLinks <= 0 {
		c.LinkDensityMaxLinks = defaultLinkDensityMaxLinks
	}
	if c.TimingMinDuration <= 0 {
		c.TimingMinDuration = defaultTimingMinDuration
	}
}
