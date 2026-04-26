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
	"testing"

	"charm.land/lipgloss/v2"
)

func TestBuildClassicTheme(t *testing.T) {
	theme := buildClassicTheme()

	if theme.Name != ThemeClassic {
		t.Errorf("Name = %q, want %q", theme.Name, ThemeClassic)
	}
	if !theme.IsDark {
		t.Error("classic theme should be marked dark")
	}
	if theme.Palette.Primary == nil {
		t.Error("Palette.Primary not populated")
	}
	if theme.ScrollOffLines <= 0 {
		t.Errorf("ScrollOffLines = %d, want positive default", theme.ScrollOffLines)
	}
}

func TestThemeStylesAllPopulated(t *testing.T) {
	theme := buildDarkTheme()

	cases := []struct {
		style lipgloss.Style
		name  string
	}{
		{name: "Title", style: theme.Title},
		{name: "Subtle", style: theme.Subtle},
		{name: "Selected", style: theme.Selected},
		{name: "Cursor", style: theme.Cursor},
		{name: "Border", style: theme.Border},
		{name: "BorderFocused", style: theme.BorderFocused},
		{name: "Panel", style: theme.Panel},
		{name: "PanelFocused", style: theme.PanelFocused},
		{name: "PanelTitle", style: theme.PanelTitle},
		{name: "Tab", style: theme.Tab},
		{name: "TabActive", style: theme.TabActive},
		{name: "TabHotkey", style: theme.TabHotkey},
		{name: "StatusBar", style: theme.StatusBar},
		{name: "StatusKey", style: theme.StatusKey},
		{name: "StatusDesc", style: theme.StatusDesc},
		{name: "StatusSep", style: theme.StatusSep},
		{name: "StatusHealthy", style: theme.StatusHealthy},
		{name: "StatusDegraded", style: theme.StatusDegraded},
		{name: "StatusUnhealthy", style: theme.StatusUnhealthy},
		{name: "StatusPending", style: theme.StatusPending},
		{name: "StatusUnknown", style: theme.StatusUnknown},
		{name: "Error", style: theme.Error},
		{name: "Warning", style: theme.Warning},
		{name: "Info", style: theme.Info},
		{name: "Dim", style: theme.Dim},
		{name: "Bold", style: theme.Bold},
		{name: "OverlayBackground", style: theme.OverlayBackground},
		{name: "Search", style: theme.Search},
		{name: "SearchLabel", style: theme.SearchLabel},
		{name: "Sparkline", style: theme.Sparkline},
		{name: "SparklineHigh", style: theme.SparklineHigh},
		{name: "SparklineLow", style: theme.SparklineLow},
		{name: "TableHeader", style: theme.TableHeader},
		{name: "TableRow", style: theme.TableRow},
		{name: "TableSelected", style: theme.TableSelected},
		{name: "TableBorder", style: theme.TableBorder},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.style.Render("x")
			if out == "" {
				t.Errorf("style %s rendered empty output", tc.name)
			}
		})
	}
}

func TestThemeHealthStateStyleAccessors(t *testing.T) {
	theme := buildDarkTheme()

	cases := []struct {
		name string
		in   HealthState
	}{
		{name: "unknown", in: HealthStateUnknown},
		{name: "healthy", in: HealthStateHealthy},
		{name: "degraded", in: HealthStateDegraded},
		{name: "unhealthy", in: HealthStateUnhealthy},
		{name: "out-of-range", in: HealthState(999)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if out := theme.HealthStateStyle(c.in).Render("x"); out == "" {
				t.Errorf("HealthStateStyle(%v) rendered empty output", c.in)
			}
		})
	}
}

func TestThemeResourceStatusStyleAccessors(t *testing.T) {
	theme := buildDarkTheme()

	cases := []ResourceStatus{
		ResourceStatusUnknown,
		ResourceStatusHealthy,
		ResourceStatusDegraded,
		ResourceStatusUnhealthy,
		ResourceStatusPending,
		ResourceStatus(999),
	}
	for _, in := range cases {
		if out := theme.ResourceStatusStyle(in).Render("x"); out == "" {
			t.Errorf("ResourceStatusStyle(%v) rendered empty output", in)
		}
	}
}

func TestThemeProviderStatusStyleAccessors(t *testing.T) {
	theme := buildDarkTheme()

	cases := []ProviderStatus{
		ProviderStatusDisconnected,
		ProviderStatusConnecting,
		ProviderStatusConnected,
		ProviderStatusError,
		ProviderStatus(999),
	}
	for _, in := range cases {
		if out := theme.ProviderStatusStyle(in).Render("x"); out == "" {
			t.Errorf("ProviderStatusStyle(%v) rendered empty output", in)
		}
	}
}

func TestThemeSeverityFor(t *testing.T) {
	theme := buildDarkTheme()

	if got := theme.SeverityFor(SeverityHealthy); got.Render("x") != theme.StatusHealthy.Render("x") {
		t.Errorf("SeverityHealthy did not map to StatusHealthy")
	}
	if got := theme.SeverityFor(SeverityWarning); got.Render("x") != theme.StatusDegraded.Render("x") {
		t.Errorf("SeverityWarning did not map to StatusDegraded")
	}
	if got := theme.SeverityFor(SeverityCritical); got.Render("x") != theme.StatusUnhealthy.Render("x") {
		t.Errorf("SeverityCritical did not map to StatusUnhealthy")
	}
	if got := theme.SeverityFor(SeveritySaturated); got.Render("x") != theme.StatusUnhealthy.Render("x") {
		t.Errorf("SeveritySaturated did not map to StatusUnhealthy")
	}
}
