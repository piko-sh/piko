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

package tui_domain

import (
	"strings"
	"testing"
)

func TestBreadcrumbRendersTitleAndChain(t *testing.T) {
	b := &Breadcrumb{
		Title:      "piko",
		Endpoint:   "localhost:9091",
		PanelChain: []string{"Watchdog", "Profiles"},
		Watch:      true,
	}

	out := b.Render(nil, 80)

	if !strings.Contains(out, "piko") {
		t.Errorf("missing title: %q", out)
	}
	if !strings.Contains(out, "Watchdog") {
		t.Errorf("missing first chain segment: %q", out)
	}
	if !strings.Contains(out, "Profiles") {
		t.Errorf("missing second chain segment: %q", out)
	}
	if !strings.Contains(out, "localhost:9091") {
		t.Errorf("missing endpoint: %q", out)
	}
	if !strings.Contains(out, watchIndicatorActive) {
		t.Errorf("missing active watch indicator: %q", out)
	}
}

func TestBreadcrumbWatchIdleIndicator(t *testing.T) {
	b := &Breadcrumb{
		Title: "piko",
		Watch: false,
	}
	out := b.Render(nil, 40)
	if !strings.Contains(out, watchIndicatorIdle) {
		t.Errorf("missing idle indicator: %q", out)
	}
	if strings.Contains(out, watchIndicatorActive) {
		t.Errorf("active indicator should not appear when Watch=false: %q", out)
	}
}

func TestBreadcrumbWidth(t *testing.T) {
	b := &Breadcrumb{
		Title:      "piko",
		Endpoint:   "localhost:9091",
		PanelChain: []string{"Watchdog", "Profiles"},
	}
	out := b.Render(nil, 60)
	if got := TextWidth(out); got != 60 {
		t.Errorf("Render width = %d, want 60: %q", got, out)
	}
}

func TestBreadcrumbZeroWidth(t *testing.T) {
	b := &Breadcrumb{Title: "piko"}
	if out := b.Render(nil, 0); out != "" {
		t.Errorf("zero-width render should return empty, got %q", out)
	}
}

func TestBreadcrumbScopeRenders(t *testing.T) {
	b := &Breadcrumb{
		Title:    "piko",
		Endpoint: "localhost:9091",
		Scope:    "ns: prod",
	}
	out := b.Render(nil, 80)
	if !strings.Contains(out, "ns: prod") {
		t.Errorf("missing scope: %q", out)
	}
}
