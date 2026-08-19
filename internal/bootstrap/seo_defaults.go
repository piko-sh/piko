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

package bootstrap

import (
	"context"
	"fmt"
	"net/url"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
)

// applySEOConfigDefaults fills the unset fields of seoConfig from the "default" struct
// tags, so a programmatic WithSEO literal behaves like a configuration the loader
// produced.
//
// Takes seoConfig (config.SEOConfig) which holds the settings the caller supplied.
//
// Returns config.SEOConfig which is the defaulted configuration, or the supplied
// configuration unchanged when the passes fail.
// Returns error when applying the defaults fails.
func applySEOConfigDefaults(ctx context.Context, seoConfig config.SEOConfig) (config.SEOConfig, error) {
	defaulted := config.SEOConfig{}
	_, err := config_domain.Load(ctx, &defaulted, config_domain.LoaderOptions{
		ProgrammaticOverrides: &seoConfig,
		PassOrder: []config_domain.Pass{
			config_domain.PassDefaults,
			config_domain.PassProgrammaticOverrides,
		},
	})
	if err != nil {
		return seoConfig, fmt.Errorf("applying SEO configuration defaults: %w", err)
	}
	return defaulted, nil
}

// isAbsoluteOrigin reports whether value carries both a scheme and a host.
//
// Takes value (string) which is the configured hostname.
//
// Returns bool which is true when value is an absolute origin.
func isAbsoluteOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.IsAbs() && parsed.Host != ""
}
