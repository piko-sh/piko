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

package seo_dto

import (
	"strings"
	"testing"
)

func TestRobotsTxtContent_RenderToString(t *testing.T) {
	r := &RobotsTxtContent{
		Groups: []RobotGroup{
			{
				UserAgents: []string{"*"},
				Disallow:   []string{"/admin", "/api"},
				Allow:      []string{"/api/public"},
			},
			{
				UserAgents: []string{"GPTBot"},
				Disallow:   []string{"/"},
			},
		},
	}

	got := r.RenderToString("https://example.com/sitemap.xml")

	if !strings.Contains(got, "User-agent: *") {
		t.Error("missing wildcard user-agent")
	}
	if !strings.Contains(got, "Disallow: /admin") {
		t.Error("missing /admin disallow")
	}
	if !strings.Contains(got, "Allow: /api/public") {
		t.Error("missing /api/public allow")
	}
	if !strings.Contains(got, "User-agent: GPTBot") {
		t.Error("missing GPTBot user-agent")
	}
	if !strings.Contains(got, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("missing sitemap URL")
	}
}

func TestRobotsTxtContent_RenderToString_NoSitemap(t *testing.T) {
	r := &RobotsTxtContent{
		Groups: []RobotGroup{
			{UserAgents: []string{"*"}, Disallow: []string{"/"}},
		},
	}

	got := r.RenderToString("")
	if strings.Contains(got, "Sitemap:") {
		t.Error("should not contain Sitemap line when URL is empty")
	}
}
