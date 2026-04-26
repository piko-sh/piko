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
)

func TestApplyThemeToLegacyGlobalsRebindsColours(t *testing.T) {
	originalPrimary := colourPrimary
	originalDanger := colourError
	originalPanelStyle := panelStyle

	t.Cleanup(func() {
		applyThemeToLegacyGlobals(themeOrDefault(ThemeClassic))
		_ = originalPrimary
		_ = originalDanger
		_ = originalPanelStyle
	})

	dark := themeOrDefault(ThemeDark)
	applyThemeToLegacyGlobals(dark)
	if colourPrimary != dark.Palette.Primary {
		t.Errorf("colourPrimary did not update for dark theme")
	}
	if colourError != dark.Palette.Danger {
		t.Errorf("colourError did not update for dark theme")
	}
	if panelStyle.Render("x") == "" {
		t.Errorf("panelStyle should still produce non-empty output after rebind")
	}

	light := themeOrDefault(ThemeLight)
	applyThemeToLegacyGlobals(light)
	if colourPrimary != light.Palette.Primary {
		t.Errorf("colourPrimary did not update for light theme")
	}
}

func TestApplyThemeToLegacyGlobalsNilNoop(t *testing.T) {
	before := colourPrimary
	applyThemeToLegacyGlobals(nil)
	if colourPrimary != before {
		t.Errorf("nil theme should not modify legacy globals")
	}
}

func TestModelSetThemePropagatesToGlobals(t *testing.T) {
	config := newTestConfig()
	model := NewModel(config)

	dark := themeOrDefault(ThemeDark)
	model.SetTheme(dark)
	if colourPrimary != dark.Palette.Primary {
		t.Errorf("Model.SetTheme did not propagate to colourPrimary")
	}

	t.Cleanup(func() {
		applyThemeToLegacyGlobals(themeOrDefault(ThemeClassic))
	})
}

func themeOrDefault(name string) *Theme {
	registry := GlobalThemeRegistry()
	theme, ok := registry.Get(name)
	if !ok {
		theme = registry.Default()
	}
	return &theme
}
