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
	"fmt"
	"image/color"
	"slices"
	"sync"

	"charm.land/lipgloss/v2"
)

const (
	// ThemeClassic is the legacy ANSI 256 palette preserved for visual
	// continuity with prior releases.
	ThemeClassic = "piko-classic"

	// ThemeDark is the modern hex palette tuned for dark terminals.
	ThemeDark = "piko-dark"

	// ThemeLight is the modern hex palette tuned for light terminals.
	ThemeLight = "piko-light"

	// ThemeMono is a grayscale palette with a single accent for accessible
	// or low-colour environments.
	ThemeMono = "piko-mono"

	// ThemeNoColor strips all colours; selected automatically when the
	// NO_COLOR environment variable is set.
	ThemeNoColor = "piko-no-color"

	// DefaultThemeName is the theme used when no explicit choice is made.
	// During the migration window this stays as ThemeClassic so existing
	// users see no visual change; flip to ThemeDark when the migration is
	// complete.
	DefaultThemeName = ThemeClassic
)

// ThemeRegistry holds the set of themes available at runtime.
type ThemeRegistry struct {
	// themes maps a registered theme name to its compiled Theme value.
	themes map[string]Theme

	// defaultName is the registry's currently-selected default theme name.
	defaultName string

	// mu guards themes and defaultName for concurrent access.
	mu sync.RWMutex
}

// NewThemeRegistry creates an empty ThemeRegistry.
//
// Returns *ThemeRegistry which is empty and ready to receive Register calls.
func NewThemeRegistry() *ThemeRegistry {
	return &ThemeRegistry{
		themes:      make(map[string]Theme),
		defaultName: DefaultThemeName,
	}
}

// Register adds a theme to the registry. Registering a theme whose name
// already exists panics so configuration errors surface at start-up rather
// than at render time.
//
// Takes theme (*Theme) which is the entry to register.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *ThemeRegistry) Register(theme *Theme) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.themes[theme.Name]; exists {
		panic(fmt.Sprintf("tui_domain: theme %q is already registered", theme.Name))
	}
	r.themes[theme.Name] = *theme
}

// Get returns the theme with the given name and a flag indicating whether
// the lookup hit.
//
// Takes name (string) which is the registered theme identifier.
//
// Returns Theme which is the resolved theme (zero value when not found).
// Returns bool which is true when a theme with the given name exists.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *ThemeRegistry) Get(name string) (Theme, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.themes[name]
	return t, ok
}

// Default returns the registry's configured default theme. If the configured
// default is missing the function panics: callers always need a usable
// theme to render with.
//
// Returns Theme which is the registered default.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *ThemeRegistry) Default() Theme {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.themes[r.defaultName]
	if !ok {
		panic(fmt.Sprintf("tui_domain: default theme %q is not registered", r.defaultName))
	}
	return t
}

// SetDefault changes the registry's default theme name. The named theme must
// already be registered.
//
// Takes name (string) which is the theme to use as the default.
//
// Panics if name is not a registered theme.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *ThemeRegistry) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.themes[name]; !ok {
		panic(fmt.Sprintf("tui_domain: cannot set default to unregistered theme %q", name))
	}
	r.defaultName = name
}

// Names returns the registered theme names sorted alphabetically.
//
// Returns []string which lists every registered theme name.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *ThemeRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.themes))
	for name := range r.themes {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

var (
	// globalThemeRegistryOnce ensures the singleton registry is initialised
	// at most once.
	globalThemeRegistryOnce sync.Once

	// globalThemeRegistryRef is the package-level registry initialised with
	// the built-in themes on first access.
	globalThemeRegistryRef *ThemeRegistry
)

// paletteSpec is a flat list of colour role strings used to compose a
// Palette. The 18 fields appear in the same order as the Palette struct
// fields so a paletteSpec literal reads top-to-bottom in the same shape.
type paletteSpec struct {
	// Background is the deepest layer colour string.
	Background string

	// Surface is the panel-content layer colour string.
	Surface string

	// SurfaceHigh is the higher-contrast surface colour string.
	SurfaceHigh string

	// SurfaceLow is the lower-contrast surface colour string.
	SurfaceLow string

	// Foreground is the body text colour string.
	Foreground string

	// ForegroundDim is the muted body text colour string.
	ForegroundDim string

	// ForegroundMuted is the most muted text colour string.
	ForegroundMuted string

	// Border is the inactive border colour string.
	Border string

	// BorderFocused is the focused border colour string.
	BorderFocused string

	// Cursor is the cursor colour string.
	Cursor string

	// Primary is the dominant accent colour string.
	Primary string

	// PrimarySoft is the softer primary accent colour string.
	PrimarySoft string

	// Accent is the secondary accent colour string.
	Accent string

	// AccentSoft is the softer secondary accent colour string.
	AccentSoft string

	// Success is the success colour string.
	Success string

	// Warning is the warning colour string.
	Warning string

	// Danger is the danger colour string.
	Danger string

	// Info is the informational colour string.
	Info string
}

// builtInThemeSpec describes a single built-in theme entry.
type builtInThemeSpec struct {
	// Spec is the palette specification for this entry.
	Spec paletteSpec

	// Name is the registered identifier of this entry.
	Name string

	// IsDark indicates whether this entry targets dark terminals.
	IsDark bool
}

// builtInThemes is the canonical list of palettes registered on first
// access of GlobalThemeRegistry. Adding a new built-in is a single row
// here rather than a new builder function.
var builtInThemes = []builtInThemeSpec{
	{
		Name:   ThemeClassic,
		IsDark: true,
		Spec: paletteSpec{
			Background: "234", Surface: "235", SurfaceHigh: "235", SurfaceLow: "234",
			Foreground: "252", ForegroundDim: "240", ForegroundMuted: "238",
			Border: "238", BorderFocused: "39", Cursor: "39",
			Primary: "39", PrimarySoft: "32", Accent: "214", AccentSoft: "172",
			Success: "42", Warning: "214", Danger: "196", Info: "39",
		},
	},
	{
		Name:   ThemeDark,
		IsDark: true,
		Spec: paletteSpec{
			Background: "#0e0f14", Surface: "#1a1d24", SurfaceHigh: "#232831", SurfaceLow: "#15171d",
			Foreground: "#d6d8dc", ForegroundDim: "#6c7281", ForegroundMuted: "#4a505d",
			Border: "#383e4a", BorderFocused: "#7aa2f7", Cursor: "#7aa2f7",
			Primary: "#7aa2f7", PrimarySoft: "#5277c5", Accent: "#e0af68", AccentSoft: "#b58950",
			Success: "#9ece6a", Warning: "#ff9e64", Danger: "#f7768e", Info: "#7dcfff",
		},
	},
	{
		Name:   ThemeLight,
		IsDark: false,
		Spec: paletteSpec{
			Background: "#f7f8fb", Surface: "#eef0f4", SurfaceHigh: "#dde1e9", SurfaceLow: "#fafbfd",
			Foreground: "#21262d", ForegroundDim: "#5e6470", ForegroundMuted: "#8a90a0",
			Border: "#cdd2db", BorderFocused: "#3a5fcd", Cursor: "#3a5fcd",
			Primary: "#3a5fcd", PrimarySoft: "#5b7fe6", Accent: "#b06f1c", AccentSoft: "#cf8e3a",
			Success: "#3a8c3a", Warning: "#a86a17", Danger: "#c92443", Info: "#0a7099",
		},
	},
	{
		Name:   ThemeMono,
		IsDark: true,
		Spec: paletteSpec{
			Background: "#0a0a0a", Surface: "#181818", SurfaceHigh: "#262626", SurfaceLow: "#0f0f0f",
			Foreground: "#e6e6e6", ForegroundDim: "#9a9a9a", ForegroundMuted: "#5e5e5e",
			Border: "#3a3a3a", BorderFocused: "#bfbfbf", Cursor: "#ffffff",
			Primary: "#ffffff", PrimarySoft: "#cccccc", Accent: "#e0af68", AccentSoft: "#b58950",
			Success: "#bfbfbf", Warning: "#e0af68", Danger: "#ffffff", Info: "#cccccc",
		},
	},
}

// GlobalThemeRegistry returns the singleton ThemeRegistry, initialising the
// built-in themes on the first call.
//
// Returns *ThemeRegistry which is the package-wide registry.
func GlobalThemeRegistry() *ThemeRegistry {
	globalThemeRegistryOnce.Do(func() {
		globalThemeRegistryRef = NewThemeRegistry()
		for i := range builtInThemes {
			entry := &builtInThemes[i]
			palette := paletteFromSpec(&entry.Spec)
			theme := buildTheme(&palette, entry.Name, entry.IsDark)
			globalThemeRegistryRef.Register(&theme)
		}
		noColor := buildNoColorTheme()
		globalThemeRegistryRef.Register(&noColor)
	})
	return globalThemeRegistryRef
}

// buildThemeByName returns the built-in theme registered under name. Used
// by tests; returns the no-colour theme when name is unknown so the
// helper is safe to call before initialisation.
//
// Takes name (string) which is the requested built-in identifier.
//
// Returns Theme which is the matching built-in or the no-colour fallback.
func buildThemeByName(name string) Theme {
	if name == ThemeNoColor {
		return buildNoColorTheme()
	}
	for i := range builtInThemes {
		entry := &builtInThemes[i]
		if entry.Name == name {
			palette := paletteFromSpec(&entry.Spec)
			return buildTheme(&palette, entry.Name, entry.IsDark)
		}
	}
	return buildNoColorTheme()
}

// buildClassicTheme returns the classic ANSI palette. Retained as a
// thin wrapper for tests; production code should use the global
// registry.
//
// Returns Theme rendered with the legacy ANSI palette.
func buildClassicTheme() Theme { return buildThemeByName(ThemeClassic) }

// buildDarkTheme returns the modern dark hex palette.
//
// Returns Theme tuned for dark terminals.
func buildDarkTheme() Theme { return buildThemeByName(ThemeDark) }

// buildLightTheme returns the modern light hex palette.
//
// Returns Theme tuned for light terminals.
func buildLightTheme() Theme { return buildThemeByName(ThemeLight) }

// buildMonoTheme returns the grayscale-with-accent palette.
//
// Returns Theme suitable for limited-colour terminals.
func buildMonoTheme() Theme { return buildThemeByName(ThemeMono) }

// paletteFromSpec converts a string-keyed palette specification into a
// fully-populated Palette by applying lipgloss.Color to each role.
//
// Takes spec (*paletteSpec) which holds the colour strings.
//
// Returns Palette which wraps each string in a lipgloss.Color.
func paletteFromSpec(spec *paletteSpec) Palette {
	return Palette{
		Background:      lipgloss.Color(spec.Background),
		Surface:         lipgloss.Color(spec.Surface),
		SurfaceHigh:     lipgloss.Color(spec.SurfaceHigh),
		SurfaceLow:      lipgloss.Color(spec.SurfaceLow),
		Foreground:      lipgloss.Color(spec.Foreground),
		ForegroundDim:   lipgloss.Color(spec.ForegroundDim),
		ForegroundMuted: lipgloss.Color(spec.ForegroundMuted),
		Border:          lipgloss.Color(spec.Border),
		BorderFocused:   lipgloss.Color(spec.BorderFocused),
		Cursor:          lipgloss.Color(spec.Cursor),
		Primary:         lipgloss.Color(spec.Primary),
		PrimarySoft:     lipgloss.Color(spec.PrimarySoft),
		Accent:          lipgloss.Color(spec.Accent),
		AccentSoft:      lipgloss.Color(spec.AccentSoft),
		Success:         lipgloss.Color(spec.Success),
		Warning:         lipgloss.Color(spec.Warning),
		Danger:          lipgloss.Color(spec.Danger),
		Info:            lipgloss.Color(spec.Info),
	}
}

// buildNoColorTheme returns a palette where every role is NoColor. Used when
// the NO_COLOR environment variable is set or when the user explicitly
// requests a colourless TUI.
//
// Returns Theme rendered without any colour application.
func buildNoColorTheme() Theme {
	none := color.Color(lipgloss.NoColor{})
	palette := Palette{
		Background:      none,
		Surface:         none,
		SurfaceHigh:     none,
		SurfaceLow:      none,
		Foreground:      none,
		ForegroundDim:   none,
		ForegroundMuted: none,
		Border:          none,
		BorderFocused:   none,
		Cursor:          none,
		Primary:         none,
		PrimarySoft:     none,
		Accent:          none,
		AccentSoft:      none,
		Success:         none,
		Warning:         none,
		Danger:          none,
		Info:            none,
	}
	return buildTheme(&palette, ThemeNoColor, false)
}
