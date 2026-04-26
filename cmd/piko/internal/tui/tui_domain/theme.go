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
	"image/color"

	"charm.land/lipgloss/v2"
)

// Severity categorises a measured utilisation or status into ordered bands.
// It maps to colour ramps in the active theme.
type Severity int

const (
	// SeverityHealthy indicates a value within nominal limits.
	SeverityHealthy Severity = iota

	// SeverityWarning indicates a value approaching its limit and worth
	// surfacing in the UI.
	SeverityWarning

	// SeverityCritical indicates a value at or above its limit and requiring
	// attention.
	SeverityCritical

	// SeveritySaturated indicates a value that has overflowed its limit.
	SeveritySaturated
)

// Palette holds the raw colour roles a theme assigns. All TUI styles are
// built from these values, so theming is a question of choosing a palette
// rather than rewriting individual styles.
type Palette struct {
	// Background is the deepest layer, used behind everything.
	Background color.Color

	// Surface is the panel-content layer, slightly raised from Background.
	Surface color.Color

	// SurfaceHigh is a higher-contrast variant of Surface for elements that
	// need to stand out (status bar background, selected row background).
	SurfaceHigh color.Color

	// SurfaceLow is a lower-contrast variant of Surface used sparingly for
	// subtle fills.
	SurfaceLow color.Color

	// Foreground is the standard body text colour.
	Foreground color.Color

	// ForegroundDim is the muted text colour for labels and secondary text.
	ForegroundDim color.Color

	// ForegroundMuted is even more muted than ForegroundDim, used for help
	// hints and the most-secondary content.
	ForegroundMuted color.Color

	// Border is the colour of inactive panel borders and separators.
	Border color.Color

	// BorderFocused is the colour of the focused panel's border.
	BorderFocused color.Color

	// Cursor is the colour of the active row indicator.
	Cursor color.Color

	// Primary is the dominant accent colour: focused borders, selections,
	// active titles.
	Primary color.Color

	// PrimarySoft is a softer variant of Primary used for subtle accents.
	PrimarySoft color.Color

	// Accent is a secondary accent colour, distinct from Primary, used for
	// hotkey letters and highlights.
	Accent color.Color

	// AccentSoft is a softer variant of Accent.
	AccentSoft color.Color

	// Success is the colour for healthy status, completed states, low-risk
	// gauges.
	Success color.Color

	// Warning is the colour for degraded states and medium-risk gauges.
	Warning color.Color

	// Danger is the colour for unhealthy states, errors, and high-risk
	// gauges.
	Danger color.Color

	// Info is the colour for informational messages and pending states.
	Info color.Color
}

// Theme holds pre-computed lipgloss styles for every named role in the TUI.
// Panels and widgets read styles from a *Theme rather than constructing them
// inline, so palette swaps are a single allocation rather than a sweeping
// edit.
type Theme struct {
	// StatusDesc styles the descriptive text portion of the status bar.
	StatusDesc lipgloss.Style

	// StatusHealthy styles content marked as healthy.
	StatusHealthy lipgloss.Style

	// TableBorder styles a table's outer border.
	TableBorder lipgloss.Style

	// TableAlternate styles alternating table rows for striping.
	TableAlternate lipgloss.Style

	// TableSelected styles the currently selected table row.
	TableSelected lipgloss.Style

	// TableRow styles the default table row.
	TableRow lipgloss.Style

	// Title styles top-level panel titles.
	Title lipgloss.Style

	// Subtle styles secondary, low-emphasis text.
	Subtle lipgloss.Style

	// Selected styles selected items in lists and menus.
	Selected lipgloss.Style

	// Cursor styles the active row indicator.
	Cursor lipgloss.Style

	// Border styles inactive panel borders and separators.
	Border lipgloss.Style

	// BorderFocused styles the focused panel's border.
	BorderFocused lipgloss.Style

	// Panel styles the panel container with rounded border.
	Panel lipgloss.Style

	// PanelFocused styles the focused panel container.
	PanelFocused lipgloss.Style

	// PanelTitle styles the title row at the top of a panel.
	PanelTitle lipgloss.Style

	// Tab styles inactive tabs in a tab bar.
	Tab lipgloss.Style

	// TabActive styles the currently active tab.
	TabActive lipgloss.Style

	// StatusSep styles separators between status-bar items.
	StatusSep lipgloss.Style

	// StatusBar styles the bottom status bar background and text.
	StatusBar lipgloss.Style

	// StatusKey styles hotkey letters in the status bar.
	StatusKey lipgloss.Style

	// TableHeader styles the header row of a table.
	TableHeader lipgloss.Style

	// SparklineLow styles low-value samples in a sparkline.
	SparklineLow lipgloss.Style

	// TabHotkey styles the hotkey letter inside a tab label.
	TabHotkey lipgloss.Style

	// StatusDegraded styles content in a degraded state.
	StatusDegraded lipgloss.Style

	// StatusUnhealthy styles content in an unhealthy state.
	StatusUnhealthy lipgloss.Style

	// StatusPending styles content in a pending state.
	StatusPending lipgloss.Style

	// StatusUnknown styles content whose state is not known.
	StatusUnknown lipgloss.Style

	// Error styles error messages.
	Error lipgloss.Style

	// Warning styles warning messages.
	Warning lipgloss.Style

	// Info styles informational messages.
	Info lipgloss.Style

	// Dim styles dimmed, low-emphasis text.
	Dim lipgloss.Style

	// Bold styles bold-weight text.
	Bold lipgloss.Style

	// OverlayBackground styles backgrounds beneath modal overlays.
	OverlayBackground lipgloss.Style

	// Search styles the search input frame.
	Search lipgloss.Style

	// SearchLabel styles the label that precedes the search input.
	SearchLabel lipgloss.Style

	// Sparkline styles the default sparkline body.
	Sparkline lipgloss.Style

	// SparklineHigh styles high-value samples in a sparkline.
	SparklineHigh lipgloss.Style

	// Palette is the source palette this theme was built from.
	Palette Palette

	// Name is the registered identifier of the theme.
	Name string

	// ScrollOffLines is the minimum line buffer at the top and bottom of a
	// scrollable view.
	ScrollOffLines int

	// IsDark indicates whether the theme is intended for dark terminals.
	IsDark bool

	// MouseHoverFocus enables focus-follows-mouse behaviour.
	MouseHoverFocus bool

	// DimInactivePanes dims panes that do not have keyboard focus.
	DimInactivePanes bool
}

// buildTheme constructs a Theme from a Palette and a name.
//
// Takes palette (*Palette) which provides the raw colour roles.
// Takes name (string) which identifies the theme in the registry.
// Takes isDark (bool) which records whether the theme targets dark
// terminals.
//
// Returns Theme with every style field populated from the palette.
func buildTheme(palette *Palette, name string, isDark bool) Theme {
	t := Theme{
		Name:           name,
		IsDark:         isDark,
		Palette:        *palette,
		ScrollOffLines: DefaultScrollOff,
	}

	t.Title = lipgloss.NewStyle().Foreground(palette.Primary).Bold(true).Padding(0, 1)
	t.Subtle = lipgloss.NewStyle().Foreground(palette.ForegroundDim)
	t.Selected = lipgloss.NewStyle().Foreground(palette.Foreground).Background(palette.SurfaceHigh).Bold(true)
	t.Cursor = lipgloss.NewStyle().Foreground(palette.Cursor)
	t.Border = lipgloss.NewStyle().Foreground(palette.Border)
	t.BorderFocused = lipgloss.NewStyle().Foreground(palette.BorderFocused)
	t.Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(palette.Border).Padding(0, 1)
	t.PanelFocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(palette.BorderFocused).Padding(0, 1)
	t.PanelTitle = lipgloss.NewStyle().Foreground(palette.Primary).Bold(true).Padding(0, 1)
	t.Tab = lipgloss.NewStyle().Foreground(palette.ForegroundDim).Padding(0, 1)
	t.TabActive = lipgloss.NewStyle().Foreground(palette.Primary).Bold(true).Padding(0, 1)
	t.TabHotkey = lipgloss.NewStyle().Foreground(palette.Accent).Bold(true)
	t.StatusBar = lipgloss.NewStyle().Foreground(palette.ForegroundDim).Background(palette.SurfaceHigh).Padding(0, 1)
	t.StatusKey = lipgloss.NewStyle().Foreground(palette.Accent).Bold(true)
	t.StatusDesc = lipgloss.NewStyle().Foreground(palette.ForegroundDim)
	t.StatusSep = lipgloss.NewStyle().Foreground(palette.Border)
	t.StatusHealthy = lipgloss.NewStyle().Foreground(palette.Success)
	t.StatusDegraded = lipgloss.NewStyle().Foreground(palette.Warning)
	t.StatusUnhealthy = lipgloss.NewStyle().Foreground(palette.Danger)
	t.StatusPending = lipgloss.NewStyle().Foreground(palette.Info)
	t.StatusUnknown = lipgloss.NewStyle().Foreground(palette.ForegroundDim)
	t.Error = lipgloss.NewStyle().Foreground(palette.Danger)
	t.Warning = lipgloss.NewStyle().Foreground(palette.Warning)
	t.Info = lipgloss.NewStyle().Foreground(palette.Info)
	t.Dim = lipgloss.NewStyle().Foreground(palette.ForegroundDim)
	t.Bold = lipgloss.NewStyle().Bold(true)
	t.OverlayBackground = lipgloss.NewStyle().Faint(true)
	t.Search = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(palette.BorderFocused)
	t.SearchLabel = lipgloss.NewStyle().Foreground(palette.Accent)
	t.Sparkline = lipgloss.NewStyle().Foreground(palette.Primary)
	t.SparklineHigh = lipgloss.NewStyle().Foreground(palette.Success)
	t.SparklineLow = lipgloss.NewStyle().Foreground(palette.Danger)
	t.TableHeader = lipgloss.NewStyle().
		Foreground(palette.Primary).
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(palette.Border)
	t.TableRow = lipgloss.NewStyle().Foreground(palette.Foreground)
	t.TableSelected = lipgloss.NewStyle().Foreground(palette.Foreground).Background(palette.SurfaceHigh).Bold(true)
	t.TableAlternate = lipgloss.NewStyle()
	t.TableBorder = lipgloss.NewStyle().Foreground(palette.Border)

	return t
}

// SeverityFor returns the style appropriate for a severity band.
//
// Takes s (Severity) which is the band to render.
//
// Returns lipgloss.Style which maps the band to its colour role.
func (t *Theme) SeverityFor(s Severity) lipgloss.Style {
	switch s {
	case SeverityHealthy:
		return t.StatusHealthy
	case SeverityWarning:
		return t.StatusDegraded
	case SeverityCritical, SeveritySaturated:
		return t.StatusUnhealthy
	default:
		return t.StatusUnknown
	}
}

// HealthStateStyle returns the style for a HealthState value.
//
// Takes s (HealthState) which is the health state to colour.
//
// Returns lipgloss.Style which is the appropriate themed style.
func (t *Theme) HealthStateStyle(s HealthState) lipgloss.Style {
	switch s {
	case HealthStateHealthy:
		return t.StatusHealthy
	case HealthStateDegraded:
		return t.StatusDegraded
	case HealthStateUnhealthy:
		return t.StatusUnhealthy
	default:
		return t.StatusUnknown
	}
}

// ResourceStatusStyle returns the style for a ResourceStatus value.
//
// Takes s (ResourceStatus) which is the resource status to colour.
//
// Returns lipgloss.Style which is the appropriate themed style.
func (t *Theme) ResourceStatusStyle(s ResourceStatus) lipgloss.Style {
	switch s {
	case ResourceStatusHealthy:
		return t.StatusHealthy
	case ResourceStatusDegraded:
		return t.StatusDegraded
	case ResourceStatusUnhealthy:
		return t.StatusUnhealthy
	case ResourceStatusPending:
		return t.StatusPending
	default:
		return t.StatusUnknown
	}
}

// ProviderStatusStyle returns the style for a ProviderStatus value.
//
// Takes s (ProviderStatus) which is the provider status to colour.
//
// Returns lipgloss.Style which is the appropriate themed style.
func (t *Theme) ProviderStatusStyle(s ProviderStatus) lipgloss.Style {
	switch s {
	case ProviderStatusConnected:
		return t.StatusHealthy
	case ProviderStatusDisconnected, ProviderStatusError:
		return t.StatusUnhealthy
	default:
		return t.StatusUnknown
	}
}
