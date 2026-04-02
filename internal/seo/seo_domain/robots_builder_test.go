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
	"slices"
	"strings"
	"testing"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestRobotsBuilder_Build_BasicPermissiveRules(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     false,
		BlockNonSeoBots: false,
		CustomRules:     []config.RobotsRuleGroup{},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	if !strings.Contains(content, "User-agent: *") {
		t.Error("Expected basic 'User-agent: *' rule")
	}
	if !strings.Contains(content, "Allow: /") {
		t.Error("Expected basic 'Allow: /' rule")
	}

	if !strings.Contains(content, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("Expected sitemap directive")
	}
}

func TestRobotsBuilder_Build_BlockAIBots(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     true,
		BlockNonSeoBots: false,
		CustomRules:     []config.RobotsRuleGroup{},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	aiBotsToCheck := []string{
		"GPTBot",
		"ChatGPT-User",
		"CCBot",
		"anthropic-ai",
		"ClaudeBot",
		"Google-Extended",
	}

	for _, bot := range aiBotsToCheck {
		expectedRule := "User-agent: " + bot
		if !strings.Contains(content, expectedRule) {
			t.Errorf("Expected to find blocking rule for %s", bot)
		}
	}

	if !strings.Contains(content, "Disallow: /") {
		t.Error("Expected 'Disallow: /' rule for blocking bots")
	}
}

func TestRobotsBuilder_Build_BlockNonSEOBots(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     false,
		BlockNonSeoBots: true,
		CustomRules:     []config.RobotsRuleGroup{},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	nonSEOBotsToCheck := []string{
		"AhrefsBot",
		"SemrushBot",
		"DotBot",
		"Baiduspider",
	}

	for _, bot := range nonSEOBotsToCheck {
		expectedRule := "User-agent: " + bot
		if !strings.Contains(content, expectedRule) {
			t.Errorf("Expected to find blocking rule for %s", bot)
		}
	}
}

func TestRobotsBuilder_Build_BlockBothAIAndNonSEO(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     true,
		BlockNonSeoBots: true,
		CustomRules:     []config.RobotsRuleGroup{},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	if !strings.Contains(content, "GPTBot") {
		t.Error("Expected AI bot GPTBot to be blocked")
	}
	if !strings.Contains(content, "AhrefsBot") {
		t.Error("Expected non-SEO bot AhrefsBot to be blocked")
	}

	if !strings.Contains(content, "User-agent: *") {
		t.Error("Expected basic 'User-agent: *' rule for good bots")
	}
}

func TestRobotsBuilder_Build_CustomRules(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     false,
		BlockNonSeoBots: false,
		CustomRules: []config.RobotsRuleGroup{
			{
				UserAgents: []string{"CustomBot"},
				Disallow:   []string{"/admin/", "/private/"},
				Allow:      []string{"/admin/public/"},
			},
			{
				UserAgents: []string{"AnotherBot"},
				Disallow:   []string{"/"},
			},
		},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	if !strings.Contains(content, "User-agent: CustomBot") {
		t.Error("Expected custom rule for CustomBot")
	}
	if !strings.Contains(content, "Disallow: /admin/") {
		t.Error("Expected Disallow rule for /admin/")
	}
	if !strings.Contains(content, "Disallow: /private/") {
		t.Error("Expected Disallow rule for /private/")
	}
	if !strings.Contains(content, "Allow: /admin/public/") {
		t.Error("Expected Allow rule for /admin/public/")
	}

	if !strings.Contains(content, "User-agent: AnotherBot") {
		t.Error("Expected custom rule for AnotherBot")
	}
}

func TestRobotsBuilder_Build_CustomRulesWithAllowAndDisallow(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     false,
		BlockNonSeoBots: false,
		CustomRules: []config.RobotsRuleGroup{
			{
				UserAgents: []string{"CustomBot"},
				Disallow:   []string{"/admin"},
				Allow:      []string{"/admin/public"},
			},
		},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	if !strings.Contains(content, "User-agent: CustomBot") {
		t.Error("Expected custom rule for CustomBot")
	}
	if !strings.Contains(content, "Disallow: /admin") {
		t.Error("Expected Disallow: /admin")
	}
	if !strings.Contains(content, "Allow: /admin/public") {
		t.Error("Expected Allow: /admin/public")
	}
}

func TestRobotsBuilder_Build_NoSitemap(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     false,
		BlockNonSeoBots: false,
		CustomRules:     []config.RobotsRuleGroup{},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := ""

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	if strings.Contains(content, "Sitemap:") {
		t.Error("Expected no sitemap directive when sitemapURL is empty")
	}
}

func TestRobotsBuilder_Build_RuleOrder(t *testing.T) {
	robotsConfig := config.RobotsConfig{
		BlockAiBots:     true,
		BlockNonSeoBots: false,
		CustomRules: []config.RobotsRuleGroup{
			{
				UserAgents: []string{"TestBot"},
				Disallow:   []string{"/test/"},
			},
		},
	}

	builder := newRobotsBuilder(robotsConfig)
	sitemapURL := "https://example.com/sitemap.xml"

	robotsTxt, err := builder.Build(context.Background(), sitemapURL)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	content := string(robotsTxt)

	permissivePosition := strings.Index(content, "User-agent: *")
	aiBotsPosition := strings.Index(content, "User-agent: GPTBot")
	customPosition := strings.Index(content, "User-agent: TestBot")
	sitemapPosition := strings.Index(content, "Sitemap:")

	if permissivePosition == -1 || aiBotsPosition == -1 || customPosition == -1 || sitemapPosition == -1 {
		t.Fatal("Not all expected sections found in robots.txt")
	}

	if permissivePosition > aiBotsPosition {
		t.Error("Permissive rules should come before AI bot blocking rules")
	}
	if aiBotsPosition > customPosition {
		t.Error("AI bot blocking rules should come before custom rules")
	}
	if customPosition > sitemapPosition {
		t.Error("Custom rules should come before sitemap directive")
	}
}

func TestRobotsDTO_AllBotsLists(t *testing.T) {
	if len(seo_dto.AIBots) == 0 {
		t.Error("AIBots list should not be empty")
	}

	if len(seo_dto.NonSEOBots) == 0 {
		t.Error("NonSEOBots list should not be empty")
	}

	if !slices.Contains(seo_dto.AIBots, "GPTBot") {
		t.Error("Expected GPTBot to be in AIBots list")
	}

	if !slices.Contains(seo_dto.NonSEOBots, "AhrefsBot") {
		t.Error("Expected AhrefsBot to be in NonSEOBots list")
	}
}
