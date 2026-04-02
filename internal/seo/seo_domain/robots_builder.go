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

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

// robotsBuilder creates robots.txt content based on settings.
type robotsBuilder struct {
	// config holds the settings for robots.txt generation.
	config config.RobotsConfig
}

// Build creates the full robots.txt content using all set rules.
//
// Takes sitemapURL (string) which specifies the sitemap location to include.
//
// Returns []byte which contains the rendered robots.txt content.
// Returns error when content creation fails.
func (b *robotsBuilder) Build(ctx context.Context, sitemapURL string) ([]byte, error) {
	_, l := logger_domain.From(ctx, log)
	content := &seo_dto.RobotsTxtContent{
		Groups: []seo_dto.RobotGroup{},
	}

	content.Groups = append(content.Groups, b.generateBaseRules())

	if b.config.BlockAiBots {
		content.Groups = append(content.Groups, b.generateAIBotRules())
		l.Trace("Added AI bot blocking rules to robots.txt")
	}

	if b.config.BlockNonSeoBots {
		content.Groups = append(content.Groups, b.generateNonSEOBotRules())
		l.Trace("Added non-SEO bot blocking rules to robots.txt")
	}

	for _, customRule := range b.config.CustomRules {
		content.Groups = append(content.Groups, seo_dto.RobotGroup{
			UserAgents: customRule.UserAgents,
			Disallow:   customRule.Disallow,
			Allow:      customRule.Allow,
		})
	}

	textContent := content.RenderToString(sitemapURL)

	l.Trace("Generated robots.txt", logger_domain.Int("rule_groups", len(content.Groups)))
	return []byte(textContent), nil
}

// generateBaseRules creates the default rules that allow all bots to access
// all paths.
//
// Returns seo_dto.RobotGroup which permits all user agents to access all paths.
func (*robotsBuilder) generateBaseRules() seo_dto.RobotGroup {
	return seo_dto.RobotGroup{
		UserAgents: []string{"*"},
		Disallow:   []string{},
		Allow:      []string{"/"},
	}
}

// generateAIBotRules creates rules that block known AI crawler bots.
//
// Returns seo_dto.RobotGroup which blocks all paths for AI bot user agents.
func (*robotsBuilder) generateAIBotRules() seo_dto.RobotGroup {
	return seo_dto.RobotGroup{
		UserAgents: seo_dto.AIBots,
		Disallow:   []string{"/"},
		Allow:      []string{},
	}
}

// generateNonSEOBotRules creates blocking rules for known web scrapers and
// non-SEO bots.
//
// Returns a RobotGroup that disallows all paths for non-SEO bot user agents.
func (*robotsBuilder) generateNonSEOBotRules() seo_dto.RobotGroup {
	return seo_dto.RobotGroup{
		UserAgents: seo_dto.NonSEOBots,
		Disallow:   []string{"/"},
		Allow:      []string{},
	}
}

// newRobotsBuilder creates a new robotsBuilder with the given settings.
//
// Takes robotsConfig (config.RobotsConfig) which specifies the
// robots.txt settings.
//
// Returns *robotsBuilder which is ready for use.
func newRobotsBuilder(robotsConfig config.RobotsConfig) *robotsBuilder {
	return &robotsBuilder{
		config: robotsConfig,
	}
}
