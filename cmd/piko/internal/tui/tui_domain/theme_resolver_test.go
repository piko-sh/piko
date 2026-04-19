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

func fixedEnv(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := values[name]
		return v, ok
	}
}

func TestResolveThemeNoColorWins(t *testing.T) {
	registry := buildTestRegistry()
	env := fixedEnv(map[string]string{
		EnvNoColor:       "1",
		EnvThemeOverride: ThemeDark,
	})

	theme := resolveThemeWithEnv(registry, ThemeLight, env)
	if theme.Name != ThemeNoColor {
		t.Errorf("Name = %q, want %q (NO_COLOR should win)", theme.Name, ThemeNoColor)
	}
}

func TestResolveThemeEnvOverride(t *testing.T) {
	registry := buildTestRegistry()
	env := fixedEnv(map[string]string{
		EnvThemeOverride: ThemeDark,
	})

	theme := resolveThemeWithEnv(registry, ThemeLight, env)
	if theme.Name != ThemeDark {
		t.Errorf("Name = %q, want %q (env override should win)", theme.Name, ThemeDark)
	}
}

func TestResolveThemeConfigFallback(t *testing.T) {
	registry := buildTestRegistry()
	env := fixedEnv(nil)

	theme := resolveThemeWithEnv(registry, ThemeLight, env)
	if theme.Name != ThemeLight {
		t.Errorf("Name = %q, want %q (config should resolve)", theme.Name, ThemeLight)
	}
}

func TestResolveThemeDefault(t *testing.T) {
	registry := buildTestRegistry()
	env := fixedEnv(nil)

	theme := resolveThemeWithEnv(registry, "", env)
	if theme.Name != DefaultThemeName {
		t.Errorf("Name = %q, want %q (registry default)", theme.Name, DefaultThemeName)
	}
}

func TestResolveThemeUnknownEnvFallsThroughToConfig(t *testing.T) {
	registry := buildTestRegistry()
	env := fixedEnv(map[string]string{
		EnvThemeOverride: "no-such-theme",
	})

	theme := resolveThemeWithEnv(registry, ThemeLight, env)
	if theme.Name != ThemeLight {
		t.Errorf("Name = %q, want %q (config should win after unknown env)", theme.Name, ThemeLight)
	}
}

func TestResolveThemeNilRegistryUsesGlobal(t *testing.T) {
	theme := resolveThemeWithEnv(nil, "", fixedEnv(nil))
	if theme.Name != DefaultThemeName {
		t.Errorf("Name = %q, want %q (global default)", theme.Name, DefaultThemeName)
	}
}

func buildTestRegistry() *ThemeRegistry {
	registry := NewThemeRegistry()
	registry.Register(new(buildClassicTheme()))
	registry.Register(new(buildDarkTheme()))
	registry.Register(new(buildLightTheme()))
	registry.Register(new(buildMonoTheme()))
	registry.Register(new(buildNoColorTheme()))
	return registry
}
