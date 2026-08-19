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

package seo_domain

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/config"
)

// sitemapIndexURL builds the absolute sitemap location advertised in robots.txt, so
// generation and serving cannot disagree about how the hostname and path are joined.
//
// Takes hostname (string) which is the configured sitemap hostname.
//
// Returns string which is the absolute sitemap URL, or empty when hostname is empty.
func sitemapIndexURL(hostname string) string {
	if hostname == "" {
		return ""
	}
	return strings.TrimSuffix(hostname, "/") + "/sitemap.xml"
}

// RenderBlockedRobotsTxt renders the robots.txt a build would have stored had it blocked
// every crawler, so a serving process can present a block without regenerating the stored
// artefact or writing to the registry.
//
// Takes seoConfig (config.SEOConfig) which carries the robots rules and the sitemap
// hostname.
//
// Returns []byte which is the rendered robots.txt.
// Returns error when rendering the content fails.
func RenderBlockedRobotsTxt(ctx context.Context, seoConfig config.SEOConfig) ([]byte, error) {
	content, err := newRobotsBuilder(seoConfig.Robots, true).
		Build(ctx, sitemapIndexURL(seoConfig.Sitemap.Hostname))
	if err != nil {
		return nil, fmt.Errorf("rendering blocked robots.txt: %w", err)
	}
	return content, nil
}
