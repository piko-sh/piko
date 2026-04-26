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
	"slices"
	"testing"
)

func TestThemeRegistryRegisterAndGet(t *testing.T) {
	registry := NewThemeRegistry()
	theme := buildDarkTheme()
	registry.Register(&theme)

	got, ok := registry.Get(theme.Name)
	if !ok {
		t.Fatalf("Get(%q) reported not found after Register", theme.Name)
	}
	if got.Name != theme.Name {
		t.Errorf("Get returned theme %q, want %q", got.Name, theme.Name)
	}
}

func TestThemeRegistryDuplicatePanics(t *testing.T) {
	registry := NewThemeRegistry()
	theme := buildDarkTheme()
	registry.Register(&theme)

	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on duplicate Register")
		}
	}()
	registry.Register(&theme)
}

func TestThemeRegistryDefaultPanicsWhenMissing(t *testing.T) {
	registry := NewThemeRegistry()

	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on Default() with no themes")
		}
	}()
	_ = registry.Default()
}

func TestThemeRegistryNamesSorted(t *testing.T) {
	registry := NewThemeRegistry()
	dark := buildDarkTheme()
	light := buildLightTheme()
	classic := buildClassicTheme()
	registry.Register(&dark)
	registry.Register(&light)
	registry.Register(&classic)

	names := registry.Names()
	expected := append([]string{}, names...)
	slices.Sort(expected)

	for i := range names {
		if names[i] != expected[i] {
			t.Errorf("Names() not sorted: got %v", names)
			break
		}
	}
}

func TestThemeRegistrySetDefault(t *testing.T) {
	registry := NewThemeRegistry()
	dark := buildDarkTheme()
	light := buildLightTheme()
	registry.Register(&dark)
	registry.Register(&light)

	registry.SetDefault(ThemeLight)
	if got := registry.Default().Name; got != ThemeLight {
		t.Errorf("Default after SetDefault = %q, want %q", got, ThemeLight)
	}

	defer func() {
		if recover() == nil {
			t.Errorf("expected panic when setting unregistered default")
		}
	}()
	registry.SetDefault("missing-theme")
}

func TestBuildThemeByNameKnown(t *testing.T) {
	for _, name := range []string{ThemeClassic, ThemeDark, ThemeLight, ThemeMono, ThemeNoColor} {
		t.Run(name, func(t *testing.T) {
			theme := buildThemeByName(name)
			if theme.Name != name {
				t.Errorf("buildThemeByName(%q).Name = %q", name, theme.Name)
			}
		})
	}
}

func TestBuildThemeByNameUnknown(t *testing.T) {
	theme := buildThemeByName("not-a-theme")
	if theme.Name != ThemeNoColor {
		t.Errorf("unknown theme name fallback = %q, want %q", theme.Name, ThemeNoColor)
	}
}

func TestGlobalThemeRegistryHasBuiltins(t *testing.T) {
	registry := GlobalThemeRegistry()

	for _, name := range []string{ThemeClassic, ThemeDark, ThemeLight, ThemeMono, ThemeNoColor} {
		t.Run(name, func(t *testing.T) {
			if _, ok := registry.Get(name); !ok {
				t.Errorf("global registry missing built-in theme %q", name)
			}
		})
	}
}
