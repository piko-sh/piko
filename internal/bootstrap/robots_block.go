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

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/seo/seo_domain"
)

const (
	// fallbackRobotsTxtBlock is served when rendering the configured block fails. An
	// explicit instruction not to index must never degrade into a permissive response.
	fallbackRobotsTxtBlock = "User-agent: *\nDisallow: /\n"
)

// robotsTxtBlock renders the robots.txt this process serves in place of the stored
// artefact, so a deploy can withhold itself from search without a rebuild.
//
// Takes seoConfigOverride (*config.SEOConfig) which is the container's SEO configuration,
// or nil when the application supplied none.
//
// Returns []byte which holds the bytes to serve, or nil to serve the stored artefact.
func robotsTxtBlock(ctx context.Context, seoConfigOverride *config.SEOConfig) []byte {
	if seoConfigOverride == nil {
		return nil
	}

	robotsConfig := seoConfigOverride.Robots
	if !robotsConfig.NeverIndex && !robotsConfig.PreviewDeployment {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Notice("SEO: this deploy withholds itself from search - /robots.txt serves a site-wide block and every response carries X-Robots-Tag: noindex, nofollow",
		logger_domain.Bool("neverIndex", robotsConfig.NeverIndex),
		logger_domain.Bool("previewDeployment", robotsConfig.PreviewDeployment),
		logger_domain.String("hostname", seoConfigOverride.Sitemap.Hostname))

	content, err := seo_domain.RenderBlockedRobotsTxt(ctx, *seoConfigOverride)
	if err != nil {
		l.Error("SEO: could not render the blocking robots.txt; serving a bare site-wide block",
			logger_domain.Error(err))
		return []byte(fallbackRobotsTxtBlock)
	}

	return content
}
