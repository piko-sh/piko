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
	"os"
	"strings"
)

const (
	// EnvNoColor is the canonical NO_COLOR signal recognised across CLI
	// tooling. When set to any non-empty value, the resolver picks the
	// no-colour theme regardless of any other configuration.
	EnvNoColor = "NO_COLOR"

	// EnvThemeOverride is the piko-specific environment variable that
	// overrides the configured theme name when set to a registered theme.
	EnvThemeOverride = "PIKO_TUI_THEME"
)

// ResolveTheme returns the theme to render with given a registry and a
// configured theme name.
//
// The resolution order is:
//
//  1. If NO_COLOR is set in the environment, the no-colour theme wins.
//  2. If PIKO_TUI_THEME names a registered theme, it wins.
//  3. If configName names a registered theme, it wins.
//  4. Otherwise the registry's default is used.
//
// Takes registry (*ThemeRegistry) which holds the registered themes.
// Takes configName (string) which is the user-configured theme identifier;
// empty string defers to the next step in the order above.
//
// Returns Theme which is the resolved theme for rendering.
func ResolveTheme(registry *ThemeRegistry, configName string) Theme {
	return resolveThemeWithEnv(registry, configName, osLookupEnv)
}

// resolveThemeWithEnv is ResolveTheme with an injected environment lookup,
// kept package-private so tests can drive the resolver deterministically.
//
// Takes registry (*ThemeRegistry) which holds the registered themes.
// Takes configName (string) which is the user-configured theme identifier.
// Takes lookup (func(string) (string, bool)) which yields environment values.
//
// Returns Theme which is the resolved theme.
func resolveThemeWithEnv(registry *ThemeRegistry, configName string, lookup func(string) (string, bool)) Theme {
	if registry == nil {
		registry = GlobalThemeRegistry()
	}

	if v, ok := lookup(EnvNoColor); ok && v != "" {
		if t, found := registry.Get(ThemeNoColor); found {
			return t
		}
		return registry.Default()
	}

	if v, ok := lookup(EnvThemeOverride); ok {
		name := strings.TrimSpace(v)
		if name != "" {
			if t, found := registry.Get(name); found {
				return t
			}
		}
	}

	name := strings.TrimSpace(configName)
	if name != "" {
		if t, found := registry.Get(name); found {
			return t
		}
	}

	return registry.Default()
}

// osLookupEnv is the production environment lookup wrapping os.LookupEnv.
//
// Takes name (string) which is the environment variable name.
//
// Returns string which is the value (empty when not set).
// Returns bool which is true when the variable is present.
func osLookupEnv(name string) (string, bool) {
	return os.LookupEnv(name)
}
