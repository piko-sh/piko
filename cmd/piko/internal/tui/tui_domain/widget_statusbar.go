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
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	// statusBarEllipsisPadding is the width kept for the ellipsis in the status bar.
	statusBarEllipsisPadding = 5

	// statusBarBindingSeparator is the space placed between key bindings.
	statusBarBindingSeparator = " "
)

// StatusBarConfig holds style settings for the status bar.
type StatusBarConfig struct {
	// Style specifies the visual style for the status bar background.
	Style lipgloss.Style

	// KeyStyle is the visual style for key binding keys in the status bar.
	KeyStyle lipgloss.Style

	// DescStyle sets the text style for descriptions and labels.
	DescStyle lipgloss.Style

	// SeparatorStyle is the style for separators between items.
	SeparatorStyle lipgloss.Style

	// HealthyStyle is the style used for providers that are connected.
	HealthyStyle lipgloss.Style

	// UnhealthyStyle is the visual style for unhealthy provider status indicators.
	UnhealthyStyle lipgloss.Style
}

// StatusBarData holds the data shown in the status bar.
type StatusBarData struct {
	// Now is the current time for age calculations; zero uses time.Now().
	Now time.Time

	// LastRefresh is when data was last updated; zero means never refreshed.
	LastRefresh time.Time

	// CurrentPanel is the name of the panel that is currently active.
	CurrentPanel string

	// KeyBindings lists the keyboard shortcuts to show in the status bar.
	KeyBindings []KeyBinding

	// Providers holds the status bar entries shown on the right side.
	Providers []ProviderInfo
}

// DefaultStatusBarConfig returns sensible defaults for status bar styling.
//
// Returns StatusBarConfig which contains pre-configured styles for the status
// bar, including colours for keys, descriptions, separators, and health states.
func DefaultStatusBarConfig() StatusBarConfig {
	return StatusBarConfig{
		Style: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1),
		KeyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		DescStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		SeparatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")),
		HealthyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		UnhealthyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
	}
}

// StatusBar renders a styled status bar with key bindings and provider states.
//
// Takes data (StatusBarData) which provides the key bindings, provider
// statuses, and last refresh time to show.
// Takes config (*StatusBarConfig) which sets the styles for each element.
// Takes width (int) which sets the total width of the bar.
//
// Returns string which is the styled status bar ready for display.
func StatusBar(data StatusBarData, config *StatusBarConfig, width int) string {
	leftParts := make([]string, 0, len(data.KeyBindings))
	rightParts := make([]string, 0, len(data.Providers)+1)

	for _, kb := range data.KeyBindings {
		key := config.KeyStyle.Render(kb.Key)
		description := config.DescStyle.Render(kb.Description)
		leftParts = append(leftParts, fmt.Sprintf("%s %s", key, description))
	}

	for _, provider := range data.Providers {
		var statusIcon string
		var style lipgloss.Style
		if provider.Status == ProviderStatusConnected {
			statusIcon = "●"
			style = config.HealthyStyle
		} else {
			statusIcon = "○"
			style = config.UnhealthyStyle
		}
		rightParts = append(rightParts, style.Render(fmt.Sprintf("%s %s", statusIcon, provider.Name)))
	}

	if !data.LastRefresh.IsZero() {
		now := data.Now
		if now.IsZero() {
			now = time.Now()
		}
		elapsed := now.Sub(data.LastRefresh)
		refreshString := fmt.Sprintf("↻ %ds ago", int(elapsed.Seconds()))
		rightParts = append(rightParts, config.DescStyle.Render(refreshString))
	}

	sep := config.SeparatorStyle.Render(" │ ")
	left := strings.Join(leftParts, sep)
	right := strings.Join(rightParts, sep)

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	spacing := max(1, width-leftLen-rightLen-4)

	bar := fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", spacing), right)

	return config.Style.Width(width).Render(bar)
}

// MiniStatusBar renders a compact single-line status bar.
//
// Takes panelName (string) which is the label shown on the left side.
// Takes itemCount (int) which is the total number of items in the panel.
// Takes selectedIndex (int) which is the zero-based index of the selected item.
// Takes config (*StatusBarConfig) which provides styling for the status bar.
// Takes width (int) which sets the total width of the rendered bar.
//
// Returns string which is the styled status bar ready for display.
func MiniStatusBar(panelName string, itemCount int, selectedIndex int, config *StatusBarConfig, width int) string {
	left := config.DescStyle.Render(panelName)

	var right string
	if itemCount > 0 {
		right = config.DescStyle.Render(fmt.Sprintf("%d/%d", selectedIndex+1, itemCount))
	}

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	spacing := max(1, width-leftLen-rightLen-4)

	bar := fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", spacing), right)

	return config.Style.Width(width).Render(bar)
}

// HelpBar renders a help bar showing key bindings that change based on context.
//
// Takes bindings ([]KeyBinding) which specifies the keys and descriptions to
// show.
// Takes config (*StatusBarConfig) which provides the styling for the bar.
// Takes width (int) which sets the maximum width of the rendered bar.
//
// Returns string which is the styled help bar. If the content is too wide, it
// is cut short and ends with an ellipsis.
func HelpBar(bindings []KeyBinding, config *StatusBarConfig, width int) string {
	parts := make([]string, 0, len(bindings))

	for _, kb := range bindings {
		key := config.KeyStyle.Render(kb.Key)
		description := config.DescStyle.Render(kb.Description)
		parts = append(parts, fmt.Sprintf("%s:%s", key, description))
	}

	sep := config.SeparatorStyle.Render(" ")
	content := strings.Join(parts, sep)

	if lipgloss.Width(content) > width-2 {
		content = ""
		for i, kb := range bindings {
			key := config.KeyStyle.Render(kb.Key)
			description := config.DescStyle.Render(kb.Description)
			part := fmt.Sprintf("%s:%s", key, description)
			if lipgloss.Width(content)+lipgloss.Width(part)+1 > width-statusBarEllipsisPadding {
				content += config.DescStyle.Render(ellipsis)
				break
			}
			if i > 0 {
				content += statusBarBindingSeparator
			}
			content += part
		}
	}

	return config.Style.Width(width).Render(content)
}
